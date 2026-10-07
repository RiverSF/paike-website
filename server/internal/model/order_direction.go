package model

import "tutoring_server/pkg/logger"

// BackfillOrderDirections 存量订单方向回填（服务启动时执行一次，幂等）：
// 早期订单没有「收入 / 开支」字段，按所属账号的注册身份推导默认方向后写入，
// 使历史数据与新增订单口径一致（老师 / 机构=收入，家长 / 个人=开支）。
func BackfillOrderDirections() {
	// 家长 / 个人 → 开支
	if err := db.Exec(`
		UPDATE "order" SET direction = ?
		 WHERE direction = ''
		   AND user_id IN (SELECT id FROM "user" WHERE user_role IN (?, ?))`,
		DirectionExpense, UserRoleParent, UserRolePersonal,
	).Error; err != nil {
		logger.Error("backfill order direction (expense) failed, err=%s", err.Error())
		return
	}
	// 其余（老师 / 机构，含身份缺失的历史账号）→ 收入
	if err := db.Exec(`
		UPDATE "order" SET direction = ?
		 WHERE direction = ''`, DirectionIncome,
	).Error; err != nil {
		logger.Error("backfill order direction (income) failed, err=%s", err.Error())
	}
}
