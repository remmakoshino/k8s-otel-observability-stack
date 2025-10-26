# トラブルシューティングガイド

このドキュメントでは、よくある問題と解決方法を説明します。

## 目次

1. [デプロイメント問題](#デプロイメント問題)
2. [ネットワーク問題](#ネットワーク問題)
3. [データ収集問題](#データ収集問題)
4. [パフォーマンス問題](#パフォーマンス問題)
5. [デバッグコマンド集](#デバッグコマンド集)

## デプロイメント問題

### Pod が起動しない (Pending状態)

**症状**:
```bash
$ kubectl get pods -n observability
NAME                    READY   STATUS    RESTARTS   AGE
prometheus-xxx          0/1     Pending   0          5m
```

**原因と解決方法**:

1. **リソース不足**
```bash
# ノードのリソース確認
kubectl describe node

# 解決: Minikubeのリソース増量
minikube delete
minikube start --cpus=4 --memory=8192
```

2. **PersistentVolumeの問題**
```bash
# PV/PVC確認
kubectl get pv,pvc -n observability

# 解決: StorageClassの確認
kubectl get storageclass
```

### Pod が CrashLoopBackOff

**症状**:
```bash
NAME                    READY   STATUS             RESTARTS   AGE
grafana-xxx             0/1     CrashLoopBackOff   5          5m
```

**診断**:
```bash
# ログ確認
kubectl logs -n observability grafana-xxx --previous

# イベント確認
kubectl describe pod -n observability grafana-xxx

# コンテナ内のシェル実行 (可能な場合)
kubectl exec -it -n observability grafana-xxx -- sh
```

**よくある原因**:

1. **設定ミス**
```bash
# ConfigMapの確認
kubectl get configmap -n observability grafana-config -o yaml

# 修正後、Podを再起動
kubectl rollout restart deployment -n observability grafana
```

2. **権限問題**
```bash
# SecurityContextの確認
kubectl get deployment -n observability grafana -o yaml | grep -A 10 securityContext
```

### ImagePullBackOff

**症状**:
```bash
NAME                    READY   STATUS             RESTARTS   AGE
backend-xxx             0/1     ImagePullBackOff   0          2m
```

**解決方法**:

1. **ローカルイメージの使用**
```bash
# MinikubeのDockerデーモンに切り替え
eval $(minikube docker-env)

# イメージビルド
docker build -t backend:latest ./apps/backend
docker build -t frontend:latest ./apps/frontend

# ImagePullPolicyを変更
kubectl patch deployment backend -p '{"spec":{"template":{"spec":{"containers":[{"name":"backend","imagePullPolicy":"Never"}]}}}}'
```

2. **イメージレジストリの確認**
```bash
# イメージ名の確認
kubectl get deployment backend -o yaml | grep image:

# 正しいイメージ名に更新
kubectl set image deployment/backend backend=backend:latest
```

## ネットワーク問題

### Service に接続できない

**症状**:
```bash
$ curl http://localhost:3000
curl: (7) Failed to connect to localhost port 3000: Connection refused
```

**診断**:

1. **Serviceの確認**
```bash
# Service一覧
kubectl get svc -n observability

# Endpoints確認
kubectl get endpoints -n observability grafana

# 詳細情報
kubectl describe svc -n observability grafana
```

2. **Port-forward再実行**
```bash
# 既存のport-forwardをkill
pkill -f "port-forward"

# 再実行
kubectl port-forward -n observability svc/grafana 3000:3000
```

### Pod間通信ができない

**症状**: Frontend から Backend に接続できない

**診断**:
```bash
# Frontendからネットワーク確認
kubectl exec -it frontend-xxx -- sh
wget -O- http://backend.default.svc.cluster.local:8080/health

# DNS確認
kubectl exec -it frontend-xxx -- nslookup backend.default.svc.cluster.local
```

**解決**:

1. **Service名の確認**
```bash
# 正しいDNS名
# <service-name>.<namespace>.svc.cluster.local
backend.default.svc.cluster.local
```

2. **NetworkPolicy確認** (有効な場合)
```bash
kubectl get networkpolicy -A
kubectl describe networkpolicy <policy-name>
```

### OTel Collectorに接続できない

**症状**: アプリケーションからメトリクス/トレースが送信されない

**診断**:
```bash
# OTel Collectorのログ確認
kubectl logs -n observability -l app=otel-collector --tail=100

# OTel CollectorのServiceエンドポイント確認
kubectl get endpoints -n observability otel-collector

# アプリからの疎通確認
kubectl exec -it frontend-xxx -- sh
telnet otel-collector.observability.svc.cluster.local 4317
```

**解決**:

1. **環境変数の確認**
```bash
kubectl exec frontend-xxx -- env | grep OTEL

# 期待値:
# OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.observability.svc.cluster.local:4317
```

2. **OTel Collector設定確認**
```bash
kubectl get configmap -n observability otel-collector-config -o yaml
```

## データ収集問題

### Prometheusにメトリクスが表示されない

**診断**:

1. **Targets確認**
```bash
# Prometheus UI
http://localhost:9090/targets

# ConfigMap確認
kubectl get configmap -n observability prometheus-config -o yaml
```

2. **OTel Collectorの確認**
```bash
# OTel Collectorのメトリクスエンドポイント
kubectl port-forward -n observability svc/otel-collector 8888:8888
curl http://localhost:8888/metrics
```

3. **アプリケーションのメトリクス確認**
```bash
# Backendのメトリクスエンドポイント
kubectl port-forward svc/backend 8080:8080
curl http://localhost:8080/metrics
```

**解決**:

```bash
# OTel Collector再起動
kubectl rollout restart deployment -n observability otel-collector

# Prometheus再起動
kubectl rollout restart deployment -n observability prometheus
```

### Jaegerにトレースが表示されない

**診断**:

1. **Jaeger UIの確認**
```bash
http://localhost:16686
# Service一覧にfrontend/backendが表示されるか
```

2. **OTel Collectorログ確認**
```bash
kubectl logs -n observability -l app=otel-collector | grep -i jaeger
```

3. **アプリケーションのトレース送信確認**
```bash
# アプリログでトレースIDを確認
kubectl logs frontend-xxx | grep -i trace
```

**解決**:

1. **OTel SDKの設定確認**
```javascript
// Frontend (Node.js)
const { TraceIdRatioBasedSampler } = require('@opentelemetry/sdk-trace-base');
// サンプリングレート確認: 1.0 = 100%
```

2. **Jaegerエンドポイント確認**
```yaml
# OTel Collector ConfigMap
exporters:
  jaeger:
    endpoint: jaeger-collector.observability.svc.cluster.local:14250
```

### Grafanaダッシュボードにデータが表示されない

**診断**:

1. **データソース確認**
```bash
# Grafana UI → Configuration → Data Sources
# Prometheus: http://prometheus.observability.svc.cluster.local:9090
# Jaeger: http://jaeger-query.observability.svc.cluster.local:16686
```

2. **クエリテスト**
```bash
# Grafana Explore
# PromQL: up{job="otel-collector"}
```

3. **時間範囲確認**
```bash
# ダッシュボード右上の時間範囲を確認
# Last 5 minutes → Last 1 hour
```

**解決**:

```bash
# データソースConfigMap確認
kubectl get configmap -n observability grafana-datasources -o yaml

# Grafana再起動
kubectl rollout restart deployment -n observability grafana
```

## パフォーマンス問題

### OTel Collectorの高CPU使用率

**診断**:
```bash
# リソース使用状況
kubectl top pod -n observability -l app=otel-collector

# 詳細メトリクス
kubectl port-forward -n observability svc/otel-collector 8888:8888
curl http://localhost:8888/metrics | grep processor
```

**解決**:

1. **Batch Processorチューニング**
```yaml
processors:
  batch:
    timeout: 10s
    send_batch_size: 1024
    send_batch_max_size: 2048
```

2. **リソース増量**
```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
```

3. **Horizontal Pod Autoscaler**
```bash
kubectl autoscale deployment -n observability otel-collector --cpu-percent=70 --min=2 --max=5
```

### Prometheusのメモリ使用量が多い

**診断**:
```bash
kubectl top pod -n observability -l app=prometheus
```

**解決**:

1. **保持期間の短縮**
```yaml
args:
  - '--storage.tsdb.retention.time=15d'  # デフォルト
  - '--storage.tsdb.retention.time=7d'   # 短縮
```

2. **メトリクスのフィルタリング**
```yaml
# 不要なメトリクスをdrop
metric_relabel_configs:
  - source_labels: [__name__]
    regex: 'go_gc_.*'
    action: drop
```

### アプリケーションのレイテンシ増加

**診断**:

1. **トレースで分析**
```bash
# Jaeger UIでレイテンシの高いトレース検索
# Operation: all
# Min Duration: 1s
```

2. **メトリクス確認**
```promql
# PromQL
histogram_quantile(0.99, 
  rate(http_request_duration_seconds_bucket[5m])
)
```

**解決**:

1. **サンプリングレート調整**
```javascript
// トレースサンプリング: 100% → 10%
const sampler = new TraceIdRatioBasedSampler(0.1);
```

2. **バッチサイズ調整**
```javascript
// Metric export間隔: 5s → 10s
const exporter = new OTLPMetricExporter({
  exportIntervalMillis: 10000,
});
```

## デバッグコマンド集

### 一般的なデバッグ

```bash
# すべてのリソース確認
kubectl get all -n observability

# イベント確認 (時系列)
kubectl get events -n observability --sort-by='.lastTimestamp'

# リソース使用状況
kubectl top nodes
kubectl top pods -n observability

# ログストリーミング
kubectl logs -f -n observability -l app=otel-collector
```

### 設定確認

```bash
# ConfigMap一覧
kubectl get configmap -n observability

# ConfigMap内容確認
kubectl describe configmap -n observability otel-collector-config

# Secret確認
kubectl get secret -n observability
```

### ネットワークデバッグ

```bash
# Pod内でネットワーク確認
kubectl exec -it -n observability otel-collector-xxx -- sh
apk add curl
curl http://prometheus:9090/-/ready

# DNS確認
kubectl exec -it -n observability otel-collector-xxx -- nslookup prometheus

# ポート確認
kubectl exec -it -n observability otel-collector-xxx -- netstat -tlnp
```

### 完全リセット

```bash
# すべて削除
kubectl delete namespace observability
kubectl delete all -l tier=application

# 再デプロイ
./scripts/deploy-all.sh
```

## よくある質問 (FAQ)

### Q: メトリクスが古い

**A**: Prometheusのscrape間隔を確認してください。

```yaml
scrape_configs:
  - job_name: 'otel-collector'
    scrape_interval: 15s  # デフォルト
```

### Q: ダッシュボードが保存されない

**A**: GrafanaのStorageを確認してください。

```bash
# PersistentVolumeを使用
kubectl get pvc -n observability grafana-storage
```

### Q: Jaegerのトレースが消える

**A**: Jaegerはデフォルトでインメモリストレージです。

```yaml
# 永続化にはCassandra/Elasticsearch使用
# または保持期間を確認
```

## サポート

問題が解決しない場合:

1. [GitHub Issues](https://github.com/remmakoshino/k8s-otel-observability-stack/issues)で質問
2. ログとエラーメッセージを含めて報告
3. 環境情報 (Kubernetes version, OS etc.) を記載

## 参考リンク

- [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug/)
- [OpenTelemetry Troubleshooting](https://opentelemetry.io/docs/collector/troubleshooting/)
- [Prometheus Troubleshooting](https://prometheus.io/docs/prometheus/latest/troubleshooting/)
