package model

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 邀请码字符集（去除易混淆字符 0/O/1/I/l）
const (
	inviteUpper  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	inviteLower  = "abcdefghijkmnpqrstuvwxyz"
	inviteDigit  = "23456789"
	inviteSymbol = "!@#$%&*"
)

var inviteAll = inviteUpper + inviteLower + inviteDigit + inviteSymbol

// InviteCode 邀请码：管理员为申请人（手机号）生成，注册时与手机号关联校验。
// 一个手机号（微信）仅可申请一个邀请码。
type InviteCode struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Code      string `gorm:"size:16;uniqueIndex;not null" json:"code"` // 8 位随机邀请码
	Phone     string `gorm:"size:32;not null" json:"phone"`            // 申请人手机号
	Used      int    `gorm:"not null;default:0" json:"used"`           // 0 未使用 1 已使用
	Invalid   int    `gorm:"not null;default:0" json:"invalid"`        // 0 正常 1 已作废（保留记录用于审计，注册时拒绝使用）
	UsedByID  uint   `gorm:"not null" json:"usedByUserId"`             // 使用该邀请码注册的用户 ID
	UsedAt    int64  `gorm:"not null" json:"usedAt"`
	CreatedAt int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
}

func (InviteCode) TableName() string { return "invite_code" }

type InviteCodeModel struct{}

func NewInviteCodeModel() *InviteCodeModel { return &InviteCodeModel{} }

var ErrInviteNotFound = errors.New("邀请码不存在")

// CodeOfUser 返回该用户注册时使用的邀请码（用于生成订单编号前缀），不存在返回空串。
func (m *InviteCodeModel) CodeOfUser(userID uint) string {
	if userID == 0 {
		return ""
	}
	var code string
	if err := db.Model(&InviteCode{}).
		Where("used_by_id = ?", userID).
		Order("id ASC").Limit(1).Pluck("code", &code).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(code)
}

func randomChar(set string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, err
	}
	return set[n.Int64()], nil
}

// GenerateInviteCode 生成 8 位邀请码：包含大写、小写、数字、特殊字符，每类至少 1 个。
func GenerateInviteCode() (string, error) {
	sets := []string{inviteUpper, inviteLower, inviteDigit, inviteSymbol}
	chars := make([]byte, 0, 8)
	for _, s := range sets {
		c, err := randomChar(s)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}
	for i := len(chars); i < 8; i++ {
		c, err := randomChar(inviteAll)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}
	// Fisher–Yates 洗牌，避免固定每类靠前
	for i := len(chars) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := n.Int64()
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars), nil
}

func (m *InviteCodeModel) GetByID(id uint) (*InviteCode, error) {
	var ic InviteCode
	if err := db.Where("id = ?", id).First(&ic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}
	return &ic, nil
}

// GetByCode 按邀请码查询。
// 大小写不敏感：邀请码含大小写字母与特殊字符，人工转述/微信传播时极易出错，统一 upper 比较。
func (m *InviteCodeModel) GetByCode(code string) (*InviteCode, error) {
	var ic InviteCode
	if err := db.Where("upper(code) = upper(?)", strings.TrimSpace(code)).First(&ic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}
	return &ic, nil
}

func (m *InviteCodeModel) GetByPhone(phone string) (*InviteCode, error) {
	var ic InviteCode
	if err := db.Where("phone = ?", strings.TrimSpace(phone)).First(&ic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, err
	}
	return &ic, nil
}

func (m *InviteCodeModel) Create(ic *InviteCode) error {
	return db.Create(ic).Error
}

func (m *InviteCodeModel) Delete(id uint) error {
	return db.Where("id = ?", id).Delete(&InviteCode{}).Error
}

// Invalidate 作废邀请码：保留记录但标记为不可用（与删除的区别：删除会移除记录）。
// 已使用 / 未使用均可作废，注册时该码将被拒绝。
func (m *InviteCodeModel) Invalidate(id uint) error {
	return db.Model(&InviteCode{}).Where("id = ?", id).Update("invalid", 1).Error
}

// IsUsable 判断邀请码当前是否可用于注册：存在、未使用且未作废。
func (ic *InviteCode) IsUsable() bool {
	return ic.Used != 1 && ic.Invalid != 1
}

// MarkUsed 标记邀请码已被某用户使用（一次性）。
// 返回 ok=false 表示并发下已被他人抢先核销（UPDATE 命中 0 行），调用方必须据此中止注册，
// 否则会出现「一码多注册」——这是邀请码作为准入凭证的底线。
func (m *InviteCodeModel) MarkUsed(code string, userID uint) (bool, error) {
	return markUsed(db, code, userID)
}

// MarkUsedTx 事务版 MarkUsed，供「创建用户 + 核销邀请码」放在同一事务内调用。
func (m *InviteCodeModel) MarkUsedTx(tx *gorm.DB, code string, userID uint) (bool, error) {
	return markUsed(tx, code, userID)
}

func markUsed(tx *gorm.DB, code string, userID uint) (bool, error) {
	res := tx.Model(&InviteCode{}).
		Where("upper(code) = upper(?) AND used = 0", strings.TrimSpace(code)).
		Updates(map[string]interface{}{"used": 1, "used_by_id": userID, "used_at": time.Now().Unix()})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// UpdatePhoneByUser 同步更新该用户名下邀请码记录上的手机号：
// 邀请码与注册手机号强关联（注册时校验、后台按手机号检索），用户变更手机号后需保持一致。
func (m *InviteCodeModel) UpdatePhoneByUser(userID uint, phone string) error {
	if userID == 0 || phone == "" {
		return nil
	}
	return db.Model(&InviteCode{}).Where("used_by_id = ?", userID).Update("phone", phone).Error
}

// CodeMapByUserIDs 返回「用户 ID -> 其注册时使用的邀请码」，用于后台列表展示。
func (m *InviteCodeModel) CodeMapByUserIDs(ids []uint) (map[uint]string, error) {
	res := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return res, nil
	}
	var list []InviteCode
	if err := db.Where("used_by_id IN ?", ids).Find(&list).Error; err != nil {
		return res, err
	}
	for _, ic := range list {
		res[ic.UsedByID] = ic.Code
	}
	return res, nil
}

// List 分页查询邀请码，keyword 可匹配邀请码或手机号。
func (m *InviteCodeModel) List(page, pageSize int, keyword string) ([]InviteCode, int64, error) {
	tx := db.Model(&InviteCode{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("code ILIKE ? OR phone ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var list []InviteCode
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
