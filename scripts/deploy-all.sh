#!/bin/bash

set -e

echo "=========================================="
echo "Deploying Observability Stack"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to wait for pods to be ready
wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=${3:-300}
    
    echo -e "${YELLOW}Waiting for pods with label ${label} in namespace ${namespace}...${NC}"
    kubectl wait --for=condition=ready pod \
        -l "${label}" \
        -n "${namespace}" \
        --timeout="${timeout}s" || true
}

# Function to check deployment status
check_deployment() {
    local namespace=$1
    local deployment=$2
    
    echo -e "${YELLOW}Checking deployment ${deployment} in namespace ${namespace}...${NC}"
    kubectl get deployment -n "${namespace}" "${deployment}"
}

echo ""
echo "Step 1: Creating Namespace"
echo "----------------------------"
kubectl apply -f kubernetes/namespaces/observability.yaml
echo -e "${GREEN}✓ Namespace created${NC}"

echo ""
echo "Step 2: Deploying OpenTelemetry Collector"
echo "----------------------------"
kubectl apply -f kubernetes/otel-collector/configmap.yaml
kubectl apply -f kubernetes/otel-collector/deployment.yaml
kubectl apply -f kubernetes/otel-collector/service.yaml
wait_for_pods "observability" "app=otel-collector"
check_deployment "observability" "otel-collector"
echo -e "${GREEN}✓ OpenTelemetry Collector deployed${NC}"

echo ""
echo "Step 3: Deploying Prometheus"
echo "----------------------------"
kubectl apply -f kubernetes/prometheus/rbac.yaml
kubectl apply -f kubernetes/prometheus/configmap.yaml
kubectl apply -f kubernetes/prometheus/deployment.yaml
kubectl apply -f kubernetes/prometheus/service.yaml
wait_for_pods "observability" "app=prometheus"
check_deployment "observability" "prometheus"
echo -e "${GREEN}✓ Prometheus deployed${NC}"

echo ""
echo "Step 4: Deploying Jaeger"
echo "----------------------------"
kubectl apply -f kubernetes/jaeger/deployment.yaml
kubectl apply -f kubernetes/jaeger/service.yaml
wait_for_pods "observability" "app=jaeger"
check_deployment "observability" "jaeger"
echo -e "${GREEN}✓ Jaeger deployed${NC}"

echo ""
echo "Step 5: Deploying Grafana"
echo "----------------------------"
kubectl apply -f kubernetes/grafana/configmap-datasources.yaml
kubectl apply -f kubernetes/grafana/configmap-dashboards.yaml
kubectl apply -f kubernetes/grafana/deployment.yaml
kubectl apply -f kubernetes/grafana/service.yaml
wait_for_pods "observability" "app=grafana"
check_deployment "observability" "grafana"
echo -e "${GREEN}✓ Grafana deployed${NC}"

echo ""
echo "Step 6: Deploying Sample Applications"
echo "----------------------------"
echo "Deploying Backend..."
kubectl apply -f kubernetes/sample-apps/backend/deployment.yaml
kubectl apply -f kubernetes/sample-apps/backend/service.yaml

echo "Deploying Frontend..."
kubectl apply -f kubernetes/sample-apps/frontend/deployment.yaml
kubectl apply -f kubernetes/sample-apps/frontend/service.yaml

wait_for_pods "default" "app=backend"
wait_for_pods "default" "app=frontend"
echo -e "${GREEN}✓ Sample applications deployed${NC}"

echo ""
echo "=========================================="
echo "Deployment Summary"
echo "=========================================="

echo ""
echo "Observability Stack Pods:"
kubectl get pods -n observability

echo ""
echo "Application Pods:"
kubectl get pods -l tier=application

echo ""
echo "Services:"
kubectl get svc -n observability
kubectl get svc -l tier=application

echo ""
echo "=========================================="
echo -e "${GREEN}Deployment completed successfully!${NC}"
echo "=========================================="

echo ""
echo "Access the services:"
echo ""
echo "Grafana (admin/admin):"
echo "  kubectl port-forward -n observability svc/grafana 3000:3000"
echo "  http://localhost:3000"
echo ""
echo "Prometheus:"
echo "  kubectl port-forward -n observability svc/prometheus 9090:9090"
echo "  http://localhost:9090"
echo ""
echo "Jaeger:"
echo "  kubectl port-forward -n observability svc/jaeger-query 16686:16686"
echo "  http://localhost:16686"
echo ""
echo "Frontend Application:"
echo "  kubectl port-forward svc/frontend 8080:8080"
echo "  http://localhost:8080"
echo ""
echo "Generate some traffic:"
echo "  curl http://localhost:8080/api/users"
echo "  curl http://localhost:8080/api/users/1"
echo ""
