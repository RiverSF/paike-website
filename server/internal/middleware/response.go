package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// abortJSON 中间件里中断请求并返回统一结构。
func abortJSON(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": code, "message": message})
}
