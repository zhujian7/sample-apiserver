# Widget API Server Deployment

This directory contains all deployment manifests for the Widget API Server.

## Directory Structure

```
deploy/
├── README.md                    # This file
├── base/                       # Core deployment files
│   ├── deploy.yaml            # RBAC, Namespace, ServiceAccount, Deployment, Service
│   └── apiservice.yaml        # APIService registration
└── certificates/              # TLS certificate management
    ├── ca.yaml               # Certificate Authority
    ├── issuer.yaml           # Certificate Issuer
    └── cert.yaml             # TLS Certificate for API server
```

## Quick Deployment

### Prerequisites

1. **Kubernetes cluster** (v1.20+)
2. **cert-manager** installed (for TLS certificates)
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

### Deployment Steps

1. **Deploy base components** (RBAC, Deployment, Service):
   ```bash
   kubectl apply -f deploy/base/deploy.yaml
   ```

2. **Set up certificates** (if using cert-manager):
   ```bash
   kubectl apply -f deploy/certificates/
   ```

3. **Register APIService**:
   ```bash
   kubectl apply -f deploy/base/apiservice.yaml
   ```

### Verification

Check deployment status:
```bash
# Check pods
kubectl get pods -n my-apiserver-system

# Check APIService
kubectl get apiservice widget-apiserver

# Check custom resources are available
kubectl api-resources | grep things.myorg.io
```

## File Descriptions

### Base Components

- **`deploy.yaml`**: Contains all core Kubernetes resources:
  - Namespace: `my-apiserver-system`
  - ServiceAccount: `widget-apiserver`
  - ClusterRole & ClusterRoleBinding: RBAC permissions
  - Deployment: Widget API server pod
  - Service: Internal service exposure

- **`apiservice.yaml`**: Registers the custom API with Kubernetes API aggregation layer

### Certificate Management

- **`ca.yaml`**: Self-signed Certificate Authority for TLS
- **`issuer.yaml`**: cert-manager Issuer using the CA
- **`cert.yaml`**: TLS certificate for the API server

## Configuration

### Environment Variables

The deployment supports these configuration options via environment variables:

- **TLS Settings**: Configured via volume mounts in `deploy.yaml`
- **Port**: Default 8443 (secure port)
- **Log Level**: Controlled by klog flags

### Resource Limits

Default resource limits in `deploy.yaml`:
- No limits set (adjust based on your requirements)

## Troubleshooting

### Common Issues

1. **APIService not Available**:
   ```bash
   kubectl describe apiservice v1alpha1.things.myorg.io
   kubectl logs -n my-apiserver-system -l app=widget-apiserver
   ```

2. **Certificate Issues**:
   ```bash
   kubectl describe certificate -n my-apiserver-system
   kubectl describe secret widget-apiserver-tls -n my-apiserver-system
   ```

3. **RBAC Issues**:
   ```bash
   kubectl auth can-i create widgets --as=system:serviceaccount:my-apiserver-system:widget-apiserver
   ```

### Logs

View API server logs:
```bash
kubectl logs -f -n my-apiserver-system -l app=widget-apiserver
```

## Testing

After deployment, test the API:

```bash
# Create a widget
kubectl apply -f - <<EOF
apiVersion: things.myorg.io/v1alpha1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  name: "Test Widget"
  description: "A test widget"
  size: 42
EOF

# List widgets
kubectl get widgets

# Get widget details
kubectl get widget test-widget -o yaml

# Create a gadget
kubectl apply -f - <<EOF
apiVersion: things.myorg.io/v1alpha1
kind: Gadget
metadata:
  name: test-gadget
  namespace: default
spec:
  type: "sensor"
  version: "v1.0"
  enabled: true
  priority: 10
EOF

# List gadgets
kubectl get gadgets
```

## Cleanup

To remove the deployment:

```bash
# Remove custom resources first
kubectl delete widgets --all --all-namespaces
kubectl delete gadgets --all --all-namespaces

# Remove APIService
kubectl delete -f deploy/base/apiservice.yaml

# Remove certificates
kubectl delete -f deploy/certificates/

# Remove base deployment
kubectl delete -f deploy/base/deploy.yaml
```
