package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/config"
	"tutoring_server/internal/handler"
	"tutoring_server/internal/middleware"
	"tutoring_server/pkg/logger"
	"tutoring_server/pkg/path"
)

// New 创建 HTTP 引擎：中间件、API 路由、上传目录与前端静态资源。
func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.LoggerMiddleware())
	// 全局请求体上限 10MB，防大请求 DoS（与 nginx/ingress 的 body size 限制一致）
	r.Use(func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
		}
		c.Next()
	})

	registerAPI(r)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// 上传文件静态目录
	r.Static("/uploads", path.Join(config.AppConfig.UploadDir))

	registerSPA(r)
	return r
}

func registerAPI(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 公开接口
		api.POST("/user/register", handler.Register)
		api.POST("/user/login", handler.Login)
		api.GET("/price", handler.GetPrice)                               // 公开：当前会员价格
		api.GET("/price/schedules/pending", handler.PublicPriceSchedules) // 公开：待生效价格预约（首页预告）
		api.POST("/upload", handler.Upload)                               // 公开：注册时证件上传不强制登录

		// 登录后可访问
		auth := api.Group("", middleware.AuthRequired())
		{
			auth.GET("/user/profile", handler.Profile)
			auth.PUT("/user/profile", handler.UpdateProfile)
			auth.PUT("/user/password", handler.ChangePassword)
			auth.POST("/user/member/activate", handler.ActivateMember)
			auth.GET("/user/payments", handler.MyPayments)
			auth.POST("/feedback", handler.FeedbackCreate)
			auth.GET("/feedback/mine", handler.MyFeedback)
			auth.GET("/user/messages", handler.UserMessages)
			auth.GET("/user/messages/unread", handler.UserMessageUnread)
			auth.PUT("/user/messages/:id/read", handler.UserMessageRead)

			// 课表 / 课程数据查看：登录即可访问（试用期结束后仍可查看已录入的数据），
			// 新增与调整操作在下方 member 组内校验会员有效期
			auth.GET("/orders", handler.OrderList)
			auth.GET("/orders/options", handler.OrderOptions)
			auth.GET("/schedule", handler.Schedule)
			auth.GET("/lessons/exception", handler.LessonExceptionList)

			// 站点拥有者 / 管理员
			staff := auth.Group("/admin", middleware.StaffRequired())
			{
				staff.GET("/users", handler.AdminUsers)
				staff.POST("/users", handler.AdminCreateAdmin) // 添加管理员（仅站长）
				staff.GET("/stats", handler.AdminStats)
				staff.PUT("/users/:id/freeze", handler.AdminFreeze)
				staff.PUT("/users/:id/member", handler.AdminRenew)
				staff.PUT("/users/:id/role", handler.AdminSetRole)
				staff.DELETE("/users/:id", handler.AdminDeleteUser)                  // 删除账号（仅站长）
				staff.PUT("/users/:id/password", handler.AdminResetPassword) // 为普通用户重置登录密码
				staff.POST("/messages", handler.AdminSendMessage)           // 发送站内信（指定用户或全员）
				staff.GET("/feedbacks", handler.AdminFeedbacks)
				staff.PUT("/feedbacks/:id", handler.AdminFeedbackReply)
				staff.PUT("/price", handler.AdminUpdatePrice)                                // 调整会员价格（整体，兼容）
				staff.POST("/price/card", handler.AdminUpdatePriceCard)                      // 按卡片调整价格 / 折扣（支持生效时间）
				staff.GET("/price/schedules", handler.AdminListPriceSchedules)               // 待生效价格预约列表
				staff.DELETE("/price/schedules/:id", handler.AdminCancelPriceSchedule)       // 撤销待生效预约
				staff.GET("/price/impact", handler.AdminPriceImpact)                         // 价格通知预计触达人数
				staff.GET("/price/history", handler.AdminPriceHistory)                       // 价格变更历史
				staff.POST("/price/history/:id/rollback", handler.AdminRollbackPrice)        // 按历史记录回滚
				staff.GET("/invite-codes", handler.AdminListInviteCodes)                     // 邀请码列表
				staff.POST("/invite-codes", handler.AdminCreateInviteCode)                   // 生成邀请码
				staff.DELETE("/invite-codes/:id", handler.AdminDeleteInviteCode)             // 删除未使用邀请码
				staff.PUT("/invite-codes/:id/invalidate", handler.AdminInvalidateInviteCode) // 作废邀请码（保留记录）
			}

			// 会员未过期才能执行写操作（查看类接口已在 auth 组放开）
			member := auth.Group("", middleware.MemberRequired())
			{
				member.POST("/orders", handler.OrderCreate)
				member.PUT("/orders/:id", handler.OrderUpdate)
				member.PUT("/orders/:id/status", handler.OrderUpdateStatus)
				member.DELETE("/orders/:id", handler.OrderDelete)
				member.GET("/orders/dashboard", handler.OrderDashboard)
				member.DELETE("/lessons/:id", handler.LessonDelete)                    // 删除一节已物化历史课次（老师未上课等）
				member.POST("/lessons/exception", handler.LessonExceptionCreate)       // 单次调课 / 停课 / 加课
				member.DELETE("/lessons/exception/:id", handler.LessonExceptionDelete) // 撤销调整
			}
		}
	}
}

// registerSPA 单机部署时由后端直接托管前端构建产物（web/dist）。
// docker compose 部署时 STATIC_DIR 为空，前端由 nginx 托管。
func registerSPA(r *gin.Engine) {
	dir := strings.TrimSpace(config.AppConfig.StaticDir)
	// docker compose 部署时前端由 nginx 托管，此处设置为 none 即可关闭
	if dir == "" || strings.EqualFold(dir, "none") {
		return
	}
	if !filepathIsDir(dir) {
		logger.Info("static dir not found, skip serving frontend: %s", dir)
		return
	}

	absDir := dir
	if !filepath.IsAbs(absDir) {
		absDir = path.Join(absDir)
	}

	r.Static("/assets", filepath.Join(absDir, "assets"))
	r.StaticFile("/favicon.svg", filepath.Join(absDir, "favicon.svg"))
	r.StaticFile("/favicon.ico", filepath.Join(absDir, "favicon.ico"))

	index := filepath.Join(absDir, "index.html")
	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		if strings.HasPrefix(urlPath, "/api") || strings.HasPrefix(urlPath, "/uploads") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}

		// 优先命中前端根目录下的静态文件（logo.png、qrcode/*、avatar-default.png 等）
		target := filepath.Join(absDir, filepath.Clean(strings.TrimPrefix(urlPath, "/")))
		if strings.HasPrefix(target, absDir) {
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				c.File(target)
				return
			}
		}

		// 其余路径回落到 index.html（Vue hash 路由）
		c.File(index)
	})
}

func filepathIsDir(dir string) bool {
	if filepath.IsAbs(dir) {
		info, err := os.Stat(dir)
		return err == nil && info.IsDir()
	}
	info, err := os.Stat(path.Join(dir))
	return err == nil && info.IsDir()
}
