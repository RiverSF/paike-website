package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务状态码
const (
	CodeOK           = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message, Data: data})
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, CodeOK, "success", data)
}

func Success(c *gin.Context, message string) {
	JSON(c, CodeOK, message, nil)
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message})
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, CodeBadRequest, message)
}

func ServerError(c *gin.Context, message string) {
	Fail(c, CodeServerError, message)
}

// PageData 统一分页返回结构。
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
