#!/bin/bash

# Kind Cluster Setup Script for MyTest API Server

set -e

CLUSTER_NAME="kind"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

echo "🚀 Setting up Kind cluster for MyTest API Server..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ ERROR: kind is not installed. Please install kind first:"
    echo "   https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ ERROR: kubectl is not installed. Please install kubectl first:"
    echo "   https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi

# Create Kind cluster
echo "1. Creating Kind cluster '$CLUSTER_NAME'..."
if kind get clusters | grep -q "^$CLUSTER_NAME$"; then
    echo "   Cluster '$CLUSTER_NAME' already exists, skipping creation"
else
    kind create cluster --name $CLUSTER_NAME --config "$SCRIPT_DIR/cluster-config.yaml"
    echo "   ✅ Cluster '$CLUSTER_NAME' created successfully"
fi

# Wait for cluster to be ready
echo "2. Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=60s

# Install cert-manager
echo "3. Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager to be ready
echo "4. Waiting for cert-manager to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-cainjector -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager

echo ""
echo "✅ Kind cluster setup completed!"
echo ""
echo "Next steps:"
echo "  1. Deploy the API server: ./deploy/deploy.sh install"
echo "  2. Test the deployment: kubectl get apiservice v1alpha1.things.myorg.io"
echo ""
echo "To delete the cluster later: kind delete cluster --name $CLUSTER_NAME"