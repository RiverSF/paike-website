<template>
  <div class="page">
    <template v-if="mainVisible">
      <section class="card">
        <div class="card-title">
          课程安排
          <span class="sub" v-show="activeTab === 'list'">共 {{ total }} 门课程，状态可直接在列表中切换</span>
        </div>

        <!-- 试用 / 过期提示条：免注册试用引导注册保存；会员过期只读提示续费 -->
        <div v-if="trialBanner" class="trial-banner" :class="trialBanner.type">
          <span>{{ trialBanner.text }}</span>
          <el-button v-if="trialStore.isGuest" type="primary" size="small" round @click="trialStore.openRegisterGuide">
            注册保存数据
          </el-button>
          <el-button v-else type="warning" size="small" round @click="needMember">去续费</el-button>
        </div>

        <div class="nav-tabs">
          <!-- 费用统计为注册用户进阶功能：免注册试用与过期账号不展示 -->
          <button v-if="!dashLocked" class="nav-tab" :class="{ active: activeTab === 'dashboard' }" @click="onTab('dashboard')">数据仪表盘</button>
          <button class="nav-tab" :class="{ active: activeTab === 'list' }" @click="onTab('list')">课程列表</button>
        </div>

        <div v-show="activeTab === 'list'">
        <div class="filters">
          <el-input
            v-model="filters.keyword"
            :placeholder="lv.keywordPlaceholder"
            clearable
            style="width: 260px"
            @keyup.enter="load"
            @clear="load"
          />
          <el-select v-model="filters.subject" placeholder="全部科目" clearable filterable style="width: 140px" @change="load">
            <el-option v-for="s in subjectOptions" :key="s" :label="s" :value="s" />
          </el-select>
          <el-select v-model="filters.grade" :placeholder="lv.gradeFilterLabel" clearable filterable style="width: 140px" @change="load">
            <el-option v-for="g in lv.gradeFilterOptions" :key="g" :label="g" :value="g" />
          </el-select>
          <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 140px" @change="load">
            <el-option v-for="(v, k) in statusMap" :key="k" :label="v" :value="k" />
          </el-select>
          <el-button @click="load">查询</el-button>
        </div>

        <div class="order-legend">
          <el-button type="primary" :icon="Plus" :disabled="readonlyMode" @click="openCreate">添加课程</el-button>
          <span v-if="trialStore.isGuest" class="legend-item trial-quota">
            试用最多添加 3 门课程，还可添加 <b>{{ trialStore.remaining }}</b> 门
          </span>
          <span class="legend-item">
            <i class="legend-block lg-pending" />未开始
          </span>
          <span class="legend-item">
            <i class="legend-block lg-running" />进行中
          </span>
          <span class="legend-item">
            <i class="legend-block lg-finished" />已结束
          </span>
          <div class="flex-spacer" />
          <el-popover placement="bottom-end" :width="300" trigger="click">
            <template #reference>
              <el-button :icon="Operation">列设置</el-button>
            </template>
            <div class="col-set">
              <div class="col-set-title">拖动 / 点箭头调整顺序，勾选控制显示</div>
              <ReorderList :model-value="colOrderItems" @update:model-value="onColumnsReorder" />
              <div class="col-vis">
                <el-checkbox
                  v-for="c in settingColumns"
                  :key="c.key"
                  v-model="visibleMap[c.key]"
                  size="small"
                  @change="persistColumns"
                >
                  {{ c.label }}
                </el-checkbox>
              </div>
              <div class="col-set-foot">
                <el-button text size="small" @click="resetColumns">恢复默认</el-button>
              </div>
            </div>
          </el-popover>
        </div>
        </div>

        <!-- 仪表盘：订单数 / 预计收益 / 真实收益 / 课程节数 + 趋势图，支持时间选择 -->
        <div v-show="activeTab === 'dashboard'">
        <div class="dashboard">
          <div class="dash-head">
            <div class="dash-title">数据仪表盘</div>
            <div class="dash-tools">
              <!-- 统计维度：按课时（一节课为单位）/ 按课程（一门课为单位），金额口径一致 -->
              <el-radio-group v-model="dashDim" size="small">
                <el-radio-button value="lesson">按课时</el-radio-button>
                <el-radio-button value="course">按课程</el-radio-button>
              </el-radio-group>
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

          <div class="stat-row">
            <!-- 主指标随维度切换：按课时看课时数，按课程看课程数 -->
            <div class="stat">
              <b>{{ dashPrimary }}</b>
              <span>{{ dashDim === 'lesson' ? '课时数' : '课程数' }}</span>
            </div>
            <div class="stat">
              <b class="money">¥{{ money(dashSummary.estimatedIncome) }}</b>
              <span>{{ dashMoneyLabel }}</span>
            </div>
            <div class="stat">
              <b class="ok">¥{{ money(dashSummary.realIncome) }}</b>
              <span>{{ dashRealMoneyLabel }}</span>
            </div>
            <div class="stat">
              <b>{{ dashSecondary }}</b>
              <span>{{ dashDim === 'lesson' ? '涉及课程数' : '课时数' }}</span>
            </div>
          </div>

          <div class="chart-box">
            <div class="chart-title">
              {{ dashTrendTitle }}
              <span class="muted small">{{ dashTrendHint }}</span>
            </div>
            <div ref="dashRef" class="chart" />
          </div>
        </div>
        </div>

        <div v-show="activeTab === 'list'">
        <el-table v-loading="loading" :data="list" border stripe :row-class-name="rowClass">
          <el-table-column
            v-for="col in visibleColumns"
            :key="col.key"
            :label="col.label"
            :width="col.width"
            :min-width="col.minWidth"
            :align="col.align || 'left'"
            :fixed="col.fixed"
            :class-name="col.className"
          >
            <template #default="{ row }">
              <template v-if="col.key === 'orderNo'">
                <span class="cell-main">{{ row.orderNo || '自动生成' }}</span>
              </template>

              <template v-else-if="col.key === 'student'">
                <div class="cell-main">{{ row.grade || '-' }}</div>
                <div class="cell-sub">
                  {{ row.studentName || '未填写' }}
                  <el-tag v-if="row.studentGender && lv.showStudentGender" size="small" effect="plain" round>{{ row.studentGender }}</el-tag>
                </div>
              </template>

              <template v-else-if="col.key === 'subjects'">
                <span v-if="subjectText(row)" class="subject-tags">
                  <el-tag v-for="s in subjectText(row).split('、')" :key="s" size="small" effect="light" round>{{ s }}</el-tag>
                </span>
                <span v-else class="muted">-</span>
              </template>

              <template v-else-if="col.key === 'address'">
                <span :title="row.address">{{ row.address || '-' }}</span>
              </template>

              <template v-else-if="col.key === 'period'">
                <span>{{ row.startDate }}</span>
                <span class="muted"> ~ </span>
                <span>{{ row.endDate || '长期' }}</span>
              </template>

              <template v-else-if="col.key === 'progress'">
                <!-- 已上 / 预计节数；长期课未填总课时时只显示已上节数 -->
                <div class="progress-cell">
                  <span class="progress-text">
                    已上 <b>{{ row.progress?.done || 0 }}</b>
                    <template v-if="row.progress?.planned">/ {{ row.progress.planned }} 节</template>
                    <template v-else>节</template>
                  </span>
                  <el-progress
                    v-if="row.progress?.planned"
                    :percentage="progressPercent(row)"
                    :show-text="false"
                    :stroke-width="6"
                    :color="progressColor(row)"
                  />
                </div>
              </template>

              <template v-else-if="col.key === 'slots'">
                <div v-for="(s, i) in slotTexts(row)" :key="i" class="slot-text">{{ s }}</div>
                <span v-if="!slotTexts(row).length" class="muted">-</span>
              </template>

              <template v-else-if="col.key === 'price'">
                <div>{{ priceText(row) }}</div>
              </template>

              <template v-else-if="col.key === 'source'">
                <div>{{ row.source || '-' }}</div>
                <div class="cell-sub">{{ row.publisher || '-' }}</div>
              </template>

              <template v-else-if="col.key === 'remark'">
                <el-tooltip v-if="row.remark" placement="top" :content="row.remark" :show-after="200">
                  <span class="remark-dot" :class="{ flag: row.remarkFlag }">
                    <el-icon><ChatDotSquare /></el-icon>
                    <i v-if="row.remarkFlag" class="red-dot" />
                  </span>
                </el-tooltip>
                <span v-else class="muted">-</span>
              </template>

              <template v-else-if="col.key === 'status'">
                <!-- 状态胶囊：圆点 + 文案；进行中可点击切换，未开始 / 已结束为静态展示 -->
                <el-dropdown
                  v-if="row.status === 'running'"
                  trigger="click"
                  @command="(v) => changeStatus(row, v)"
                >
                  <span class="status-pill is-running"><i class="dot" />{{ statusMap[row.status] }}</span>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item v-for="k in statusOptions(row)" :key="k" :command="k">
                        {{ statusMap[k] }}
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
                <span v-else class="status-pill" :class="`is-${row.status}`">
                  <i class="dot" />{{ statusMap[row.status] }}
                </span>
              </template>

              <template v-else-if="col.key === 'op'">
                <el-button link type="primary" @click="openEdit(row)">
                  {{ row.status === 'finished' ? '查看' : '编辑' }}
                </el-button>
                <el-button
                  v-if="!readonlyMode"
                  link
                  type="danger"
                  :disabled="row.status === 'running' || row.status === 'finished'"
                  @click="remove(row)"
                  >删除</el-button
                >
              </template>
            </template>
          </el-table-column>

          <template #empty>
            <el-empty description="还没有课程，点击右上角「添加课程」开始" />
          </template>
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
        </div>
      </section>
    </template>

    <!-- 新增 / 编辑订单（已结束订单仅可查看） -->
    <el-drawer
      v-model="drawer"
      :title="readonly ? '查看课程' : editing ? '编辑课程' : '添加课程'"
      size="640px"
      close-on-click-modal
    >
      <el-form :model="form" label-width="92px" class="order-form" :disabled="readonly">
        <template v-if="fv.showBasic">
          <div class="form-section">
          <div class="section-head">{{ fv.basicTitle }}</div>
          <el-row :gutter="16">
            <el-col v-if="fv.showSource" :span="12" :xs="24">
              <el-form-item label="信息来源">
                <el-input v-model="form.source" placeholder="如：某某家教群 / 中介" />
              </el-form-item>
            </el-col>
            <el-col v-if="fv.showSource" :span="12" :xs="24">
              <el-form-item label="发布人">
                <el-input v-model="form.publisher" placeholder="如：王老师" />
              </el-form-item>
            </el-col>
            <el-col v-if="fv.showSource" :span="12" :xs="24">
              <el-form-item label="发布时间">
                <el-date-picker
                  v-model="form.publishedAt"
                  type="date"
                  value-format="YYYY-MM-DD"
                  placeholder="选择发布时间"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
            <el-col v-if="fv.showOrderNo" :span="12" :xs="24">
              <el-form-item label="课程编号">
                <el-input v-model="form.orderNo" :placeholder="fv.orderNoPlaceholder" />
              </el-form-item>
            </el-col>
          </el-row>
          </div>
        </template>

        <div class="form-section">
        <div class="section-head">{{ fv.studentTitle }}</div>
        <el-row :gutter="16">
          <el-col :span="12" :xs="24">
            <el-form-item :label="fv.gradeLabel" required>
              <el-select v-model="form.grade" filterable allow-create default-first-option placeholder="选择或输入" style="width: 100%">
                <el-option v-for="g in fv.gradeOptions" :key="g" :label="g" :value="g" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item label="科目">
              <el-select
                v-model="form.subjects"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="选填，可多选"
                style="width: 100%"
              >
                <el-option v-for="s in subjectOptions" :key="s" :label="s" :value="s" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item :label="fv.studentLabel">
              <el-input v-model="form.studentName" :placeholder="fv.studentPlaceholder" />
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item v-if="fv.showStudentGender" label="学生性别">
              <el-radio-group v-model="form.studentGender">
                <el-radio value="男">男</el-radio>
                <el-radio value="女">女</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12" :xs="24">
            <el-form-item :label="fv.phoneLabel" :class="{ 'has-error': phoneError }">
              <el-input
                v-model="form.contactPhone"
                :placeholder="fv.phonePlaceholder"
                maxlength="11"
                @input="validatePhone"
              />
              <div v-if="phoneError" class="field-error">{{ phoneError }}</div>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="地址" required>
              <el-input v-model="form.address" placeholder="如：xx 小区 x 栋" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="辅导内容">
              <el-input v-model="form.content" type="textarea" :rows="2" placeholder="如：初二数学同步辅导 + 错题讲解" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item :label="fv.situationLabel">
              <el-input v-model="form.studentSituation" type="textarea" :rows="2" :placeholder="fv.situationPlaceholder" />
            </el-form-item>
          </el-col>
        </el-row>
        </div>

        <div class="form-section">
          <div class="section-head">
            {{ fv.lessonTitle }}
            <span class="section-note">日期含当天，结束日期留空=长期；填写总课时可自动推算上到哪天</span>
          </div>
          <el-row :gutter="16">
            <!-- 两列栅格：所有输入控件都撑满各自列宽（width:100%），
                 对齐由栅格保证，不依赖固定 px，避免出现右侧留白导致的参差 -->
            <el-col :span="12" :xs="24">
              <el-form-item label="开始日期" required>
                <el-date-picker v-model="form.startDate" type="date" value-format="YYYY-MM-DD" placeholder="开始上课" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="12" :xs="24">
              <el-form-item label="结束日期">
                <el-date-picker v-model="form.endDate" type="date" value-format="YYYY-MM-DD" placeholder="留空 = 长期" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="课程费用" required>
                <div class="price-input">
                  <el-input-number
                    v-model="activePrice"
                    :min="0"
                    :precision="0"
                    :controls="false"
                    class="money-input"
                  />
                  <!-- 计费单位：紧跟金额输入框，宽度固定，不参与输入框对齐 -->
                  <el-select v-model="priceMode" class="price-unit" @change="onPriceModeChange">
                    <el-option value="hourly" label="元/时" />
                    <el-option value="lesson" label="元/次" />
                    <el-option value="total" label="总费用" />
                  </el-select>
                  <el-tooltip v-if="priceMode === 'total'" placement="top" :width="260" :content="totalPriceTip">
                    <el-icon class="price-help"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="总课时">
                <div class="price-input">
                  <el-input-number
                    v-model="form.totalLessons"
                    :min="0"
                    :max="2000"
                    :controls="false"
                    class="money-input"
                  />
                  <span class="unit-text">节</span>
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="周时间段" required>
                <WeeklySlotEditor v-model="form.weeklySlots" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="课时进度">
                <div class="progress-hint">
                  <span class="progress-chip" :class="{ 'is-empty': !form.totalLessons }">{{ totalLessonsTip }}</span>
                  <el-button v-if="endEstimate" link type="primary" class="estimate-btn" @click="applyEndEstimate">
                    填入结束日期
                  </el-button>
                </div>
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <div class="form-section">
          <div class="section-head">{{ fv.extraTitle }}</div>
          <el-form-item label="备注">
            <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="特殊要求、注意事项等" />
          </el-form-item>
          <el-form-item label="特殊备注">
            <el-switch v-model="form.remarkFlag" active-text="课表红点标记" />
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="form.status" :disabled="editing?.status === 'finished'">
              <el-radio-button v-for="(v, k) in editableStatusMap" :key="k" :value="k">{{ v }}</el-radio-button>
            </el-radio-group>
            <span v-if="editing?.status === 'finished'" class="muted small" style="margin-left:10px">
              已结束课程不可再切换状态
            </span>
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="drawer = false">{{ readonly ? '关闭' : '取消' }}</el-button>
        <el-button v-if="!readonly" type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ChatDotSquare, Operation, Plus, QuestionFilled } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import dayjs from 'dayjs'
import WeeklySlotEditor from '@/components/WeeklySlotEditor.vue'
import ReorderList from '@/components/ReorderList.vue'
import { orderApi } from '@/api'
import { useNavTab } from '@/stores/navTab'
import { useUserStore } from '@/stores/user'
import { useTrialStore, TRIAL_COURSE_LIMIT } from '@/stores/trial'
import { sanitizeText, validateTexts } from '@/utils/text'

const userStore = useUserStore()

// 订单状态：进行中（列表置顶并高亮） / 已结束
const statusMap = {
  pending: '未开始',
  running: '进行中',
  finished: '已结束'
}
// 表单中可手动选择的状态（未开始由开始日期自动判定，不在此暴露，避免误设）
const editableStatusMap = {
  running: '进行中',
  finished: '已结束'
}
const defaultGrades = ['小班', '中班', '大班', '一年级', '二年级', '三年级', '四年级', '五年级', '六年级', '初一', '初二', '初三', '高一', '高二', '高三', '成人']
const defaultSubjects = ['语文', '数学', '英语', '物理', '化学', '生物', '政治', '历史', '地理', '奥数', '钢琴', '书法', '美术', '编程', '其他']

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const filters = reactive({ keyword: '', status: '', grade: '', subject: '', direction: '' })
const gradeOptions = ref(defaultGrades)
const subjectOptions = ref(defaultSubjects)

const drawer = ref(false)
const editing = ref(null)
const saving = ref(false)
// 已结束订单：抽屉只读，仅可查看
const readonly = ref(false)

// 字段口径由「使用身份」决定：
//   老师 / 机构 → 收入（我提供服务，需记录学员与接单来源）；
//   家长 / 个人 → 已融合为一个「付费方」身份，统一按开支（我购买服务）渲染，
//                 年级 / 课程类型均可填，覆盖给孩子排课与给自己排课两类场景。
// 方向不再手动选择，固定跟随身份；字段文案与显隐也随之适配，不改变数据结构与计费口径。
const personalTypes = ['兴趣课', '技能训练', '健身运动', '语言学习', '艺术修养', '考级备考', '其他']
const fv = computed(() => {
  const role = userStore.userRole
  const isTeacher = role === 'teacher'
  const isOrg = role === 'org'
  // 家长与个人已融合为一个「付费方」身份（均为开支口径）
  const isPayer = role === 'parent' || role === 'personal'
  // 方向固定跟随身份：老师 / 机构=收入，付费方=开支
  const isIncome = defaultDirection.value === 'income'
  // 接单来源：只有「我收的课」且从外部接单的老师 / 机构才需要记录
  const showSource = (isTeacher || isOrg) && isIncome
  return {
    isIncome,
    showBasic: true,
    showSource,
    // 付费方（家长 / 个人）也保留课程编号概念，统一与老师 / 机构一致
    showOrderNo: true,
    orderNoPlaceholder: showSource ? '选填，留空自动生成' : '自定义编号',
    // 学员主体：收入时是我的学生；开支时是我购买的课程所对应的上课人（孩子或本人）
    studentLabel: isIncome ? '学生姓名' : '学员姓名',
    studentPlaceholder: isIncome ? '如：小明' : '如：小明 / 钢琴课',
    showStudentGender: true,
    // 联系电话：收入时对接学员家长，开支时对接授课老师
    phoneLabel: isIncome ? '家长电话' : '老师电话',
    phonePlaceholder: isIncome ? '家长/学生联系电话' : '老师联系电话',
    // 年级 / 课程类型：付费方两类场景都支持
    gradeLabel: isPayer ? '年级 / 类型' : '年级',
    gradeOptions: isPayer ? [...gradeOptions.value, ...personalTypes] : gradeOptions.value,
    // 学生情况：收入时记录学员基础，开支时写自己/孩子想达到的目标
    situationLabel: isIncome ? '学生情况' : '学习目标',
    situationPlaceholder: isIncome ? '学习成绩、性格、薄弱环节等' : '想达到的效果、当前进度等',
    // 区块标题：按口径命名，避免「课时安排」里夹着费用这类名实不符
    basicTitle: showSource ? '来源与编号' : '课程编号',
    studentTitle: '学员信息',
    lessonTitle: '课时与费用',
    extraTitle: '备注与状态'
  }
})

// 列表列 / 筛选 / 仪表盘的展示口径同样随身份适配（与表单 fv 同源，保证同一身份口径一致）
const lv = computed(() => {
  const role = userStore.userRole
  const isTeacher = role === 'teacher'
  const isOrg = role === 'org'
  // 家长与个人已融合为一个「付费方」身份
  const isPayer = role === 'parent' || role === 'personal'
  return {
    showOrderNo: true,
    showSource: isTeacher || isOrg,
    showStudentGender: true,
    // 学员主体文案：老师 / 机构看学生，付费方（家长 / 个人）看学员
    studentColLabel: isTeacher || isOrg ? '年级 / 学生' : '年级 / 学员',
    // 费用列名固定为「课程费用」：收/支由「收支」列表达，不再按身份改列名
    priceColLabel: '课程费用',
    // 搜索提示：与后端关键词匹配口径一致（学生姓名 / 地址 / 备注 / 来源）
    keywordPlaceholder: isPayer
      ? '搜索学员姓名 / 地址 / 备注'
      : '搜索学生姓名 / 地址 / 备注 / 来源',
    gradeFilterLabel: isPayer ? '全部年级 / 类型' : '全部年级',
    gradeFilterOptions: isPayer ? [...gradeOptions.value, ...personalTypes] : gradeOptions.value
  }
})

// 仪表盘金额口径：随「收 / 支」筛选切换（收入=预计收入，开支=预计开支）
const dashMoneyLabel = computed(() => (defaultDirection.value === 'income' ? '预计收入' : '预计开支'))
const dashRealMoneyLabel = computed(() => (defaultDirection.value === 'income' ? '真实收入' : '真实开支'))
const dashMoneyAxis = computed(() => (defaultDirection.value === 'income' ? '收入(元)' : '开支(元)'))
const dashTrendTitle = computed(() => {
  const unit = dashDim.value === 'course' ? '课程' : '课时'
  const kind = defaultDirection.value === 'income' ? '收入' : '开支'
  return `${kind}与${unit}趋势`
})
const dashTrendHint = computed(() => {
  const kind = defaultDirection.value === 'income' ? '真实收入' : '真实开支'
  return `（${kind} = 区间内已实际上课且未停课的课程金额）`
})

function emptyForm() {
  return {
    source: '',
    publishedAt: '',
    publisher: '',
    grade: '',
    orderNo: '',
    address: '',
    subject: '',
    subjects: [],
    content: '',
    studentName: '',
    studentGender: '',
    startDate: '',
    endDate: '',
    totalLessons: 0,
    weeklySlots: [],
    hourlyRate: 0,
    lessonPrice: 0,
    totalAmount: 0,
    contactPhone: '',
    studentSituation: '',
    remark: '',
    remarkFlag: false,
    status: 'running'
  }
}

const form = ref(emptyForm())

// 列表字段定义（key 顺序即默认展示顺序，可自定义拖拽 / 显隐）
// 顺序按使用频率排列：学员 / 科目 → 课程安排 → 进度费用 → 状态操作；
// 编号、地址、来源、备注默认收起，需要时在「列设置」里打开。
const COLUMN_BASE = [
  { key: 'student', width: 140 },
  { key: 'subjects', width: 120 },
  { key: 'period', width: 175 },
  { key: 'slots', width: 170 },
  { key: 'progress', width: 140 },
  { key: 'price', width: 120 },
  { key: 'status', width: 110 },
  { key: 'op', width: 120, fixed: 'right' },
  { key: 'orderNo', width: 130 },
  { key: 'address', minWidth: 160 },
  { key: 'source', width: 150 },
  { key: 'remark', width: 76, align: 'center' }
]

// 默认显示的列（其余列在「列设置」里按需开启）
const DEFAULT_VISIBLE_KEYS = ['student', 'subjects', 'period', 'slots', 'progress', 'price', 'status', 'op']

// 当前身份是否适用某列：个人无「课程编号 / 来源」，家长无「来源 / 发布人」（与表单 fv 口径一致）
function applicable(key) {
  if (key === 'orderNo') return lv.value.showOrderNo
  if (key === 'source') return lv.value.showSource
  return true
}

// 列标题随身份变化（学员主体、费用口径）
function columnLabel(key) {
  switch (key) {
    case 'orderNo':
      return '课程编号'
    case 'student':
      return lv.value.studentColLabel
    case 'subjects':
      return '科目'
    case 'address':
      return '地址'
    case 'period':
      return '补课周期'
    case 'progress':
      return '课程进度'
    case 'slots':
      return '周时间段'
    case 'price':
      return lv.value.priceColLabel
    case 'source':
      return '来源 / 发布人'
    case 'remark':
      return '备注'
    case 'status':
      return '状态'
    case 'op':
      return '操作'
    default:
      return key
  }
}

const columnMeta = computed(() => COLUMN_BASE.map((c) => ({ ...c, label: columnLabel(c.key) })))
// 列设置里仅展示当前身份适用的列
const settingColumns = computed(() => columnMeta.value.filter((c) => applicable(c.key)))
// v2：列顺序与默认显隐做过一次重排，升级 key 让新的默认列生效（用户的旧偏好不再沿用）
const COL_ORDER_KEY = 'orders.columnOrder.v2'
const COL_VIS_KEY = 'orders.columnVisible.v2'

function loadColumnOrder() {
  try {
    const v = JSON.parse(localStorage.getItem(COL_ORDER_KEY) || '[]')
    if (Array.isArray(v) && v.length) {
      const valid = v.filter((k) => COLUMN_BASE.some((c) => c.key === k))
      // 补齐可能新增的字段
      for (const c of COLUMN_BASE) if (!valid.includes(c.key)) valid.push(c.key)
      return valid
    }
  } catch (e) {
    /* ignore */
  }
  return COLUMN_BASE.map((c) => c.key)
}

const order = ref(loadColumnOrder())
const visibleMap = reactive(loadColumnVisible())

function loadColumnVisible() {
  let saved = {}
  try {
    saved = JSON.parse(localStorage.getItem(COL_VIS_KEY) || '{}') || {}
  } catch (e) {
    saved = {}
  }
  const map = {}
  // 用户手动设置过则沿用，否则按「核心列默认显示、其余默认收起」
  for (const c of COLUMN_BASE) {
    map[c.key] = saved[c.key] === undefined ? DEFAULT_VISIBLE_KEYS.includes(c.key) : saved[c.key] !== false
  }
  return map
}

// 列设置中的可排序项：仅当前身份适用的列
const colOrderItems = computed(() =>
  order.value
    .map((k) => columnMeta.value.find((c) => c.key === k))
    .filter((c) => c && applicable(c.key))
    .map((c) => ({ key: c.key, label: c.label }))
)
// 实际渲染的列：按自定义顺序，过滤不适用的列与未勾选的列
const visibleColumns = computed(() =>
  order.value
    .map((k) => columnMeta.value.find((c) => c.key === k))
    .filter((c) => c && applicable(c.key) && visibleMap[c.key])
)

function persistColumns() {
  localStorage.setItem(COL_ORDER_KEY, JSON.stringify(order.value))
  localStorage.setItem(COL_VIS_KEY, JSON.stringify(visibleMap))
}

function onColumnsReorder(val) {
  const keys = val.map((x) => x.key)
  // 未在列表中的列（不适用当前身份）保留在末尾，避免丢失顺序
  const rest = order.value.filter((k) => !keys.includes(k))
  order.value = [...keys, ...rest]
  persistColumns()
}

function resetColumns() {
  order.value = COLUMN_BASE.map((c) => c.key)
  for (const c of COLUMN_BASE) visibleMap[c.key] = DEFAULT_VISIBLE_KEYS.includes(c.key)
  persistColumns()
}

// 行颜色标记：进行中高亮蓝底，已结束灰底
function rowClass({ row }) {
  if (row.status === 'running') return 'row-running'
  if (row.status === 'finished') return 'row-finished'
  return ''
}

function clockToMin(str) {
  if (!str || !/^\d{1,2}:\d{2}$/.test(str)) return 0
  const [h, m] = str.split(':').map(Number)
  return h * 60 + m
}

function slotTexts(row) {
  try {
    const slots = typeof row.weeklySlots === 'string' ? JSON.parse(row.weeklySlots || '[]') : row.weeklySlots || []
    const names = ['一', '二', '三', '四', '五', '六', '日']
    return slots.map((s) => {
      const days = (s.days || []).map((d) => `周${names[d - 1] || ''}`)
      const eff =
        s.effectiveFrom || s.effectiveTo
          ? `（${s.effectiveFrom || '起'}~${s.effectiveTo || '长期'}）`
          : ''
      const dur = Math.max(0, clockToMin(s.end) - clockToMin(s.start))
      const durText = dur > 0 ? `（${dur}分钟）` : ''
      return `${days.join('、') || '未选'} ${s.start}-${s.end}${eff}${durText}`
    })
  } catch (e) {
    return []
  }
}

function subjectText(row) {
  const arr = Array.isArray(row.subjects) ? row.subjects : []
  if (arr.length) return arr.join('、')
  return row.subject || ''
}

// 课程进度：已上 / 预计节数（口径见后端 order_progress.go）
function progressPercent(row) {
  const planned = Number(row.progress?.planned || 0)
  const done = Number(row.progress?.done || 0)
  if (planned <= 0) return 0
  return Math.min(100, Math.round((done / planned) * 100))
}

function progressColor(row) {
  const pct = progressPercent(row)
  if (pct >= 100) return '#22a06b'
  if (pct >= 70) return '#e6a23c'
  return '#2f6fed'
}

function priceText(row) {
  // 按计费方式展示：总费用模式附带分摊后的单节价
  if (row.billMode === 'total') {
    const amount = Number(row.totalAmount || 0)
    if (amount <= 0) return '-'
    const planned = Number(row.progress?.planned || 0)
    const per = planned > 0 ? ` · 约 ¥${(amount / planned).toFixed(2)}/节` : ''
    return `总费用 ¥${amount}${per}`
  }
  const parts = []
  if (Number(row.hourlyRate) > 0) parts.push('时薪 ¥' + row.hourlyRate + '/时')
  if (Number(row.lessonPrice) > 0) parts.push('单次 ¥' + row.lessonPrice + '/次')
  return parts.length ? parts.join(' ｜ ') : '-'
}

const money = (v) => Number(v || 0).toFixed(2)

// 课程费用：单输入框 + 单位选择（元/时 ｜ 元/次 ｜ 总费用），切换时清空其它金额，保证只保留一种口径
const priceMode = ref('hourly')
const activePrice = computed({
  get: () => {
    if (priceMode.value === 'hourly') return Number(form.value.hourlyRate) || 0
    if (priceMode.value === 'lesson') return Number(form.value.lessonPrice) || 0
    return Number(form.value.totalAmount) || 0
  },
  set: (v) => {
    const n = Number(v) || 0
    form.value.hourlyRate = priceMode.value === 'hourly' ? n : 0
    form.value.lessonPrice = priceMode.value === 'lesson' ? n : 0
    form.value.totalAmount = priceMode.value === 'total' ? n : 0
  }
})
function onPriceModeChange() {
  if (priceMode.value !== 'hourly') form.value.hourlyRate = 0
  if (priceMode.value !== 'lesson') form.value.lessonPrice = 0
  if (priceMode.value !== 'total') form.value.totalAmount = 0
}

// 总费用模式下按计划总课时分摊到每节课（与后端 lessonIncomeWithPlan 口径一致）
const totalPriceTip = computed(() => {
  const amount = Number(form.value.totalAmount || 0)
  if (amount <= 0) return '填写整期总费用，将按计划总课时分摊到每节课'
  const planned = plannedLessons()
  if (planned <= 0) return '需填写总课时或结束日期，才能把总费用分摊到每节课'
  return `分摊到每节课约 ¥${(amount / planned).toFixed(2)}（共 ${planned} 节）`
})

// 计划总节数：优先填写的总课时，其次按开始日期与结束日期之间的实际排课节数估算
function plannedLessons() {
  const total = Number(form.value.totalLessons || 0)
  if (total > 0) return total
  if (!form.value.startDate || !form.value.endDate) return 0
  const slots = (form.value.weeklySlots || []).filter((s) => s && Array.isArray(s.days) && s.days.length && s.start)
  if (!slots.length) return 0
  const end = dayjs(form.value.endDate)
  let cur = dayjs(form.value.startDate)
  if (!cur.isValid() || !end.isValid()) return 0
  let count = 0
  const maxDays = 3650
  for (let i = 0; i <= maxDays && !cur.isAfter(end); i++) {
    const dayStr = cur.format('YYYY-MM-DD')
    const wd = cur.day() === 0 ? 7 : cur.day()
    for (const s of slots) {
      if (s.days.includes(wd) && slotActiveOn(s, dayStr)) count++
    }
    cur = cur.add(1, 'day')
  }
  return count
}

// 联系电话合法性校验（11 位手机号）
const phoneError = ref('')
function validatePhone(val) {
  const v = (val || '').trim()
  if (!v) {
    phoneError.value = ''
    return
  }
  phoneError.value = /^1[3-9]\d{9}$/.test(v) ? '' : '请输入正确的 11 位手机号'
}

// 订单编号前缀：有邀请码用邀请码；站长 / 管理员无邀请码时按角色回退，保证可读
function orderNoPrefix() {
  // 免注册试用：本地编号统一 TRIAL 前缀
  if (trialStore.isGuest) return 'TRIAL'
  const p = userStore.profile
  if (p?.inviteCode) return p.inviteCode.trim()
  if (p?.role === 'owner') return 'OWNER'
  if (p?.isStaff) return 'ADMIN'
  return 'ORD'
}

async function load() {
  // 免注册试用：数据保存在本机浏览器，直接读本地并做客户端筛选（一次性分页）
  if (trialStore.isGuest) {
    loading.value = true
    try {
      let orders = [...trialStore.orders]
      if (filters.keyword) {
        const kw = filters.keyword.toLowerCase()
        orders = orders.filter((o) =>
          [o.orderNo, o.studentName, o.grade, o.subject, o.address, o.remark]
            .some((v) => (v || '').toLowerCase().includes(kw))
        )
      }
      if (filters.status) orders = orders.filter((o) => o.status === filters.status)
      if (filters.grade) orders = orders.filter((o) => o.grade === filters.grade)
      if (filters.subject) orders = orders.filter((o) => (o.subject || '') === filters.subject)
      if (filters.direction) orders = orders.filter((o) => (o.direction || defaultDirection.value) === filters.direction)
      total.value = orders.length
      const startIdx = (page.value - 1) * pageSize.value
      list.value = orders.slice(startIdx, startIdx + pageSize.value).map((o) => ({
        ...o,
        progress: trialStore.progressOf(o)
      }))
    } finally {
      loading.value = false
    }
    return
  }
  loading.value = true
  try {
    const res = await orderApi.list({
      page: page.value,
      pageSize: pageSize.value,
      keyword: filters.keyword,
      status: filters.status,
      grade: filters.grade,
      subject: filters.subject,
      direction: filters.direction
    })
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function loadOptions() {
  if (trialStore.isGuest) return // 免注册试用使用内置默认选项
  try {
    const res = await orderApi.options()
    if (res.data?.grades?.length) gradeOptions.value = res.data.grades
    if (res.data?.subjects?.length) subjectOptions.value = res.data.subjects
  } catch (e) {
    /* 失败时使用默认选项 */
  }
}

// 收支方向：固定跟随注册身份（老师 / 机构=收入，家长 / 个人=开支），不再手动调整；
// 免注册试用默认按老师（收入）方向
const defaultDirection = computed(() => {
  const role = userStore.userRole
  return role === 'parent' || role === 'personal' ? 'expense' : 'income'
})

function openCreate() {
  // 试用上限：免注册期间最多添加 3 门课程，超出则引导注册
  if (trialStore.isGuest && trialStore.orderCount >= TRIAL_COURSE_LIMIT) {
    ElMessage.warning('免费试用期间最多可添加 3 门课程，注册后不限课程数量')
    trialStore.openRegisterGuide()
    return
  }
  editing.value = null
  readonly.value = false
  form.value = emptyForm()
  drawer.value = true
}

function openEdit(row) {
  editing.value = row
  // 已结束订单仅可查看，不可编辑修改
  readonly.value = row.status === 'finished'
  let slots = []
  try {
    slots = typeof row.weeklySlots === 'string' ? JSON.parse(row.weeklySlots || '[]') : row.weeklySlots || []
  } catch (e) {
    slots = []
  }
  const subjects = Array.isArray(row.subjects) ? row.subjects : row.subject ? [row.subject] : []
  form.value = {
    ...emptyForm(),
    ...row,
    weeklySlots: slots,
    subjects,
    totalLessons: row.totalLessons || 0,
    hourlyRate: row.hourlyRate || 0,
    lessonPrice: row.lessonPrice || 0,
    contactPhone: row.contactPhone || ''
  }
  delete form.value.progress // 进度为只读统计字段，不参与提交
  // 收支方向不再作为课程属性：提交时按注册身份统一写入（老师 / 机构=收入，学员=开支），用户无需选择
  form.value.direction = defaultDirection.value
  // 计费方式按订单回显；其它金额字段清零，保证只保留一种口径
  priceMode.value = ['hourly', 'lesson', 'total'].includes(row.billMode) ? row.billMode : 'hourly'
  onPriceModeChange()
  phoneError.value = ''
  drawer.value = true
}

// 单次时段在某天是否生效（与后端 WeeklySlot.Active 一致：含首尾，留空表示不限）
function slotActiveOn(slot, dayStr) {
  if (slot.effectiveFrom && dayStr < slot.effectiveFrom) return false
  if (slot.effectiveTo && dayStr > slot.effectiveTo) return false
  return true
}

// 按总课时与每周时段推算结束日期（口径与后端 order_progress.go 的周期展开一致）：
// 从开始日期起逐日累计课次，达到总课时的当天即为结束日期。
function estimateEndDate(startDate, weeklySlots, totalLessons) {
  const total = Number(totalLessons || 0)
  if (!startDate || total <= 0) return null
  const slots = (weeklySlots || []).filter((s) => s && Array.isArray(s.days) && s.days.length && s.start)
  if (!slots.length) return null
  let cur = dayjs(startDate)
  if (!cur.isValid()) return null

  let count = 0
  const maxDays = 3650 // 最多推算 10 年，防止无效配置导致死循环
  for (let i = 0; i < maxDays; i++) {
    const dayStr = cur.format('YYYY-MM-DD')
    const wd = cur.day() === 0 ? 7 : cur.day() // dayjs：0=周日，订单时段用 1=周一 ... 7=周日
    let hits = 0
    for (const s of slots) {
      if (!s.days.includes(wd) || !slotActiveOn(s, dayStr)) continue
      hits++
    }
    if (hits > 0) {
      count += hits
      if (count >= total) return { date: dayStr, lessons: count, perWeek: countPerWeek(slots) }
    }
    cur = cur.add(1, 'day')
  }
  return null
}

// 每周课次（按生效周期内的第一个自然周估算，仅用于文案展示）
function countPerWeek(slots) {
  return (slots || []).reduce((n, s) => n + (Array.isArray(s.days) ? s.days.length : 0), 0)
}

const endEstimate = computed(() =>
  estimateEndDate(form.value.startDate, form.value.weeklySlots, form.value.totalLessons)
)

// 提示固定为「推算文案」：不论是否已填结束日期，都按当前总课时与时段给出同一口径的推算结果
const totalLessonsTip = computed(() => {
  const total = Number(form.value.totalLessons || 0)
  if (total <= 0) return '填写总课时后，自动推算上到哪天'
  const est = endEstimate.value
  if (!est) return '填好开始日期与周时间段后，自动推算上到哪天'
  return `共 ${total} 节 · 每周约 ${est.perWeek} 节 · 预计上到 ${est.date}`
})

function applyEndEstimate() {
  if (!endEstimate.value) return
  form.value.endDate = endEstimate.value.date
  ElMessage.success(`已按 ${form.value.totalLessons} 节课填入结束日期 ${endEstimate.value.date}`)
}

async function submit() {
  if (!String(form.value.grade || '').trim()) {
    ElMessage.warning('请填写年级')
    return
  }
  if (!String(form.value.address || '').trim()) {
    ElMessage.warning('请填写地址')
    return
  }
  if (!form.value.weeklySlots || form.value.weeklySlots.length === 0) {
    ElMessage.warning('请添加周时间段')
    return
  }
  // 多个时间段：同一星期在相同生效周期内不可重复安排（避免同一周出现两种上课安排）
  const slots = form.value.weeklySlots
  for (let i = 0; i < slots.length; i++) {
    for (let j = i + 1; j < slots.length; j++) {
      const a = slots[i]
      const b = slots[j]
      if (!(a.days || []).some((d) => (b.days || []).includes(d))) continue
      const af = a.effectiveFrom || ''
      const at = a.effectiveTo || ''
      const bf = b.effectiveFrom || ''
      const bt = b.effectiveTo || ''
      if ((af === '' || bt === '' || af <= bt) && (bf === '' || at === '' || bf <= at)) {
        ElMessage.warning('多个时间段的生效周期在同一星期上重叠，请调整各时间段的生效日期')
        return
      }
    }
  }
  if (!form.value.startDate) {
    ElMessage.warning('请填写补课开始日期')
    return
  }
  // 边界：补课周期与各时段生效区间均为「含首尾」的自然日；无交集时该时段永远不会生成课次
  const periodEnd = form.value.endDate || ''
  if (periodEnd && periodEnd < form.value.startDate) {
    ElMessage.warning('补课结束日期不能早于开始日期')
    return
  }
  if (periodEnd) {
    for (const s of slots) {
      const ef = s.effectiveFrom || ''
      const et = s.effectiveTo || ''
      if (ef && ef > periodEnd) {
        ElMessage.warning('时间段的生效起始日期晚于补课结束日期，该时段不会生成课次')
        return
      }
      if (et && et < form.value.startDate) {
        ElMessage.warning('时间段的生效截止日期早于补课开始日期，该时段不会生成课次')
        return
      }
    }
  }
  // 课程费用：按所选计费方式校验对应金额
  if (priceMode.value === 'total') {
    if (Number(form.value.totalAmount) <= 0) {
      ElMessage.warning('请填写课程总费用')
      return
    }
    if (plannedLessons() <= 0) {
      ElMessage.warning('按总费用计费需填写总课时或结束日期，才能把费用分摊到每节课')
      return
    }
  } else if (Number(activePrice.value) <= 0) {
    ElMessage.warning(priceMode.value === 'hourly' ? '请填写时薪' : '请填写单次课时价')
    return
  }
  const phone = (form.value.contactPhone || '').trim()
  if (phone && phoneError.value) {
    ElMessage.warning(phoneError.value)
    return
  }
  // 自由文本字段统一做长度与违禁词校验
  const textCheck = validateTexts({
    [form.value.orderNo]: '课程编号',
    [form.value.grade]: '年级',
    [form.value.studentName]: '学生姓名',
    [form.value.subject]: '科目',
    [form.value.address]: '地址',
    [form.value.content]: '辅导内容',
    [form.value.studentSituation]: '学生情况',
    [form.value.remark]: '备注',
    [form.value.source]: '来源',
    [form.value.publisher]: '发布人'
  })
  if (textCheck) {
    ElMessage.warning(textCheck)
    return
  }
  for (const key of ['grade', 'studentName', 'address', 'content', 'studentSituation', 'remark', 'source', 'publisher']) {
    form.value[key] = sanitizeText(form.value[key])
  }
  // 开支订单不记录接单来源：切换方向后清空，避免残留与当前口径无关的渠道信息
  if (!fv.value.showSource) {
    form.value.source = ''
    form.value.publisher = ''
    form.value.publishedAt = ''
  }
  // 订单编号为空时按「邀请码-年月日-4位随机数」自动生成（管理员/站长按角色回退前缀）
  if (!form.value.orderNo || !form.value.orderNo.trim()) {
    const code = orderNoPrefix()
    const date = dayjs().format('YYYYMMDD')
    const rand = String(Math.floor(Math.random() * 10000)).padStart(4, '0')
    form.value.orderNo = `${code}-${date}-${rand}`
  }
  saving.value = true
  try {
    // 免注册试用：写入本地缓存（数据临时保存在本机浏览器）
    if (trialStore.isGuest) {
      if (editing.value) {
        trialStore.updateOrder(editing.value.id, form.value)
        ElMessage.success('课程已更新（保存在本机浏览器）')
      } else {
        trialStore.addOrder(form.value)
        ElMessage.success('课程已添加（数据暂时保存在本机浏览器）')
      }
    } else if (editing.value) {
      await orderApi.update(editing.value.id, form.value)
      ElMessage.success('课程已更新')
    } else {
      await orderApi.create(form.value)
      ElMessage.success('课程已添加')
    }
    drawer.value = false
    load()
  } finally {
    saving.value = false
  }
}

// 状态下拉的可选项：未开始可切到进行中/已结束；进行中不可回退为未开始（未开始由开始日期自动判定）
function statusOptions(row) {
  if (row.status === 'pending') return ['pending', 'running', 'finished']
  if (row.status === 'running') return ['running', 'finished']
  return ['finished']
}

async function changeStatus(row, status) {
  // 进行中 / 已结束不可变更为「未开始」
  if (status === 'pending' && row.status !== 'pending') {
    ElMessage.warning('进行中或已结束的课程不可变更为未开始')
    load()
    return
  }
  // 未开始 → 进行中：若开始日期仍在未来，需做时间校验（提前开始会把开始日期校正为今日）
  if (status === 'running' && row.status === 'pending') {
    const start = row.startDate ? dayjs(row.startDate) : null
    if (start && start.isAfter(dayjs().startOf('day'))) {
      try {
        await ElMessageBox.confirm(
          `该课程补课开始日期为 ${row.startDate}，尚未到开始时间。确认提前开始？系统将把开始日期调整为今日，并从今日起计算课时。`,
          '提前开始确认',
          { type: 'warning', confirmButtonText: '确认提前开始', cancelButtonText: '取消' }
        )
      } catch (e) {
        load()
        return
      }
    }
  }
  // 手动结束订单需二次确认（结束后不可再切换状态）
  if (status === 'finished') {
    try {
      await ElMessageBox.confirm(
        `确认将课程「${row.orderNo || row.studentName || '#' + row.id}」标记为已结束？结束后补课结束日期将同步更新为今日，且不可再切换为其他状态。`,
        '结束确认',
        { type: 'warning', confirmButtonText: '确认结束', cancelButtonText: '取消' }
      )
    } catch (e) {
      load() // 取消时还原下拉显示
      return
    }
  }
  try {
    if (trialStore.isGuest) {
      trialStore.changeStatus(row.id, status)
      ElMessage.success(status === 'finished' ? '已结束，结束日期已同步更新为今日' : '状态已更新')
    } else {
      await orderApi.updateStatus(row.id, status)
      ElMessage.success(status === 'finished' ? '已结束，结束日期已同步更新为今日' : '状态已更新')
    }
    load()
  } catch (e) {
    load()
  }
}

async function remove(row) {
  // 进行中 / 已结束订单保留历史记录，不可删除；仅未开始订单可删
  if (row.status === 'running' || row.status === 'finished') {
    ElMessage.warning('进行中或已结束的课程不可删除，可保留历史记录')
    return
  }
  await ElMessageBox.confirm(`确认删除课程「${row.orderNo || row.studentName || row.id}」？`, '提示', {
    type: 'warning'
  })
  if (trialStore.isGuest) {
    trialStore.removeOrder(row.id)
  } else {
    await orderApi.remove(row.id)
  }
  ElMessage.success('已删除')
  load()
}

function needMember() {
  window.dispatchEvent(new CustomEvent('open-profile'))
}

/* ---------------- 二级导航：仪表盘 / 订单列表（记住上次所在页签） ---------------- */
const activeTab = useNavTab('orders', 'dashboard')

// 切换页签时，切回仪表盘需重算图表尺寸（v-show 切换后容器宽度恢复）
function onTab(tab) {
  if (tab === 'dashboard' && dashLocked.value) {
    // 进阶功能拦截：费用统计需注册解锁
    trialStore.requireAuth('费用统计')
    return
  }
  activeTab.value = tab
  if (tab === 'dashboard') nextTick(() => dashChart && dashChart.resize())
}

/* ---------------- 仪表盘 ---------------- */
const dashPreset = ref(30)
const dashRange = ref([dayjs().subtract(29, 'day').format('YYYY-MM-DD'), dayjs().format('YYYY-MM-DD')])
const dashSummary = reactive({ orderCount: 0, courseCount: 0, estimatedIncome: 0, realIncome: 0, lessonCount: 0 })
// 统计维度：lesson=按课时（一节课为单位，默认）；course=按课程（一门课为单位）
const dashDim = ref('lesson')
const dashPrimary = computed(() =>
  dashDim.value === 'course' ? dashSummary.courseCount || dashSummary.orderCount : dashSummary.lessonCount
)
const dashSecondary = computed(() =>
  dashDim.value === 'course' ? dashSummary.lessonCount : dashSummary.courseCount || dashSummary.orderCount
)
const dashDaily = ref([])
const dashRef = ref(null)
let dashChart = null

// 快捷时间选择：将实际起止日期写入时间控件，使其内部直接显示日期文案
function onDashPreset(val) {
  const n = Number(val)
  const end = dayjs()
  const start = end.subtract(n - 1, 'day')
  dashRange.value = [start.format('YYYY-MM-DD'), end.format('YYYY-MM-DD')]
  loadDashboard()
}

function onDashRange(val) {
  if (val && val.length === 2) {
    dashPreset.value = null // 手动选择后取消快捷高亮
    loadDashboard()
  }
}

async function loadDashboard() {
  const params = {}
  if (dashRange.value && dashRange.value.length === 2) {
    params.start = dashRange.value[0]
    params.end = dashRange.value[1]
  } else {
    params.range = dashPreset.value
  }
  // 收支方向：固定跟随注册身份（老师 / 机构=收入，付费方=开支），后端按方向统计
  params.direction = defaultDirection.value
  try {
    const res = await orderApi.dashboard(params)
    const d = res.data || {}
    Object.assign(dashSummary, d.summary || {})
    dashDaily.value = d.daily || []
    renderDashChart()
  } catch (e) {
    /* 仪表盘加载失败不影响列表 */
  }
}

function renderDashChart() {
  if (!dashChart) return
  const dates = dashDaily.value.map((x) => x.date.slice(5))
  // 计数系列随维度切换：课时数（节）或课程数（门）
  const byCourse = dashDim.value === 'course'
  const counts = dashDaily.value.map((x) => (byCourse ? x.courseCount : x.lessonCount))
  const countName = byCourse ? '课程数' : '课时数'
  const estimated = dashDaily.value.map((x) => x.estimatedIncome)
  const real = dashDaily.value.map((x) => x.realIncome)
  dashChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: [countName, dashMoneyLabel.value, dashRealMoneyLabel.value], top: 0, icon: 'roundRect' },
    grid: { left: 48, right: 60, top: 36, bottom: 28 },
    xAxis: { type: 'category', data: dates, axisLabel: { fontSize: 11 } },
    yAxis: [
      { type: 'value', name: byCourse ? '课程数(门)' : '课时数(节)', minInterval: 1, axisLabel: { fontSize: 11 } },
      { type: 'value', name: dashMoneyAxis.value, axisLabel: { fontSize: 11 } }
    ],
    series: [
      {
        name: countName,
        type: 'bar',
        yAxisIndex: 0,
        data: counts,
        barMaxWidth: 16,
        itemStyle: { color: '#9aa7ff', borderRadius: [3, 3, 0, 0] }
      },
      {
        name: dashMoneyLabel.value,
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        data: estimated,
        symbolSize: 5,
        itemStyle: { color: '#2f6fed' },
        areaStyle: { color: 'rgba(47,111,237,0.10)' }
      },
      {
        name: dashRealMoneyLabel.value,
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        data: real,
        symbolSize: 5,
        itemStyle: { color: '#18a058' },
        areaStyle: { color: 'rgba(24,160,88,0.10)' }
      }
    ]
  })
}

// 维度变化时重新统计（后端按订单方向过滤）
watch(dashDim, () => loadDashboard())

function initDash() {
  if (dashChart) {
    dashChart.resize()
    return
  }
  if (!dashRef.value) return
  dashChart = echarts.init(dashRef.value)
  loadDashboard()
}

function onResize() {
  dashChart?.resize()
}

/* ---------------- 使用模式：正常会员 / 免注册试用 / 会员过期只读 ---------------- */
const trialStore = useTrialStore()
// 三种模式共用列表区域：登录会员正常使用；未登录进入免注册试用；会员过期只读查看
const mainVisible = computed(() => true)
// 会员过期（含试用到期）：只读模式，可查看不可新增 / 调课
const readonlyMode = computed(() => userStore.isLogin && !userStore.isMember)
// 费用统计为注册用户进阶功能：免注册试用与过期账号锁定；
// 锁定时强制停在课程列表，避免默认停留在看不见的布局入口上
const dashLocked = computed(() => trialStore.isGuest || readonlyMode.value)
watch(
  dashLocked,
  (locked) => {
    if (locked) activeTab.value = 'list'
  },
  { immediate: true }
)

// 顶部提示条文案
const trialBanner = computed(() => {
  if (trialStore.isGuest) {
    return {
      type: 'trial',
      text: '试用模式：您的排课数据暂时保存在本机浏览器，注册即可永久保存并随时查看。'
    }
  }
  if (readonlyMode.value) {
    return {
      type: 'expired',
      text: '会员已过期：已录入的课程与课表仍可查看，新增课程与调课功能需续费后使用。'
    }
  }
  return null
})

// 本地试用数据变化时刷新列表（新增 / 编辑 / 删除 / 状态切换均已本地落盘）
watch(
  () => trialStore.orderCount,
  () => {
    if (trialStore.isGuest) load()
  }
)

watch(
  mainVisible,
  (v) => {
    if (v) {
      // 收支方向固定跟随注册身份：老师/机构=收入，付费方（家长/个人）=开支
      filters.direction = defaultDirection.value
      load()
      loadOptions()
      if (!dashLocked.value) nextTick(initDash)
    }
  },
  { immediate: true }
)

// 维度变化时重绘图表（金额标签与计数系列随之切换）
watch(dashDim, () => {
  if (dashChart) nextTick(renderDashChart)
})

onMounted(() => {
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  dashChart?.dispose()
})
</script>

<style scoped>
.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.flex-spacer {
  flex: 1;
}

/* 二级导航：仪表盘 / 订单列表 */
.nav-tabs {
  display: flex;
  gap: 4px;
  margin: 4px 0 16px;
  border-bottom: 1px solid var(--brand-line);
}

.nav-tab {
  position: relative;
  appearance: none;
  border: none;
  background: transparent;
  padding: 9px 16px;
  font-size: 14px;
  color: var(--brand-sub);
  cursor: pointer;
  border-radius: 8px 8px 0 0;
  transition: color 0.15s, background 0.15s;
}

.nav-tab:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.nav-tab.active {
  color: var(--el-color-primary);
  font-weight: 600;
}

.nav-tab.active::after {
  content: '';
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: -1px;
  height: 2px;
  border-radius: 2px;
  background: var(--el-color-primary);
}

/* 订单颜色标记：进行中高亮蓝底，已结束灰底 */
.order-legend {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 10px;
  font-size: 12px;
  color: var(--brand-sub);
  flex-wrap: wrap;
}

/* 试用课程额度提示 */
.trial-quota b {
  color: var(--el-color-primary);
  margin: 0 2px;
}

/* 试用 / 过期提示条 */
.trial-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 13px;
}

.trial-banner.trial {
  background: #f0f7ff;
  color: #2b6cb0;
  border: 1px solid #cfe4ff;
}

.trial-banner.expired {
  background: #fff9e6;
  color: #8a6d3b;
  border: 1px solid #f5e0a3;
}

.trial-banner span {
  flex: 1;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

/* 图例圆点：与状态胶囊的圆点配色一致 */
.legend-block {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.lg-pending {
  background-color: #9aa7bd;
}

.lg-running {
  background-color: #37985a;
}

.lg-finished {
  background-color: #c3c9d4;
}

:deep(.el-table__row.row-running) > td.el-table__cell,
:deep(.el-table__row.row-running.el-table__row--striped) > td.el-table__cell {
  background-color: #f5f9ff;
}

:deep(.el-table__row.row-running:hover) > td.el-table__cell {
  background-color: #eaf2ff;
}

:deep(.el-table__row.row-finished) > td.el-table__cell,
:deep(.el-table__row.row-finished.el-table__row--striped) > td.el-table__cell {
  background-color: #f7f8fa;
  color: var(--brand-muted);
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.cell-main {
  font-weight: 600;
}

.cell-sub {
  font-size: 12px;
  color: var(--brand-muted);
}

.slot-text {
  font-size: 12px;
  line-height: 1.6;
  color: var(--brand-sub);
}

.subject-tags {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 表单分区：统一浅色卡片 + 标题（替代 el-divider） */
.form-section {
  margin-bottom: 14px;
  padding: 14px 16px 0;
  border: 1px solid #eaeff7;
  border-radius: 12px;
  background: #fbfcfe;
}

.section-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--brand-ink);
}

.section-head::before {
  content: '';
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--el-color-primary);
}

.section-note {
  font-size: 12px;
  font-weight: 400;
  color: var(--brand-muted);
  line-height: 1.5;
}

/* 收支类型置顶区块：无 label，分段按钮与区块标题左对齐，底部留白与其它区块一致 */
.direction-item {
  margin-bottom: 16px;
}

.direction-item :deep(.el-form-item__content) {
  margin-left: 0;
}

/* 数字输入框左对齐：默认居中会让空框显得松散、数字跳动 */
.order-form :deep(.el-input-number .el-input__inner) {
  text-align: left;
}

/* 金额 / 节数输入框：宽度与日期框（栅格 span12 内的控件）严格一致，随抽屉宽度自适应。
   推导（100% = 本行 form-item 内容宽 C）：
     C            = 抽屉内容宽 W − 本行列内边距 16 − 标签 92 = W − 108
     span12 控件宽 = W / 2 − 列内边距 16 − 标签 92 = W / 2 − 108
     代入 W = C + 108 ⇒ 控件宽 = C / 2 − 54 = 50% − 54px */
.money-input {
  flex: none;
  width: calc(50% - 54px);
}

/* 费用 / 课时行：输入框 + 单位（元/时 ｜ 元/次 ｜ 总费用、节）横向排布 */
.price-input {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.price-unit {
  flex: none;
  width: 96px;
}

.unit-text {
  flex: none;
  font-size: 13px;
  color: var(--brand-sub);
}

.estimate-btn {
  flex: none;
  white-space: nowrap;
}

/* 课时进度提示：固定展示推算结果 */
.progress-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  flex-wrap: wrap;
}

.progress-chip {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 13px;
  color: #2f6fed;
  background: #eef3ff;
  border: 1px solid #dde8ff;
}

/* 未填总课时：不做成"框"，用浅色提示文字，避免看起来像禁用输入框 */
.progress-chip.is-empty {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--brand-muted);
}

/* 收支标签：箭头 + 胶囊，收入绿、开支橙，一眼区分资金方向 */
.dir-tag {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.dir-tag .el-icon {
  font-size: 11px;
}

/* 收支列较窄：收紧单元格内边距，保证胶囊完整显示不出现省略号 */
:deep(.col-direction .cell) {
  padding-left: 8px;
  padding-right: 8px;
}

.dir-tag.is-in {
  background: #eaf7ee;
  color: #2f8a52;
}

.dir-tag.is-out {
  background: #fff3e6;
  color: #cf7a12;
}

/* 状态胶囊：圆点 + 文案；进行中可点击切换 */
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 22px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  outline: none;
}

.status-pill .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.7;
}

.status-pill.is-pending {
  background: #f1f4f9;
  color: #5f6f8c;
}

.status-pill.is-running {
  background: #eaf7ee;
  color: #2f8a52;
  cursor: pointer;
}

.status-pill.is-finished {
  background: #f3f4f6;
  color: #8a94a6;
}

/* 课程进度：已上 / 预计节数 + 进度条 */
.progress-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.progress-text {
  font-size: 13px;
  color: var(--brand-sub);
}

.progress-text b {
  color: var(--brand-ink);
  font-size: 14px;
}

.remark-dot {
  position: relative;
  display: inline-flex;
  color: var(--el-color-primary);
  cursor: pointer;
}

.red-dot {
  position: absolute;
  top: -2px;
  right: -6px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--brand-danger);
  box-shadow: 0 0 0 2px rgba(227, 77, 89, 0.18);
}

.order-form :deep(.el-divider__text) {
  font-size: 13px;
  color: var(--brand-sub);
}

/* 列设置弹层 */
.col-set-title {
  font-size: 12px;
  color: var(--brand-muted);
  margin-bottom: 10px;
}

.col-vis {
  margin-top: 10px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 10px;
  border-top: 1px dashed var(--brand-line);
  padding-top: 10px;
}

.col-set-foot {
  margin-top: 8px;
  text-align: right;
}

/* 仪表盘 */
.dashboard {
  margin: 6px 0 16px;
  padding: 16px;
  border: 1px solid var(--brand-line);
  border-radius: 12px;
  background: linear-gradient(180deg, #fbfcff 0%, #fff 100%);
}

.dash-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.dash-title {
  font-size: 15px;
  font-weight: 700;
}

.dash-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.dash-range {
  font-size: 12px;
  color: var(--brand-muted);
}

.stat-row {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.stat {
  flex: 1 1 140px;
  padding: 14px 16px;
  border-radius: 10px;
  background: #fff;
  border: 1px solid var(--brand-line);
}

.stat b {
  display: block;
  font-size: 22px;
  color: var(--el-color-primary-dark-2);
}

.stat b.ok {
  color: var(--brand-success);
}

.stat b.money {
  color: var(--brand-success);
}

.stat span {
  font-size: 12px;
  color: var(--brand-muted);
}

.chart-box {
  border: 1px solid var(--brand-line);
  border-radius: 12px;
  padding: 12px;
  background: #fff;
}

.chart-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}

.chart {
  width: 100%;
  height: 280px;
}

/* 操作列固定右侧：不透明背景，避免下方单元格内容与行高亮透出（与各行背景保持一致） */
:deep(.el-table .el-table__cell.el-table-fixed-column--right) {
  background-color: #fff;
}
:deep(.el-table .el-table__row--striped .el-table__cell.el-table-fixed-column--right) {
  background-color: var(--el-fill-color-lighter, #fafafa);
}
:deep(.el-table__row.row-running .el-table__cell.el-table-fixed-column--right),
:deep(.el-table__row.row-running.el-table__row--striped .el-table__cell.el-table-fixed-column--right) {
  background-color: #f5f9ff;
}
:deep(.el-table__row.row-finished .el-table__cell.el-table-fixed-column--right),
:deep(.el-table__row.row-finished.el-table__row--striped .el-table__cell.el-table-fixed-column--right) {
  background-color: #f7f8fa;
}
:deep(.el-table__row:hover .el-table__cell.el-table-fixed-column--right) {
  background-color: var(--el-table-row-hover-bg-color, #f5f7fa);
}
:deep(.el-table__row.row-running:hover .el-table__cell.el-table-fixed-column--right) {
  background-color: #eaf2ff;
}

/* 表单提示 / 错误 */
.form-hint {
  font-size: 12px;
  color: var(--brand-muted);
  margin-top: 4px;
  line-height: 1.4;
}

.field-error {
  font-size: 12px;
  color: var(--el-color-danger);
  margin-top: 4px;
  line-height: 1.4;
}

.has-error :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--el-color-danger) inset;
}

/* 「总费用」计费方式的说明图标：跟在单位下拉后，不参与输入框对齐 */
.price-help {
  flex: none;
  color: var(--brand-muted);
  cursor: help;
}
</style>
