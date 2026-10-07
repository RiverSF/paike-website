package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 学生身份申请状态
const (
	StudentAppPending  = "pending"  // 待审核
	StudentAppApproved = "approved" // 已通过
	StudentAppRejected = "rejected" // 已驳回
)

// StudentAppStatusName 申请状态中文名
var StudentAppStatusName = map[string]string{
	StudentAppPending:  "待审核",
	StudentAppApproved: "已通过",
	StudentAppRejected: "未通过",
}

// StudentApplication 大学生家教身份申请记录。
// 注册时选择「大学生家教」，或用户重新发起申请时，各生成一条记录，便于留档与追溯。
type StudentApplication struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	UserID         uint   `gorm:"not null" json:"userId"`
	Username       string `gorm:"size:64;not null" json:"username"`
	Phone          string `gorm:"size:32;not null" json:"phone"`
	InviteCode     string `gorm:"size:16;not null" json:"inviteCode"`
	StudentCardURL string `gorm:"size:255;not null" json:"studentCardUrl"`
	IdCardURL      string `gorm:"size:255;not null" json:"idCardUrl"`
	Status         string `gorm:"size:16;not null;default:pending" json:"status"` // pending / approved / rejected
	ExpireAt       int64  `gorm:"not null" json:"expireAt"`                       // 审核通过时设置的毕业时间（秒级时间戳）
	ReviewerID     uint   `gorm:"not null" json:"reviewerId"`
	ReviewerName   string `gorm:"size:32;not null" json:"reviewerName"`
	Remark         string `gorm:"size:255;not null" json:"remark"`
	CreatedAt      int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
	ReviewedAt     int64  `gorm:"not null" json:"reviewedAt"`
}

func (StudentApplication) TableName() string { return "student_application" }

type StudentApplicationModel struct{}

func NewStudentApplicationModel() *StudentApplicationModel { return &StudentApplicationModel{} }

func (m *StudentApplicationModel) Create(app *StudentApplication) error {
	if app.Status == "" {
		app.Status = StudentAppPending
	}
	return db.Create(app).Error
}

// CreateTx 事务版 Create，供注册流程与用户创建、邀请码核销放在同一事务内调用。
func (m *StudentApplicationModel) CreateTx(tx *gorm.DB, app *StudentApplication) error {
	if app.Status == "" {
		app.Status = StudentAppPending
	}
	return tx.Create(app).Error
}

func (m *StudentApplicationModel) GetByID(id uint) (*StudentApplication, error) {
	var app StudentApplication
	if err := db.First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// LatestPendingByUser 用户最近一条待审核申请；无则返回 nil。
func (m *StudentApplicationModel) LatestPendingByUser(userID uint) (*StudentApplication, error) {
	var app StudentApplication
	err := db.Where("user_id = ? AND status = ?", userID, StudentAppPending).
		Order("id DESC").First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// LatestByUser 用户最近一条申请记录（任意状态）；无则返回 nil。
func (m *StudentApplicationModel) LatestByUser(userID uint) (*StudentApplication, error) {
	var app StudentApplication
	err := db.Where("user_id = ?", userID).Order("id DESC").First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// StudentAppQuery 申请记录查询条件。
type StudentAppQuery struct {
	Keyword  string // 匹配用户名 / 手机号 / 邀请码
	Status   string
	Page     int
	PageSize int
}

// List 分页查询申请记录（按时间倒序）。
// 只返回普通用户的申请：站长 / 管理员账号不参与学生身份认证，
// 历史遗留的管理账号记录不应出现在审核队列与待审核计数中
// （子查询形式保证 user 记录缺失时也不丢数据）。
func (m *StudentApplicationModel) List(q StudentAppQuery) ([]StudentApplication, int64, error) {
	tx := db.Model(&StudentApplication{}).
		Where(`user_id NOT IN (SELECT id FROM "user" WHERE role <> ?)`, RoleUser)
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("username ILIKE ? OR phone ILIKE ? OR invite_code ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	var list []StudentApplication
	// 排序：待审核置顶（需尽快处理）→ 未通过 → 已通过；同状态内按申请时间倒序
	err := tx.Order("CASE status WHEN 'pending' THEN 0 WHEN 'rejected' THEN 1 WHEN 'approved' THEN 2 ELSE 3 END, id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// ReviewAllPendingByUser 收口该用户全部待审核记录（审核通过 / 驳回时调用）。
// 历史数据下同一用户可能存在多条 pending（早期补建逻辑非幂等导致），
// 若只处理最新一条，其余 pending 会永久挂在审核队列里无法收尾。
func (m *StudentApplicationModel) ReviewAllPendingByUser(userID uint, status string, expireAt int64, reviewerID uint, reviewerName, remark string) error {
	return db.Model(&StudentApplication{}).
		Where("user_id = ? AND status = ?", userID, StudentAppPending).
		Updates(map[string]interface{}{
			"status":        status,
			"expire_at":     expireAt,
			"reviewer_id":   reviewerID,
			"reviewer_name": reviewerName,
			"remark":        remark,
			"reviewed_at":   time.Now().Unix(),
		}).Error
}

// BackfillStudentApplications 为历史「待审核学生」用户补建申请记录。
// 同时把历史学生用户的注册身份补为 student，保证「申请/重新申请」入口可见。
//
// 幂等性要求（重要）：本函数在每次服务启动的 AutoMigrate 中执行，必须保证
// 「同一用户最多补建一次」。因此判重条件是「该用户不存在任何申请记录」，
// 而不是「不存在 pending 记录」——后者会在「补建 pending → 管理员驳回 → 重启」
// 的循环中反复新增记录，导致同一账号堆积多条申请。
// 同时排除站长 / 管理员：管理账号不参与学生身份认证，不应出现在审核队列。
func BackfillStudentApplications() error {
	if err := db.Exec(`UPDATE "user" SET reg_teacher_type = 'student' WHERE teacher_type = ? AND role = ? AND (reg_teacher_type IS NULL OR reg_teacher_type = '')`, TeacherTypeStudent, RoleUser).Error; err != nil {
		return err
	}
	var users []User
	if err := db.Where("teacher_type = ? AND verified = ? AND role = ?", TeacherTypeStudent, 0, RoleUser).Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		// 任意状态的记录存在即视为「已留档」，不再补建
		var cnt int64
		if err := db.Model(&StudentApplication{}).
			Where("user_id = ?", u.ID).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			continue
		}
		code := ""
		if m, err := NewInviteCodeModel().CodeMapByUserIDs([]uint{u.ID}); err == nil {
			code = m[u.ID]
		}
		app := &StudentApplication{
			UserID:         u.ID,
			Username:       u.Username,
			Phone:          u.Phone,
			InviteCode:     code,
			StudentCardURL: u.StudentCardURL,
			IdCardURL:      u.IdCardURL,
			Status:         StudentAppPending,
		}
		if err := db.Create(app).Error; err != nil {
			return err
		}
	}
	return nil
}
