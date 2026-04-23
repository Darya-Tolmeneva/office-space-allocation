#!/usr/bin/env bash
#
# Deploy Prometheus + Grafana monitoring stack to k3s
#
# Environment variables:
#   GRAFANA_ADMIN_PASSWORD — Grafana admin password (auto-generated if not set)
#
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
NAMESPACE="monitoring"

echo "=== Deploying monitoring stack ==="
echo "  Namespace: ${NAMESPACE}"
echo ""

# -------------------------------------------------------
# Step 1: Create namespace
# -------------------------------------------------------
echo "=== [1/3] Creating namespace ==="
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# -------------------------------------------------------
# Step 2: Create Grafana secrets
# -------------------------------------------------------
echo ""
echo "=== [2/3] Creating Grafana secrets ==="

SECRET_EXISTS=false
if kubectl get secret grafana-secrets -n "${NAMESPACE}" &>/dev/null; then
  SECRET_EXISTS=true
fi

if [[ -n "${GRAFANA_ADMIN_PASSWORD:-}" ]] || [[ "$SECRET_EXISTS" == "false" ]]; then
  if [[ -n "${GRAFANA_ADMIN_PASSWORD:-}" ]]; then
    ADMIN_PASS="${GRAFANA_ADMIN_PASSWORD}"
  else
    ADMIN_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
    echo "  Generated Grafana admin password: ${ADMIN_PASS}"
    echo "  (save this — it won't be shown again)"
  fi

  kubectl create secret generic grafana-secrets \
    --namespace="${NAMESPACE}" \
    --from-literal="admin-password=${ADMIN_PASS}" \
    --dry-run=client -o yaml | kubectl apply -f -

  echo "  Secret 'grafana-secrets' applied"
else
  echo "  Secret 'grafana-secrets' already exists — keeping existing password"
fi

# -------------------------------------------------------
# Step 3: Apply monitoring manifests
# -------------------------------------------------------
echo ""
echo "=== [3/3] Applying monitoring manifests ==="
kubectl apply -k "${PROJECT_ROOT}/k8s/base/monitoring"

echo ""
echo "Waiting for Prometheus rollout..."
kubectl rollout status deployment/prometheus -n "${NAMESPACE}" --timeout=120s

echo ""
echo "Waiting for Grafana rollout..."
kubectl rollout status deployment/grafana -n "${NAMESPACE}" --timeout=120s

echo ""
echo "============================================"
echo "  Monitoring stack deployed!"
echo "============================================"
echo ""
echo "Pods:"
kubectl get pods -n "${NAMESPACE}"
echo ""
echo "Services:"
kubectl get svc -n "${NAMESPACE}"
echo ""
echo "Access:"
echo "  Grafana:    http://<VM_IP>/grafana"
echo "  Login:      admin / <password from above or existing secret>"
echo ""
echo "  Prometheus: Internal only (ClusterIP on port 9090)"
echo "  To access:  kubectl port-forward -n monitoring svc/prometheus 9090:9090"
echo ""
