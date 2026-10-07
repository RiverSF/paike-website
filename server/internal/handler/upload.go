package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/config"
	"tutoring_server/pkg/logger"
	"tutoring_server/pkg/path"
)

const maxUploadSize = 5 << 20 // 5MB

// allowImageExt 允许的文件扩展名（已移除 .svg：SVG 可被浏览器渲染执行脚本，存在存储型 XSS 风险）
var allowImageExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}

// allowedMIME 允许的真实 MIME 类型（基于文件头 magic number 检测）
var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// 匿名上传频控（按客户端 IP），单实例有效；多副本部署需改为共享后端（如 Redis）
const (
	uploadRateWindowSec = 60
	uploadRateMax       = 20
)

var (
	uploadMu    sync.Mutex
	uploadCount = map[string]int{}
	uploadStart = map[string]int64{}
)

func uploadRateAllow(ip string) bool {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	now := time.Now().Unix()
	if s, ok := uploadStart[ip]; !ok || now-s >= uploadRateWindowSec {
		uploadStart[ip] = now
		uploadCount[ip] = 0
	}
	uploadCount[ip]++
	return uploadCount[ip] <= uploadRateMax
}

// Upload 上传图片（头像、注册证件等），返回可访问 URL。
// 注册流程需未登录上传证件，故不校验登录态。
func Upload(c *gin.Context) {
	// 匿名上传频控（按客户端 IP），防止未登录接口被滥用耗尽存储
	if !uploadRateAllow(c.ClientIP()) {
		Fail(c, 429, "上传过于频繁，请稍后再试")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "请选择要上传的图片")
		return
	}
	if file.Size > maxUploadSize {
		BadRequest(c, "图片不能超过 5MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowImageExt[ext] {
		BadRequest(c, "仅支持 jpg/png/gif/webp 格式")
		return
	}

	// 真实类型校验：读取文件头 magic number，防止伪造扩展名上传可执行/脚本内容
	src, err := file.Open()
	if err != nil {
		ServerError(c, "上传失败")
		return
	}
	head := make([]byte, 512)
	n, rerr := src.Read(head)
	src.Close()
	if rerr != nil && n == 0 {
		ServerError(c, "上传失败")
		return
	}
	if !allowedMIME[http.DetectContentType(head[:n])] {
		BadRequest(c, "文件类型不合法，仅支持 jpg/png/gif/webp 图片")
		return
	}

	dir := path.Join(config.AppConfig.UploadDir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		logger.Error("mkdir upload dir failed, err=%s", err.Error())
		ServerError(c, "上传失败")
		return
	}

	name := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102150405"), randomHex(6), ext)
	dst := filepath.Join(dir, name)
	if err := saveFile(file, dst); err != nil {
		logger.Error("save upload file failed, err=%s", err.Error())
		ServerError(c, "上传失败")
		return
	}

	OK(c, gin.H{"url": "/uploads/" + name, "name": name})
}

func saveFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "000000"
	}
	return hex.EncodeToString(buf)
}
