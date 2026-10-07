package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders 统一设置基础安全响应头：
//   - X-Content-Type-Options: nosniff  防止 MIME 嗅探导致上传文件被当作可执行脚本执行
//   - X-Frame-Options: DENY            禁止被 iframe 嵌套，防范点击劫持
//   - Referrer-Policy: no-referrer     减少 Referer 头泄露来源路径
//   - Content-Security-Policy: frame-ancestors 'none'  进一步禁止跨域嵌套
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "frame-ancestors 'none'")
		c.Next()
	}
}
