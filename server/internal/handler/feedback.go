package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/logger"
)

// 反馈提交限制
const (
	feedbackMaxLen      = 200  // 最多 200 字
	feedbackIntervalSec = 3600 // 同一用户每小时只能提交一次
)

// 违禁词（命中则拒绝提交），可按需扩展
var bannedWords = []string{
	"赌博", "博彩", "刷单", "返利", "兼职日结", "贷款", "裸聊", "色情", "约炮",
	"代开发票", "发票", "私彩", "洗钱", "银行卡出售", "刷赞", "外挂",
}

type feedbackReq struct {
	Content string `json:"content" binding:"required"`
	Contact string `json:"contact"`
}

// MyFeedback 我提交的反馈（仅本人可见，需登录）。公开列表已移除，未登录/登录用户均不可查看他人反馈。
func MyFeedback(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	list, err := model.NewFeedbackModel().ListByUser(user.ID)
	if err != nil {
		logger.Error("list my feedback failed, err=%s", err.Error())
		ServerError(c, "查询反馈失败")
		return
	}

	nextAt := int64(0)
	if len(list) > 0 {
		nextAt = list[0].CreatedAt + feedbackIntervalSec
	}
	OK(c, gin.H{"list": list, "nextAllowedAt": nextAt})
}

// FeedbackCreate 提交反馈/问题（字数、频率、违禁词校验）。
func FeedbackCreate(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录后再提交问题")
		return
	}

	var req feedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请填写反馈内容")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		BadRequest(c, "请填写反馈内容")
		return
	}
	if runeLen(content) > feedbackMaxLen {
		BadRequest(c, "反馈内容最多 200 字")
		return
	}
	if word := hitBannedWord(content); word != "" {
		BadRequest(c, "内容包含不允许发布的词汇，请修改后再提交")
		return
	}

	// 频率限制：每个用户每小时只能提交一次
	last, err := model.NewFeedbackModel().LastByUser(user.ID)
	if err == nil && last != nil {
		if left := last.CreatedAt + feedbackIntervalSec - time.Now().Unix(); left > 0 {
			minutes := (left + 59) / 60
			Fail(c, CodeBadRequest, "提交过于频繁，请 "+itoa(int(minutes))+" 分钟后再试")
			return
		}
	}

	f := &model.Feedback{
		UserID:   user.ID,
		Username: user.Username,
		Content:  content,
		Contact:  strings.TrimSpace(req.Contact),
		Status:   model.FeedbackPending,
	}
	if err := model.NewFeedbackModel().Create(f); err != nil {
		logger.Error("create feedback failed, err=%s", err.Error())
		ServerError(c, "提交失败，请稍后重试")
		return
	}
	OK(c, f)
}

func runeLen(s string) int {
	return len([]rune(s))
}

func hitBannedWord(content string) string {
	lower := strings.ToLower(content)
	for _, w := range bannedWords {
		if strings.Contains(lower, strings.ToLower(w)) {
			return w
		}
	}
	return ""
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
