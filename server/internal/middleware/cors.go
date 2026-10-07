package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware 跨域控制：默认拒绝任何跨域请求（不返回 CORS 头，浏览器将直接拦截），
// 仅当请求的 Origin 命中 CORS_ALLOW_ORIGINS 白名单时才放行并允许携带凭证。
//
// ⚠️ 安全约束：严禁在未配置白名单时反射客户端 Origin 或开放 "*"+凭证。
// 旧实现会回显任意 Origin 并允许 credentials，导致任意第三方站点可携带用户凭证
// 跨域调用本站 API，造成账号/订单/会员数据泄露。上线必须通过环境变量配置明确的前端域名。
func CorsMiddleware() gin.HandlerFunc {
	allowOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	allowed := splitOrigins(allowOrigins)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// 同源或浏览器之外的请求（无 Origin）直接放行，不做跨域处理
		if origin == "" {
			c.Next()
			return
		}
		// 未配置白名单：拒绝跨域，不返回任何 CORS 头，浏览器会拦截
		if len(allowed) == 0 {
			c.Next()
			return
		}
		// 命中白名单：放行并允许携带凭证
		if containsOrigin(allowOrigins, origin) {
			setCors(c, origin)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// splitOrigins 解析逗号分隔的允许源，忽略空白项。
func splitOrigins(list string) []string {
	raw := strings.Split(list, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if v := strings.TrimSpace(item); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func setCors(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Vary", "Origin")
	c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization,Accept,X-Requested-With")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Max-Age", "86400")
}

func containsOrigin(list, origin string) bool {
	for _, item := range strings.Split(list, ",") {
		if strings.TrimSpace(item) == origin {
			return true
		}
	}
	return false
}
