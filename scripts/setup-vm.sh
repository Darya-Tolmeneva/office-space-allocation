#!/usr/bin/env bash
#
# setup-vm.sh — Bootstrap a single Yandex Cloud VM for running
# FlowDesk on k3s
#
set -euo pipefail

echo "=== [1/6] Updating system packages ==="
apt-get update -y
apt-get upgrade -y
apt-get install -y curl git

echo "=== [2/6] Installing Docker ==="
if ! command -v docker &>/dev/null; then
  curl -fsSL https://get.docker.com | sh
  # Add current sudo user to docker group if running via sudo
  if [ -n "${SUDO_USER:-}" ]; then
    usermod -aG docker "$SUDO_USER"
  fi
else
  echo "Docker is already installed, skipping"
fi

echo "=== [3/6] Installing k3s (lightweight Kubernetes) ==="
if ! command -v k3s &>/dev/null; then
  curl -sfL https://get.k3s.io | sh -s - \
    --write-kubeconfig-mode 644 \
    --disable traefik
  echo "Waiting for k3s to be ready..."
  sleep 10
  kubectl wait --for=condition=Ready node --all --timeout=120s
else
  echo "k3s is already installed, skipping"
fi

echo "=== [4/6] Installing NGINX Ingress Controller ==="
if ! kubectl get namespace ingress-nginx &>/dev/null 2>&1; then
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/baremetal/deploy.yaml
  echo "Waiting for ingress-nginx to be ready (this may take a minute)..."
  sleep 15
  kubectl wait --namespace ingress-nginx \
    --for=condition=Ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=300s
else
  echo "ingress-nginx is already installed, skipping"
fi

echo "=== [5/6] Patching ingress-nginx to use hostNetwork ==="
# This allows the ingress controller to bind to ports 80/443 on the VM
kubectl patch deployment ingress-nginx-controller \
  -n ingress-nginx \
  --type=json \
  -p='[
    {"op": "add", "path": "/spec/template/spec/hostNetwork", "value": true},
    {"op": "replace", "path": "/spec/template/spec/containers/0/ports", "value": [
      {"containerPort": 80, "hostPort": 80, "protocol": "TCP"},
      {"containerPort": 443, "hostPort": 443, "protocol": "TCP"}
    ]}
  ]' 2>/dev/null || echo "Patch already applied or not needed"

# Wait for the controller to restart
sleep 5
kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=120s

# Delete the admission webhook — it can't work with hostNetwork because
# the service endpoints become unreachable. This webhook only validates
# Ingress resources before creation; the controller works fine without it.
kubectl delete validatingwebhookconfiguration ingress-nginx-admission 2>/dev/null || true

echo "=== [6/6] Verifying local-path StorageClass ==="
# k3s ships with local-path-provisioner by default
if kubectl get storageclass local-path &>/dev/null 2>&1; then
  echo "local-path storageclass is available"
else
  echo "WARNING: local-path storageclass not found. PVCs may not work."
  echo "Install manually: kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.26/deploy/local-path-storage.yaml"
fi

echo ""
echo "============================================"
echo "  VM setup complete!"
echo "============================================"
echo ""
echo "k3s version:"
k3s --version
echo ""
echo "Docker version:"
docker --version
echo ""
echo "Nodes:"
kubectl get nodes
echo ""
echo "Next steps:"
echo "  1. Clone the repository to this VM"
echo "  2. Create secrets for the target environment"
echo "  3. Run scripts/deploy.sh <test|prod>"
echo "  4. See README.md for detailed instructions"
echo ""
