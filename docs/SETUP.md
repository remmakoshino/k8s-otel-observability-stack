# セットアップガイド

このガイドでは、Kubernetes Observability Stackの詳細なセットアップ手順を説明します。

## 目次

1. [前提条件](#前提条件)
2. [Minikubeセットアップ](#minikubeセットアップ)
3. [手動デプロイ](#手動デプロイ)
4. [Helmデプロイ](#helmデプロイ)
5. [検証](#検証)
6. [アクセス方法](#アクセス方法)

## 前提条件

### 必須ツール

```bash
# kubectl
kubectl version --client

# minikube
minikube version

# Docker
docker --version

# Helm (オプション)
helm version
```

### バージョン要件

- Kubernetes: 1.24+
- kubectl: 1.24+
- minikube: 1.30+
- Docker: 20.10+
- Helm: 3.10+ (オプション)

## Minikubeセットアップ

### 1. Minikube起動

```bash
# スクリプトを使用 (推奨)
./scripts/setup-minikube.sh

# または手動で
minikube start \
  --cpus=4 \
  --memory=8192 \
  --disk-size=20g \
  --driver=docker \
  --kubernetes-version=v1.28.0
```

### 2. Minikubeアドオン有効化

```bash
# メトリクスサーバー
minikube addons enable metrics-server

# Ingress (オプション)
minikube addons enable ingress

# ダッシュボード (オプション)
minikube addons enable dashboard
```

### 3. 動作確認

```bash
# クラスター情報
kubectl cluster-info

# ノード確認
kubectl get nodes

# コンテキスト確認
kubectl config current-context
```

## 手動デプロイ

### Step 1: Namespace作成

```bash
kubectl apply -f kubernetes/namespaces/observability.yaml

# 確認
kubectl get namespaces
```

### Step 2: OpenTelemetry Collector

```bash
# ConfigMap
kubectl apply -f kubernetes/otel-collector/configmap.yaml

# Deployment & Service
kubectl apply -f kubernetes/otel-collector/deployment.yaml
kubectl apply -f kubernetes/otel-collector/service.yaml

# 確認
kubectl get pods -n observability -l app=otel-collector
```

### Step 3: Prometheus

```bash
# RBAC
kubectl apply -f kubernetes/prometheus/rbac.yaml

# ConfigMap
kubectl apply -f kubernetes/prometheus/configmap.yaml

# Deployment & Service
kubectl apply -f kubernetes/prometheus/deployment.yaml
kubectl apply -f kubernetes/prometheus/service.yaml

# 確認
kubectl get pods -n observability -l app=prometheus
```

### Step 4: Jaeger

```bash
# Deployment & Service
kubectl apply -f kubernetes/jaeger/deployment.yaml
kubectl apply -f kubernetes/jaeger/service.yaml

# 確認
kubectl get pods -n observability -l app=jaeger
```

### Step 5: Grafana

```bash
# ConfigMaps
kubectl apply -f kubernetes/grafana/configmap-datasources.yaml
kubectl apply -f kubernetes/grafana/configmap-dashboards.yaml

# Deployment & Service
kubectl apply -f kubernetes/grafana/deployment.yaml
kubectl apply -f kubernetes/grafana/service.yaml

# 確認
kubectl get pods -n observability -l app=grafana
```

### Step 6: サンプルアプリケーション

```bash
# Backendアプリ
kubectl apply -f kubernetes/sample-apps/backend/deployment.yaml
kubectl apply -f kubernetes/sample-apps/backend/service.yaml

# Frontendアプリ
kubectl apply -f kubernetes/sample-apps/frontend/deployment.yaml
kubectl apply -f kubernetes/sample-apps/frontend/service.yaml

# 確認
kubectl get pods -l tier=application
```

### 全体確認

```bash
# Observabilityスタック
kubectl get all -n observability

# アプリケーション
kubectl get all -l tier=application
```

## Helmデプロイ

### 1. Helmチャートの確認

```bash
cd helm/observability-stack

# 依存関係の更新
helm dependency update

# チャートの検証
helm lint .
```

### 2. values.yamlのカスタマイズ

```yaml
# helm/observability-stack/values.yaml

prometheus:
  retention: 30d
  resources:
    limits:
      memory: 2Gi

grafana:
  adminPassword: "your-secure-password"
```

### 3. インストール

```bash
# Dry-run
helm install observability-stack ./helm/observability-stack \
  --namespace observability \
  --create-namespace \
  --dry-run --debug

# 実際のインストール
helm install observability-stack ./helm/observability-stack \
  --namespace observability \
  --create-namespace
```

### 4. アップグレード

```bash
helm upgrade observability-stack ./helm/observability-stack \
  --namespace observability
```

### 5. アンインストール

```bash
helm uninstall observability-stack --namespace observability
```

## 検証

### 1. Pod起動確認

```bash
# すべてのPodがRunning状態か確認
kubectl get pods -n observability
kubectl get pods -l tier=application

# 詳細確認
kubectl describe pod -n observability <pod-name>
```

### 2. ログ確認

```bash
# OTel Collector
kubectl logs -n observability -l app=otel-collector --tail=50

# Prometheus
kubectl logs -n observability -l app=prometheus --tail=50

# Jaeger
kubectl logs -n observability -l app=jaeger --tail=50

# Grafana
kubectl logs -n observability -l app=grafana --tail=50
```

### 3. サービス確認

```bash
# サービス一覧
kubectl get svc -n observability

# エンドポイント確認
kubectl get endpoints -n observability
```

### 4. 設定確認

```bash
# OTel Collector設定
kubectl get configmap -n observability otel-collector-config -o yaml

# Prometheus設定
kubectl get configmap -n observability prometheus-config -o yaml
```

## アクセス方法

### Port Forward方式

```bash
# Grafana (推奨)
kubectl port-forward -n observability svc/grafana 3000:3000
# http://localhost:3000 (admin/admin)

# Prometheus
kubectl port-forward -n observability svc/prometheus 9090:9090
# http://localhost:9090

# Jaeger
kubectl port-forward -n observability svc/jaeger-query 16686:16686
# http://localhost:16686

# Frontend App
kubectl port-forward svc/frontend 8080:8080
# http://localhost:8080

# Backend App (直接アクセス - デバッグ用)
kubectl port-forward svc/backend 8081:8080
# http://localhost:8081
```

### Minikube Service方式

```bash
# Grafana
minikube service grafana -n observability

# Prometheus
minikube service prometheus -n observability

# Jaeger
minikube service jaeger-query -n observability

# Frontend
minikube service frontend
```

### Ingress方式 (オプション)

Ingressコントローラーを有効化している場合:

```bash
# Ingressリソース作成
kubectl apply -f kubernetes/ingress.yaml

# Minikube IP取得
minikube ip

# /etc/hostsに追加
echo "$(minikube ip) grafana.local prometheus.local jaeger.local" | sudo tee -a /etc/hosts
```

アクセス:
- http://grafana.local
- http://prometheus.local
- http://jaeger.local

## 動作テスト

### 1. アプリケーションテスト

```bash
# Frontendにリクエスト送信
curl http://localhost:8080/

# APIエンドポイント
curl http://localhost:8080/api/users
curl http://localhost:8080/api/health
```

### 2. メトリクス確認

```bash
# Prometheusでクエリ
# PromQL例: rate(http_requests_total[5m])
open http://localhost:9090/graph
```

### 3. トレース確認

```bash
# Jaeger UIでトレース検索
# Service: frontend, backend
open http://localhost:16686
```

### 4. ダッシュボード確認

```bash
# Grafana
open http://localhost:3000

# ログイン: admin/admin
# Dashboards → Browse → Application Metrics
```

## トラブルシューティング

### Podが起動しない

```bash
# イベント確認
kubectl get events -n observability --sort-by='.lastTimestamp'

# Pod詳細
kubectl describe pod -n observability <pod-name>

# ログ確認
kubectl logs -n observability <pod-name> --previous
```

### イメージがプルできない

```bash
# Minikubeのdockerデーモン使用
eval $(minikube docker-env)

# イメージビルド
docker build -t frontend:latest ./apps/frontend
docker build -t backend:latest ./apps/backend

# イメージ確認
docker images | grep -E 'frontend|backend'
```

### リソース不足

```bash
# Minikube再起動 (リソース増量)
minikube delete
minikube start --cpus=4 --memory=8192 --disk-size=20g
```

### ConfigMap/Secret更新が反映されない

```bash
# Pod再起動
kubectl rollout restart deployment -n observability <deployment-name>

# または削除して再作成
kubectl delete pod -n observability -l app=<app-name>
```

## 次のステップ

- [アーキテクチャ詳細](ARCHITECTURE.md)を確認
- [トラブルシューティング](TROUBLESHOOTING.md)を確認
- カスタムダッシュボードの作成
- アラートルールの設定
- 本番環境への展開計画

## 参考資料

- [Kubernetes公式ドキュメント](https://kubernetes.io/docs/)
- [Minikube公式ドキュメント](https://minikube.sigs.k8s.io/docs/)
- [Helm公式ドキュメント](https://helm.sh/docs/)
