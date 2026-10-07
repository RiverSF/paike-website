# 排课助手 · 课程排期与课表管理

面向家长、老师与个人的排课工具：录入课程后，系统按「补课周期 + 每周时段」自动生成周课表。

- 前端：Vue 3 + Vite + Element Plus + Pinia
- 后端：Go + Gin + GORM
- 数据库：PostgreSQL
- 部署：单机部署 / Kubernetes 部署（两种可切换，见下文）

导航栏：**首页 / 订单 / 课表 / 使用指南**，右上角头像可查看个人信息与会员信息。前端访问端口为 **80**。

> 使用身份：用户表 `user_role` 字段（`teacher` 老师 / `parent` 家长 / `personal` 个人 / `org` 机构），注册时选择，**仅影响课程表单字段与文案**（家长隐藏「信息来源 / 发布人 / 发布时间」，个人隐藏「课程编号 / 学生性别」并把「年级」改为「课程类型」、「学生姓名」改为「课程名称」），**不参与计费**；计费仍由师资身份 `teacher_type` 决定：大学生家教（需上传学生证与身份证并通过审核）享学生折扣，其余身份（含机构）统一走标准价。仅「老师」可选择大学生家教身份。
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
│   │   ├── stores/           Pinia（用户/会员）
│   │   ├── styles/           全局样式与主题色
│   │   └── views/            首页 / 订单 / 课表 / 使用指南
│   └── public/               logo、默认头像、二维码目录(qrcode/)
├── server/                   后端（Gin）
│   ├── main.go
│   ├── internal/
│   │   ├── conf/             配置 app.ini（同 .example）
│   │   ├── config/           配置加载（支持环境变量覆盖）
│   │   ├── handler/          业务处理（user/order/schedule/feedback/upload）
│   │   ├── middleware/       JWT 鉴权、会员校验、CORS、日志
│   │   ├── model/            GORM 模型（user/order/feedback）
│   │   └── router/           路由注册 + SPA 托管
│   ├── migrations/           初始化 SQL（启动也会自动 AutoMigrate）
│   └── pkg/                  日志、路径、通用工具
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
| 首页 | 站点简介、核心功能、三步上手、会员权益；底部提供「微信好友 / 微信收款 / 支付宝收款」二维码位（图片后续上传） |
| 课程安排 | 课程列表（状态可直接在列表中切换）、新增/编辑/删除，字段含：信息来源、发布时间、发布人、年级、订单编号、地址、科目、辅导内容、补课时间、周时间段（支持多个）、单次时长、时薪、学生情况、学生性别、备注、状态 |
| 课表 | 按自然周展示（默认当前周），x 轴周一~周日、y 轴按小时；课程按「时间范围 + 每周时段」自动展开；备注较长的课程用红点标记，鼠标悬浮查看完整信息；支持上一周/本周/下一周与日期选择器切换 |
| 使用指南 | 使用提示、常见问题 FAQ、反馈提交框与已反馈内容列表 |
| 角色 | `user` 普通用户 / `admin` 管理员 / `owner` 站点拥有者；**ID=1 的用户为站点拥有者**，具备管理员权限且为永久会员 |
| 注册/会员 | **手机号必填**（11 位中国大陆号码，与邮箱均做查重与格式校验），注册即送 **30 天免费会员**；课程与课表仅对会员未过期用户开放；支持包月 / 包年续费 |
| 登录方式 | 用户名 / 手机号 / 邮箱 均可登录 |
| 管理端 | 站点拥有者、管理员可见「会员管理」「用户反馈」两个菜单：会员管理含用户列表（冻结/解冻、**充值续费**）与注册/会员趋势图；用户反馈含答复与审核 |
| 充值与累计 | **自助续费入口已关闭**，续费统一由管理员在会员管理中填写**充值金额**与**续费时长（一月/一年）**；系统自动在原到期时间上顺延、累加**累计充值金额**并生成充值记录；个人中心可查看每次充值金额、时长与到期变化 |
| 会员价格 | 两档：**标准价**（老师 / 家长 / 个人 / 机构，默认包月 79 / 包季 225 / 包年 790）+ **大学生折扣价**（标准价 × 折扣 8.5 折向上取整，需认证）；机构当前与个人老师同价；用户扫码支付后联系管理员人工开通 |
| 个人中心 | 注册信息（用户名、头像、手机号、邮箱、注册时长）+ 会员信息（会员类型、包月/包年、有效时段、剩余天数） |

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

---

## 七、接口一览

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/api/user/register` | 注册（送 30 天会员） | 公开 |
| POST | `/api/user/login` | 登录 | 公开 |
| GET | `/api/user/profile` | 个人信息 + 会员信息（含 `role`/`roleName`/`isStaff`） | 登录 |
| PUT | `/api/user/profile` | 修改头像/手机号/邮箱 | 登录 |
| POST | `/api/user/member/activate` | 已关闭（返回 403），续费统一由管理员后台操作 | — |
| POST | `/api/upload` | 上传图片（头像） | 登录 |
| GET | `/api/orders` | 订单列表（分页/状态/关键词/年级） | 会员 |
| POST | `/api/orders` | 新增订单 | 会员 |
| PUT | `/api/orders/:id` | 编辑订单 | 会员 |
| PUT | `/api/orders/:id/status` | 改状态 | 会员 |
| DELETE | `/api/orders/:id` | 删除订单 | 会员 |
| GET | `/api/orders/options` | 状态/年级/科目枚举 | 会员 |
| GET | `/api/schedule` | 周课表（`date` 指定所在周，`status` 过滤） | 会员 |
| GET | `/api/feedback` | 已反馈列表 | 公开 |
| POST | `/api/feedback` | 提交反馈（≤200 字、每小时 1 次、违禁词校验） | 登录 |
| GET | `/api/feedback/mine` | 我的反馈 + 下次可提交时间 | 登录 |
| GET | `/api/admin/users` | 后台用户列表（拥有者看全部，管理员看普通会员） | 管理员 |
| GET | `/api/admin/stats` | 概览 + 每日注册人数 / 有效会员人数趋势 | 管理员 |
| PUT | `/api/admin/users/:id/freeze` | 冻结 / 解冻普通会员（冻结后禁止登录） | 管理员 |
| PUT | `/api/admin/users/:id/member` | 会员续费（`amount` + `period=monthly/yearly`），自动算到期并计入累计充值 | 管理员 |
| GET | `/api/user/recharges` | 我的充值记录 + 累计充值金额 | 登录 |
| GET | `/api/admin/feedbacks` | 后台反馈列表（`all=1` 含管理员提交） | 管理员 |
| PUT | `/api/admin/feedbacks/:id` | 答复反馈并设置审核状态 | 管理员 |
| GET | `/api/feedback/mine` | 我的反馈 | 登录 |
| GET | `/ping` | 健康检查 | 公开 |

订单「周时间段」存储格式（`weekly_slots` 字段，JSON 字符串）：

```json
[{"days":[1,2,3,4],"start":"18:00","end":"20:00"}]
```

`days` 取值 `1=周一 ... 7=周日`，一个订单可配置多个时间段，课表会自动展开。

### 数据库

- 表：`user`、`order`、`feedback`（GORM 启动时 AutoMigrate 自动建表 / 补列）
- **时间字段统一为秒级时间戳（BIGINT，Unix 秒）**：`created_at`、`updated_at`、`member_start`、`member_expire`
  （订单的业务字段 `published_at` 仍为 `TIMESTAMP`，由用户填写发布时间）
- 表与字段注释：启动时由 `model.EnsureComments()` 自动写入，也可手工执行 `server/migrations/` 下的 SQL
  - `001_init.sql` 建表 + 注释
  - `002_role.sql` 角色字段 + ID=1 拥有者初始化
  - `003_comments.sql` 仅注释（已有库可直接执行补全）
  - `004_timestamp.sql` 已有库的时间字段转为时间戳（`EXTRACT(EPOCH)` 无损转换，需停机执行一次）
  - `005_user_status.sql` 账号状态字段（冻结）+ 手机号/邮箱唯一索引
  - `006_recharge.sql` 累计充值金额字段 + 充值记录表 `recharge`

---

## 八、二维码替换说明

首页底部三个二维码位预留未上传。替换方式：

1. 把图片放到 `web/public/qrcode/`，例如 `wechat-friend.png`、`wechat-pay.png`、`alipay.png`；
2. 在 `web/src/config/site.js` 中填写路径：

```js
export const qrcodes = {
  wechatFriend: '/qrcode/wechat-friend.png',
  wechatPay: '/qrcode/wechat-pay.png',
  alipay: '/qrcode/alipay.png'
}
```

3. 重新构建：单机 `make web-build`；k8s `make images-build` 后 `kubectl -n tutoring rollout restart deploy/web`。未上传时显示「二维码待上传」占位，不影响其他功能。

---

## 九、后续可迭代方向

- 订单导入（微信群聊文本解析批量导入）
- 课表冲突检测与提醒、按月视图
- 支付回调自动开通会员（现为线下转账 + 自助开通）
- 多老师 / 机构多账号协作与数据隔离
- 课表导出（图片 / PDF）、课时统计报表
