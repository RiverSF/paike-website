<template>
  <div class="page">
    <section class="card">
      <div class="card-title">
        <span>管理中心
          <span class="sub">
            {{ userStore.isOwner ? '站长：可查看全部账号' : '管理员：可查看普通会员账号' }}
          </span>
        </span>
      </div>

      <!-- 二级导航工具条：菜单排序 -->
      <div class="tab-tools">
        <el-popover placement="bottom-end" :width="300" trigger="click">
          <template #reference>
            <el-button link type="primary" size="small">菜单排序</el-button>
          </template>
          <div class="col-set">
            <div class="col-set-title">拖动 / 点箭头调整页签顺序（总览固定置顶）</div>
            <ReorderList :model-value="menuEditItems" @update:model-value="onMenuReorder" />
            <div class="col-set-foot">
              <el-button text size="small" @click="resetOrder">恢复默认</el-button>
            </div>
          </div>
        </el-popover>
      </div>

      <el-tabs v-model="activeTab" class="admin-tabs" @tab-change="onTabChange">
        <el-tab-pane v-for="t in orderedTabs" :key="t.name" :name="t.name" :lazy="!!t.lazy">
          <template #label>
            <span
              class="tab-label"
              :class="{
                dragging: dragName === t.name,
                over: dragName && dragName !== t.name && hoverName === t.name
              }"
              draggable="true"
              @dragstart="onDragStart(t.name)"
              @dragover.prevent="hoverName = t.name"
              @drop.prevent="onDrop(t.name)"
              @dragend="onDragEnd"
            >
              {{ t.label }}
              <!-- 待办角标：用普通元素渲染，避免组件内部定位/过渡导致不显示或被裁切 -->
              <span v-if="t.badge" class="tab-count" :title="`待处理 ${t.badge} 项`">
                {{ t.badge > 99 ? '99+' : t.badge }}
              </span>
            </span>
          </template>

          <!-- 会员总览 -->
          <template v-if="t.name === 'overview'">
          <div class="stat-row">
            <div class="stat"><b>{{ summary.totalUsers }}</b><span>总用户数</span></div>
            <div class="stat"><b>{{ summary.activeMembers }}</b><span>有效会员</span></div>
            <div class="stat"><b class="danger">{{ summary.frozenUsers }}</b><span>已冻结</span></div>
            <div class="stat"><b class="ok">{{ summary.todayRegister }}</b><span>今日注册</span></div>
            <div class="stat"><b class="money">¥{{ money(summary.totalPaid) }}</b><span>累计充值金额</span></div>
          </div>

          <div class="chart-head">
            <div class="chart-title">数据趋势</div>
            <div class="chart-tools">
              <el-radio-group v-model="dashPreset" size="small" @change="onDashPreset">
                <el-radio-button :value="7">近 7 天</el-radio-button>
                <el-radio-button :value="30">近 30 天</el-radio-button>
                <el-radio-button :value="90">近 90 天</el-radio-button>
              </el-radio-group>
              <el-date-picker
                v-model="dashRange"
                type="daterange"
                value-format="YYYY-MM-DD"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                size="small"
                style="width: 240px"
                @change="onDashRange"
              />
            </div>
          </div>
          <div class="chart-row">
            <div class="chart-box">
              <div class="chart-title">每日注册人数</div>
              <div ref="barRef" class="chart"></div>
            </div>
            <div class="chart-box">
              <div class="chart-title">已开通会员人数变化</div>
              <div ref="lineRef" class="chart"></div>
            </div>
          </div>
          </template>

          <!-- 用户列表 -->
          <template v-else-if="t.name === 'users'">
          <div class="filters">
            <el-input
              v-model="filters.keyword"
              placeholder="搜索 ID / 用户名 / 手机号 / 邮箱"
              clearable
              style="width: 240px"
              @keyup.enter="searchUsers"
              @clear="searchUsers"
            />
            <el-select v-model="filters.status" placeholder="账号状态" clearable style="width: 120px" @change="searchUsers">
              <el-option label="正常" value="normal" />
              <el-option label="已冻结" value="frozen" />
            </el-select>
            <el-select v-model="filters.memberType" placeholder="会员类型" clearable style="width: 130px" @change="searchUsers">
              <el-option label="免费试用" value="trial" />
              <el-option label="付费会员" value="paid" />
              <el-option label="永久会员" value="permanent" />
            </el-select>
            <el-button @click="searchUsers">查询</el-button>
            <el-button
              v-if="userStore.isOwner"
              type="primary"
              class="push-right"
              @click="openCreateAdmin"
            >
              添加管理员
            </el-button>
          </div>

          <el-table v-loading="loading" :data="list" border stripe>
            <el-table-column prop="id" label="账号 ID" width="70" />
            <el-table-column prop="username" label="用户名" width="120" show-overflow-tooltip />
            <el-table-column label="登录账号（手机 / 邮箱）" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                <div>{{ row.phone || '-' }}</div>
                <div class="cell-sub">{{ row.email || '-' }}</div>
              </template>
            </el-table-column>
            <el-table-column label="角色" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.isStaff" type="warning" size="small" effect="dark" round>
                  {{ row.roleName }}
                </el-tag>
                <span v-else>{{ row.roleName }}</span>
              </template>
            </el-table-column>
            <el-table-column label="身份" width="118">
              <template #default="{ row }">
                <el-tag
                  class="id-tag"
                  :class="identityClass(row)"
                  size="small"
                  effect="light"
                  round
                >
                  {{ row.identityName }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="账号状态" width="96">
              <template #default="{ row }">
                <el-tag :type="row.status === 'frozen' ? 'danger' : 'success'" size="small" effect="light" round>
                  {{ row.statusName }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="会员类型" width="118">
              <template #default="{ row }">
                <div>{{ row.memberTypeName }}</div>
                <div class="cell-sub">
                  {{ row.memberType === 'permanent' ? '长期有效' : fmtDate(row.memberExpire) + ' 到期' }}
                </div>
              </template>
            </el-table-column>
            <el-table-column label="剩余天数" width="96">
              <template #default="{ row }">
                <span :class="row.memberActive ? 'ok' : 'expired'">
                  {{ row.memberType === 'permanent' ? '永久' : row.memberActive ? row.memberLeftDays + ' 天' : '已过期' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="累计充值" width="110">
              <template #default="{ row }">
                <!-- 站长 / 管理员为后台账号，不参与充值统计 -->
                <span v-if="!row.isStaff" class="money-cell">¥{{ money(row.totalPaid) }}</span>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="注册时间" width="110">
              <template #default="{ row }">{{ fmtDate(row.registeredAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
              <template #default="{ row }">
                <template v-if="row.role === 'user'">
                  <div class="op-stack">
                    <div class="op-row">
                      <el-button link type="primary" @click="openRenew(row)">续期</el-button>
                      <el-button
                        link
                        :type="row.status === 'frozen' ? 'success' : 'danger'"
                        @click="toggleFreeze(row)"
                      >
                        {{ row.status === 'frozen' ? '解冻' : '冻结' }}
                      </el-button>
                    </div>
                    <div class="op-row">
                      <el-button v-if="userStore.isOwner" link type="warning" @click="setRole(row, 'admin')">设为管理员</el-button>
                      <el-button link type="info" @click="openResetPwd(row)">重置密码</el-button>
                    </div>
                  </div>
                </template>
                <template v-else-if="row.role === 'admin'">
                  <div class="op-stack">
                    <div class="op-row">
                      <el-button
                        link
                        :type="row.status === 'frozen' ? 'success' : 'danger'"
                        @click="toggleFreeze(row)"
                      >
                        {{ row.status === 'frozen' ? '解冻' : '冻结' }}
                      </el-button>
                      <el-button link type="info" @click="openResetPwd(row)">重置密码</el-button>
                    </div>
                    <div v-if="userStore.isOwner" class="op-row">
                      <el-button link type="danger" @click="removeUser(row)">删除账号</el-button>
                    </div>
                  </div>
                </template>
                <template v-else-if="row.role === 'owner'">
                  <div class="op-stack">
                    <div class="op-row">
                      <el-button link type="info" @click="openResetPwd(row)">重置密码</el-button>
                    </div>
                  </div>
                </template>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
          </el-table>

          <div class="pager">
            <el-pagination
              v-model:current-page="page"
              :page-size="pageSize"
              :total="total"
              layout="total, prev, pager, next"
              background
              @current-change="load"
            />
          </div>
          </template>

          <!-- 价格设置 -->
          <template v-else-if="t.name === 'price'">
            <div class="pane-desc">各卡片可独立设置并指定生效日期（立即生效 / 指定日期当天 00:00 生效）。保存（或到点）后首页、支付弹窗与续费金额生效，并给对应账号发送该卡片的站内信。</div>

            <!-- 总价预览：与首页 / 支付弹窗同源，便于运营观察最终效果 -->
            <div class="price-preview">
              <div class="pv-head">
                <div class="pv-title">
                  总价预览<span class="pv-sub">按下方设置实时计算 · 首页 / 支付弹窗展示效果</span>
                </div>
              </div>
              <!-- 标签组：位于价格表右上角 -->
              <div class="pv-tags-bar">
                <span v-if="previewDirty" class="pv-unsaved"><i class="dot" />含未保存修改</span>
                <el-tag v-for="d in previewDiscounts" :key="d.key" size="small" :type="d.type" effect="light" round>
                  {{ d.label }}
                </el-tag>
                <el-tag size="small" :type="finalDiscountTag.type" effect="dark" round>
                  {{ finalDiscountTag.label }}
                </el-tag>
              </div>
              <div class="pv-grid">
                <div class="pv-cell pv-th">价格档位</div>
                <div v-for="c in previewCols" :key="c.key" class="pv-cell pv-th">{{ c.label }}</div>
                <template v-for="row in previewRows" :key="row.key">
                  <div class="pv-cell pv-name">{{ row.name }}</div>
                  <div v-for="(c, i) in row.cells" :key="c.key" class="pv-cell pv-price">
                    <span class="pv-eff">
                      ¥{{ c.eff }}<i class="pv-unit">{{ previewCols[i].unit }}</i>
                    </span>
                    <span v-if="c.origin" class="pv-origin">¥{{ c.origin }}</span>
                  </div>
                </template>
              </div>
              <div class="pv-note">
                寒暑假高峰按原价。预览随下方设置实时变化，保存后立即同步首页、支付弹窗与个人中心续费价。
              </div>
              <div v-if="priceWarnings.length" class="pv-warns">
                <div v-for="w in priceWarnings" :key="w" class="pv-warn-item">
                  <span class="pv-warn-dot">!</span>
                  <span>{{ w }}</span>
                </div>
              </div>
            </div>

            <div class="price-grid">
              <div class="price-card" :class="{ 'pc-dirty': cardDirty.pro }">
                <div class="pc-head">
                  <div class="pc-title">
                    标准价<span class="pc-sub">专职老师 / 学员 / 机构</span>
                    <span v-if="cardDirty.pro" class="pc-dirty-dot" />
                  </div>
                </div>
                <div class="pc-item">
                  <span class="pc-label">包月</span>
                  <el-input-number v-model="priceForm.proMonthlyAmount" :min="0" :step="1" :precision="2" controls-position="right" />
                  <span class="pc-unit">元</span>
                </div>
                <div class="pc-item">
                  <span class="pc-label">包季</span>
                  <el-input-number v-model="priceForm.proQuarterlyAmount" :min="0" :step="1" :precision="2" controls-position="right" />
                  <span class="pc-unit">元</span>
                </div>
                <div class="pc-item">
                  <span class="pc-label">包年</span>
                  <el-input-number v-model="priceForm.proYearlyAmount" :min="0" :step="1" :precision="2" controls-position="right" />
                  <span class="pc-unit">元</span>
                </div>
                <div class="pc-note">老师、学员、机构统一适用的包月 / 包季 / 包年标准价（机构当前与学员同价）</div>
                <div class="pc-foot">
                  <price-eff v-model:mode="priceEff.pro.mode" v-model:time="priceEff.pro.time" />
                  <div class="pc-foot-btns">
                    <div class="pc-foot-info">
                      <span v-if="cardDirty.pro" class="pc-unsaved">已修改 · 未保存</span>
                      <span v-if="pendingByCard.pro" class="pc-pending">待生效 {{ fmtSchedule(pendingByCard.pro) }}</span>
                    </div>
                    <div class="pc-foot-actions">
                      <el-button v-if="cardDirty.pro" link size="small" @click="resetCard('pro')">放弃修改</el-button>
                      <el-button type="primary" size="small" :loading="priceSaving === 'pro'" :disabled="!cardDirty.pro" @click="submitCard('pro')">保存</el-button>
                    </div>
                  </div>
                </div>
              </div>

              <div class="price-card" :class="{ 'pc-off': !priceForm.renewalEnabled, 'pc-dirty': cardDirty.renewal }">
                <div class="pc-head">
                  <div class="pc-title">
                    平季续费优惠
                    <span v-if="cardDirty.renewal" class="pc-dirty-dot" />
                  </div>
                  <el-switch v-model="priceForm.renewalEnabled" active-text="启用" size="small" />
                </div>
                <div class="pc-item">
                  <span class="pc-label">平季折扣</span>
                  <el-input-number v-model="priceForm.renewalDiscount" :min="0.1" :max="0.99" :step="0.01" :precision="2" controls-position="right" :disabled="!priceForm.renewalEnabled" />
                </div>
                <div class="pc-note">开学平季续费按此折扣（默认 0.9 即 9 折）；寒暑假高峰续费维持原价。启用后首页套餐与支付弹窗才按折扣计价</div>
                <div class="pc-foot">
                  <price-eff v-model:mode="priceEff.renewal.mode" v-model:time="priceEff.renewal.time" />
                  <div class="pc-foot-btns">
                    <div class="pc-foot-info">
                      <span v-if="cardDirty.renewal" class="pc-unsaved">已修改 · 未保存</span>
                      <span v-if="pendingByCard.renewal" class="pc-pending">待生效 {{ fmtSchedule(pendingByCard.renewal) }}</span>
                    </div>
                    <div class="pc-foot-actions">
                      <el-button v-if="cardDirty.renewal" link size="small" @click="resetCard('renewal')">放弃修改</el-button>
                      <el-button type="primary" size="small" :loading="priceSaving === 'renewal'" :disabled="!cardDirty.renewal" @click="submitCard('renewal')">保存</el-button>
                    </div>
                  </div>
                </div>
              </div>

              <div class="price-card" :class="{ 'pc-off': !priceForm.activityEnabled, 'pc-dirty': cardDirty.activity }">
                <div class="pc-head">
                  <div class="pc-title">
                    限时活动<span class="pc-sub">自定义</span>
                    <span v-if="cardDirty.activity" class="pc-dirty-dot" />
                  </div>
                  <el-switch v-model="priceForm.activityEnabled" active-text="启用" size="small" />
                </div>
                <div class="pc-item">
                  <span class="pc-label">活动名称</span>
                  <el-input v-model="priceForm.activityName" maxlength="32" placeholder="如：开学季优惠" style="flex: 1" />
                </div>
                <div class="pc-item">
                  <span class="pc-label">活动折扣</span>
                  <el-input-number v-model="priceForm.activityDiscount" :min="0.1" :max="0.99" :step="0.01" :precision="2" controls-position="right" :disabled="!priceForm.activityEnabled" />
                </div>
                <div class="pc-note">启用后与平季折扣叠加作用于首页与支付价格；停用或未启用时按标准价（及学生折扣）计价</div>
                <div class="pc-foot">
                  <price-eff v-model:mode="priceEff.activity.mode" v-model:time="priceEff.activity.time" />
                  <div class="pc-foot-btns">
                    <div class="pc-foot-info">
                      <span v-if="cardDirty.activity" class="pc-unsaved">已修改 · 未保存</span>
                      <span v-if="pendingByCard.activity" class="pc-pending">待生效 {{ fmtSchedule(pendingByCard.activity) }}</span>
                    </div>
                    <div class="pc-foot-actions">
                      <el-button v-if="cardDirty.activity" link size="small" @click="resetCard('activity')">放弃修改</el-button>
                      <el-button type="primary" size="small" :loading="priceSaving === 'activity'" :disabled="!cardDirty.activity" @click="submitCard('activity')">保存</el-button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 待生效计划：预约的价格调整可在此核对与撤销 -->
            <div class="price-panel">
              <div class="pp-head">
                <div class="pp-title">
                  待生效计划
                  <span class="pp-sub">这些调整会在所选日期 00:00 自动生效并通知对应用户</span>
                </div>
                <el-tag v-if="!schedules.length" size="small" type="info" effect="light" round>暂无预约</el-tag>
              </div>
              <el-table v-if="schedules.length" :data="schedules" size="small" border>
                <el-table-column label="卡片" width="120">
                  <template #default="{ row }">{{ cardLabel(row.card) }}</template>
                </el-table-column>
                <el-table-column label="生效日期" width="140">
                  <template #default="{ row }">{{ fmtSchedule(row) }}</template>
                </el-table-column>
                <el-table-column label="调整内容" min-width="240">
                  <template #default="{ row }">
                    <span class="pp-summary">{{ payloadSummary(row.card, row.payload) }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="90" align="right">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="cancelSchedule(row)">撤销</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- 变更历史：可核对每次调价并一键回滚 -->
            <div class="price-panel">
              <div class="pp-head">
                <div class="pp-title">
                  变更历史
                  <span class="pp-sub">最近 20 条；回滚会把该卡片恢复到本次变更前的值</span>
                </div>
                <el-button link size="small" :loading="historyLoading" @click="loadHistory">刷新</el-button>
              </div>
              <el-table v-if="historyList.length" :data="historyList" size="small" border>
                <el-table-column label="时间" width="150">
                  <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
                </el-table-column>
                <el-table-column label="卡片" width="120">
                  <template #default="{ row }">{{ cardLabel(row.card) }}</template>
                </el-table-column>
                <el-table-column label="来源" width="90">
                  <template #default="{ row }">
                    <el-tag size="small" effect="light" :type="sourceTag(row.source).type">{{ sourceTag(row.source).label }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="调整内容" min-width="240">
                  <template #default="{ row }">
                    <span class="pp-summary">{{ historySummary(row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作人" width="110">
                  <template #default="{ row }">{{ row.operatorName || '系统' }}</template>
                </el-table-column>
                <el-table-column label="操作" width="90" align="right">
                  <template #default="{ row }">
                    <el-button link type="primary" size="small" @click="rollbackHistory(row)">回滚</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div v-else class="pp-empty">暂无变更记录</div>
            </div>
          </template>

          <!-- 邀请码 -->
          <template v-else-if="t.name === 'invite'">
            <InviteCodePanel />
          </template>

          <!-- 用户反馈 -->
          <template v-else-if="t.name === 'feedback'">
            <FeedbackPanel />
          </template>

          <!-- 站内信 -->
          <template v-else-if="t.name === 'messages'">
          <div class="pane-desc">可发送给全员或指定用户（按用户名），用于优惠通知、到期提醒等。</div>
          <el-form label-position="top" class="msg-form">
            <el-form-item label="接收对象">
              <el-radio-group v-model="msgForm.target">
                <el-radio value="all">全员广播</el-radio>
                <el-radio value="user">指定用户</el-radio>
              </el-radio-group>
              <el-input
                v-if="msgForm.target === 'user'"
                v-model="msgForm.username"
                placeholder="请输入接收用户名"
                style="width: 200px; margin-left: 12px"
              />
            </el-form-item>
            <el-form-item label="类型">
              <el-radio-group v-model="msgForm.type">
                <el-radio value="system">系统通知</el-radio>
                <el-radio value="promo">优惠活动</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="标题">
              <el-input v-model="msgForm.title" placeholder="站内信标题" style="max-width: 360px" />
            </el-form-item>
            <el-form-item label="内容">
              <el-input
                v-model="msgForm.content"
                type="textarea"
                :rows="3"
                placeholder="站内信正文内容"
                style="max-width: 520px"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="msgSending" @click="sendMessage">发送</el-button>
            </el-form-item>
          </el-form>
          </template>
        </el-tab-pane>
      </el-tabs>
    </section>

    <!-- 续费（填写支付金额 + 续费时长） -->
    <el-dialog v-model="renewVisible" title="会员续费" width="460px" class="renew-dialog">
      <div v-if="renewUser" class="renew-user">
        <div class="ru-main">
          <span class="ru-label">账号</span>
          <b class="ru-name">{{ renewUser.username }}</b>
          <el-tag size="small" effect="light" round>{{ renewUser.memberTypeName }}</el-tag>
        </div>
        <div class="ru-meta">
          当前到期 {{ fmtDate(renewUser.memberExpire) }} · 累计充值 ¥{{ money(renewUser.totalPaid) }}
        </div>
      </div>
      <el-form label-position="top" class="renew-form">
        <el-form-item label="支付金额">
          <div class="amount-row">
            <el-input-number v-model="renewForm.amount" :min="0" :step="10" :precision="2" controls-position="right" style="width: 180px" />
            <span class="unit">元</span>
          </div>
        </el-form-item>
        <el-form-item label="续费时长">
          <div class="period-row">
            <el-radio-group v-model="renewForm.period">
              <el-radio-button value="daily">按天</el-radio-button>
              <el-radio-button value="monthly">一月</el-radio-button>
              <el-radio-button value="quarterly">一季度</el-radio-button>
              <el-radio-button value="yearly">一年</el-radio-button>
            </el-radio-group>
            <div v-if="renewForm.period === 'daily'" class="days-row">
              <el-input-number
                v-model="renewForm.days"
                :min="1"
                :max="3650"
                :step="1"
                controls-position="right"
                size="small"
                style="width: 120px"
              />
              <span class="unit">天</span>
              <span class="quick-days">
                <el-button v-for="d in [7, 30, 90]" :key="d" link type="primary" size="small" @click="renewForm.days = d">
                  {{ d }} 天
                </el-button>
              </span>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="备注（选填）">
          <el-input v-model="renewForm.remark" placeholder="如：微信转账 / 支付宝" />
        </el-form-item>
      </el-form>

      <div class="renew-preview">
        <span>续费后到期时间</span>
        <b>{{ previewExpire }}</b>
        <div class="muted small">在原到期时间上顺延（已过期则从今天起算），并累加到累计充值金额</div>
      </div>

      <template #footer>
        <el-button @click="renewVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitRenew">确认续费</el-button>
      </template>
    </el-dialog>

    <!-- 添加管理员（仅站长）：只需填写登录信息 -->
    <el-dialog v-model="adminVisible" title="添加管理员" width="420px" @keyup.enter="submitCreateAdmin">
      <div class="muted small" style="margin-bottom:12px">
        填写登录信息即可：创建后该账号即为管理员，享有永久会员权益，无需邀请码；请妥善保存密码，管理员登录后可在个人信息中自行修改。
      </div>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="用户名">
          <el-input
            v-model="adminForm.username"
            maxlength="32"
            placeholder="2-32 个字符，可使用中文、字母、数字，不可含空格"
          />
        </el-form-item>
        <el-form-item label="手机号（登录账号）">
          <el-input v-model="adminForm.phone" maxlength="11" placeholder="11 位手机号，用于登录与找回" />
        </el-form-item>
        <el-form-item label="登录密码">
          <el-input
            v-model="adminForm.password"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="6-32 位，无需二次确认"
          >
            <template #append>
              <el-button @click="genAdminPassword">随机生成</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="adminVisible = false">取消</el-button>
        <el-button type="primary" :loading="adminSaving" @click="submitCreateAdmin">确认添加</el-button>
      </template>
    </el-dialog>

    <!-- 重置登录密码（普通用户 / 管理员 / 站长本人）：无需原密码，可自定义或留空由服务端随机生成 -->
    <el-dialog v-model="pwdVisible" title="重置登录密码" width="420px">
      <div v-if="pwdUser" class="renew-user">
        <div class="ru-main">
          <span class="ru-label">账号</span>
          <b class="ru-name">{{ pwdUser.username }}</b>
          <el-tag size="small" effect="light" round>{{ pwdUser.roleName }}</el-tag>
        </div>
        <div class="ru-meta">
          {{ pwdUser.phone || pwdUser.email || '-' }} · {{ pwdUser.identityName }}
        </div>
      </div>
      <el-form label-position="top" class="renew-form" @submit.prevent>
        <el-form-item label="新密码">
          <el-input
            v-model="pwdForm.password"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="6-32 位；留空则自动生成 12 位随机密码"
          >
            <template #append>
              <el-button @click="genResetPassword">随机生成</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <div class="muted small">重置后原密码立即失效，请把新密码告知本人；对方登录后可在个人信息中自行修改。</div>

      <div v-if="pwdResult" class="renew-preview">
        <span>新密码</span>
        <b>{{ pwdResult }}</b>
        <el-button link type="primary" size="small" @click="copyResetPwd">复制</el-button>
      </div>

      <template #footer>
        <el-button @click="closeResetPwd">{{ pwdResult ? '完成' : '取消' }}</el-button>
        <el-button v-if="!pwdResult" type="primary" :loading="pwdSaving" @click="submitResetPwd">确认重置</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import { adminApi, messageApi, priceApi } from '@/api'
import { useNavTab } from '@/stores/navTab'
import { usePriceStore } from '@/stores/price'
import { useUserStore } from '@/stores/user'
import FeedbackPanel from '@/components/FeedbackPanel.vue'
import { isValidPhone, sanitizeText, validateText, validateTexts } from '@/utils/text'
import { zhe, priceCardTags } from '@/utils/priceTags'
import InviteCodePanel from '@/components/InviteCodePanel.vue'
import ReorderList from '@/components/ReorderList.vue'
import PriceEff from '@/components/PriceEff.vue'

const userStore = useUserStore()
const priceStore = usePriceStore()
// 支持从其它入口（如 /admin/feedbacks）跳转到指定页签：?tab=feedback
const route = useRoute()

// 二级导航页签：weight 为默认权重；智能排序时按待办数量（badge）自动提前
const TAB_META = [
  { name: 'overview', label: '会员总览', weight: 10 },
  { name: 'users', label: '用户列表', weight: 30 },
  { name: 'price', label: '价格设置', weight: 40 },
  { name: 'invite', label: '邀请码', weight: 50, lazy: true },
  { name: 'feedback', label: '用户反馈', weight: 60, lazy: true },
  { name: 'messages', label: '站内信', weight: 70 }
]

const ORDER_KEY = 'admin.members.tabOrder'

function loadOrder() {
  try {
    const v = JSON.parse(localStorage.getItem(ORDER_KEY) || '[]')
    return Array.isArray(v) ? v : []
  } catch {
    return []
  }
}

// 页签记忆：路由 ?tab= 优先，其次恢复上次所在页签，默认总览
const activeTab = useNavTab(
  'admin-members',
  'overview',
  TAB_META.some((t) => t.name === route.query.tab) ? route.query.tab : ''
)
// 页签顺序：手动排序（拖拽 / 排序面板），持久化到 localStorage
const manualOrder = ref(loadOrder())
const dragName = ref('')
const hoverName = ref('')
const pendingFeedbacks = ref(0)

const badgeMap = computed(() => ({
  feedback: pendingFeedbacks.value
}))

const tabs = computed(() => TAB_META.map((t) => ({ ...t, badge: badgeMap.value[t.name] || 0 })))

const orderedTabs = computed(() => {
  const list = tabs.value
  const order = manualOrder.value.length ? manualOrder.value : TAB_META.map((t) => t.name)
  const sorted = order.map((n) => list.find((t) => t.name === n)).filter(Boolean)
  // 总览固定置顶，其余按自定义顺序
  return [...sorted.filter((t) => t.name === 'overview'), ...sorted.filter((t) => t.name !== 'overview')]
})

function currentOrder() {
  return orderedTabs.value.map((t) => t.name)
}

function onDragStart(name) {
  dragName.value = name
}

function onDrop(name) {
  if (!dragName.value || dragName.value === name) return
  const order = currentOrder()
  const from = order.indexOf(dragName.value)
  const to = order.indexOf(name)
  if (from < 0 || to < 0) return
  order.splice(to, 0, order.splice(from, 1)[0])
  manualOrder.value = order
}

function onDragEnd() {
  dragName.value = ''
  hoverName.value = ''
}

function resetOrder() {
  manualOrder.value = []
  dragName.value = ''
  hoverName.value = ''
}

// 待办数量：用于页签徽标（进入管理中心即检测，并有定时/聚焦刷新）
// 静默请求（silent）：轮询失败不弹全局错误提示，保留上一次的角标数量，下一轮自动重试
async function loadBadges() {
  try {
    const res = await adminApi.feedbacks({ status: 'pending', page: 1, pageSize: 1 }, { silent: true })
    pendingFeedbacks.value = res?.data?.total || 0
  } catch {
    // 保留上次结果，下一轮自动重试
  }
}

watch(manualOrder, (v) => localStorage.setItem(ORDER_KEY, JSON.stringify(v)), { deep: true })

const dashPreset = ref(30)
const dashRange = ref([dayjs().subtract(29, 'day').format('YYYY-MM-DD'), dayjs().format('YYYY-MM-DD')])
const dashRangeStart = ref('')
const dashRangeEnd = ref('')
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const summary = ref({ totalUsers: 0, activeMembers: 0, frozenUsers: 0, todayRegister: 0, totalRecharge: 0 })
const money = (v) => Number(v || 0).toFixed(2)
const filters = reactive({ keyword: '', status: '', memberType: '' })

const barRef = ref(null)
const lineRef = ref(null)
let barChart = null
let lineChart = null
let badgeTimer = null // 待办角标定时检测句柄

const renewVisible = ref(false)
const renewUser = ref(null)
const saving = ref(false)
const renewForm = reactive({ amount: 0, period: 'monthly', days: 30, remark: '' })

// 添加管理员（仅站长）：仅填写登录信息
const adminVisible = ref(false)
const adminSaving = ref(false)
const adminForm = reactive({ username: '', phone: '', password: '' })

function openCreateAdmin() {
  adminForm.username = ''
  adminForm.phone = ''
  adminForm.password = ''
  adminVisible.value = true
}

// 身份标签配色：站长 / 管理员用后台色，普通用户按使用身份区分
function identityClass(row) {
  return (
    {
      owner: 'is-owner',
      admin: 'is-admin',
      teacher: 'is-pro',
      parent: 'is-user',
      personal: 'is-user',
      org: 'is-org'
    }[row.identityType] || 'is-pro'
  )
}

// 随机生成 12 位登录密码（大小写字母 + 数字，去除易混淆字符）
function genRandomPassword() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789'
  let pwd = ''
  const rnd = new Uint32Array(12)
  crypto.getRandomValues(rnd)
  for (let i = 0; i < 12; i++) pwd += chars[rnd[i] % chars.length]
  return pwd
}

function genAdminPassword() {
  adminForm.password = genRandomPassword()
}

// 重置普通用户密码：无需原密码；留空则由服务端随机生成并回传明文
const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdUser = ref(null)
const pwdForm = reactive({ password: '' })
const pwdResult = ref('')

function openResetPwd(row) {
  pwdUser.value = row
  pwdForm.password = ''
  pwdResult.value = ''
  pwdVisible.value = true
}

function closeResetPwd() {
  pwdVisible.value = false
  pwdUser.value = null
  pwdForm.password = ''
  pwdResult.value = ''
}

function genResetPassword() {
  pwdForm.password = genRandomPassword()
}

async function submitResetPwd() {
  if (pwdSaving.value || !pwdUser.value) return
  const pwd = pwdForm.password
  if (pwd && (pwd.length < 6 || pwd.length > 32)) {
    ElMessage.warning('密码需为 6-32 位')
    return
  }
  pwdSaving.value = true
  try {
    const res = await adminApi.resetPassword(pwdUser.value.id, pwd)
    const generated = res?.data?.password || ''
    if (generated) {
      pwdResult.value = generated
      pwdForm.password = ''
      ElMessage.success('密码已重置，请复制后告知本人')
    } else {
      ElMessage.success('密码已重置，请将新密码告知本人')
      closeResetPwd()
    }
  } finally {
    pwdSaving.value = false
  }
}

async function copyResetPwd() {
  try {
    await navigator.clipboard.writeText(pwdResult.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选中复制')
  }
}

async function submitCreateAdmin() {
  if (adminSaving.value) return
  const username = sanitizeText(adminForm.username)
  adminForm.username = username
  if (username.length < 2 || username.length > 32) {
    ElMessage.warning('用户名需为 2-32 个字符，可使用中文、字母、数字，不可含空格')
    return
  }
  let msg = validateText(username, '用户名', { max: 32 })
  if (!msg) msg = validateText(adminForm.phone, '手机号', { max: 11, required: true })
  if (msg) {
    ElMessage.warning(msg)
    return
  }
  const phone = adminForm.phone.trim()
  if (!isValidPhone(phone)) {
    ElMessage.warning('请填写正确的 11 位手机号')
    return
  }
  if (adminForm.password.length < 6 || adminForm.password.length > 32) {
    ElMessage.warning('密码需为 6-32 位')
    return
  }
  adminSaving.value = true
  try {
    await adminApi.createAdmin({ username, phone, password: adminForm.password })
    ElMessage.success(`管理员「${username}」已添加，请将登录信息告知对方`)
    adminVisible.value = false
    load()
    loadStats()
  } finally {
    adminSaving.value = false
  }
}

// 会员价格设置：管理员按卡片调整，可指定生效时间（已作为二级导航页签）
const priceSaving = ref('') // 记录正在保存的卡片，避免互相干扰
const priceForm = reactive({
  proMonthlyAmount: 0, proQuarterlyAmount: 0, proYearlyAmount: 0,
  renewalDiscount: 0.9,
  renewalEnabled: true,
  activityName: '',
  activityDiscount: 0.9,
  activityEnabled: false
})
// 每个卡片的生效时间控制：mode=now 立即 / later 指定时间；time 为 datetime 字符串
const priceEff = reactive({
  pro: { mode: 'now', time: '' },
  renewal: { mode: 'now', time: '' },
  activity: { mode: 'now', time: '' }
})
const schedules = ref([]) // 待生效的价格预约
const historyList = ref([]) // 价格变更历史
const historyLoading = ref(false)
const impact = ref({ all: 0, students: 0 }) // 站内信预计触达人数（按账号身份统计，与在线状态无关）

// ---- 总价预览：由下方表单实时驱动（改数字 / 折扣 / 开关即刻反映），算法与后端 GetPrice 完全一致 ----
// 生效折扣 = 平季折扣（启用且处于平季）× 活动折扣（启用），与后端 renewalOn / activityOn 判定一致
const previewDisc = computed(() => {
  const f = priceForm
  const peak = priceStore.price.season === 'peak'
  let d = 1
  if (f.renewalEnabled && !peak && Number(f.renewalDiscount) > 0 && Number(f.renewalDiscount) < 1) {
    d *= Number(f.renewalDiscount)
  }
  if (f.activityEnabled && Number(f.activityDiscount) > 0 && Number(f.activityDiscount) < 1) {
    d *= Number(f.activityDiscount)
  }
  return d
})

// 折扣标签：与首页权益卡片共用同一套口径（活动 / 寒暑假 / 平季），见 utils/priceTags
const previewDiscounts = computed(() => priceCardTags(priceForm, { viewType: 'professional', season: priceStore.price.season }))

// 最终折扣：把「平季 × 活动」叠加后折算成最低折扣，给运营一个直观的总优惠强度
const finalDiscount = computed(() => previewDisc.value)
const finalDiscountTag = computed(() => {
  const d = finalDiscount.value
  if (d >= 0.995) return { label: '最终按标准价', type: 'info' }
  return { label: `最终最低 ${zhe(d)} 折`, type: d < 0.5 ? 'danger' : 'warning' }
})

// 套餐合理性提示：包季 / 包年应低于包月 ×3 / ×12，否则套餐没有优惠（仅提示，不阻断保存）
const proComboWarn = computed(() => {
  const m = Number(priceForm.proMonthlyAmount) || 0
  const q = Number(priceForm.proQuarterlyAmount) || 0
  const y = Number(priceForm.proYearlyAmount) || 0
  if (m <= 0) return ''
  if (q >= m * 3) return `包季 ¥${q} 不低于包月 ¥${m} × 3，未形成套餐优惠`
  if (y >= m * 12) return `包年 ¥${y} 不低于包月 ¥${m} × 12，未形成套餐优惠`
  return ''
})

// 折上折护栏 + 组合提醒：与后端 priceWarnings 口径一致，仅提示不阻断
const priceWarnings = computed(() => {
  const list = []
  const m = Number(priceForm.proMonthlyAmount) || 0
  const q = Number(priceForm.proQuarterlyAmount) || 0
  const y = Number(priceForm.proYearlyAmount) || 0
  if (m > 0 && q >= m * 3) list.push(`包季 ¥${q} 不低于包月 ¥${m} × 3，未形成套餐优惠`)
  if (m > 0 && y >= m * 12) list.push(`包年 ¥${y} 不低于包月 ¥${m} × 12，未形成套餐优惠`)
  const d = finalDiscount.value
  if (d < 0.3) list.push(`折上折后最低约 ${zhe(d)} 折，价格明显偏低，请确认是否符合预期`)
  return list
})

// 卡片 / 来源的中文名，用于预约列表与变更历史展示
const cardLabel = (card) =>
  ({ pro: '标准价', student: '学生价', renewal: '平季续费优惠', activity: '限时活动' }[card] || card)
const sourceTag = (source) =>
  ({
    immediate: { label: '立即保存', type: 'success' },
    schedule: { label: '预约生效', type: 'warning' },
    rollback: { label: '回滚', type: 'danger' }
  }[source] || { label: source || '-', type: 'info' })
const fmtTime = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD HH:mm') : '-')

// 把卡片字段 JSON 渲染成「旧值 → 新值」的可读摘要
function describeCard(card, data) {
  if (!data) return '-'
  const num = (v) => (Number(v) || 0).toString()
  if (card === 'pro') {
    return `包月 ¥${num(data.proMonthlyAmount)} / 包季 ¥${num(data.proQuarterlyAmount)} / 包年 ¥${num(data.proYearlyAmount)}`
  }
  if (card === 'student') return `学生折扣 ${zhe(data.studentDiscount)} 折`
  if (card === 'renewal') {
    return data.renewalEnabled ? `平季 ${zhe(data.renewalDiscount)} 折（启用）` : '平季优惠（已关闭）'
  }
  if (card === 'activity') {
    const name = data.activityName || '未命名活动'
    return data.activityEnabled ? `「${name}」${zhe(data.activityDiscount)} 折（启用）` : `「${name}」（已关闭）`
  }
  return '-'
}
const parseJson = (s) => {
  try {
    return JSON.parse(s || '{}')
  } catch (e) {
    return {}
  }
}
const payloadSummary = (card, payload) => describeCard(card, parseJson(payload))
const historySummary = (row) => `${describeCard(row.card, parseJson(row.before))} → ${describeCard(row.card, parseJson(row.after))}`
// 确认框使用 HTML 换行，涉及活动名称等用户输入时先转义，避免被当作标签渲染
const escHtml = (s) =>
  String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))

// 价格表：行 = 标准价，列 = 包月 / 包季 / 包年；划线原价统一取「标准价」（与首页一致）
// 注意：师资身份计费档位已下线，全站统一按标准价
const previewRows = computed(() => {
  const f = priceForm
  const disc = previewDisc.value
  const base = [Number(f.proMonthlyAmount) || 0, Number(f.proQuarterlyAmount) || 0, Number(f.proYearlyAmount) || 0]
  const keys = ['monthly', 'quarterly', 'yearly']
  const cellsOf = (effList) =>
    keys.map((key, i) => ({
      key,
      eff: effList[i],
      origin: effList[i] !== base[i] ? base[i] : ''
    }))
  const proEff = base.map((v) => Math.ceil(v * disc))
  return [{ key: 'professional', name: '标准价', cells: cellsOf(proEff) }]
})

// 各卡片是否有未保存修改（用于卡片角标 + 预览「含未保存修改」提示）
const numOf = (v) => Number(v) || 0
const cardDirty = computed(() => {
  const p = priceStore.price
  const f = priceForm
  return {
    pro:
      numOf(f.proMonthlyAmount) !== numOf(p.professional.monthlyAmount) ||
      numOf(f.proQuarterlyAmount) !== numOf(p.professional.quarterlyAmount) ||
      numOf(f.proYearlyAmount) !== numOf(p.professional.yearlyAmount),
    renewal:
      !!f.renewalEnabled !== !!p.renewalEnabled ||
      (!!f.renewalEnabled && numOf(f.renewalDiscount) !== numOf(p.renewalDiscount)),
    activity:
      !!f.activityEnabled !== !!p.activityEnabled ||
      // 活动折扣仅在启用时比较：停用时表单保留占位值（0.9），不参与计价也就不算改动
      (!!f.activityEnabled && numOf(f.activityDiscount) !== numOf(p.activityDiscount)) ||
      (f.activityName || '') !== (p.activityName || '')
  }
})
const previewDirty = computed(() => Object.values(cardDirty.value).some(Boolean))

// 放弃某张卡片的未保存修改，恢复为当前已生效配置
function resetCard(card) {
  const p = priceStore.price
  if (card === 'pro') {
    priceForm.proMonthlyAmount = numOf(p.professional.monthlyAmount)
    priceForm.proQuarterlyAmount = numOf(p.professional.quarterlyAmount)
    priceForm.proYearlyAmount = numOf(p.professional.yearlyAmount)
  } else if (card === 'renewal') {
    priceForm.renewalDiscount = numOf(p.renewalDiscount)
    priceForm.renewalEnabled = p.renewalEnabled !== false
  } else if (card === 'activity') {
    priceForm.activityName = p.activityName || ''
    priceForm.activityDiscount = p.activityEnabled ? numOf(p.activityDiscount) || 0.9 : 0.9
    priceForm.activityEnabled = !!p.activityEnabled
  }
}

const previewCols = [
  { key: 'monthly', label: '包月', unit: '/ 月' },
  { key: 'quarterly', label: '包季', unit: '/ 季' },
  { key: 'yearly', label: '包年', unit: '/ 年' }
]

// 各卡片首条待生效预约（用于卡片上展示「待生效」）
const pendingByCard = computed(() => {
  const map = {}
  for (const s of schedules.value) {
    if (!map[s.card]) map[s.card] = s
  }
  return map
})

const fmtSchedule = (sch) => dayjs((sch.effectiveAt || 0) * 1000).format('YYYY-MM-DD')

async function loadSchedules() {
  try {
    const res = await priceApi.schedules()
    schedules.value = res.data || []
  } catch (e) {
    schedules.value = []
  }
}

// 站内信预计触达人数：只按账号身份统计，与登录 / 在线状态无关
async function loadImpact() {
  try {
    const res = await priceApi.impact()
    impact.value = { all: res.data?.all || 0, students: res.data?.students || 0 }
  } catch (e) {
    impact.value = { all: 0, students: 0 }
  }
}

async function loadHistory() {
  historyLoading.value = true
  try {
    const res = await priceApi.history({ limit: 20 })
    historyList.value = res.data || []
  } catch (e) {
    historyList.value = []
  } finally {
    historyLoading.value = false
  }
}

// 撤销一条待生效预约
async function cancelSchedule(row) {
  try {
    await ElMessageBox.confirm(
      `确认撤销「${cardLabel(row.card)}」在 ${fmtSchedule(row)} 的生效计划？`,
      '撤销预约',
      { type: 'warning', confirmButtonText: '撤销', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await priceApi.cancelSchedule(row.id)
    ElMessage.success('已撤销该预约')
    loadSchedules()
  } catch (e) {
    /* 错误提示由请求拦截器统一处理 */
  }
}

// 按历史记录回滚对应卡片
async function rollbackHistory(row) {
  try {
    await ElMessageBox.confirm(
      `将「${escHtml(cardLabel(row.card))}」回滚到 ${fmtTime(row.createdAt)} 变更前的值：<br><b>${escHtml(
        describeCard(row.card, parseJson(row.before))
      )}</b><br><br>回滚会立即生效并通知对应用户。`,
      '回滚价格',
      {
        dangerouslyUseHTMLString: true,
        type: 'warning',
        confirmButtonText: '确认回滚',
        cancelButtonText: '取消'
      }
    )
  } catch (e) {
    return
  }
  try {
    await priceApi.rollback(row.id)
    ElMessage.success('已回滚到该次变更前的价格')
    priceStore.fetch()
    loadSchedules()
    loadHistory()
  } catch (e) {
    /* 错误提示由请求拦截器统一处理 */
  }
}

async function loadPrice() {
  await priceStore.fetch()
  const pro = priceStore.price.professional
  const p = priceStore.price
  priceForm.proMonthlyAmount = pro.monthlyAmount ?? 0
  priceForm.proQuarterlyAmount = pro.quarterlyAmount ?? 0
  priceForm.proYearlyAmount = pro.yearlyAmount ?? 0
  priceForm.renewalDiscount = p.renewalDiscount ?? 0.9
  priceForm.renewalEnabled = p.renewalEnabled !== false
  priceForm.activityName = p.activityName || ''
  priceForm.activityDiscount = p.activityEnabled ? p.activityDiscount || 0.9 : 0.9
  priceForm.activityEnabled = !!p.activityEnabled
  await Promise.all([loadSchedules(), loadImpact(), loadHistory()])
}

// 页签切换即加载价格配置（immediate：因记忆页签直接在「价格设置」打开时也要加载）。
// 放在价格状态定义之后注册，避免 immediate 触发时访问尚未初始化的变量。
function onTabChange(val) {
  if (val === 'price') loadPrice()
}

watch(
activeTab,
(val) => {
  if (val === 'price') loadPrice()
  if (val === 'feedback') loadBadges()
},
{ immediate: true }
)

// 按卡片保存价格 / 折扣：立即生效直接写入并通知，指定时间写入预约由调度器到点应用
async function submitCard(card) {
  // 未进行任何有效改动时直接提示并返回，避免无意义的覆盖保存与站内信通知
  if (!cardDirty.value[card]) {
    ElMessage.info('该卡片内容未修改，无需保存')
    return
  }
  const eff = priceEff[card]
  let effectiveAt = 0
  if (eff.mode === 'later') {
    if (!eff.time) {
      ElMessage.warning('请选择生效日期')
      return
    }
    // 生效时间精确到日：所选日期当天 00:00 起生效
    effectiveAt = dayjs(eff.time).startOf('day').unix()
    if (effectiveAt <= Math.floor(Date.now() / 1000)) {
      ElMessage.warning('生效日期需晚于今天')
      return
    }
    // 未启用的价卡（当前与目标均为关闭）不会有实际变化，禁止为其指定未来生效日期
    const p = priceStore.price
    if (card === 'renewal' && !priceForm.renewalEnabled && !p.renewalEnabled) {
      ElMessage.warning('「平季续费优惠」当前未启用，无需指定生效日期')
      return
    }
    if (card === 'activity' && !priceForm.activityEnabled && !p.activityEnabled) {
      ElMessage.warning('「限时活动」当前未启用，无需指定生效日期')
      return
    }
  }
  const payload = { card, effectiveAt }
  if (card === 'pro') {
    // 防呆：价格未加载完成（表单仍为初始 0）时禁止保存，避免把 0 写进价格配置
    if (!Number(priceForm.proMonthlyAmount) && !Number(priceForm.proQuarterlyAmount) && !Number(priceForm.proYearlyAmount)) {
      ElMessage.warning('价格尚未加载完成，已重新加载，请确认后再保存')
      loadPrice()
      return
    }
    payload.proMonthlyAmount = Number(priceForm.proMonthlyAmount) || 0
    payload.proQuarterlyAmount = Number(priceForm.proQuarterlyAmount) || 0
    payload.proYearlyAmount = Number(priceForm.proYearlyAmount) || 0
  } else if (card === 'renewal') {
    if (priceForm.renewalEnabled && (priceForm.renewalDiscount < 0.1 || priceForm.renewalDiscount > 0.99)) {
      ElMessage.warning('平季折扣需在 0.1~0.99 之间')
      return
    }
    payload.renewalDiscount = Number(priceForm.renewalDiscount)
    payload.renewalEnabled = priceForm.renewalEnabled
  } else if (card === 'activity') {
    if (priceForm.activityEnabled) {
      if (!priceForm.activityName.trim()) {
        ElMessage.warning('请填写活动名称')
        return
      }
      if (priceForm.activityDiscount < 0.1 || priceForm.activityDiscount > 0.99) {
        ElMessage.warning('活动折扣需在 0.1~0.99 之间')
        return
      }
    }
    payload.activityName = priceForm.activityName.trim()
    payload.activityDiscount = Number(priceForm.activityDiscount)
    payload.activityEnabled = priceForm.activityEnabled
  }
  // 保存前二次确认：说明生效时间、优惠强度与站内信触达人数
  const audience = `全部注册账号（约 ${impact.value.all} 个）`
  const lines = []
  if (eff.mode === 'later') {
    lines.push(`生效时间：${fmtSchedule({ effectiveAt })} 00:00 自动生效`)
    lines.push(`届时将给${audience}发送站内信`)
  } else {
    lines.push(`生效时间：立即生效，并给${audience}发送站内信`)
  }
  lines.push(`优惠强度：${finalDiscountTag.value.label}`)
  if (priceWarnings.value.length) {
    lines.push('', '请注意：')
    priceWarnings.value.forEach((w) => lines.push(`· ${w}`))
  }
  try {
    await ElMessageBox.confirm(lines.join('<br>'), `确认保存「${cardLabel(card)}」`, {
      dangerouslyUseHTMLString: true,
      type: priceWarnings.value.length ? 'warning' : 'info',
      confirmButtonText: '确认保存',
      cancelButtonText: '再检查一下'
    })
  } catch (e) {
    return
  }
  priceSaving.value = card
  try {
    const res = await priceApi.updateCard(payload)
    if (res.data?.applied) {
      ElMessage.success(`已保存并生效，站内信已发送给${audience}`)
    } else {
      ElMessage.success(`已预约于 ${fmtSchedule({ effectiveAt })} 生效`)
    }
    ;(res.data?.warnings || []).forEach((w) => ElMessage.warning(w))
    priceStore.fetch()
    loadSchedules()
    loadHistory()
  } finally {
    priceSaving.value = ''
  }
}

// 续费后的到期时间预览（与后端算法一致：未过期顺延，已过期从今天起算）
const previewExpire = computed(() => {
  if (!renewUser.value) return '-'
  const now = Math.floor(Date.now() / 1000)
  const base = Math.max(renewUser.value.memberExpire || 0, now)
  const d = dayjs(base * 1000)
  if (renewForm.period === 'daily') {
    const days = Math.max(1, Math.min(3650, Math.floor(Number(renewForm.days) || 0)))
    return d.add(days, 'day').format('YYYY-MM-DD')
  }
  const addMap = { yearly: [1, 'year'], quarterly: [3, 'month'], monthly: [1, 'month'] }
  const [n, unit] = addMap[renewForm.period] || [1, 'month']
  const next = d.add(n, unit)
  return next.format('YYYY-MM-DD')
})

const fmtDate = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD') : '-')

// 查询用户列表：筛选变化时回到第一页，避免停留在超出范围的页码看不到账号
function searchUsers() {
  page.value = 1
  load()
}

async function load() {
  loading.value = true
  try {
    const res = await adminApi.users({
      page: page.value,
      pageSize: pageSize.value,
      keyword: filters.keyword,
      status: filters.status,
      memberType: filters.memberType
    })
    list.value = res.data.list || []
    total.value = res.data.total || 0
    // 删掉本页最后一条时回退一页，避免停留在空白页
    if (!list.value.length && page.value > 1) {
      page.value -= 1
      return load()
    }
  } catch (e) {
    /* 查询失败：保留现有数据，不抛出以免中断其它模块加载 */
  } finally {
    loading.value = false
  }
}

// 趋势图时间选择：自定义起止日期优先，否则按预设天数
// 最近一次趋势数据：总览页签重新激活后用于重绘图表
const lastDaily = ref([])

async function loadStats() {
  const params = {}
  if (dashRange.value && dashRange.value.length === 2) {
    params.start = dashRange.value[0]
    params.end = dashRange.value[1]
  } else {
    params.days = dashPreset.value
  }
  try {
    // 静默请求：概览统计是非关键数据（服务重启 / 迁移竞态时可能瞬时失败），
    // 失败不弹全局错误提示，保留上次数据，重新进入或切换页签会自动重试
    const res = await adminApi.stats(params, { silent: true })
    summary.value = res.data.summary || summary.value
    dashRangeStart.value = res.data?.rangeStart || ''
    dashRangeEnd.value = res.data?.rangeEnd || ''
    renderCharts(res.data.daily || [])
  } catch (e) {
    /* 统计失败不影响其它模块：不弹 toast 打扰运营，但必须在控制台留痕，
       否则后端统计接口一旦报错，页面会一直静默显示全 0 而无人察觉 */
    console.warn('[会员总览] 统计数据加载失败：', e?.message || e)
  }
}

// 快捷时间选择：将实际起止日期写入时间控件，使其内部直接显示日期文案
function onDashPreset(val) {
  const n = Number(val)
  const end = dayjs()
  const start = end.subtract(n - 1, 'day')
  dashRange.value = [start.format('YYYY-MM-DD'), end.format('YYYY-MM-DD')]
  loadStats()
}

function onDashRange(val) {
  if (val && val.length === 2) {
    dashPreset.value = null // 手动选择后取消快捷高亮
    loadStats()
  }
}

// 菜单排序可编辑列表（总览固定置顶）
const menuEditItems = computed(() => {
  const all = TAB_META.map((t) => ({ key: t.name, label: t.label, locked: t.name === 'overview' }))
  const order = manualOrder.value.length ? manualOrder.value : TAB_META.map((t) => t.name)
  return order.map((n) => all.find((t) => t.key === n)).filter(Boolean)
})

function onMenuReorder(val) {
  manualOrder.value = val.map((x) => x.key)
  smartOrder.value = false
}

function renderCharts(daily) {
  lastDaily.value = daily || []
  const dates = daily.map((d) => d.date.slice(5))
  const regs = daily.map((d) => d.registerCount)
  const payments = daily.map((d) => d.paidCount || 0)

  barChart?.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 16, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: dates, axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      {
        name: '注册人数',
        type: 'bar',
        data: regs,
        barMaxWidth: 26,
        itemStyle: { color: '#ff7a45', borderRadius: [4, 4, 0, 0] }
      }
    ]
  })

  lineChart?.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 16, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: dates, boundaryGap: false, axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      {
        name: '已开通会员',
        type: 'line',
        data: payments,
        smooth: true,
        symbolSize: 6,
        itemStyle: { color: '#2f6fed' },
        areaStyle: { color: 'rgba(47,111,237,0.12)' }
      }
    ]
  })
}

async function toggleFreeze(row) {
  const frozen = row.status !== 'frozen'
  await ElMessageBox.confirm(`确认${frozen ? '冻结' : '解冻'}账号「${row.username}」？`, '提示', { type: 'warning' })
  await adminApi.freeze(row.id, frozen)
  ElMessage.success(frozen ? '已冻结' : '已解冻')
  load()
  loadStats()
}

async function setRole(row, role) {
  const text = role === 'admin' ? '设为管理员' : '取消管理员'
  await ElMessageBox.confirm(`确认将「${row.username}」${text}？`, '提示', { type: 'warning' })
  await adminApi.setRole(row.id, role)
  ElMessage.success(`已${text}`)
  load()
}

// 删除账号（仅站长）：不可撤销，账号登录信息与附属数据一并清除
async function removeUser(row) {
  await ElMessageBox.confirm(
    `确认删除账号「${row.username}」？删除后该账号将无法登录，且不可恢复。`,
    '删除确认',
    { type: 'error', confirmButtonText: '确认删除', confirmButtonClass: 'el-button--danger' }
  )
  await adminApi.removeUser(row.id)
  ElMessage.success('账号已删除')
  load()
  loadStats()
}

// 页签激活时加载对应数据：不依赖 onMounted 的串行链路，避免「徽标有数字但列表为空」
watch(
  activeTab,
  (val) => {
    if (val === 'users') load()
  },
  { immediate: true }
)

// 发送站内信
const msgForm = reactive({ target: 'all', username: '', type: 'system', title: '', content: '' })
const msgSending = ref(false)
async function sendMessage() {
  if (!msgForm.title.trim() || !msgForm.content.trim()) {
    ElMessage.warning('请填写标题与内容')
    return
  }
  let bannedMsg = validateTexts({ [msgForm.title]: '站内信标题', [msgForm.content]: '站内信内容' })
  if (bannedMsg) {
    ElMessage.warning(bannedMsg)
    return
  }
  let userId = 0
  if (msgForm.target === 'user') {
    const name = msgForm.username.trim()
    if (!name) {
      ElMessage.warning('请输入接收用户名')
      return
    }
    const res = await adminApi.users({ keyword: name, pageSize: 50 })
    const found = (res.data?.list || []).find((u) => u.username === name)
    if (!found) {
      ElMessage.warning('未找到该用户')
      return
    }
    userId = found.id
  }
  msgSending.value = true
  try {
    await messageApi.send({
      userId,
      title: msgForm.title.trim(),
      content: msgForm.content.trim(),
      type: msgForm.type
    })
    ElMessage.success(msgForm.target === 'all' ? '已向全员发送站内信' : '站内信已发送')
    msgForm.title = ''
    msgForm.content = ''
    msgForm.username = ''
  } finally {
    msgSending.value = false
  }
}

function openRenew(row) {
  renewUser.value = row
  renewForm.amount = 0
  renewForm.period = 'monthly'
  renewForm.days = 30
  renewForm.remark = ''
  renewVisible.value = true
}

async function submitRenew() {
  const bannedMsg = validateText(renewForm.remark, '备注')
  if (bannedMsg) {
    ElMessage.warning(bannedMsg)
    return
  }
  if (renewForm.period === 'daily') {
    const days = Math.floor(Number(renewForm.days) || 0)
    if (days < 1 || days > 3650) {
      ElMessage.warning('按天续费请填写 1-3650 天')
      return
    }
  }
  saving.value = true
  try {
    await adminApi.renew(renewUser.value.id, {
      amount: Number(renewForm.amount || 0),
      period: renewForm.period,
      days: renewForm.period === 'daily' ? Math.floor(Number(renewForm.days) || 0) : 0,
      remark: sanitizeText(renewForm.remark)
    })
    ElMessage.success('续费成功')
    renewVisible.value = false
    load()
    loadStats()
  } finally {
    saving.value = false
  }
}

function onResize() {
  barChart?.resize()
  lineChart?.resize()
}

// 图表按需初始化：容器可能尚未渲染（例如记忆页签直接停在其它页签），此时 ref 为空，
// 直接 echarts.init(null) 会抛错并中断 onMounted 后续初始化，造成「页签有徽标但列表为空」。
// 注意：模板 ref 位于 el-tab-pane 的 v-for 内，Vue 会收集成数组，需取第一个元素。
function refEl(v) {
  if (Array.isArray(v)) return v[0] || null
  return v || null
}
function initCharts() {
  const bar = refEl(barRef.value)
  const line = refEl(lineRef.value)
  if (!barChart && bar) barChart = echarts.init(bar)
  if (!lineChart && line) lineChart = echarts.init(line)
}

// 切回「会员总览」时补初始化、重绘并重新计算尺寸（避免隐藏期间尺寸为 0）
watch(activeTab, (v) => {
  if (v === 'overview') {
    nextTick(() => {
      initCharts()
      if (lastDaily.value.length) renderCharts(lastDaily.value)
      onResize()
    })
  }
})

onMounted(async () => {
  initCharts()
  window.addEventListener('resize', onResize)
  // 回到页面 / 切回本页时刷新待办徽标：进入管理中心即开始检测待审核数量，无需手动刷新
  window.addEventListener('focus', loadBadges)
  // 定时检测（每分钟）：有待审核记录时数字角标常显，无需管理员手动刷新页面
  badgeTimer = window.setInterval(loadBadges, 60000)
  // 总览统计 + 页签徽标；列表类数据由各页签激活时按需加载（见 activeTab 监听），避免重复请求。
  // 用 allSettled 并行加载：任一失败不影响另一个，也不会中断后续流程。
  await Promise.allSettled([loadStats(), loadBadges()])
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.removeEventListener('focus', loadBadges)
  if (badgeTimer) {
    window.clearInterval(badgeTimer)
    badgeTimer = null
  }
  barChart?.dispose()
  lineChart?.dispose()
})

watch(page, () => load())
</script>

<style scoped>
.admin-tabs {
  margin-top: 4px;
}

/* 二级导航工具条：菜单排序入口 */
.tab-tools {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
  margin: 10px 0 2px;
  min-height: 24px;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 2px;
  cursor: inherit;
}

.tab-label.dragging {
  opacity: 0.5;
}

.tab-label.over {
  box-shadow: inset 0 -2px 0 var(--el-color-primary);
}

/* 会员续费弹窗 */
.renew-dialog .renew-user {
  margin-bottom: 16px;
  padding: 12px 14px;
  border-radius: 10px;
  background: #f7f9ff;
  border: 1px solid #e8eeff;
}

.renew-dialog .ru-main {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.renew-dialog .ru-label {
  font-size: 13px;
  color: #8a94a6;
}

.renew-dialog .ru-name {
  color: #303133;
}

.renew-dialog .ru-meta {
  margin-top: 6px;
  font-size: 12px;
  color: #8a94a6;
}

.renew-dialog .amount-row,
.renew-dialog .period-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.renew-dialog .days-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.renew-dialog .quick-days {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.renew-dialog .unit {
  font-size: 13px;
  color: #8a94a6;
}

.renew-dialog .renew-preview {
  padding: 12px 14px;
  border-radius: 10px;
  background: #fff8f2;
  border: 1px solid #ffe3d3;
  font-size: 13px;
  color: #606266;
}

.renew-dialog .renew-preview b {
  margin-left: 6px;
  font-size: 16px;
  color: #ff7a45;
}

/* 身份标签：柔和胶囊，去掉边框并加宽内边距，长文案不再被列宽截断 */
.id-tag {
  height: 22px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.2px;
  border: none;
  white-space: nowrap;
}

/* 普通用户按使用身份：老师（绿） / 学员（蓝） / 机构（青） */
.id-tag.is-pro {
  background: #eaf7ee;
  color: #37985a;
}

.id-tag.is-user {
  background: #eaf2ff;
  color: #3b7fd4;
}

.id-tag.is-org {
  background: #e6f7f5;
  color: #16a589;
}

/* 后台身份：站长（橙）与管理员（紫），与角色标签区分 */
.id-tag.is-owner {
  background: #fff3e6;
  color: #e08214;
}

.id-tag.is-admin {
  background: #f6f0ff;
  color: #7a5bd6;
}

/* 用户列表操作列：设为管理员独立一行，压缩列宽 */
.op-stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.5;
}

.op-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.op-row :deep(.el-button + .el-button) {
  margin-left: 0;
}

/* 待办角标：内联胶囊，稳定可见（不依赖组件内部定位/过渡） */
.tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--el-color-danger);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  box-shadow: 0 0 0 2px #fff;
}

/* 价格设置：总价预览板块（与首页同源，供运营观察效果） */
.price-preview {
  margin-bottom: 16px;
  padding: 16px 18px;
  border-radius: 12px;
  border: 1px solid #e3ebff;
  background: linear-gradient(180deg, #f7f9ff 0%, #ffffff 100%);
}

.price-preview .pv-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 4px;
}

.price-preview .pv-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.price-preview .pv-title::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: #2f6fed;
}

.price-preview .pv-sub {
  font-size: 12px;
  font-weight: 400;
  color: #a0a6b1;
}

/* 标签组：置于价格表右上角 */
.price-preview .pv-tags-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 8px;
  min-height: 24px;
}

/* 未保存提示：橙色胶囊 + 脉冲点（与折扣标签同形，靠圆点与配色区分「状态」） */
.price-preview .pv-unsaved {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 9px;
  border-radius: 12px;
  border: 1px solid #f5dab1;
  background: #fdf6ec;
  font-size: 12px;
  line-height: 1;
  color: #e6a23c;
}

.price-preview .pv-unsaved .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f0a020;
  animation: pc-pulse 1.4s ease-in-out infinite;
}

.price-preview .pv-grid {
  display: grid;
  grid-template-columns: 96px repeat(3, 1fr);
  border: 1px solid #e8eef7;
  border-radius: 10px;
  overflow: hidden;
  background: #fff;
}

.price-preview .pv-cell {
  padding: 10px 12px;
  border-bottom: 1px solid #eef2f8;
  font-size: 13px;
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.price-preview .pv-cell:nth-child(4n + 2),
.price-preview .pv-cell:nth-child(4n + 3),
.price-preview .pv-cell:nth-child(4n + 4) {
  border-left: 1px solid #eef2f8;
}

.price-preview .pv-grid > .pv-cell:nth-last-child(-n + 4) {
  border-bottom: none;
}

.price-preview .pv-th {
  background: #f5f8ff;
  color: #8a94a6;
  font-size: 12px;
  font-weight: 500;
}

.price-preview .pv-name {
  color: #606266;
}

.price-preview .pv-price .pv-eff {
  font-size: 16px;
  font-weight: 600;
  color: #2f6fed;
  white-space: nowrap;
}

.price-preview .pv-price .pv-unit {
  margin-left: 2px;
  font-size: 12px;
  font-style: normal;
  font-weight: 400;
  color: #a0a6b1;
}

.price-preview .pv-price .pv-origin {
  font-size: 12px;
  color: #c0c4cc;
  text-decoration: line-through;
  white-space: nowrap;
}

.price-preview .pv-note {
  margin-top: 10px;
  font-size: 12px;
  line-height: 1.6;
  color: #a0a6b1;
}

/* 价格软提醒：组合未形成优惠 / 折上折过低，仅提示不阻断保存 */
.price-preview .pv-warns {
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid #ffe0b2;
  border-radius: 8px;
  background: #fff8ee;
}

.price-preview .pv-warn-item {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: #b77414;
}

.price-preview .pv-warn-item + .pv-warn-item {
  margin-top: 4px;
}

.price-preview .pv-warn-dot {
  flex: none;
  width: 14px;
  height: 14px;
  margin-top: 2px;
  border-radius: 50%;
  background: #f0a020;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 14px;
  text-align: center;
}

.price-card .pc-warn {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: #b77414;
}

/* 待生效计划 / 变更历史 */
.price-panel {
  margin-bottom: 18px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #ebeef3;
  border-radius: 12px;
  box-shadow: 0 1px 4px rgba(31, 41, 55, 0.04);
}

.price-panel .pp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.price-panel .pp-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.price-panel .pp-sub {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 400;
  color: #a0a6b1;
}

.price-panel .pp-summary {
  font-size: 12px;
  line-height: 1.6;
  color: #606266;
  word-break: break-all;
}

.price-panel .pp-empty {
  font-size: 12px;
  color: #a0a6b1;
}

/* 价格设置：卡片式分组 */
.price-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.price-card {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px 18px;
  background: #fff;
  border: 1px solid #ebeef3;
  border-radius: 12px;
  box-shadow: 0 1px 4px rgba(31, 41, 55, 0.04);
  transition: box-shadow 0.2s;
}

.price-card:hover {
  box-shadow: 0 4px 14px rgba(31, 41, 55, 0.08);
}

.price-card .pc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.price-card .pc-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px dashed #ebeef3;
  font-weight: 600;
  font-size: 14px;
  color: #303133;
  flex: 1;
}

.price-card.pc-off .pc-title,
.price-card.pc-off .pc-label {
  color: #a0a6b1;
}

.price-card .pc-title::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: #ff7a45;
}

.price-card .pc-sub {
  font-weight: 400;
  font-size: 12px;
  color: #8a94a6;
}

.price-card .pc-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.price-card .pc-item:last-of-type {
  margin-bottom: 0;
}

/* 数值输入统一宽度，保证左右两列各行齐平 */
.price-card .pc-item .el-input-number {
  width: 150px;
}

.price-card .pc-label {
  flex-shrink: 0;
  width: 60px;
  font-size: 13px;
  color: #606266;
}

.price-card .pc-unit {
  flex-shrink: 0;
  font-size: 13px;
  color: #8a94a6;
}

.price-card .pc-note {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed #ebeef3;
  font-size: 12px;
  line-height: 1.6;
  color: #a0a6b1;
}

.price-card .pc-foot {
  margin-top: auto; /* 贴底：同一行各卡片页脚纵向对齐 */
  padding-top: 12px;
  border-top: 1px solid #eef1f6;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.price-card .pc-foot-btns {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.price-card .pc-foot-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-width: 0;
}

.price-card .pc-foot-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.price-card .pc-pending {
  font-size: 12px;
  line-height: 1.4;
  color: #e6a23c;
}

/* 有未保存修改的卡片：橙色描边 + 标题角标，提示「这里改过但还没保存」 */
.price-card.pc-dirty {
  border-color: #f5c26b;
  box-shadow: 0 0 0 2px rgba(240, 160, 32, 0.1);
}

.price-card .pc-unsaved {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  line-height: 1.4;
  color: #e6a23c;
}

.price-card .pc-unsaved::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f0a020;
  animation: pc-pulse 1.4s ease-in-out infinite;
}

.price-card .pc-dirty-dot {
  flex-shrink: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #f0a020;
  box-shadow: 0 0 0 3px rgba(240, 160, 32, 0.15);
  animation: pc-pulse 1.4s ease-in-out infinite;
}

@keyframes pc-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.35;
  }
}

.price-actions {
  display: flex;
  justify-content: flex-end;
}

.pane-desc {
  font-size: 13px;
  color: var(--brand-muted);
  margin-bottom: 12px;
}

.chart-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 18px 0 10px;
}

.chart-head .chart-title {
  margin-bottom: 0;
}


.cert-row {
  display: flex;
  align-items: center;
  gap: 14px;
}

.cert {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.cert-img {
  width: 56px;
  height: 56px;
  border-radius: 6px;
  border: 1px solid var(--brand-line);
  background: #fff;
  cursor: zoom-in;
}

.cert-label {
  font-size: 11px;
  color: var(--brand-muted);
}

.stat-row {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.stat {
  flex: 1 1 140px;
  padding: 14px 16px;
  border-radius: 10px;
  background: #fafbfc;
}

.stat b {
  display: block;
  font-size: 22px;
  color: var(--el-color-primary-dark-2);
}

.stat b.ok {
  color: var(--brand-success);
}

.stat b.danger {
  color: var(--brand-danger);
}

.stat b.money {
  color: var(--brand-success);
}

.money-cell {
  color: var(--brand-success);
  font-weight: 600;
}

.stat span {
  font-size: 12px;
  color: var(--brand-muted);
}

.chart-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 16px;
}

.chart-box {
  border: 1px solid var(--brand-line);
  border-radius: 12px;
  padding: 12px;
}

.chart-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}

.chart {
  width: 100%;
  height: 260px;
}

.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.push-right {
  margin-left: auto;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

/* 列 / 菜单排序弹层 */
.col-set-title {
  font-size: 12px;
  color: var(--brand-muted);
  margin-bottom: 10px;
}

.col-set-foot {
  margin-top: 8px;
  text-align: right;
}

.chart-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.dash-range {
  font-size: 12px;
  color: var(--brand-muted);
}
</style>
