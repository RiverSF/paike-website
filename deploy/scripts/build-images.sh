#!/usr/bin/env bash
# 构建 k8s 部署所需的两个镜像
# 用法：
#   ./deploy/scripts/build-images.sh                 # 本地构建 tutoring-server/tutoring-web:latest
#   REGISTRY=registry.example.com/tutoring TAG=v1.0.0 ./deploy/scripts/build-images.sh
#   LOAD_KIND=1 ./deploy/scripts/build-images.sh     # 构建后载入 kind 集群
#   LOAD_MINIKUBE=1 ./deploy/scripts/build-images.sh # 构建后载入 minikube 集群
set -euo pipefail

cd "$(dirname "$0")/../.."

REGISTRY="${REGISTRY:-}"
TAG="${TAG:-latest}"
SERVER_IMAGE="${REGISTRY:+$REGISTRY/}tutoring-server:${TAG}"
WEB_IMAGE="${REGISTRY:+$REGISTRY/}tutoring-web:${TAG}"

echo ">>> build ${SERVER_IMAGE}"
docker build -t "${SERVER_IMAGE}" -f deploy/docker/Dockerfile.server .

echo ">>> build ${WEB_IMAGE}"
docker build -t "${WEB_IMAGE}" -f deploy/docker/Dockerfile.web .

if [ "${LOAD_KIND:-0}" = "1" ]; then
  echo ">>> load into kind"
  kind load docker-image "${SERVER_IMAGE}" "${WEB_IMAGE}"
fi

if [ "${LOAD_MINIKUBE:-0}" = "1" ]; then
  echo ">>> load into minikube"
  minikube image load "${SERVER_IMAGE}"
  minikube image load "${WEB_IMAGE}"
fi

if [ -n "${REGISTRY}" ] && [ "${PUSH:-0}" = "1" ]; then
  echo ">>> push"
  docker push "${SERVER_IMAGE}"
  docker push "${WEB_IMAGE}"
fi

echo ">>> done: ${SERVER_IMAGE} ${WEB_IMAGE}"
