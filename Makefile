# 家教助手 - 构建与部署 Makefile
# 三种启动模式：standalone(单机) / dev(热更新) / k8s

.PHONY: help web-install web-dev web-build server-build server-run server-dev \
        standalone stop restart \
        dev dev-stop dev-restart \
        images-build k8s-apply k8s-delete k8s-status k8s-logs k8s-port-forward k8s-restart

# ===== 端口配置（stop / dev-stop 依此释放端口）=====
# 单机部署：后端监听端口
STANDALONE_PORT ?= 80
# 开发热更新：前端 vite 端口
DEV_WEB_PORT    ?= 5173
# 开发热更新：后端端口
DEV_API_PORT    ?= 9090
IMAGE_TAG       ?= latest

help:
	@echo "家教助手 - 常用命令"
	@echo ""
	@echo "【模式一 · 单机部署 standalone】后端 :$(STANDALONE_PORT) 直接托管前端（演示/生产用）"
	@echo "  make standalone   启动：构建前端 + 编译后端 + 监听 :$(STANDALONE_PORT)"
	@echo "  make stop         停止：杀掉占用 :$(STANDALONE_PORT) 的后端进程"
	@echo "  make restart      重启：= stop + standalone（改代码后一键重启）"
	@echo "  make web-build    仅构建前端到 web/dist        make server-build 仅编译后端到 server/bin/"
	@echo ""
	@echo "【模式二 · 开发 dev】前端 Vite HMR 热更新（改前端代码自动生效）；后端 go run 非热更新，改 Go 代码后需 make dev-restart"
	@echo "  make dev          启动：后端 :$(DEV_API_PORT) + 前端 :$(DEV_WEB_PORT)（一条命令后台启动）"
	@echo "  make server-dev   仅后端热更新启动 (:$(DEV_API_PORT)，另开终端)"
	@echo "  make web-dev      仅前端热更新启动 (:$(DEV_WEB_PORT)，另开终端)"
	@echo "  make dev-stop     停止：杀掉 :$(DEV_WEB_PORT)/:$(DEV_API_PORT) 及 vite、go run 进程"
	@echo "  make dev-restart  重启：= dev-stop + dev"
	@echo "  访问 http://127.0.0.1:$(DEV_WEB_PORT) （/api 经 vite 代理到 :$(DEV_API_PORT)）"
	@echo ""
	@echo "【模式三 · k8s 部署】"
	@echo "  make images-build     构建镜像 (tutoring-server / tutoring-web:$(IMAGE_TAG))"
	@echo "  make k8s-apply        部署：kubectl apply -k deploy/k8s"
	@echo "  make k8s-restart      重启：rollout restart deploy/server deploy/web"
	@echo "  make k8s-delete       卸载：kubectl delete -k deploy/k8s"
	@echo "  make k8s-status       状态：pods/svc/ingress        make k8s-logs 后端日志"
	@echo "  make k8s-port-forward  端口转发 (http://127.0.0.1:8080)"

# ===== 公共：依赖安装与产物构建 =====
web-install:
	cd web && npm install

web-build:
	cd web && npm run build

server-build:
	cd server && GOTOOLCHAIN=local go build -o bin/tutoring-server .

# ===== 模式一 · 单机部署 standalone（后端 :$(STANDALONE_PORT) 托管前端）=====
# 以源码直接运行后端（默认端口 :$(STANDALONE_PORT)），调试用
server-run:
	cd server && GOTOOLCHAIN=local go run .

# 构建前端 + 编译后端，并启动监听 :$(STANDALONE_PORT)
standalone: web-build server-build
	cd server && GOTOOLCHAIN=local ./bin/tutoring-server

# 停掉占用单机端口的旧后端进程（避免 listen :$(STANDALONE_PORT) address already in use）
stop:
	@echo "==> stopping standalone server on :$(STANDALONE_PORT) ..."
	@pids=$$(lsof -iTCP:$(STANDALONE_PORT) -sTCP:LISTEN -t 2>/dev/null); \
	if [ -n "$$pids" ]; then \
	  for p in $$pids; do \
	    if kill -9 $$p 2>/dev/null; then echo "  [ok] killed :$(STANDALONE_PORT) pid $$p"; else echo "  [fail] cannot kill pid $$p"; fi; \
	  done; \
	else \
	  echo "  no process listening on :$(STANDALONE_PORT)"; \
	fi
	@if pkill -9 -f 'bin/tutoring-ser[v]er' 2>/dev/null; then echo "  [ok] killed leftover bin/tutoring-server"; fi
	@if lsof -iTCP:$(STANDALONE_PORT) -sTCP:LISTEN -t 2>/dev/null | grep -q .; then echo "  [warn] :$(STANDALONE_PORT) still in use"; else echo "==> :$(STANDALONE_PORT) freed"; fi

# 重启：先停旧进程，再重建并启动
restart: stop standalone

# ===== 模式二 · 开发热更新 dev（vite HMR，改代码浏览器自动刷新）=====
# 仅后端热更新（端口 :$(DEV_API_PORT)，前端经 vite :$(DEV_WEB_PORT) 代理 /api）
server-dev:
	cd server && HTTP_PORT=$(DEV_API_PORT) GOTOOLCHAIN=local go run .

# 仅前端热更新（端口 :$(DEV_WEB_PORT)）
web-dev:
	cd web && npm run dev

# 一条命令后台启动 后端(:$(DEV_API_PORT)) + 前端(:$(DEV_WEB_PORT))；分开调试可用 server-dev / web-dev
dev:
	@echo "starting dev mode: web(:$(DEV_WEB_PORT)) + api(:$(DEV_API_PORT)) ..."
	cd web && nohup npm run dev > /tmp/web-dev.log 2>&1 &
	cd server && nohup env HTTP_PORT=$(DEV_API_PORT) GOTOOLCHAIN=local go run . > /tmp/server-dev.log 2>&1 &
	@echo "done. open http://127.0.0.1:$(DEV_WEB_PORT)"
	@echo "note: 前端 HMR 热更新，改前端代码自动生效；后端改 Go 代码后请执行 make dev-restart"

# 停止开发模式：释放前端/后端端口并清理 vite、go run 进程
dev-stop:
	@echo "==> stopping dev mode (web :$(DEV_WEB_PORT) / api :$(DEV_API_PORT)) ..."
	@for port in $(DEV_WEB_PORT) $(DEV_API_PORT); do \
	  pids=$$(lsof -iTCP:$$port -sTCP:LISTEN -t 2>/dev/null); \
	  if [ -n "$$pids" ]; then \
	    for p in $$pids; do \
	      if kill -9 $$p 2>/dev/null; then echo "  [ok] killed :$$port pid $$p"; else echo "  [fail] cannot kill :$$port pid $$p"; fi; \
	    done; \
	  else \
	    echo "  no process listening on :$$port"; \
	  fi; \
	done
	@if pkill -9 -f '[v]ite' 2>/dev/null; then echo "  [ok] killed vite processes"; fi
	@if pkill -9 -f '[g]o run' 2>/dev/null; then echo "  [ok] killed go run processes"; fi
	@still=""; \
	for port in $(DEV_WEB_PORT) $(DEV_API_PORT); do \
	  if lsof -iTCP:$$port -sTCP:LISTEN -t 2>/dev/null | grep -q .; then still="$$still :$$port"; fi; \
	done; \
	if [ -n "$$still" ]; then echo "  [warn] still listening on:$$still"; else echo "==> dev ports freed"; fi

# 重启开发模式
dev-restart: dev-stop dev

# ===== 模式三 · k8s 部署 =====
images-build:
	TAG=$(IMAGE_TAG) ./deploy/scripts/build-images.sh

k8s-apply:
	kubectl apply -k deploy/k8s

k8s-delete:
	kubectl delete -k deploy/k8s

k8s-restart:
	kubectl -n tutoring rollout restart deploy/server deploy/web

k8s-status:
	kubectl -n tutoring get pods,svc,ingress,pvc -o wide

k8s-logs:
	kubectl -n tutoring logs -f deploy/server

k8s-port-forward:
	kubectl -n tutoring port-forward svc/web 8080:80
