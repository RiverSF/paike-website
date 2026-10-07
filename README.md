# 排课助手 · 课程排期与课表管理

![vibe coding](https://img.shields.io/badge/vibe--coding-AI%20%E9%A9%B1%E5%8A%A8-ff69b4)
![Vue 3](https://img.shields.io/badge/Vue-3-42b883)
![Go](https://img.shields.io/badge/Go-Gin-00add8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-336791)

面向家长、老师与个人的排课工具：录入课程后，系统按「补课周期 + 每周时段」自动生成周课表。

- 前端：Vue 3 + Vite + Element Plus + Pinia
- 后端：Go + Gin + GORM
- 数据库：PostgreSQL
- 部署：单机部署 / Kubernetes 部署（两种可切换，见下文）

导航栏：**首页 / 订单 / 课表 / 使用指南**，右上角头像可查看个人信息与会员信息。前端访问端口为 **80**。

## 功能演示

<video src="docs/demo/demo.mp4" poster="docs/demo/demo-cover.png" controls muted width="100%"></video>

> 视频源文件在 `docs/demo/`（`demo.mp4` 与封面 `demo-cover.png`）；站内的演示视频由 `web/public/videos/guide-demo.mp4` 单独维护。
> 若此处无法播放，可直接打开 [docs/demo/demo.mp4](docs/demo/demo.mp4)。

## Vibe Coding

本项目**全程 Vibe Coding**：不写长篇设计文档再照图施工，而是「说清楚想要什么 → 让 AI 实现 → 跑起来看对不对 → 再说下一句」的对话式开发。前端、后端、K8s 编排与文档都在这种循环里长出来的。

| | 人负责 | AI 负责 |
|---|---|---|
| 需求 | 讲清目标、边界与优先级，拍板取舍 | 拆解为可执行的改动与验证步骤 |
| 实现 | 只 Review 关键逻辑与 SQL | 写代码、重构、补注释与文档 |
| 质量 | 定义验收标准、决定何时够用 | 写迁移脚本、补索引、对齐口径 |
| 兜底 | 承担最终判断与线上责任 | 提供方案与风险提示，不替你决策 |

三条实际约定：

1. **能跑通优先，再谈优雅** —— 每个功能先落到可点击 / 可调用，再收敛结构；
2. **库表与接口是契约** —— 字段、注释、索引由 `model/schema.go` + `comments.go` 在启动时自动对齐，改代码即改库；
3. **文档随代码更新** —— README 与 `docs/` 由同一轮对话维护，避免文档漂移。

> 因此代码里保留了不少「当时就这么定了」的权衡痕迹。想接着改，直接开 issue 说清目标即可，AI 会照上面的循环继续往下写。

> 使用身份：用户表 `user_role` 字段（`teacher` 老师 / `parent`+`personal` 学员 / `org` 机构），注册时选择，**仅影响课程表单字段与文案**（学员隐藏「信息来源 / 发布人 / 发布时间」，机构为多学员排课口径），**不参与计费**；计费仍由师资身份 `teacher_type` 决定：`student` 大学生家教（需上传学生证与身份证并通过审核）享折扣，`professional` 专职老师走标准价。仅「老师」身份可选择大学生家教。
>
> 角色规则：用户表 `role` 字段（`user`/`admin`/`owner`）。服务每次启动会执行 `SeedOwner()`，把 **ID=1 的用户**校正为站点拥有者（`owner` + 永久会员，到期 2099-12-31），管理员与拥有者不受会员有效期限制。

---

## 一、目录结构

```
tutoring-website/
├── web/                      前端（Vue3）
│   ├── src/
│   │   ├── api/              接口封装
│   │   ├── components/       公共组件（头部、登录、个人中心、周时段编辑器...）
│   │   ├── layouts/          主布局
│   │   ├── router/           路由
│   │   ├── stores/           Pinia（用户 / 会员 / 免注册试用 / 价格）
│   │   ├── styles/           全局样式与主题色
│   │   └── views/            首页 / 课程安排 / 课表 / 使用指南 / 管理端（总览·用户·价格·反馈·站内信）
│   └── public/               logo、默认头像、二维码目录(qrcode/)、演示视频(videos/)
├── server/                   后端（Gin）
│   ├── main.go
│   ├── internal/
│   │   ├── conf/             配置 app.ini（同 .example）
│   │   ├── config/           配置加载（支持环境变量覆盖）
│   │   ├── handler/          业务处理（user/order/schedule/feedback/message/price/admin/upload）
│   │   ├── middleware/       JWT 鉴权、会员校验、CORS、日志、注册 / 登录限流
│   │   ├── model/            GORM 模型；库表规范由 schema.go（非空+索引）与 comments.go（注释）在启动时自动维护
│   │   └── router/           路由注册 + SPA 托管
│   ├── migrations/           初始化 SQL（启动也会自动 AutoMigrate）
│   └── pkg/                  日志、路径、通用工具
├── docs/                     文档：部署安全加固、产品设计说明、演示视频（docs/demo/）
├── deploy/
│   ├── docker/               两个 Dockerfile（server / web 镜像）
│   ├── k8s/                  k8s 编排：namespace / configmap / secret / postgres / server / web / ingress
│   ├── nginx/                nginx 配置（web 镜像内使用，反代 /api 与 /uploads）
│   └── scripts/              镜像构建脚本（支持 kind / minikube 载入）
└── Makefile                  常用命令
```

---

## 二、功能说明

| 模块 | 说明 |
|------|------|
| 首页 | 站点简介、核心功能、三步上手、会员权益与实时价格卡片（含待生效的价格预告）；底部提供「微信好友 / 微信收款 / 支付宝收款」二维码位 |
| 免注册试用 | **不登录也能排课**：数据存在本机浏览器（localStorage），最多 3 门课程；导出课表、费用统计、调课提醒等进阶功能提示注册解锁，注册后可把试用数据导入账号 |
| 课程安排 | 课程列表（状态可直接在列表中切换、进度与收入统计）、新增/编辑/删除；字段含：信息来源、发布时间、发布人、年级、订单编号、地址、科目（支持多选）、辅导内容、补课起止日期、周时间段（多段，支持生效区间）、计划总课时、计费方式（按小时 / 按单次课时 / 按整期总额）、时薪 / 课时价 / 总费用、收支方向、学生情况、学生性别、联系电话、备注、状态 |
| 课表 | 按自然周展示（默认当前周），x 轴周一~周日、y 轴按小时；课程按「时间范围 + 每周时段」展开为**已物化的课次**；支持**调课 / 停课 / 临时加课**（单次例外，保留完整调整路径与撤销）、删除已上的课次（作废而非物理删除）；备注较长的课程用红点标记；支持上一周/本周/下一周、日期选择器切换与**导出图片（带 logo 水印）** |
| 使用指南 | 使用提示、常见问题 FAQ、演示视频、反馈提交框与已反馈内容列表 |
| 站内信 | 管理员 / 站长可发给指定用户或全员广播；系统自动推送开课提醒、会员到期提醒、价格调整通知与试用期引导（均有去重，不会重复打扰） |
| 角色 | `user` 普通用户 / `admin` 管理员 / `owner` 站点拥有者；**ID=1 的用户为站点拥有者**，具备管理员权限且为永久会员 |
| 注册/会员 | **手机号必填**（11 位中国大陆号码，做查重与格式校验），注册即送 **30 天免费试用**；课程与课表**始终可查看**，新增 / 调整等写操作需会员未过期；支持按天 / 包月 / 包季 / 包年续费 |
| 登录方式 | **手机号 + 密码**（用户名不能登录）；手机号每月仅可修改一次 |
| 大学生认证 | 注册时可选「大学生家教」身份并上传学生证 / 身份证；审核数据结构与流转已就绪（`student_application` 表 + 毕业时间缓冲一年、逾期自动转回专职老师），**后台审核入口尚未接入前端** |
| 管理端 | 站点拥有者、管理员可见「会员总览 / 用户列表 / 价格设置 / 用户反馈 / 站内信」五个页签：总览含注册与会员趋势图；用户列表支持冻结解冻、续费、改角色、重置密码、删除账号；价格设置支持按卡片调价、预约生效、撤销预约与按历史回滚 |
| 充值与累计 | **自助续费入口已关闭**，续费统一由管理员在后台填写**充值金额**与**续费时长（按天/包月/包季/包年）**；系统自动在原到期时间上顺延、累加**累计充值金额**并生成充值记录；个人中心可查看每次充值金额、时长与到期变化 |
| 会员价格 | 三档：**标准价**（默认包月 79 / 包季 225 / 包年 790）+ **大学生折扣价**（标准价 × 8.5 折向上取整，需认证）+ **平季续费折扣**（9 折，可开关）；另有**限时活动**自定义折扣，可叠加；价格改动支持「立即生效 / 预约生效」，每次改动入历史可一键回滚；用户扫码支付后联系管理员人工开通 |
| 个人中心 | 注册信息（用户名、头像、手机号、注册时长）+ 会员信息（会员类型、有效时段、剩余天数）+ 充值记录与累计金额 |

---

## 三、开发模式（推荐先跑这个）

1. 准备 PostgreSQL 并创建数据库：

```bash
createdb tutoring
# 或修改 server/internal/conf/app.ini 中的 [postgres] 配置
cp server/internal/conf/app.ini.example server/internal/conf/app.ini
```

2. 启动后端（开发态用 9090，避免与前端 80 冲突）：

```bash
make server-dev        # HTTP_PORT=9090 go run .
```

3. 启动前端（监听 80）：

```bash
make web-install
sudo make web-dev      # macOS/Linux 绑定 80 需要 sudo
# 访问 http://127.0.0.1
```

> 开发态前端由 vite 提供，后端不需要托管前端产物；`/api`、`/uploads` 由 vite 代理到 9090。
> 若本机 80 端口已被其他服务（如 Homebrew nginx）占用，前端会启动失败或只能走 IPv6，见第四节「端口占用」。

---

## 四、部署方式一：单机部署（standalone）

**特点**：只跑一个后端进程，后端直接托管前端构建产物（`web/dist`），无需 nginx。

```bash
# 1. 构建前端产物
make web-build          # 产物在 web/dist

# 2. 编译并启动后端（默认读取 server/internal/conf/app.ini）
make standalone         # 等价 web-build + server-build + 启动

# 访问 http://127.0.0.1（端口 80）
```

关键配置 `server/internal/conf/app.ini`：

```ini
[app]
STATIC_DIR = ../web/dist    # 后端托管前端产物；设为 none 则关闭托管
UPLOAD_DIR = data/uploads   # 头像上传目录
[server]
HTTP_PORT = 80              # 前端访问端口
```

### 端口占用

80 端口需要 root 权限；若本机已有服务占用（常见为 Homebrew nginx，`nginx/1.19.2` 监听 IPv4 的 80），会出现「能启动但只能走 IPv6 / `localhost` 访问不通」的情况。三种处理方式：

1. **停掉占用者**：`sudo nginx -s stop`（确认该 nginx 没有承载其他站点）
2. **让 nginx 反代**（推荐，可并存）：后端改回 `HTTP_PORT=9090`，在 nginx 增加：
   ```nginx
   server {
       listen 80;
       server_name localhost;
       location /api/    { proxy_pass http://127.0.0.1:9090/api/; }
       location /uploads/{ proxy_pass http://127.0.0.1:9090/uploads/; }
       location /        { proxy_pass http://127.0.0.1:9090/; }
   }
   ```
3. **改用其他端口**：把 `HTTP_PORT` 改成 8080 等未被占用的端口

也可以只编译二进制后用 systemd / supervisor 托管：

```bash
cd server && go build -o bin/tutoring-server .
APP_ROOT=/path/to/tutoring-website/server ./bin/tutoring-server
```

> `APP_ROOT` 用于二进制不在项目根目录运行时定位 `internal/conf/app.ini`、`data/uploads`。

---

## 五、部署方式二：Kubernetes 部署

**特点**：资源集中在 `tutoring` 命名空间 —— postgres(StatefulSet) + server(Deployment) + web(nginx Deployment)；前端由 web Pod 托管并反代 `/api`、`/uploads`；健康检查 `/ping`，后端启动前用 initContainer 等待数据库就绪。

**前置**：任意可用集群（Docker Desktop Kubernetes / kind / minikube / 云厂商托管集群），ingress-nginx 可选（没有就用 `port-forward` 访问）。

### 1. 构建镜像

```bash
make images-build                       # 产出 tutoring-server:latest / tutoring-web:latest
LOAD_KIND=1 make images-build           # kind 集群：构建后自动 load 进集群
LOAD_MINIKUBE=1 make images-build       # minikube 集群：构建后自动 load
REGISTRY=registry.example.com/tutoring TAG=v1.0.0 PUSH=1 make images-build   # 推私有仓库
```

使用私有仓库时同步镜像地址：

```bash
cd deploy/k8s && kustomize edit set image tutoring-server=registry.example.com/tutoring/tutoring-server:v1.0.0
```

### 2. 修改敏感配置（务必）

```bash
kubectl create namespace tutoring
kubectl -n tutoring create secret generic tutoring-secret \
  --from-literal=POSTGRES_PASSWORD='强密码' \
  --from-literal=PG_PASSWORD='强密码' \
  --from-literal=JWT_SECRET='随机长字符串'
```

> 也可以直接改 `deploy/k8s/02-secret.yaml` 后 `make k8s-apply`，但不建议把明文密码入库。

### 3. 部署与访问

```bash
make k8s-apply      # kubectl apply -k deploy/k8s
make k8s-status     # 查看 Pod / Service / Ingress / PVC

# 有 ingress：把域名指向 ingress 地址后访问 http://tutoring.local
# 无 ingress：make k8s-port-forward → http://127.0.0.1:8080
```

### 4. 常用运维

```bash
make k8s-logs                                                   # 后端日志
kubectl -n tutoring scale deploy/web --replicas=3               # 扩容前端（无状态，可直接扩）
# server 扩容前需先把上传目录改为 ReadWriteMany 或对象存储，否则多余副本 Pending
kubectl -n tutoring set image deploy/server server=tutoring-server:v1.0.1   # 更新镜像
make k8s-restart                                                # 滚动重启
make k8s-delete                                                 # 卸载（PVC 需单独删除）
```

### 资源清单

| 文件 | 内容 |
|------|------|
| `00-namespace.yaml` | `tutoring` 命名空间 |
| `01-configmap.yaml` | 非敏感配置：`PG_HOST=postgres`、`STATIC_DIR=none`、`LOG_STDOUT=1`、`HTTP_PORT` 等 |
| `02-secret.yaml` | `POSTGRES_PASSWORD` / `PG_PASSWORD` / `JWT_SECRET`（**上线必须改**） |
| `03-postgres.yaml` | Headless Service + StatefulSet + 10Gi PVC，含 `pg_isready` 就绪探针 |
| `04-server.yaml` | Service(9090) + Deployment（1 副本，等待 DB 的 initContainer、`/ping` 探活）+ 上传目录 PVC |
| `05-web.yaml` | Service(80) + nginx Deployment（2 副本） |
| `06-ingress.yaml` | `tutoring.local`，`/api`、`/uploads` → server，`/` → web |
| `kustomization.yaml` | `kubectl apply -k deploy/k8s` 入口，含镜像 tag 管理 |

> 生产建议：数据库换成云托管 PostgreSQL 或独立 StatefulSet + 备份；上传目录改为对象存储（当前 PVC 为 ReadWriteOnce，多副本场景下头像上传建议改 ReadWriteMany 或统一存储）。

### 两种部署方式如何切换

| 项 | 单机部署 | Kubernetes 部署 |
|----|---------|----------------|
| 前端托管 | 后端 Gin 托管（`STATIC_DIR = ../web/dist`） | web(nginx) Pod 托管（`STATIC_DIR = none`） |
| 对外端口 | 80（后端直接监听） | 80（nginx/Ingress 对外，后端 Service 9090 仅集群内） |
| 数据库 | 本机 / 外部 PostgreSQL | 集群内 `postgres` StatefulSet（或外部数据库） |
| 日志 | 写文件 `server/logs/` | 标准输出（`LOG_STDOUT=1`），`kubectl logs` 查看 |
| 配置来源 | `internal/conf/app.ini` + 环境变量 | ConfigMap + Secret（`envFrom` 注入） |
| 扩缩容 | 手动 | `kubectl scale`（web 默认 2 副本；server 默认 1 副本，扩容前需先改造上传存储） |
| 适用场景 | 低配单机、快速自用 | 生产环境、需要高可用与弹性 |

切换要点（**核心只改一个开关**）：

- 单机 → k8s：`STATIC_DIR` 改为 `none`（后端不再托管前端，前端交给 nginx 镜像）；`PG_HOST` 指向 `postgres` Service；配置改为 ConfigMap/Secret 注入。
- k8s → 单机：把 `STATIC_DIR` 指回 `../web/dist`（需先 `make web-build`），`PG_HOST` 指回本机 PostgreSQL，直接运行后端二进制即可。

> `STATIC_DIR` 为 `none` 或为空时，后端只提供 `/api`、`/uploads`、`/ping`，不会托管页面；两种部署方式共用同一批接口与同一份代码。

---

## 六、配置项与环境变量

配置文件：`server/internal/conf/app.ini`（复制自 `app.ini.example`），**所有配置均可被同名环境变量覆盖**。
k8s 部署时无需改 ini 文件，环境变量由 `deploy/k8s/01-configmap.yaml` 与 `02-secret.yaml` 通过 `envFrom` 注入（同名 key 即覆盖）：

| 配置 | 环境变量 | 说明 |
|------|---------|------|
| `RUN_MODE` | `RUN_MODE` | debug / release |
| `server.HTTP_PORT` | `HTTP_PORT` | 监听端口，默认 9090 |
| `app.STATIC_DIR` | `STATIC_DIR` | 前端产物目录，`none` 表示不托管 |
| `app.UPLOAD_DIR` | `UPLOAD_DIR` | 上传目录，默认 `data/uploads` |
| `postgres.*` | `PG_HOST/PG_PORT/PG_USER/PG_PASSWORD/PG_DB/PG_SSLMODE` | 数据库连接 |
| `jwt.JWT_SECRET` | `JWT_SECRET` | **上线务必修改** |
| `jwt.JWT_EXPIRE_HOURS` | `JWT_EXPIRE_HOURS` | Token 有效期，默认 168 小时 |
| — | `CONFIG_PATH` | 指定配置文件路径（容器内使用） |
| — | `APP_ROOT` | 应用根目录（二进制部署使用） |
| — | `LOG_STDOUT` | `=1` 时日志只输出到标准输出 |
| — | `CORS_ALLOW_ORIGINS` | 逗号分隔的允许跨域源，默认允许全部 |
| `server.TRUSTED_PROXIES` | `TRUSTED_PROXIES` | 可信反向代理 IP / CIDR（逗号分隔），只有它们的 `X-Forwarded-For` 会被采信；留空＝不信任代理（客户端 IP 取连接地址）。nginx / ingress 部署时应填私网段，否则注册与登录的 IP 限流会被绕过或误伤 |

---

## 七、接口一览

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/api/user/register` | 注册（送 30 天免费试用） | 公开 |
| POST | `/api/user/login` | 登录（手机号 + 密码） | 公开 |
| GET | `/api/price` | 当前会员价格（标准价 / 学生价 / 平季续费 / 限时活动） | 公开 |
| GET | `/api/price/schedules/pending` | 待生效的价格预约（首页预告） | 公开 |
| POST | `/api/upload` | 上传图片（头像 / 证件，注册时不强制登录） | 公开 |
| GET | `/ping` | 健康检查 | 公开 |
| GET | `/api/user/profile` | 个人信息 + 会员信息（含 `role`/`roleName`/`isStaff`） | 登录 |
| PUT | `/api/user/profile` | 修改头像 / 用户名 / 手机号 | 登录 |
| PUT | `/api/user/password` | 修改登录密码 | 登录 |
| POST | `/api/user/member/activate` | 已关闭（返回 403），续费统一由管理员后台操作 | — |
| GET | `/api/user/payments` | 我的充值记录 + 累计充值金额 | 登录 |
| POST | `/api/feedback` | 提交反馈（≤200 字、每小时 1 次、违禁词校验） | 登录 |
| GET | `/api/feedback/mine` | 我的反馈 + 下次可提交时间 | 登录 |
| GET | `/api/user/messages` | 我的站内信列表 | 登录 |
| GET | `/api/user/messages/unread` | 未读站内信数量 | 登录 |
| PUT | `/api/user/messages/:id/read` | 标记已读 | 登录 |
| GET | `/api/orders` | 课程列表（分页 / 状态 / 关键词），登录即可查看 | 登录 |
| GET | `/api/orders/options` | 状态 / 年级 / 科目等枚举 | 登录 |
| GET | `/api/schedule` | 周课表（`date` 指定所在周，`status` 过滤） | 登录 |
| GET | `/api/lessons/exception` | 某订单的调课 / 停课 / 加课记录 | 登录 |
| POST | `/api/orders` | 新增课程 | 会员 |
| PUT | `/api/orders/:id` | 编辑课程 | 会员 |
| PUT | `/api/orders/:id/status` | 改状态 | 会员 |
| DELETE | `/api/orders/:id` | 删除课程 | 会员 |
| GET | `/api/orders/dashboard` | 课程进度与收入概览 | 会员 |
| DELETE | `/api/lessons/:id` | 作废一节已上的课次（老师未上课等） | 会员 |
| POST | `/api/lessons/exception` | 单次调课 / 停课 / 临时加课 | 会员 |
| DELETE | `/api/lessons/exception/:id` | 撤销最后一次调整 | 会员 |
| GET | `/api/admin/users` | 后台用户列表（站长看全部，管理员看普通用户） | 管理员 |
| POST | `/api/admin/users` | 添加管理员（仅站长） | 站长 |
| GET | `/api/admin/stats` | 总览 + 每日注册 / 有效会员 / 付费人数趋势 | 管理员 |
| PUT | `/api/admin/users/:id/freeze` | 冻结 / 解冻（冻结后禁止登录） | 管理员 |
| PUT | `/api/admin/users/:id/member` | 续费（`amount` + `period=daily/monthly/quarterly/yearly`），自动顺延并计入累计充值 | 管理员 |
| PUT | `/api/admin/users/:id/role` | 调整角色 | 站长 |
| DELETE | `/api/admin/users/:id` | 删除账号（有订单或充值的账号需先清空） | 站长 |
| PUT | `/api/admin/users/:id/password` | 重置普通用户登录密码 | 管理员 |
| POST | `/api/admin/messages` | 发送站内信（指定用户或全员广播） | 管理员 |
| GET | `/api/admin/feedbacks` | 后台反馈列表 | 管理员 |
| PUT | `/api/admin/feedbacks/:id` | 答复反馈并设置状态 | 管理员 |
| PUT | `/api/price` | 整体调整会员价格（兼容旧入口） | 管理员 |
| POST | `/api/price/card` | 按卡片调价 / 改折扣，支持指定生效时间 | 管理员 |
| GET | `/api/price/schedules` | 待生效价格预约列表 | 管理员 |
| DELETE | `/api/price/schedules/:id` | 撤销待生效预约 | 管理员 |
| GET | `/api/price/impact` | 价格通知预计触达人数 | 管理员 |
| GET | `/api/price/history` | 价格变更历史 | 管理员 |
| POST | `/api/price/history/:id/rollback` | 按历史记录一键回滚 | 管理员 |

订单「周时间段」存储格式（`weekly_slots` 字段，JSON 字符串）：

```json
[{"days":[1,2,3,4],"start":"18:00","end":"20:00"}]
```

`days` 取值 `1=周一 ... 7=周日`，一个订单可配置多个时间段，课表会自动展开并物化为课次。
`effectiveFrom` / `effectiveTo`（`yyyy-mm-dd`，含首尾、留空表示不限）是时段规则的生效区间：
「从某日起改为周日」只需给旧时段填 `effectiveTo`、新时段填 `effectiveFrom`，历史课表仍按当时的规则重现，不会随规则修改整体漂移。

### 数据库

启动时 `model.AutoMigrate()` 自动建表 / 补列，并由 `model/schema.go`（非空与索引）、`comments.go`（注释）对齐库表规范，无需手工执行 SQL。

| 表 | 用途 | 关键字段 |
|------|------|------|
| `user` | 账号 | `username` / `phone`（唯一登录账号）、`role`（owner/admin/user）、`status`（normal/frozen）、`user_role`（teacher 老师 / parent+personal 学员 / org 机构）、`teacher_type`（student/professional）、`verified`、`student_card_url`、`id_card_url`、`student_verified_at`、`student_expire_at`、`member_type`（trial/daily/monthly/quarterly/yearly/permanent）、`member_start`、`member_expire`、`total_recharge`、`register_ip`、`register_src`（phone/wechat/admin）、`phone_changed_at` |
| `order` | 课程 | `source`、`published_at`、`publisher`、`grade`、`order_no`、`address`、`subject` / `subjects`（多选）、`content`、`student_name`、`start_date` / `end_date`（时间戳，0＝长期）、`total_lessons`（计划课时）、`weekly_slots`（JSON）、`bill_mode`（hourly/lesson/total）、`hourly_rate` / `lesson_price` / `total_amount`、`direction`（income/expense）、`student_situation`、`student_gender`、`contact_phone`、`remark` / `remark_flag`、`status`（running/ended） |
| `lesson` | 已物化的课次快照 | `order_id`、`date`、`weekday`、`slot_index`、`start_time` / `end_time`、`duration_minutes`、`grade`、`student_name`、`subject`、`address`、`hourly_rate`、`income`、`status`（regular/canceled/movedOut/movedIn/time/extra）、`voided`（作废隐藏，不物理删除）、`exception_id`、`origin_date` / `origin_weekday`、`moved_to_date` / `moved_to_start`、`adjust_note` |
| `lesson_exception` | 单次调课 / 停课 / 加课 | `order_id`、`source_date`、`slot_index`（extra 为 -1）、`type`（cancel/move/time/extra）、`new_date` / `new_start` / `new_end`、`duration_minutes`、`income`、`note`、`history`（调整路径 JSON，撤销只回退最后一步） |
| `feedback` | 用户反馈 | `user_id`、`username`、`content`、`contact`、`reply`、`status`（pending/resolved） |
| `recharge` | 充值 / 开通记录（model 名 `Payment`） | `user_id`、`amount`、`period`（daily/monthly/quarterly/yearly）、`days`、`member_type`、`before_expire` / `after_expire`、`operator_id`、`source`（admin/self） |
| `price_config` | 会员价格配置（单行 id=1） | `pro_monthly_amount` / `pro_quarterly_amount` / `pro_yearly_amount`、`student_discount`、`renewal_discount` / `renewal_enabled`、`activity_name` / `activity_discount` / `activity_enabled` |
| `price_schedule` | 价格调整预约 | `card`（pro/student/renewal/activity）、`payload`（卡片字段 JSON）、`effective_at`（0＝立即）、`status`（pending/applied） |
| `price_history` | 价格变更历史（可回滚） | `card`、`before_value` / `after_value`（字段快照 JSON）、`source`（immediate/schedule/rollback）、`operator_id` |
| `message` | 站内信 | `user_id`（0＝广播，展开为各自记录）、`sender_id`、`sender_name`、`title`、`content`、`type`（system/promo）、`read_at`（0＝未读） |
| `student_application` | 大学生身份认证申请 | `user_id`、`student_card_url`、`id_card_url`、`status`（pending/approved/rejected）、`expire_at`、`reviewer_id`、`reviewed_at` |
| `lesson_reminder` | 开课提醒去重 | `user_id` + `order_id` + `date` + `start_time` 唯一，保证不重复推送 |
| `trial_nudge` | 试用期引导去重 | `user_id` + `kind`（activate/value/preview/urgency）唯一 |

约定与维护方式：

- **时间字段统一为秒级时间戳（BIGINT，Unix 秒）**：`created_at`、`updated_at`、`member_start`、`member_expire`、`start_date`、`end_date`；
  订单的业务字段 `published_at` 为 `TIMESTAMP`（用户填写的发布时间，零值表示未填写），`lesson.created_at/updated_at` 为 `TIMESTAMPTZ`。
- **全字段 NOT NULL**：由 `model.EnsureNotNull()` 在启动时回填默认值并补约束；索引由 `EnsureIndexes()` 统一建立与清理（model 上不再散落 index tag）。
- **`user.email`、`invite_code` 表、`student_application.invite_code` 已下线**（`015_drop_invite_code_and_email.sql`），邀请码功能整体移除。
- `server/migrations/*.sql` 为历史脚本，仅作存档与审阅：**新库不需要执行**，已有库的字段、注释、索引以代码为准。
  `008_schema_optimize.sql` 的内容已整体迁入 `model/schema.go` + `comments.go`，不再维护。

---

## 八、二维码替换说明

首页底部三个二维码位（微信好友、微信收款、支付宝收款）已通过 `web/src/config/site.js` 配置，默认指向 `web/public/qrcode/` 下的 `wechat-river.png` / `wechat-pay.png` / `zhifubao-pay.png`。替换方式：

1. 把新的图片放到 `web/public/qrcode/`（文件名保持一致，或改 `site.js` 中的路径）；
2. 重新构建：单机 `make web-build`；k8s `make images-build` 后 `kubectl -n tutoring rollout restart deploy/web`。未上传时显示「二维码待上传」占位，不影响其他功能。

---

## 九、后续可迭代方向

- 大学生认证的后台审核入口与微信扫码支付回调自动开通（当前为线下转账 + 管理员人工开通）
- 订单导入（微信群聊文本解析批量导入）
- 课表冲突检测与提醒、按月视图
- 课程 / 课时统计报表导出（课表图片导出已支持）
- 多老师 / 机构多账号协作与数据隔离
- 邀请码 / 分销体系（当前已整体下线，见 `015_drop_invite_code_and_email.sql`）
