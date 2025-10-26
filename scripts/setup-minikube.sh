#!/bin/bash

set -e

echo "=========================================="
echo "Minikube Setup for Observability Stack"
echo "=========================================="

# Check if minikube is installed
if ! command -v minikube &> /dev/null; then
    echo "Error: minikube is not installed"
    echo "Please install minikube: https://minikube.sigs.k8s.io/docs/start/"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi

# Check if docker is running
if ! docker info &> /dev/null; then
    echo "Error: Docker is not running"
    echo "Please start Docker"
    exit 1
fi

echo ""
echo "Starting Minikube..."
minikube start \
  --cpus=2 \
  --memory=4096 \
  --disk-size=20g \
  --driver=docker \
  --kubernetes-version=v1.28.0 \
  --force

echo ""
echo "Enabling Minikube addons..."
minikube addons enable metrics-server
minikube addons enable ingress

echo ""
echo "Verifying cluster status..."
kubectl cluster-info
kubectl get nodes

echo ""
echo "Setting up Docker environment for Minikube..."
eval $(minikube docker-env)

echo ""
echo "Building Docker images..."
echo "Building backend image..."
docker build -t backend:latest ./apps/backend

echo "Building frontend image..."
docker build -t frontend:latest ./apps/frontend

echo ""
echo "Verifying images..."
docker images | grep -E 'backend|frontend'

echo ""
echo "=========================================="
echo "Minikube setup completed successfully!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Deploy all components: ./scripts/deploy-all.sh"
echo "2. Access Grafana: kubectl port-forward -n observability svc/grafana 3000:3000"
echo "3. Access Jaeger: kubectl port-forward -n observability svc/jaeger-query 16686:16686"
echo "4. Access Frontend: kubectl port-forward svc/frontend 8080:8080"
echo ""
echo "To use Minikube's Docker daemon in your current shell:"
echo "  eval \$(minikube docker-env)"
echo ""
