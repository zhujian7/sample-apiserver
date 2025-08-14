#!/bin/bash

# MyTest API Server Deployment Script

set -e

NAMESPACE="my-apiserver-system"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

print_usage() {
    echo "Usage: $0 [install|uninstall|status]"
    echo ""
    echo "Commands:"
    echo "  install   - Deploy the MyTest API Server"
    echo "  uninstall - Remove the MyTest API Server"
    echo "  status    - Check deployment status"
}

check_prerequisites() {
    echo "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        echo "ERROR: kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info &> /dev/null; then
        echo "ERROR: Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if cert-manager is installed
    if ! kubectl get crd certificates.cert-manager.io &> /dev/null; then
        echo "WARNING: cert-manager CRDs not found. Please install cert-manager:"
        echo "  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml"
        echo ""
        read -p "Continue without cert-manager? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    echo "Prerequisites check completed."
}

install() {
    echo "Installing MyTest API Server..."
    
    check_prerequisites
    
    # Apply base deployment
    echo "1. Applying base deployment..."
    kubectl apply -f "$SCRIPT_DIR/base/deploy.yaml"
    
    # Wait for deployment to be ready
    echo "2. Waiting for deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/mytest-apiserver -n $NAMESPACE
    
    # Apply certificates (if cert-manager is available)
    if kubectl get crd certificates.cert-manager.io &> /dev/null; then
        echo "3. Setting up TLS certificates..."
        kubectl apply -f "$SCRIPT_DIR/certificates/"
        
        # Wait for certificate to be ready
        echo "4. Waiting for certificate to be ready..."
        kubectl wait --for=condition=ready --timeout=120s certificate/mytest-apiserver-cert -n $NAMESPACE
    else
        echo "3. Skipping certificate setup (cert-manager not available)"
    fi
    
    # Register APIService
    echo "5. Registering APIService..."
    kubectl apply -f "$SCRIPT_DIR/base/apiservice.yaml"
    
    # Wait for APIService to be available
    echo "6. Waiting for APIService to be available..."
    for i in {1..30}; do
        if kubectl get apiservice v1alpha1.things.myorg.io -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' | grep -q "True"; then
            break
        fi
        echo "   Waiting for APIService... ($i/30)"
        sleep 2
    done
    
    echo ""
    echo "✅ MyTest API Server installation completed!"
    echo ""
    status
}

uninstall() {
    echo "Uninstalling MyTest API Server..."
    
    # Remove custom resources first
    echo "1. Removing custom resources..."
    kubectl delete widgets --all --all-namespaces --ignore-not-found=true
    kubectl delete gadgets --all --all-namespaces --ignore-not-found=true
    
    # Remove APIService
    echo "2. Removing APIService..."
    kubectl delete -f "$SCRIPT_DIR/base/apiservice.yaml" --ignore-not-found=true
    
    # Remove certificates
    echo "3. Removing certificates..."
    kubectl delete -f "$SCRIPT_DIR/certificates/" --ignore-not-found=true
    
    # Remove base deployment
    echo "4. Removing base deployment..."
    kubectl delete -f "$SCRIPT_DIR/base/deploy.yaml" --ignore-not-found=true
    
    echo ""
    echo "✅ MyTest API Server uninstalled successfully!"
}

status() {
    echo "MyTest API Server Status:"
    echo "========================="
    
    # Check namespace
    if kubectl get namespace $NAMESPACE &> /dev/null; then
        echo "✅ Namespace: $NAMESPACE exists"
    else
        echo "❌ Namespace: $NAMESPACE not found"
        return 1
    fi
    
    # Check deployment
    if kubectl get deployment mytest-apiserver -n $NAMESPACE &> /dev/null; then
        READY=$(kubectl get deployment mytest-apiserver -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
        DESIRED=$(kubectl get deployment mytest-apiserver -n $NAMESPACE -o jsonpath='{.spec.replicas}')
        if [ "$READY" = "$DESIRED" ]; then
            echo "✅ Deployment: mytest-apiserver ($READY/$DESIRED ready)"
        else
            echo "⚠️  Deployment: mytest-apiserver ($READY/$DESIRED ready)"
        fi
    else
        echo "❌ Deployment: mytest-apiserver not found"
        return 1
    fi
    
    # Check APIService
    if kubectl get apiservice v1alpha1.things.myorg.io &> /dev/null; then
        AVAILABLE=$(kubectl get apiservice v1alpha1.things.myorg.io -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')
        if [ "$AVAILABLE" = "True" ]; then
            echo "✅ APIService: v1alpha1.things.myorg.io available"
        else
            echo "⚠️  APIService: v1alpha1.things.myorg.io not available"
        fi
    else
        echo "❌ APIService: v1alpha1.things.myorg.io not found"
        return 1
    fi
    
    # Check custom resources
    if kubectl api-resources | grep -q "things.myorg.io"; then
        echo "✅ Custom resources registered:"
        kubectl api-resources | grep "things.myorg.io" | awk '{print "   - " $1}'
    else
        echo "❌ Custom resources not registered"
    fi
    
    # Show pods
    echo ""
    echo "Pods in $NAMESPACE namespace:"
    kubectl get pods -n $NAMESPACE
    
    echo ""
    echo "Recent logs (last 10 lines):"
    kubectl logs -n $NAMESPACE -l app=mytest-apiserver --tail=10
}

# Main script logic
case "$1" in
    install)
        install
        ;;
    uninstall)
        uninstall
        ;;
    status)
        status
        ;;
    *)
        print_usage
        exit 1
        ;;
esac
