# Widget API Server - Kubernetes Aggregate API Example

This is a complete example of a Kubernetes Aggregate API server that implements a custom `Widget` resource with full CRUD operations using in-memory storage.

## Overview

The Widget API server demonstrates:
- **Custom Resource Definition**: `Widget` with a simple `size` specification
- **In-Memory Storage**: Thread-safe storage implementing `rest.StandardStorage`
- **CRUD Operations**: Create, Read, Update, Delete, and List operations
- **Kubernetes Integration**: Uses `apiserver-runtime` for easy setup

## Resource Definition

```go
// Widget represents a custom resource
type Widget struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              WidgetSpec `json:"spec,omitempty"`
}

type WidgetSpec struct {
    Size int `json:"size"`
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
   go build -o widget-apiserver main.go
   ```

2. **Run locally** (for development):
   ```bash
   ./widget-apiserver --secure-port=8443 --etcd-servers=http://localhost:2379
   ```

3. **Deploy to Kubernetes**:
   ```bash
   # Build Docker image
   docker build -t widget-apiserver:latest .
   
   # Deploy to cluster
   kubectl apply -f deploy.yaml
   ```

## CRUD Examples

### Create a Widget
```bash
curl -X POST https://localhost:8443/apis/things.myorg.io/v1alpha1/namespaces/default/widgets \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "things.myorg.io/v1alpha1",
    "kind": "Widget",
    "metadata": {
      "name": "my-widget"
    },
    "spec": {
      "size": 42
    }
  }' \
  --insecure
```

### Get a Widget
```bash
curl https://localhost:8443/apis/things.myorg.io/v1alpha1/namespaces/default/widgets/my-widget --insecure
```

### Update a Widget
```bash
curl -X PUT https://localhost:8443/apis/things.myorg.io/v1alpha1/namespaces/default/widgets/my-widget \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "things.myorg.io/v1alpha1",
    "kind": "Widget",
    "metadata": {
      "name": "my-widget"
    },
    "spec": {
      "size": 100
    }
  }' \
  --insecure
```

### List Widgets
```bash
curl https://localhost:8443/apis/things.myorg.io/v1alpha1/namespaces/default/widgets --insecure
```

### Delete a Widget
```bash
curl -X DELETE https://localhost:8443/apis/things.myorg.io/v1alpha1/namespaces/default/widgets/my-widget --insecure
```

## Using kubectl (after deployment)

Once deployed with the APIService, you can use kubectl:

```bash
# List widgets
kubectl get widgets

# Create a widget
kubectl apply -f - <<EOF
apiVersion: things.myorg.io/v1alpha1
kind: Widget
metadata:
  name: kubectl-widget
  namespace: default
spec:
  size: 75
EOF

# Get widget details
kubectl get widget kubectl-widget -o yaml

# Delete widget
kubectl delete widget kubectl-widget
```

## Testing

Run the unit tests:
```bash
go test -v
```

## Key Components

### 1. Widget Resource (`main.go:19-68`)
- Implements `runtime.Object` interface
- Defines the resource schema and metadata
- Provides validation hooks

### 2. In-Memory Storage (`main.go:73-149`)
- Thread-safe storage with mutex protection
- Implements `rest.StandardStorage` interface
- Provides all CRUD operations

### 3. Storage Provider (`main.go:154-156`)
- Factory function for creating storage instances
- Used by the apiserver-runtime framework

### 4. Main Server (`main.go:161-168`)
- Sets up the API server using `apiserver-runtime`
- Registers the Widget resource and storage
- Enables local debugging

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

- `k8s.io/apimachinery`: Kubernetes API machinery
- `k8s.io/apiserver`: Kubernetes API server library
- `sigs.k8s.io/apiserver-runtime`: Simplified API server framework