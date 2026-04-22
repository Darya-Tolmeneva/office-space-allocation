#!/usr/bin/env bash
#
# deploy.sh — Build Docker images and deploy to k3s on the current VM.
#
# Environment variables:
#   POSTGRES_PASSWORD  — database password (auto-generated if not set)
#   JWT_SIGNING_KEY    — JWT secret key (auto-generated if not set)
#   POSTGRES_DB        — database name (default: office_space_allocation)
#   POSTGRES_USER      — database user (default: postgres)
#   VITE_API_BASE_URL  — frontend API base URL (default: /v1)
#
set -euo pipefail

ENV="${1:-}"

if [[ -z "$ENV" ]] || [[ "$ENV" != "test" && "$ENV" != "prod" ]]; then
  echo "Usage: $0 <test|prod>"
  exit 1
fi

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${ENV}"
IMAGE_BACKEND="office-space-allocation/backend:${TAG}"
IMAGE_FRONTEND="office-space-allocation/frontend:${TAG}"
NAMESPACE="flowdesk-${ENV}"

echo "=== Deploying environment: ${ENV} ==="
echo "  Backend image:  ${IMAGE_BACKEND}"
echo "  Frontend image: ${IMAGE_FRONTEND}"
echo "  Namespace:      ${NAMESPACE}"
echo ""

# -------------------------------------------------------
# Step 1: Build Docker images
# -------------------------------------------------------
echo "=== [1/6] Building backend Docker image ==="
docker build -t "${IMAGE_BACKEND}" "${PROJECT_ROOT}/apps/backend"

echo ""
echo "=== [2/6] Building frontend Docker image ==="
FRONTEND_API_URL="${VITE_API_BASE_URL:-/v1}"
docker build \
  --build-arg "VITE_API_BASE_URL=${FRONTEND_API_URL}" \
  -t "${IMAGE_FRONTEND}" \
  "${PROJECT_ROOT}/apps/frontend"

# -------------------------------------------------------
# Step 2: Import images into k3s
# -------------------------------------------------------
echo ""
echo "=== [3/6] Importing images into k3s ==="
docker save "${IMAGE_BACKEND}" | k3s ctr images import -
echo "Backend image imported"
docker save "${IMAGE_FRONTEND}" | k3s ctr images import -
echo "Frontend image imported"

# -------------------------------------------------------
# Step 3: Create namespace
# -------------------------------------------------------
echo ""
echo "=== [4/6] Creating namespace ==="
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# -------------------------------------------------------
# Step 4: Create migrations ConfigMap + secrets
# -------------------------------------------------------
echo ""
echo "=== [5/6] Creating ConfigMap and secrets ==="

kubectl create configmap migrations \
  --namespace="${NAMESPACE}" \
  --from-file="${PROJECT_ROOT}/apps/backend/migrations/" \
  --dry-run=client -o yaml | kubectl apply -f -

SECRETS_FROM_ENV=false
if [[ -n "${POSTGRES_PASSWORD:-}" ]] || [[ -n "${JWT_SIGNING_KEY:-}" ]]; then
  SECRETS_FROM_ENV=true
fi

SECRET_EXISTS=false
if kubectl get secret app-secrets -n "${NAMESPACE}" &>/dev/null; then
  SECRET_EXISTS=true
fi

# Recreate secret if env vars are provided (GitHub Secrets), or create if missing
if [[ "$SECRETS_FROM_ENV" == "true" ]] || [[ "$SECRET_EXISTS" == "false" ]]; then
  if [[ "$SECRET_EXISTS" == "true" ]] && [[ "$SECRETS_FROM_ENV" == "true" ]]; then
    echo "Updating 'app-secrets' from environment variables..."
  else
    echo "Creating 'app-secrets'..."
  fi

  # Use env vars if provided, otherwise generate defaults
  DB_NAME="${POSTGRES_DB:-office_space_allocation}"
  DB_USER="${POSTGRES_USER:-postgres}"

  if [[ -n "${POSTGRES_PASSWORD:-}" ]]; then
    DB_PASS="${POSTGRES_PASSWORD}"
  else
    DB_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 32)"
    echo "  Generated POSTGRES_PASSWORD (saved in k8s secret)"
  fi

  if [[ -n "${JWT_SIGNING_KEY:-}" ]]; then
    JWT_KEY="${JWT_SIGNING_KEY}"
  else
    JWT_KEY="$(openssl rand -base64 48 | tr -d '/+=' | head -c 64)"
    echo "  Generated JWT_SIGNING_KEY (saved in k8s secret)"
  fi

  DB_DSN="postgres://${DB_USER}:${DB_PASS}@postgres:5432/${DB_NAME}?sslmode=disable"

  kubectl create secret generic app-secrets \
    --namespace="${NAMESPACE}" \
    --from-literal="POSTGRES_DB=${DB_NAME}" \
    --from-literal="POSTGRES_USER=${DB_USER}" \
    --from-literal="POSTGRES_PASSWORD=${DB_PASS}" \
    --from-literal="POSTGRES_DSN=${DB_DSN}" \
    --from-literal="JWT_SIGNING_KEY=${JWT_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -

  echo "  Secret 'app-secrets' applied in namespace '${NAMESPACE}'"
else
  echo "Secret 'app-secrets' already exists, no env vars provided — keeping existing values"
fi

# -------------------------------------------------------
# Step 5: Apply Kubernetes manifests
# -------------------------------------------------------
echo ""
echo "=== [6/6] Applying Kubernetes manifests ==="

kubectl apply -k "${PROJECT_ROOT}/k8s/overlays/${ENV}"

echo ""
echo "Waiting for postgres to be ready..."
kubectl rollout status statefulset/postgres -n "${NAMESPACE}" --timeout=180s

echo ""
echo "Waiting for backend rollout..."
kubectl rollout status deployment/backend -n "${NAMESPACE}" --timeout=180s

echo ""
echo "Waiting for frontend rollout..."
kubectl rollout status deployment/frontend -n "${NAMESPACE}" --timeout=120s

echo ""
echo "============================================"
echo "  Deployment to '${ENV}' complete!"
echo "============================================"
echo ""
echo "Pods:"
kubectl get pods -n "${NAMESPACE}"
echo ""
echo "Services:"
kubectl get svc -n "${NAMESPACE}"
echo ""
echo "Ingress:"
kubectl get ingress -n "${NAMESPACE}"
echo ""
