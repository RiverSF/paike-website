package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/config"
	"tutoring_server/internal/handler"
	"tutoring_server/internal/model"
	"tutoring_server/internal/router"
	"tutoring_server/pkg/logger"
)

func main() {
	// 容器部署推荐 LOG_STDOUT=1：日志直接输出到标准输出，不写文件
	logger.DisableFile = os.Getenv("LOG_STDOUT") == "1"

	if err := logger.Init(); err != nil {
		log.Fatalf("fail to init logger, err=%s", err.Error())
	}
	if err := config.Init(); err != nil {
		log.Fatalf("fail to init config, err=%s", err.Error())
	}
	// 生产环境强制使用强 JWT 密钥，杜绝默认/弱密钥上线导致令牌被伪造、系统被接管。
	// Docker/k8s 部署请通过环境变量 JWT_SECRET（或 k8s Secret）注入高熵随机值（>=32 字节）。
	if config.AppConfig.RunMode == "release" {
		if config.JwtConfig.Secret == "" || isWeakJwtSecret(config.JwtConfig.Secret) {
			log.Fatalf("安全启动被拒绝：release 模式下 JWT_SECRET 必须为高熵随机值（>=32 字节），禁止沿用默认或弱密钥。请通过环境变量 JWT_SECRET 注入。")
		}
	}
	if err := model.Init(); err != nil {
		log.Fatalf("fail to init db, err=%s", err.Error())
	}
	defer model.Close()
	defer logger.Close()

	// 冷启动初始化站点拥有者（无 owner 时由 INIT_OWNER_* 环境变量自动创建首个管理员）
	model.SeedOwner()

	// 后台定时任务：每日扫描即将到期的用户（会员 7 天），自动发送站内信提醒
	startExpiringNotifier()
	// 后台定时任务：每日按注册天数给免费试用中的账号推送转化站内信（激活 / 价值回顾 / 预警 / 限时优惠）
	startTrialNudger()
	// 后台定时任务：每日把订单的历史课次物化到 lesson 表（与订单配置解耦，详见 lesson_materialize.go）
	startLessonMaterializer()
	// 后台定时任务：每分钟扫描到点的价格卡片调整，应用并通知已登录用户
	startPriceScheduler()
	// 后台定时任务：每 10 分钟扫描课表，距开课 2 小时内的课次给对应账号发送站内信提醒
	startLessonReminder()

	gin.SetMode(config.AppConfig.RunMode)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.ServerConfig.ServerPort),
		Handler:        router.New(),
		ReadTimeout:    time.Duration(config.ServerConfig.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.ServerConfig.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		logger.Info("listen and serve on 0.0.0.0:%d, runMode=%s", config.ServerConfig.ServerPort, config.AppConfig.RunMode)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("fail to listenAndServe, err=%s", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed, err=%s", err.Error())
	}
}

// weakJwtSecrets 已知的弱/默认 JWT 密钥集合，release 模式下命中即拒绝启动。
var weakJwtSecrets = map[string]bool{
	"":                          true,
	"tutoring-website-secret":   true,
	"please-change-this-secret": true,
}

// isWeakJwtSecret 判断给定密钥是否过弱（命中默认集合或长度不足）。
func isWeakJwtSecret(s string) bool {
	if weakJwtSecrets[s] {
		return true
	}
	return len(s) < 32
}

// startExpiringNotifier 启动到期前站内信提醒定时任务（每日 09:00 执行一次）：
// 会员有效期 7 天内到期 → system 提醒。
func startExpiringNotifier() {
	const memberNotifyDays = 7
	run := func() {
		if n, err := model.NotifyExpiringMembers(memberNotifyDays); err != nil {
			logger.Error("notify expiring members failed, err=%s", err.Error())
		} else if n > 0 {
			logger.Info("notified %d expiring member(s)", n)
		}
	}
	// 启动时先执行一次（稍延迟，确保 DB 已就绪）
	go func() {
		time.Sleep(3 * time.Second)
		run()
	}()
	// 每日 09:00 定时执行
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			select {
			case <-time.After(next.Sub(now)):
				run()
			}
		}
	}()
}

// startTrialNudger 启动试用期转化推送定时任务（每日 10:00 执行一次）：
// 按注册天数给免费试用中的账号推送激活引导、价值回顾、到期预警与限时优惠站内信。
func startTrialNudger() {
	run := func() {
		handler.RunTrialNudges()
	}
	// 启动时先执行一次（稍延迟，确保 DB 已就绪）
	go func() {
		time.Sleep(6 * time.Second)
		run()
	}()
	// 每日 10:00 定时执行
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			select {
			case <-time.After(next.Sub(now)):
				run()
			}
		}
	}()
}

// startLessonMaterializer 启动历史课次物化定时任务（每日 01:00 执行一次）：
// 把各订单截至昨日的历史课次落库到 lesson 表，作为与订单配置解耦的冻结快照。
// 订单保存时也会即时兜底物化，此处为主力、保证即便无人操作也会持续补齐历史。
func startLessonMaterializer() {
	run := func() {
		handler.MaterializeHistoryUntilYesterday()
	}
	// 启动时先执行一次（稍延迟，确保 DB 已就绪）
	go func() {
		time.Sleep(5 * time.Second)
		handler.RepairAdjustedLessons() // 修复存量「原课次与调整后课时并存」的数据
		handler.RepairLessonIncomes()   // 修复存量「订单后补价格导致课时费为 0」的冻结快照
		model.BackfillOrderDirections() // 存量订单按注册身份回填「收入 / 开支」方向
		run()
	}()
	// 每日 01:00 定时执行
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			select {
			case <-time.After(next.Sub(now)):
				run()
			}
		}
	}()
}

// startPriceScheduler 启动价格预约调度：每分钟扫描已到生效时间的价格卡片调整，
// 写入当前价格配置、记录变更历史，并向对应账号发送站内信（仅调整的卡片通知，与登录状态无关）。
func startPriceScheduler() {
	run := func() {
		due, err := model.NewPriceScheduleModel().PendingDue(time.Now().Unix())
		if err != nil {
			logger.Error("load due price schedules failed, err=%s", err.Error())
			return
		}
		for _, sch := range due {
			var in model.PriceCardInput
			if err := json.Unmarshal([]byte(sch.Payload), &in); err != nil {
				logger.Error("parse price schedule %d failed, err=%s", sch.ID, err.Error())
				_ = model.NewPriceScheduleModel().MarkApplied(sch.ID) // 解析失败直接标记，避免反复重试
				continue
			}
			// 未启用价卡的无效预约（当前与目标均为关闭）：无实际效果，直接标记，跳过应用与通知
			if cur, cerr := model.NewPriceModel().Get(); cerr == nil && !model.PriceScheduleMeaningful(in, cur) {
				logger.Info("skip meaningless price schedule %d, card=%s disabled", sch.ID, sch.Card)
				_ = model.NewPriceScheduleModel().MarkApplied(sch.ID)
				continue
			}
			if _, err := model.NewPriceScheduleModel().ApplyCardWithHistory(in, "schedule", 0, "系统调度"); err != nil {
				logger.Error("apply price schedule %d failed, err=%s", sch.ID, err.Error())
				continue
			}
			if err := model.NotifyPriceCard(in); err != nil {
				logger.Error("notify price schedule %d failed, err=%s", sch.ID, err.Error())
			}
			_ = model.NewPriceScheduleModel().MarkApplied(sch.ID)
		}
	}
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

// startLessonReminder 启动开课提醒定时任务：每 10 分钟扫描一次课表，
// 对「距开始时间 2 小时内」的课次给对应账号发送站内信（同一节课只提醒一次）。
// 未来课次按订单周期实时展开，因此刚新增 / 调整的课程也会被纳入提醒。
func startLessonReminder() {
	run := func() {
		handler.NotifyUpcomingLessons()
	}
	// 启动时先执行一次（稍延迟，确保 DB 已就绪）
	go func() {
		time.Sleep(8 * time.Second)
		run()
	}()
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}
