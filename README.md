# Widget API Server - Kubernetes Aggregate API Example

This is a complete example of a Kubernetes Aggregate API server that implements a custom `Widget` resource with full CRUD operations using in-memory storage.

## Overview

The Widget API server demonstrates:
- **Custom Resource Definition**: `Widget` with name, description, and size specification
- **In-Memory Storage**: Thread-safe storage with mutex protection
- **CRUD Operations**: Create, Read, Update, Delete, and List operations
- **Kubernetes Integration**: Direct integration with Kubernetes API server framework
- **Aggregate API**: Extends Kubernetes API with custom resources

## Resource Definition

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

## API Endpoints

Once deployed, the API server exposes these endpoints:

- **Create**: `POST /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets`
- **Get**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`
- **Update**: `PUT /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`
- **List**: `GET /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets`
- **Delete**: `DELETE /apis/things.myorg.io/v1alpha1/namespaces/{namespace}/widgets/{name}`

## Building and Running

1. **Build the server**:
   ```bash
   go build -o widget-apiserver .
   ```

2. **Build Docker image**:
   ```bash
   docker build -t quay.io/zhujian/widget-apiserver:dev .
   ```

3. **Deploy to Kubernetes**:
   ```bash
   # Apply RBAC and deployment
   kubectl apply -f deploy.yaml

   # Create APIService
   kubectl apply -f apiservice.yaml

   # Check deployment status
   kubectl get pods -n widget-system
   kubectl get apiservice widget-apiserver
   ```

## CRUD Examples

### Create a Widget
```bash
kubectl update -f - <<EOF
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

## Troubleshooting

### Common Issues

1. **APIService not Available**: Check pod status and logs
   ```bash
   kubectl get pods -n widget-system
   kubectl logs -f -n widget-system -l app=widget-apiserver
   ```

2. **Permission Errors**: Ensure RBAC is properly configured
   ```bash
   kubectl get clusterrole widget-apiserver-auth-reader
   kubectl get clusterrolebinding widget-apiserver-auth-reader
   ```

3. **List Operation Fails**: Fixed in latest version with proper interface implementation

## Testing

Run the unit tests:
```bash
go test -v
```

## Key Components

### 1. Widget Resource (`main.go:35-77`)
- Implements `runtime.Object` interface with DeepCopy methods
- Defines the resource schema with spec and status
- Includes TypeMeta and ObjectMeta for Kubernetes integration

### 2. In-Memory Storage (`main.go:85-175`)
- Thread-safe storage with mutex protection
- Implements Create, Read, Update, Delete, and List operations
- Provides automatic metadata management (UID, timestamps, etc.)

### 3. WidgetREST (`main.go:177-253`)
- Implements multiple REST interfaces (Creater, Lister, Getter, etc.)
- Bridges between HTTP API and storage layer
- Provides namespace scoping and singular name support

### 4. API Server Setup (`main.go:255-360`)
- Configures generic API server with custom options
- Registers Widget API group and resources
- Handles authentication delegation and TLS

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   kubectl/curl  │───▶│  Widget API      │───▶│  In-Memory      │
│   HTTP Client   │    │  Server          │    │  Storage        │
└─────────────────┘    │  (REST handlers) │    │  (Thread-safe)  │
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

- ✅ Custom resource with spec and status
- ✅ In-memory storage with thread safety
- ✅ Full CRUD operations (Create, Read, Update, Delete, List)
- ✅ Kubernetes API server integration
- ✅ Authentication delegation
- ✅ RBAC integration
- ✅ Namespace scoping
- ✅ API discovery and OpenAPI schema
- ✅ Docker containerization
- ✅ Kubernetes deployment manifests
