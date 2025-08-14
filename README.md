# MyTest API Server - Kubernetes Aggregate API Example

This is a complete example of a Kubernetes Aggregate API server that implements custom `Widget` and `Gadget` resources with full CRUD operations using in-memory storage.

## Overview

The MyTest API server demonstrates:
- **Custom Resource Definitions**: `Widget` and `Gadget` resources with their own specifications
- **In-Memory Storage**: Thread-safe storage with mutex protection
- **CRUD Operations**: Create, Read, Update, Delete, and List operations
- **Kubernetes Integration**: Direct integration with Kubernetes API server framework
- **Aggregate API**: Extends Kubernetes API with custom resources

## Resource Definitions

### Widget Resource

```go
// Widget represents a custom resource
type Widget struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              WidgetSpec   `json:"spec,omitempty"`
    Status            WidgetStatus `json:"status,omitempty"`
}

type WidgetSpec struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Size        int32  `json:"size"`
}

type WidgetStatus struct {
    Phase string `json:"phase,omitempty"`
}
```

### Gadget Resource

```go
// Gadget represents a custom resource
type Gadget struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              GadgetSpec   `json:"spec,omitempty"`
    Status            GadgetStatus `json:"status,omitempty"`
}

type GadgetSpec struct {
    Type     string `json:"type"`
    Version  string `json:"version"`
    Enabled  bool   `json:"enabled"`
    Priority int32  `json:"priority"`
}

type GadgetStatus struct {
    State string `json:"state,omitempty"`
}
```

## API Endpoints

Once deployed, the API server exposes these endpoints:

### Widget Endpoints

- **Create**: `POST /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets`
- **Get**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`
- **Update**: `PUT /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`
- **List**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets`
- **Delete**: `DELETE /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`

### Gadget Endpoints

- **Create**: `POST /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/gadgets`
- **Get**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/gadgets/{name}`
- **Update**: `PUT /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/gadgets/{name}`
- **List**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/gadgets`
- **Delete**: `DELETE /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/gadgets/{name}`

## Quick Start

### Option 1: Kind Cluster (Recommended for testing)

1. **Set up Kind cluster with cert-manager**:
   ```bash
   ./deploy/kind/setup.sh
   ```

2. **Deploy the API server**:
   ```bash
   ./deploy/deploy.sh install
   ```

3. **Verify deployment**:
   ```bash
   kubectl get apiservice v1alpha1.things.myorg.io
   kubectl get pods -n my-apiserver-system
   ```

### Option 2: Existing Kubernetes Cluster

1. **Prerequisites**: Ensure cert-manager is installed
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

2. **Deploy the API server**:
   ```bash
   ./deploy/deploy.sh install
   ```

## Building and Running

1. **Build the server**:
   ```bash
   go build -o mytest-apiserver .
   ```

2. **Build Docker image**:
   ```bash
   docker build -t quay.io/zhujian/mytest-apiserver:dev .
   ```

## CRUD Examples

### Widget Examples

### Create a Widget
```bash
kubectl apply -f - <<EOF
apiVersion: things.myorg.io/v1alpha1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  name: "My Test Widget"
  description: "A test widget for demonstration"
  size: 42
EOF
```

### Get a Widget
```bash
kubectl get widget test-widget -n default -o yaml
```

### Update a Widget
```bash
kubectl patch widget test-widget -n default --type='merge' -p='{"spec":{"size":100}}'
```

### List Widgets
```bash
kubectl get widgets -n default
```

### Delete a Widget
```bash
kubectl delete widget test-widget -n default
```

### Gadget Examples

### Create a Gadget
```bash
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
```

### Get a Gadget
```bash
kubectl get gadget test-gadget -n default -o yaml
```

### Update a Gadget
```bash
kubectl patch gadget test-gadget -n default --type='merge' -p='{"spec":{"priority":20}}'
```

### List Gadgets
```bash
kubectl get gadgets -n default
```

### Delete a Gadget
```bash
kubectl delete gadget test-gadget -n default
```

## Troubleshooting

### Common Issues

1. **APIService not Available**: Check pod status and logs
   ```bash
   kubectl get pods -n my-apiserver-system
   kubectl logs -f -n my-apiserver-system -l app=mytest-apiserver
   ```

2. **Permission Errors**: Ensure RBAC is properly configured
   ```bash
   kubectl get clusterrole mytest-apiserver-auth-reader
   kubectl get clusterrolebinding mytest-apiserver-auth-reader
   ```

3. **List Operation Fails**: Fixed in latest version with proper interface implementation

## Testing

### Unit Tests
```bash
go test -v
```

### Integration Testing
After deploying to Kind cluster, test the full workflow:
```bash
# Test Widget operations
kubectl apply -f - <<EOF
apiVersion: things.myorg.io/v1alpha1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  name: "Test Widget"
  description: "Integration test widget"
  size: 100
EOF

# Test Gadget operations  
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
  priority: 5
EOF

# Verify resources
kubectl get widgets,gadgets
```

## Key Components

### 1. Resource Definitions (`pkg/apis/*/`)
- Widget and Gadget resources with their own specifications
- Implements `runtime.Object` interface with DeepCopy methods
- Includes TypeMeta and ObjectMeta for Kubernetes integration

### 2. In-Memory Storage
- Thread-safe storage with mutex protection for both resources
- Implements Create, Read, Update, Delete, and List operations
- Provides automatic metadata management (UID, timestamps, etc.)

### 3. REST Interfaces
- Implements multiple REST interfaces (Creater, Lister, Getter, etc.)
- Bridges between HTTP API and storage layer
- Provides namespace scoping and singular name support

### 4. API Server Setup (`main.go`)
- Configures generic API server with custom options
- Registers Widget API group and resources
- Handles authentication delegation and TLS

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   kubectl/curl  │───▶│  MyTest API      │───▶│  In-Memory      │
│   HTTP Client   │    │  Server          │    │  Storage        │
└─────────────────┘    │  (REST handlers) │    │  (Widget &      │
                       │  Widget/Gadget   │    │   Gadget)       │
                       └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  Kubernetes      │
                       │  API Aggregation │
                       └──────────────────┘
```

## Production Considerations

For production use, consider:

1. **Persistent Storage**: Replace in-memory storage with etcd or database
2. **Authentication**: Add proper authentication and authorization
3. **Validation**: Implement comprehensive validation logic
4. **Monitoring**: Add metrics and health checks
5. **High Availability**: Deploy multiple replicas
6. **TLS**: Proper certificate management
7. **RBAC**: Define appropriate role-based access controls

## Dependencies

- `k8s.io/apimachinery`: Kubernetes API machinery and runtime types
- `k8s.io/apiserver`: Kubernetes API server framework and utilities
- Direct implementation without external frameworks for learning purposes

## Features Implemented

- ✅ Multiple custom resources (Widget and Gadget) with spec and status
- ✅ In-memory storage with thread safety for both resources
- ✅ Full CRUD operations (Create, Read, Update, Delete, List)
- ✅ Kubernetes API server integration
- ✅ Authentication delegation
- ✅ RBAC integration
- ✅ Namespace scoping
- ✅ API discovery and OpenAPI schema
- ✅ Docker containerization
- ✅ Kubernetes deployment manifests
- ✅ Automated deployment scripts
- ✅ Modular package structure
- ✅ Kind cluster setup for easy testing
- ✅ Automatic CA injection for TLS certificates

## Cleanup

### Remove API Server
```bash
./deploy/deploy.sh uninstall
```

### Delete Kind Cluster (if using Kind)
```bash
kind delete cluster --name kind
```

## Directory Structure

```
.
├── main.go                          # API server main entry point
├── go.mod                           # Go module definition
├── Dockerfile                       # Container build file
├── README.md                        # This file
├── pkg/                            # Go packages
│   ├── apis/                       # API resource definitions
│   │   ├── widgets/                # Widget resource implementation
│   │   └── gadgets/                # Gadget resource implementation
│   └── common/                     # Shared constants and utilities
└── deploy/                         # Deployment manifests
    ├── deploy.sh                   # Automated deployment script
    ├── README.md                   # Deployment documentation
    ├── base/                       # Core Kubernetes manifests
    │   ├── deploy.yaml            # RBAC, Deployment, Service
    │   └── apiservice.yaml        # API registration
    ├── certificates/               # TLS certificate setup
    │   ├── ca.yaml               # Certificate Authority
    │   ├── issuer.yaml           # cert-manager Issuer
    │   └── cert.yaml             # API server certificate
    └── kind/                       # Kind cluster setup
        ├── cluster-config.yaml    # Kind cluster configuration
        └── setup.sh              # Automated Kind setup
```
