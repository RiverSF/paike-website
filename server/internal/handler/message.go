package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/logger"
)

// UserMessages 我的站内信列表（含未读数量）。
func UserMessages(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := model.NewMessageModel().ListForUser(user.ID, page, pageSize)
	if err != nil {
		ServerError(c, "查询站内信失败")
		return
	}
	unread, err := model.NewMessageModel().UnreadCount(user.ID)
	if err != nil {
		unread = 0
	}
	OK(c, gin.H{"list": list, "total": total, "unread": unread})
}

// UserMessageUnread 未读站内信数量（用于头部角标）。
func UserMessageUnread(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	n, err := model.NewMessageModel().UnreadCount(user.ID)
	if err != nil {
		ServerError(c, "查询失败")
		return
	}
	OK(c, gin.H{"count": n})
}

// UserMessageRead 标记单条站内信已读。
func UserMessageRead(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, CodeBadRequest, "参数错误")
		return
	}
	if err := model.NewMessageModel().MarkRead(uint(id), user.ID); err != nil {
		ServerError(c, "标记已读失败")
		return
	}
	OK(c, gin.H{"ok": true})
}

// AdminSendMessage 发送站内信：userId>0 指定用户，userId=0 全员广播。
func AdminSendMessage(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	var in struct {
		UserID  uint   `json:"userId"` // 0 = 全员广播
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, CodeBadRequest, "参数有误："+err.Error())
		return
	}
	if in.Type == "" {
		in.Type = model.MessageTypeSystem
	}
	if in.Type != model.MessageTypeSystem && in.Type != model.MessageTypePromo {
		Fail(c, CodeBadRequest, "站内信类型不合法")
		return
	}
	msg := &model.Message{
		SenderID:   user.ID,
		SenderName: user.Username,
		Title:      in.Title,
		Content:    in.Content,
		Type:       in.Type,
		CreatedAt:  time.Now().Unix(),
	}
	if err := model.NewMessageModel().Send(msg, in.UserID); err != nil {
		logger.Error("send message failed, err=%s", err.Error())
		ServerError(c, "发送失败")
		return
	}
	OK(c, gin.H{"ok": true})
}
