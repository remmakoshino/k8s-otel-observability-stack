# Helm templates placeholder
# 
# To generate templates from existing Kubernetes manifests:
# 1. Copy files from kubernetes/ directory
# 2. Replace hard-coded values with template variables
# 3. Use values.yaml for configuration
#
# Example template structure:
# - namespace.yaml
# - otel-collector/
# - prometheus/
# - grafana/
# - jaeger/
# - sample-apps/
#
# For now, use kubectl apply -f kubernetes/ for deployment
# or use the provided scripts/deploy-all.sh
#
# Future enhancement: Convert all manifests to Helm templates
