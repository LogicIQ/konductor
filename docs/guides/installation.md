# Installation Guide

This guide covers installing Konductor in your Kubernetes cluster using various methods.

## Prerequisites

- Kubernetes cluster (v1.19+)
- `kubectl` configured to access your cluster
- Helm 3.x (for Helm installation)

## Helm Installation (Recommended)

### Add Repository

```bash
# Add the LogicIQ Helm repository
helm repo add logiciq https://logiciq.github.io/helm-charts
helm repo update
```

### Install Konductor

```bash
# Install with default values
helm install konductor logiciq/konductor

# Install in specific namespace
helm install konductor logiciq/konductor --namespace konductor-system --create-namespace

# Install with custom values
helm install konductor logiciq/konductor -f values.yaml
```

### Configuration Options

Create a `values.yaml` file to customize the installation:

```yaml
# values.yaml
replicaCount: 2

image:
  repository: logiciq/konductor
  tag: "v0.1.0"
  pullPolicy: IfNotPresent

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector: {}
tolerations: []
affinity: {}

# RBAC configuration
rbac:
  create: true

# Service account
serviceAccount:
  create: true
  name: ""

# Metrics
metrics:
  enabled: true
  port: 8080

# Webhook configuration
webhook:
  enabled: false
  port: 9443
```

## Kubectl Installation

### Install CRDs

```bash
# Install Custom Resource Definitions
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/bases/konductor.io_semaphores.yaml
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/bases/konductor.io_barriers.yaml
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/bases/konductor.io_leases.yaml
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/bases/konductor.io_gates.yaml
```

### Install Operator

```bash
# Create namespace
kubectl create namespace konductor-system

# Install RBAC
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/rbac/

# Install operator
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/manager/
```

## Kustomize Installation

Create a `kustomization.yaml` file:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: konductor-system

resources:
- https://github.com/LogicIQ/konductor/config/default

images:
- name: controller
  newName: logiciq/konductor
  newTag: v0.1.0

patchesStrategicMerge:
- manager_patch.yaml
```

Apply with:

```bash
kubectl apply -k .
```

## CLI Installation

### Binary Download

```bash
# Linux
curl -LO https://github.com/LogicIQ/konductor/releases/latest/download/koncli-linux-amd64
chmod +x koncli-linux-amd64
sudo mv koncli-linux-amd64 /usr/local/bin/koncli

# macOS
curl -LO https://github.com/LogicIQ/konductor/releases/latest/download/koncli-darwin-amd64
chmod +x koncli-darwin-amd64
sudo mv koncli-darwin-amd64 /usr/local/bin/koncli

# Windows
curl -LO https://github.com/LogicIQ/konductor/releases/latest/download/koncli-windows-amd64.exe
```

### Go Install

```bash
go install github.com/LogicIQ/konductor/cli@latest
```

### Container Image

```bash
# Pull CLI container
docker pull logiciq/koncli:latest

# Use in Kubernetes
kubectl run koncli --image=logiciq/koncli:latest --rm -it -- /bin/sh
```

## Verification

### Check Installation

```bash
# Check operator pods
kubectl get pods -n konductor-system

# Check CRDs
kubectl get crd | grep konductor

# Check CLI
koncli version
```

### Test Basic Functionality

```bash
# Create a test semaphore
kubectl apply -f - <<EOF
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: test-semaphore
spec:
  permits: 3
EOF

# Check status
kubectl get semaphore test-semaphore

# Test CLI
koncli semaphore status test-semaphore

# Cleanup
kubectl delete semaphore test-semaphore
```

## RBAC Configuration

Konductor requires specific permissions to function properly:

```yaml
# konductor-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: konductor-manager
rules:
- apiGroups: ["konductor.io"]
  resources: ["semaphores", "barriers", "leases", "gates"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: konductor-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: konductor-manager
subjects:
- kind: ServiceAccount
  name: konductor-controller-manager
  namespace: konductor-system
```

## User RBAC

For users and applications using Konductor:

```yaml
# konductor-user-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: konductor-user
rules:
- apiGroups: ["konductor.io"]
  resources: ["semaphores", "barriers", "leases", "gates"]
  verbs: ["get", "list", "watch", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: konductor-users
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: konductor-user
subjects:
- kind: User
  name: system:serviceaccounts  # All service accounts
  apiGroup: rbac.authorization.k8s.io
```

## Upgrading

### Helm Upgrade

```bash
# Update repository
helm repo update

# Upgrade installation
helm upgrade konductor logiciq/konductor

# Upgrade with new values
helm upgrade konductor logiciq/konductor -f new-values.yaml
```

### Manual Upgrade

```bash
# Update CRDs first
kubectl apply -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/

# Update operator
kubectl set image deployment/konductor-controller-manager \
  manager=logiciq/konductor:v0.2.0 \
  -n konductor-system
```

## Uninstallation

### Helm Uninstall

```bash
# Uninstall release
helm uninstall konductor

# Remove CRDs (optional)
kubectl delete crd semaphores.konductor.io
kubectl delete crd barriers.konductor.io
kubectl delete crd leases.konductor.io
kubectl delete crd gates.konductor.io
```

### Manual Uninstall

```bash
# Delete operator
kubectl delete -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/manager/

# Delete RBAC
kubectl delete -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/rbac/

# Delete CRDs
kubectl delete -f https://raw.githubusercontent.com/LogicIQ/konductor/main/config/crd/

# Delete namespace
kubectl delete namespace konductor-system
```

## Troubleshooting

### Common Issues

**Operator not starting:**
```bash
# Check logs
kubectl logs -n konductor-system deployment/konductor-controller-manager

# Check RBAC
kubectl auth can-i create semaphores --as=system:serviceaccount:konductor-system:konductor-controller-manager
```

**CLI connection issues:**
```bash
# Test connectivity
koncli version

# Check kubeconfig
kubectl config current-context
kubectl config view
```

**Permission denied:**
```bash
# Check user permissions
kubectl auth can-i get semaphores
kubectl auth can-i update semaphores

# Apply user RBAC
kubectl apply -f konductor-user-rbac.yaml
```

## Next Steps

- **[Quick Start](./quick-start.md)** - Try basic examples
- **[Core Concepts](../introduction/concepts.md)** - Understand the primitives
- **[CLI Usage](./cli-usage.md)** - Learn the command-line tool
- **[Examples](../examples/overview.md)** - See real-world usage patterns