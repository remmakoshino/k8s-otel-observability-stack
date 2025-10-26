# Kubernetes Observability Stack - Additional Resources

## Quick Reference

### Port Forwarding Commands

```bash
# Grafana
kubectl port-forward -n observability svc/grafana 3000:3000

# Prometheus
kubectl port-forward -n observability svc/prometheus 9090:9090

# Jaeger
kubectl port-forward -n observability svc/jaeger-query 16686:16686

# Frontend
kubectl port-forward svc/frontend 8080:8080

# Backend (direct access)
kubectl port-forward svc/backend 8081:8080
```

### Useful Commands

```bash
# Check all pods
kubectl get pods -A

# View logs
kubectl logs -n observability -l app=otel-collector --tail=100
kubectl logs -l app=frontend --tail=100

# Describe resources
kubectl describe pod -n observability <pod-name>

# Scale deployment
kubectl scale deployment -n observability otel-collector --replicas=3

# Restart deployment
kubectl rollout restart deployment -n observability grafana

# Delete all resources
kubectl delete namespace observability
kubectl delete all -l tier=application
```

## Grafana Dashboards

### Default Credentials
- Username: `admin`
- Password: `admin`

### Pre-configured Dashboards
1. **Observability Stack Overview** - Overall system health
2. **SLI/SLO Dashboard** - Service level indicators
3. **Application Metrics** - App-specific metrics

### Creating Custom Dashboards

1. Navigate to Dashboards → New Dashboard
2. Add Panel
3. Use PromQL queries:
   ```promql
   # Request rate
   rate(http_requests_total[5m])
   
   # Error rate
   rate(http_requests_total{status=~"5.."}[5m])
   
   # Latency p99
   histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
   ```

## Monitoring Best Practices

### Golden Signals
- **Latency**: How long it takes to serve a request
- **Traffic**: How much demand is placed on your system
- **Errors**: Rate of requests that fail
- **Saturation**: How "full" your service is

### RED Method (for requests)
- **Rate**: Number of requests per second
- **Errors**: Number of failed requests
- **Duration**: Time taken to serve requests

### USE Method (for resources)
- **Utilization**: Percentage of time resource is busy
- **Saturation**: Amount of work resource cannot service
- **Errors**: Count of error events

## Development Workflow

### Local Development

1. Start Minikube:
   ```bash
   ./scripts/setup-minikube.sh
   ```

2. Make changes to application code

3. Rebuild and redeploy:
   ```bash
   eval $(minikube docker-env)
   docker build -t backend:latest ./apps/backend
   kubectl rollout restart deployment backend
   ```

### Testing

```bash
# Health checks
curl http://localhost:8080/health

# API endpoints
curl http://localhost:8080/api/users
curl http://localhost:8080/api/users/1
curl -X POST http://localhost:8080/api/process

# Load test
curl "http://localhost:8080/api/load-test?requests=100"
```

## Production Considerations

### High Availability

1. **Multiple Replicas**
   ```yaml
   replicas: 3
   ```

2. **Pod Disruption Budgets**
   ```yaml
   apiVersion: policy/v1
   kind: PodDisruptionBudget
   metadata:
     name: otel-collector-pdb
   spec:
     minAvailable: 1
     selector:
       matchLabels:
         app: otel-collector
   ```

3. **Resource Requests/Limits**
   - Set appropriate values based on load testing
   - Monitor actual usage and adjust

### Security

1. **Network Policies**
   - Restrict traffic between namespaces
   - Allow only necessary connections

2. **RBAC**
   - Principle of least privilege
   - Service account per component

3. **Secrets Management**
   - Use Kubernetes Secrets for sensitive data
   - Consider external secret management (Vault, etc.)

### Storage

1. **Persistent Volumes**
   ```yaml
   volumeClaimTemplates:
   - metadata:
       name: prometheus-storage
     spec:
       accessModes: ["ReadWriteOnce"]
       resources:
         requests:
           storage: 50Gi
   ```

2. **Backup Strategy**
   - Regular snapshots
   - Retention policies
   - Disaster recovery plan

### Scaling

1. **Horizontal Pod Autoscaler**
   ```bash
   kubectl autoscale deployment otel-collector \
     --cpu-percent=70 \
     --min=2 \
     --max=10
   ```

2. **Vertical Pod Autoscaler** (optional)
   - Automatically adjust resource requests/limits

## Troubleshooting Tips

### Pod Won't Start

1. Check events: `kubectl describe pod <pod-name>`
2. Check logs: `kubectl logs <pod-name> --previous`
3. Verify image exists: `docker images`
4. Check resource quotas: `kubectl describe node`

### Metrics Not Appearing

1. Verify OTel Collector is running
2. Check application logs for export errors
3. Verify network connectivity
4. Check Prometheus targets: http://localhost:9090/targets

### High Resource Usage

1. Check metrics in Grafana
2. Review application code for inefficiencies
3. Adjust resource limits
4. Scale horizontally

## Additional Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Grafana Tutorials](https://grafana.com/tutorials/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [SRE Book](https://sre.google/books/)
