# 排课助手 · 课程排期与课表管理

![vibe coding](https://img.shields.io/badge/vibe--coding-AI%20%E9%A9%B1%E5%8A%A8-ff69b4)
![Vue 3](https://img.shields.io/badge/Vue-3-42b883)
![Go](https://img.shields.io/badge/Go-Gin-00add8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-336791)

面向家长、老师与个人的排课工具：录入课程后，系统按「补课周期 + 每周时段」自动生成周课表。

- 前端：Vue 3 + Vite + Element Plus + Pinia
- 后端：Go + Gin + GORM
- 数据库：PostgreSQL
- 部署：单机部署 / Kubernetes 部署（两种可切换）

导航栏：**首页 / 课程安排 / 课表 / 使用指南**，右上角头像查看个人信息与会员信息。前端访问端口为 **80**。

## 功能演示

![功能演示](web/public/videos/guide-demo.gif)

> 演示视频约 60 秒，与前端「使用指南 / 首页」同款；上方为静音 GIF 预览，GitHub 网页与本地 IDE 均会内联渲染。完整带声版本见 `web/public/videos/guide-demo.mp4`（[在线观看](https://github.com/RiverSF/paike-website/raw/main/web/public/videos/guide-demo.mp4)）。

## 快速开始（开发模式）

1. 准备 PostgreSQL 并创建数据库：

```bash
createdb tutoring
cp server/internal/conf/app.ini.example server/internal/conf/app.ini   # 修改 [postgres] 配置
```

2. 启动后端（开发态 9090，避免与前端 80 冲突）：

```bash
make server-dev        # HTTP_PORT=9090 go run .
```

3. 启动前端（监听 80）：

```bash
make web-install
sudo make web-dev      # macOS/Linux 绑定 80 需要 sudo
# 访问 http://127.0.0.1
```

> 开发态前端由 vite 提供，后端不托管前端产物；`/api`、`/uploads` 由 vite 代理到 9090。若本机 80 被占用（如 Homebrew nginx），见「单机部署 · 端口占用」。

---

## 功能说明

| 模块 | 说明 |
|------|------|
| 首页 | 站点简介、核心功能、三步上手、会员权益与实时价格卡片（含待生效的价格预告）；底部提供「微信好友 / 微信收款 / 支付宝收款」二维码位 |
| 免注册试用 | **不登录也能排课**：数据存在本机浏览器（localStorage），最多 3 门课程；导出课表、费用统计、调课提醒等进阶功能提示注册解锁，注册后可把试用数据导入账号 |
| 课程安排 | 课程列表（状态可直接切换、进度与收入统计）、新增/编辑/删除；字段含：信息来源、发布时间、发布人、年级、订单编号、地址、科目（多选）、辅导内容、补课起止日期、周时间段（多段，支持生效区间）、计划总课时、计费方式（按小时 / 按单次课时 / 按整期总额）、时薪 / 课时价 / 总费用、收支方向、学生情况、学生性别、联系电话、备注、状态 |
| 课表 | 按自然周展示（默认当前周），x 轴周一~周日、y 轴按小时；课程按「时间范围 + 每周时段」展开为**已物化的课次**；支持**调课 / 停课 / 临时加课**（单次例外，保留调整路径与撤销）、删除已上课次（作废而非物理删除）；备注较长课程用红点标记；支持上/本/下周、日期选择器切换与**导出图片（带 logo 水印）** |
| 使用指南 | 使用提示、常见问题 FAQ、演示视频、反馈提交框与已反馈内容列表 |
| 站内信 | 管理员 / 站长可发给指定用户或全员广播；系统自动推送开课提醒、会员到期提醒、价格调整通知与试用期引导（均有去重，不会重复打扰） |
| 角色 | `user` 普通用户 / `admin` 管理员 / `owner` 站点拥有者；**ID=1 的用户为站点拥有者**，具备管理员权限且为永久会员 |
| 注册/会员 | **手机号必填**（11 位中国大陆号码，查重 + 格式校验），注册即送 **30 天免费试用**；课程与课表**始终可查看**，新增 / 调整等写操作需会员未过期；支持按天 / 包月 / 包季 / 包年续费 |
| 登录方式 | **手机号 + 密码**（用户名不能登录）；手机号每月仅可修改一次 |
| 使用身份 | `user_role`（`teacher` 老师 / `parent` 学员 / `org` 机构）注册时选择，**仅影响课程表单字段与文案，不参与计费**。师资身份 `teacher_type`（`student` 大学生家教 / `professional` 专职老师）**已从注册流程下线**：新用户统一按专职标准价计费；学生折扣价仅对历史大学生家教账号生效，相关审核入口尚未接入前端 |
| 会员价格 | **标准价**（默认包月 79 / 包季 225 / 包年 790）+ **大学生折扣价**（标准价 × 学生折扣，向上取整，仅历史学生账号适用）+ **平季续费折扣**（9 折，可开关）+ **限时活动**（自定义折扣，可叠加）；价格改动支持「立即生效 / 预约生效」，每次改动入历史可一键回滚；用户扫码支付后联系管理员人工开通 |
| 充值与累计 | **自助续费入口已关闭**，续费统一由管理员在后台填写**充值金额**与**续费时长（按天/包月/包季/包年）**；系统自动在原到期时间上顺延、累加**累计充值金额**并生成充值记录；个人中心可查看每次充值金额、时长与到期变化 |
| 管理端 | 站点拥有者、管理员可见「会员总览 / 用户列表 / 价格设置 / 用户反馈 / 站内信」五个页签：总览含注册与会员趋势图；用户列表支持冻结解冻、续费、改角色、重置密码、删除账号；价格设置支持按卡片调价、预约生效、撤销预约与按历史回滚 |
| 个人中心 | 注册信息（用户名、头像、手机号、注册时长）+ 会员信息（会员类型、有效时段、剩余天数）+ 充值记录与累计金额 |

---

## 目录结构

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
│   │   └── views/            首页 / 课程安排 / 课表 / 使用指南 / 管理端
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

## 部署方式一：单机部署（standalone）

**特点**：只跑一个后端进程，后端直接托管前端构建产物（`web/dist`），无需 nginx。

```bash
make web-build          # 构建前端产物到 web/dist
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

80 端口需要 root 权限；若本机已有服务占用（常见为 Homebrew nginx），会出现「能启动但只能走 IPv6 / `localhost` 访问不通」。三种处理方式：

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

也可只编译二进制后用 systemd / supervisor 托管：

```bash
cd server && go build -o bin/tutoring-server .
APP_ROOT=/path/to/tutoring-website/server ./bin/tutoring-server
```

> `APP_ROOT` 用于二进制不在项目根目录运行时定位 `internal/conf/app.ini`、`data/uploads`。

---

## 部署方式二：Kubernetes 部署

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

## 配置项与环境变量

配置文件：`server/internal/conf/app.ini`（复制自 `app.ini.example`），**所有配置均可被同名环境变量覆盖**。k8s 部署时环境变量由 `deploy/k8s/01-configmap.yaml`（非敏感）与 `02-secret.yaml`（敏感）通过 `envFrom` 注入。

| 配置项 (ini) | 环境变量 | 默认值 | 说明 |
|------|---------|------|------|
| `RUN_MODE` | `RUN_MODE` | `debug` | `debug` / `release` |
| `server.HTTP_PORT` | `HTTP_PORT` | `80`（开发态 `9090`） | 监听端口 |
| `server.READ_TIMEOUT` / `WRITE_TIMEOUT` | `READ_TIMEOUT` / `WRITE_TIMEOUT` | `60` | 读写超时（秒） |
| `server.TRUSTED_PROXIES` | `TRUSTED_PROXIES` | 私网段 | 可信代理 IP / CIDR；nginx / ingress 后部署填私网段，直连对外可留空（否则注册 / 登录的 IP 限流会被绕过） |
| `app.STATIC_DIR` | `STATIC_DIR` | 空 | 前端产物目录；单机部署设为 `../web/dist`，`none` / 空则由 nginx 托管 |
| `app.UPLOAD_DIR` | `UPLOAD_DIR` | `data/uploads` | 上传目录 |
| `postgres.*` | `PG_HOST` / `PG_PORT` / `PG_USER` / `PG_PASSWORD` / `PG_DB` / `PG_SSLMODE` | `127.0.0.1` / `5432` / `postgres` / `tutoring` / `disable` | 数据库连接 |
| `jwt.JWT_SECRET` | `JWT_SECRET` | — | **上线务必修改** |
| `jwt.JWT_EXPIRE_HOURS` | `JWT_EXPIRE_HOURS` | `168` | Token 有效期（小时） |
| — | `CONFIG_PATH` | — | 指定配置文件路径（容器内使用） |
| — | `APP_ROOT` | — | 应用根目录（二进制部署使用） |
| — | `LOG_STDOUT` | — | `=1` 时日志只输出到标准输出 |
| — | `CORS_ALLOW_ORIGINS` | 全部 | 逗号分隔的允许跨域源 |

