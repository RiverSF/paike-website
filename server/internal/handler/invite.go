package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/model"
	"tutoring_server/pkg/logger"
)

// AdminCreateInviteCode 管理员为申请人（手机号）生成邀请码：一个手机号仅可申请一个。
func AdminCreateInviteCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请填写手机号")
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if !phoneRegexp.MatchString(phone) {
		BadRequest(c, "请填写正确的 11 位手机号")
		return
	}

	im := model.NewInviteCodeModel()
	if existing, err := im.GetByPhone(phone); err == nil && existing != nil {
		BadRequest(c, "该手机号已申请过邀请码，请勿重复申请")
		return
	}

	code, err := model.GenerateInviteCode()
	if err != nil {
		ServerError(c, "生成邀请码失败")
		return
	}
	// 极小概率冲突时重试，保证唯一
	for i := 0; i < 5; i++ {
		if _, err := im.GetByCode(code); err == model.ErrInviteNotFound {
			break
		}
		if code, err = model.GenerateInviteCode(); err != nil {
			ServerError(c, "生成邀请码失败")
			return
		}
	}

	ic := &model.InviteCode{Code: code, Phone: phone}
	if err := im.Create(ic); err != nil {
		logger.Error("create invite code failed, err=%s", err.Error())
		ServerError(c, "创建邀请码失败")
		return
	}
	OK(c, ic)
}

// AdminListInviteCodes 后台邀请码列表（分页，可按邀请码或手机号搜索）。
func AdminListInviteCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := model.NewInviteCodeModel().List(page, pageSize, c.Query("keyword"))
	if err != nil {
		logger.Error("list invite codes failed, err=%s", err.Error())
		ServerError(c, "查询邀请码失败")
		return
	}
	OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

// AdminInvalidateInviteCode 作废邀请码：保留记录但标记为不可用（与删除的区别：删除会移除记录）。
// 已使用 / 未使用均可作废，注册时该码将被拒绝。
func AdminInvalidateInviteCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "邀请码 ID 有误")
		return
	}
	im := model.NewInviteCodeModel()
	if _, err := im.GetByID(uint(id)); err != nil {
		Fail(c, CodeNotFound, "邀请码不存在")
		return
	}
	if err := im.Invalidate(uint(id)); err != nil {
		logger.Error("invalidate invite code failed, err=%s", err.Error())
		ServerError(c, "作废失败")
		return
	}
	Success(c, "已作废")
}

// AdminDeleteInviteCode 删除邀请码（仅未使用可删除）。
func AdminDeleteInviteCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "邀请码 ID 有误")
		return
	}
	im := model.NewInviteCodeModel()
	ic, err := im.GetByID(uint(id))
	if err != nil {
		Fail(c, CodeNotFound, "邀请码不存在")
		return
	}
	if ic.Used == 1 {
		BadRequest(c, "该邀请码已被使用，无法删除")
		return
	}
	if err := im.Delete(uint(id)); err != nil {
		logger.Error("delete invite code failed, err=%s", err.Error())
		ServerError(c, "删除失败")
		return
	}
	Success(c, "已删除")
}
