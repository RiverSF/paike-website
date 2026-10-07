# 上线安全加固记录（SECURITY HARDENING）

> 本文档记录家教助手站点上线前已完成的安全修复，以及仍需运维/部署侧落实的检查项。
> 代码侧修改均已通过 `go build` 验证。

---

## 一、本次已修复（代码层）

### 1. CORS 跨域策略收紧（高危）
- **风险**：原 `cors.go` 在 `CORS_ALLOW_ORIGINS` 未配置时，会把浏览器发来的任意 `Origin` 反射到 `Access-Control-Allow-Origin`，同时返回 `Access-Control-Allow-Credentials: true`。任意第三方网站可携带用户凭证跨域调用本站 API，读取/操作用户订单、会员信息。
- **修复**：`server/internal/middleware/cors.go`
  - 默认**拒绝一切跨域请求**（不返回 CORS 头，浏览器直接拦截）。
  - 仅当请求 `Origin` 命中 `CORS_ALLOW_ORIGINS` 白名单时才放行并允许凭证。
  - 新增 `splitOrigins` 解析逗号分隔白名单。
- **验证**：未配置 `CORS_ALLOW_ORIGINS` 时，跨域前端调用应被浏览器拦截；配置为你的前端域名后同源/白名单域名可正常访问。

### 2. 生产环境强制强 JWT 密钥（严重）
- **风险**：`JWT_SECRET` 多处存在默认弱值（`tutoring-website-secret` / `please-change-this-secret`），Docker 镜像与 k8s Secret 默认即此值。攻击者可用其签发任意用户（含 owner/admin）令牌，完全接管系统。
- **修复**：`server/main.go`
  - 新增 `isWeakJwtSecret()` 与弱密钥集合（含已知的 `tutoring-website-secret`、`please-change-this-secret` 及空值）。
  - `RUN_MODE=release` 时，若 `JWT_SECRET` 为空或命中弱密钥集合或长度 `<32`，**直接拒绝启动**（`log.Fatalf`）。
- **验证**：在 release 模式下不设置/设置弱 `JWT_SECRET` 启动，进程应立即退出并报错；设置 ≥32 字节高熵随机值后可正常启动。

### 3. 上传接口加固（高危）
- **风险**：`/api/upload` 为公开接口（注册需未登录上传证件），原仅校验扩展名且允许 `.svg`，存在：①匿名无限上传耗尽磁盘；②上传 SVG 触发存储型 XSS；③伪造扩展名上传可执行内容。
- **修复**：`server/internal/handler/upload.go`
  - 从允许列表**移除 `.svg`**。
  - 新增基于文件头 magic number 的**真实 MIME 校验**（`http.DetectContentType`，仅放行 jpeg/png/gif/webp）。
  - 新增按客户端 IP 的**匿名上传频控**（单实例，60s 内 ≤20 次）。
- **注意**：频控为进程内实现，多副本部署不共享计数，需后续改为 Redis 或对象存储。
- **验证**：上传 `.svg` 或非图片文件应被拒绝；正常图片可上传；同一 IP 高频上传将返回 429。

### 4. 统一安全响应头（中）
- **修复**：新增 `server/internal/middleware/security.go` 并在 `router.go` 全局注册
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: no-referrer`
  - `Content-Security-Policy: frame-ancestors 'none'`（防 iframe 点击劫持，不影响前端功能）
- **验证**：任意响应头应包含上述字段。

### 5. 全局请求体大小限制（中）
- **风险**：除上传 5MB 外，其他接口无请求体上限，存在大请求 DoS 风险。
- **修复**：`server/internal/router/router.go` 新增全局中间件，使用 `http.MaxBytesReader` 限制请求体 ≤10MB（与 nginx/ingress 的 `10m` 保持一致）。
- **说明**：未使用 `http.Server.MaxBodyBytes` 字段，以兼容较低版本 Go。

### 6. 反馈列表仅本人可见（中）
- **风险**：`GET /api/feedback` 公开返回全部用户的反馈正文，未登录/登录用户均可查看他人反馈内容，存在隐私泄露（正文可能含手机号、姓名等 PII）。
- **修复**：移除公开的 `FeedbackList` 接口与路由；反馈查询统一收敛到需登录的 `GET /api/feedback/mine`，仅返回当前用户本人提交的反馈，杜绝查看他人内容。
- **验证**：未登录或登录其他账号调用 `/api/feedback` 均返回 404；`/api/feedback/mine` 仅返回本人数据。

### 7. 部署配置补充（提示）
- `deploy/k8s/01-configmap.yaml`：补充 `CORS_ALLOW_ORIGINS` 配置说明，默认留空（拒绝跨域），上线请填前端域名。
- `deploy/k8s/02-secret.yaml`：仍为示例弱值，**上线前必须替换**（见下文检查项 2）。

---

### 8. 站点拥有者（owner）冷启动初始化（严重）
- **风险**：原系统无任何 owner 时，因邀请码需管理员发放、管理员需 owner 创建，形成"无 owner 则无管理员"的冷启动死锁；且 `model/user.go:35` 注释暗示"ID=1 即 owner"，存在首注册者误成超级管理员的隐患。
- **修复**：新增 `server/internal/model/seed.go` 的 `SeedOwner()`，在 `main.go` 启动 DB 后调用。逻辑：
  - 已有 owner 则跳过（幂等）；
  - 否则读取 `INIT_OWNER_USERNAME` / `INIT_OWNER_PHONE` / `INIT_OWNER_PASSWORD` 三件套，**仅当齐全且格式合法**（用户名 2-32 位、11 位手机号、密码 ≥8 位、用户名/手机号不重复）时才自动创建首个 owner（永久会员）；
  - 未配置则仅打印告警日志，系统处于待初始化状态，需手工执行 `server/migrations/002_role.sql`。
- **验证**：首次启动配置三件套后，日志应打印"已初始化站点拥有者账号"；未配置且库空时打印告警且不创建任何账号。
- **注意**：绝不回退到默认账号/密码；初始化后请尽快登录后台修改密码。

### 9. K8s Ingress TLS（严重）
- **修复**：`deploy/k8s/06-ingress.yaml` 增加 `spec.tls` 引用 `tutoring-tls` Secret，并加注解 `ssl-redirect: "true"` 强制 HTTPS。
- **仍需**：创建证书 Secret（`kubectl create secret tls tutoring-tls --cert=xxx.crt --key=xxx.key`，或 cert-manager 自动签发）；将 host 由 `tutoring.local` 改为真实域名。**Web Pod 内部 nginx 不终止 TLS**（TLS 在 ingress 层终止），无需改动 `nginx.conf`。

---

## 二、仍需上线前落实（运维 / 部署侧，代码已就位但未自动解决）

| # | 事项 | 必须 | 操作指引 |
|---|------|------|----------|
| 1 | **HTTPS / TLS 证书** | 是 | K8s Ingress 已加 `spec.tls` 与 `ssl-redirect`，但**还需创建证书 Secret**：`kubectl create secret tls tutoring-tls --cert=xxx.crt --key=xxx.key`（或 cert-manager）。将 host 由 `tutoring.local` 改为真实域名。单机/Gin 直听场景需在前置 nginx 做 443 + HTTP→HTTPS 跳转并开启 HSTS（当前 `deploy/nginx/nginx.conf` 仅 `listen 80`）。|
| 2 | **强 JWT_SECRET 注入** | 是 | Docker：`docker run -e JWT_SECRET=$(openssl rand -hex 32) ...`；k8s：`kubectl -n tutoring create secret generic tutoring-secret --from-literal=JWT_SECRET=$(openssl rand -hex 32) --dry-run=client -o yaml \| kubectl apply -f -`（或编辑 `02-secret.yaml` 后 `make k8s-apply`）。未注入则 release 模式启动失败。|
| 3 | **数据库强密码 + SSL** | 是 | 修改 `02-secret.yaml` 的 `POSTGRES_PASSWORD`/`PG_PASSWORD` 为强随机值；`PG_SSLMODE` 改为 `require` 并配置证书；数据库仅允许应用网段访问（安全组 / NetworkPolicy）。|
| 4 | **CORS 白名单** | 是 | 配置 `CORS_ALLOW_ORIGINS` 为你的前端域名（如 `https://www.river.site`），**切勿设为 `*`**。|
| 5 | **owner 初始化** | 是 | 已支持自动初始化：首次启动配置 `INIT_OWNER_USERNAME`/`INIT_OWNER_PHONE`/`INIT_OWNER_PASSWORD`（三件套齐全且合法）即自动创建首个 owner（见 seed.go）。未配置则系统待初始化，需手工执行 `server/migrations/002_role.sql`。初始化后请尽快登录后台修改密码。|
| 6 | **上传存储改为对象存储 / RWM** | 建议 | 当前 k8s 上传 PVC 为 ReadWriteOnce，server 扩容前须改对象存储（S3/OSS/COS）或 RWM 共享卷，否则多副本丢数据（README 已提示）。|
| 7 | **敏感证件合规** | 建议 | 用户上传的身份证/学生证属 PII，建议加密/权限隔离存储、仅审核员可见、遵守最小必要原则。|
| 8 | **备份与数据持久化** | 建议 | PostgreSQL 数据（充值/会员）重要，建立定期备份；生产建议云托管 + 自动备份。|
| 9 | **依赖与供应链审计** | 建议 | 上线前执行 `go mod tidy` + `npm audit`，确认无已知高危漏洞。|
| 10 | **登录/注册限流增强** | 建议 | `loginlimit.go` 为进程内、仅账号维度；建议改为 Redis 共享 + 增加 IP 维度；注册接口加验证码/邀请码速率限制。|

---

## 三、验证清单（上线前逐项确认）

- [ ] `cd server && go build .` 通过（已验证 BUILD_OK）
- [ ] release 模式下**未**设置强 JWT_SECRET 时进程拒绝启动（已验证逻辑）
- [ ] release 模式下设置 `JWT_SECRET=$(openssl rand -hex 32)` 可正常启动
- [ ] 未配置 `CORS_ALLOW_ORIGINS` 时跨域请求被拦截；配置白名单后正常
- [ ] `/api/upload` 拒绝 `.svg` 与非图片文件；高频上传返回 429
- [ ] 响应头包含 `X-Content-Type-Options` / `X-Frame-Options` / `Content-Security-Policy`
- [ ] `/api/feedback` 未登录响应不含用户名/联系方式
- [ ] 全站 HTTPS 已启用，HTTP 跳转 HTTPS
- [ ] 数据库强密码 + SSL 已启用，网络仅对应用开放
- [ ] `CORS_ALLOW_ORIGINS` 已设为前端域名（非 `*`）
- [ ] owner 账号已初始化，注册入口对外前 owner 已就位
- [ ] 上传存储（对象存储/RWM）已就绪或 server 维持单副本
- [ ] 数据库备份策略已建立
- [ ] K8s `tutoring-tls` 证书 Secret 已创建（或 cert-manager 自动签发），Ingress host 已改为真实域名
- [ ] 已配置 `INIT_OWNER_*` 三件套完成首个 owner 初始化（日志可见"已初始化站点拥有者账号"）
- [ ] 初始化完成后已登录后台修改 owner 密码

---

## 四、修改文件清单

- `server/internal/middleware/cors.go` — CORS 白名单化
- `server/internal/middleware/security.go` — 新增安全响应头中间件（新建）
- `server/internal/router/router.go` — 注册安全头 + 请求体限制中间件
- `server/internal/handler/upload.go` — 禁 SVG、MIME 校验、IP 频控
- `server/internal/handler/feedback.go` — 公开反馈脱敏
- `server/main.go` — release 模式 JWT 强密钥校验、弱密钥集合
- `deploy/k8s/01-configmap.yaml` — CORS 白名单配置说明
- `server/internal/model/seed.go` — 新增 SeedOwner（owner 冷启动初始化）
- `deploy/k8s/06-ingress.yaml` — 增加 TLS 与 HTTPS 强制跳转
- `deploy/k8s/02-secret.yaml` — 补充 INIT_OWNER_* 初始化说明
