# 動作確認手順

このドキュメントでは、Kubernetes Observability Stack with OpenTelemetryのデプロイ後の動作確認手順を説明します。

## 前提条件

- Minikubeクラスタが起動していること
- 全てのコンポーネントがデプロイされていること
- `kubectl`コマンドが使用可能であること

## 1. デプロイメント状態の確認

### 1.1 Observabilityスタックの確認

```bash
kubectl get pods -n observability
```

**期待される出力:**
```
NAME                              READY   STATUS    RESTARTS   AGE
grafana-xxxxx                     1/1     Running   0          Xm
jaeger-xxxxx                      1/1     Running   0          Xm
otel-collector-xxxxx              1/1     Running   0          Xm
otel-collector-xxxxx              1/1     Running   0          Xm
prometheus-xxxxx                  1/1     Running   0          Xm
```

全てのPodが`Running`状態で`READY`が`1/1`または`2/2`になっていることを確認してください。

### 1.2 アプリケーションPodの確認

```bash
kubectl get pods -l tier=application
```

**期待される出力:**
```
NAME                        READY   STATUS    RESTARTS   AGE
backend-xxxxx               1/1     Running   0          Xm
backend-xxxxx               1/1     Running   0          Xm
frontend-xxxxx              1/1     Running   0          Xm
frontend-xxxxx              1/1     Running   0          Xm
```

### 1.3 サービスの確認

```bash
# Observabilityスタックのサービス
kubectl get svc -n observability

# アプリケーションサービス
kubectl get svc -l tier=application
```

## 2. ポートフォワードの設定

各サービスにアクセスするため、ポートフォワードを設定します。

### 2.1 Grafanaへのアクセス設定

```bash
kubectl port-forward -n observability svc/grafana 3000:3000
```

ブラウザで http://localhost:3000 にアクセス

- **ユーザー名:** `admin`
- **パスワード:** `admin`

> **Note:** 初回ログイン時にパスワード変更を求められる場合があります。

### 2.2 Prometheusへのアクセス設定

```bash
kubectl port-forward -n observability svc/prometheus 9090:9090
```

ブラウザで http://localhost:9090 にアクセス

### 2.3 Jaegerへのアクセス設定

```bash
kubectl port-forward -n observability svc/jaeger-query 16686:16686
```

ブラウザで http://localhost:16686 にアクセス

### 2.4 Frontendアプリケーションへのアクセス設定

```bash
kubectl port-forward svc/frontend 8080:8080
```

ブラウザで http://localhost:8080 にアクセス

> **Tip:** 全てのポートフォワードをバックグラウンドで実行する場合は、コマンドの最後に`&`を追加してください。
> ```bash
> kubectl port-forward -n observability svc/grafana 3000:3000 > /dev/null 2>&1 &
> ```

## 3. アプリケーション動作確認

### 3.1 ユーザー一覧の取得

```bash
curl http://localhost:8080/api/users
```

**期待されるレスポンス:**
```json
{
  "count": 3,
  "users": [
    {
      "id": 1,
      "name": "Alice",
      "email": "alice@example.com",
      "created_at": "2025-10-26T09:46:14.343116378Z"
    },
    {
      "id": 2,
      "name": "Bob",
      "email": "bob@example.com",
      "created_at": "2025-10-25T09:46:14.343116378Z"
    },
    {
      "id": 3,
      "name": "Charlie",
      "email": "charlie@example.com",
      "created_at": "2025-10-24T09:46:14.343116378Z"
    }
  ]
}
```

### 3.2 特定ユーザーの取得

```bash
curl http://localhost:8080/api/users/1
```

**期待されるレスポンス:**
```json
{
  "id": 1,
  "name": "Alice",
  "email": "alice@example.com",
  "created_at": "2025-10-26T09:46:14.343116378Z"
}
```

### 3.3 テストトラフィックの生成

複数のリクエストを送信してトレースとメトリクスを生成します。

```bash
# 5回のリクエストを送信
for i in {1..5}; do
  curl -s http://localhost:8080/api/users > /dev/null
  echo "Request $i: Success"
  sleep 1
done

# 個別ユーザーへのリクエスト
for user_id in {1..3}; do
  curl -s http://localhost:8080/api/users/$user_id > /dev/null
  echo "Request for user $user_id: Success"
  sleep 1
done
```

### 3.4 エラーケースのテスト

```bash
# 存在しないユーザーへのリクエスト
curl -i http://localhost:8080/api/users/999
```

**期待されるレスポンス:**
```
HTTP/1.1 404 Not Found
...
{"error":"User not found"}
```

## 4. Observabilityスタックの確認

### 4.1 Prometheusでメトリクスを確認

1. http://localhost:9090 にアクセス
2. 以下のクエリを実行して、メトリクスが収集されていることを確認:

#### OTel Collectorの稼働確認

```promql
up{job="otel-collector"}
```

期待値: `1` (稼働中)

#### HTTPリクエスト数の確認

```promql
otel_http_server_duration_count
```

#### HTTPリクエストレイテンシーの確認

```promql
histogram_quantile(0.95, rate(otel_http_server_duration_bucket[5m]))
```

#### バックエンドのカスタムメトリクス確認

```promql
otel_http_requests_total
```

### 4.2 Jaegerで分散トレースを確認

1. http://localhost:16686 にアクセス
2. **Service** ドロップダウンから `frontend` を選択
3. **Find Traces** をクリック
4. トレース一覧が表示されることを確認

#### 確認すべきポイント

- **Frontend → Backend** の通信トレースが存在すること
- トレースに以下のSpanが含まれていること:
  - `GET /api/users` (Frontend)
  - `HTTP GET` (Backend呼び出し)
  - `GET /users` (Backend)
- 各Spanにタグ（メタデータ）が付与されていること:
  - `http.method`
  - `http.url`
  - `http.status_code`

#### トレース詳細の確認

トレースをクリックして、以下を確認:

1. **Span間の依存関係**: Frontend → Backend の順序
2. **レイテンシー**: 各Spanの実行時間
3. **Logs**: Span内のイベントログ
4. **Tags**: HTTP情報、サービス情報など

### 4.3 Grafanaでダッシュボードを確認

1. http://localhost:3000 にアクセス（admin/admin）
2. 左メニューから **Dashboards** を選択
3. **OpenTelemetry Dashboard** を開く

#### 確認すべきメトリクス

- **Request Rate**: 1秒あたりのリクエスト数
- **Error Rate**: エラー率
- **Response Time**: レスポンスタイムの分布（P50, P95, P99）
- **Throughput**: スループット
- **Active Connections**: アクティブな接続数

#### データソースの確認

1. **Configuration** (歯車アイコン) → **Data Sources** を選択
2. **Prometheus** データソースが設定されていることを確認
3. **Test** ボタンをクリックして接続を確認

## 5. ログの確認

### 5.1 Frontendのログ確認

```bash
kubectl logs -l app=frontend --tail=50
```

**確認すべきログ:**
```
{"level":"info","message":"OpenTelemetry initialized successfully",...}
{"level":"info","message":"Server listening on port 8080",...}
{"level":"info","message":"GET /api/users",...}
```

### 5.2 Backendのログ確認

```bash
kubectl logs -l app=backend --tail=50
```

**確認すべきログ:**
```
{"level":"info","msg":"Starting backend server on :8080",...}
{"level":"info","msg":"GET /users",...}
```

### 5.3 OTel Collectorのログ確認

```bash
kubectl logs -n observability -l app=otel-collector --tail=50
```

**確認すべきログ:**
```
info    service@v0.91.0/service.go:145  Starting otelcol-contrib...
info    service@v0.91.0/service.go:171  Everything is ready. Begin running and processing data.
info    MetricsExporter {"kind": "exporter", "data_type": "metrics", ...}
```

## 6. トラブルシューティング

### 6.1 Podが起動しない場合

```bash
# Pod詳細の確認
kubectl describe pod <pod-name> -n <namespace>

# イベントの確認
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
```

### 6.2 メトリクスが表示されない場合

```bash
# OTel Collectorのログを確認
kubectl logs -n observability deployment/otel-collector

# Prometheusのターゲット確認
# http://localhost:9090/targets にアクセスして、全ターゲットが「UP」であることを確認
```

### 6.3 トレースが表示されない場合

```bash
# Jaegerのログを確認
kubectl logs -n observability deployment/jaeger

# OTel CollectorがJaegerにエクスポートしているか確認
kubectl logs -n observability -l app=otel-collector | grep jaeger
```

### 6.4 ポートフォワードが動作しない場合

```bash
# 既存のポートフォワードプロセスを確認
ps aux | grep port-forward

# プロセスを終了
kill <PID>

# 再度ポートフォワードを実行
kubectl port-forward -n observability svc/grafana 3000:3000
```

## 7. クリーンアップ

テスト完了後、リソースをクリーンアップする場合:

```bash
# アプリケーションの削除
kubectl delete -f kubernetes/sample-apps/

# Observabilityスタックの削除
kubectl delete -f kubernetes/grafana/
kubectl delete -f kubernetes/jaeger/
kubectl delete -f kubernetes/prometheus/
kubectl delete -f kubernetes/otel-collector/

# Namespaceの削除
kubectl delete namespace observability

# Minikubeクラスタの削除（完全に削除する場合）
minikube delete
```

## 8. 高度な検証

### 8.1 負荷テスト

```bash
# Apache Benchを使用した負荷テスト
ab -n 1000 -c 10 http://localhost:8080/api/users

# または hey を使用
hey -n 1000 -c 10 http://localhost:8080/api/users
```

負荷テスト後、Grafanaで以下を確認:

- レイテンシーの変化
- スループットの変化
- エラー率

### 8.2 カスタムメトリクスの追加確認

Prometheusで以下のカスタムメトリクスを確認:

```promql
# ユーザー作成カウンター（Backendで実装されている場合）
users_created_total

# アクティブリクエスト数
active_requests

# データベース接続プール（実装されている場合）
db_connections_active
```

### 8.3 SLI/SLOの評価

Grafanaダッシュボードで以下のSLIを確認:

- **Availability SLI**: 稼働率 > 99.9%
- **Latency SLI**: P95レイテンシー < 200ms
- **Error Rate SLI**: エラー率 < 0.1%

## 9. まとめ

以下が正常に動作していることを確認できれば、デプロイは成功です:

- ✅ 全てのPodが`Running`状態
- ✅ FrontendからBackendへのAPI呼び出しが成功
- ✅ Prometheusでメトリクスが収集されている
- ✅ Jaegerでトレースが表示される
- ✅ Grafanaでダッシュボードが表示される
- ✅ OTel Collectorがテレメトリデータを正常に転送している

## 10. 次のステップ

- カスタムダッシュボードの作成
- アラートルールの設定
- 長期的なメトリクス保存の設定
- 本番環境への展開準備
- CI/CDパイプラインとの統合

詳細は以下のドキュメントを参照してください:

- [ARCHITECTURE.md](./ARCHITECTURE.md) - アーキテクチャ詳細
- [SETUP.md](./SETUP.md) - セットアップ手順
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - トラブルシューティングガイド
- [ADDITIONAL_RESOURCES.md](./ADDITIONAL_RESOURCES.md) - 追加リソース
