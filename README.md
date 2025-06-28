# LINE TODO Bot (Go) on Google Cloud

個人向けのLINE TODOボットをGoで構築し、Google Cloud上で運用するプロジェクトです。

> **注記**: このプロジェクトはAIエージェント（Claude Code）によって設計・実装されました。

## 機能

- **TODOの登録**: `todo <タイトル>` でTODOを登録
- **期限設定**: クイックリプライで期限を選択（今日中、明日まで、今週中、今月中、期限なし、日時指定）
- **TODO一覧表示**: `TODO一覧` で未完了タスクをFlex Messageで表示
- **TODO完了**: 各タスクの「完了」ボタンで完了扱い

## 技術構成

- **言語**: Go 1.24
- **Framework**: Echo
- **データベース**: Cloud Firestore (Native)
- **SDK**: LINE Bot SDK for Go v8
- **インフラ**: Cloud Run, Cloud Scheduler, Secret Manager
- **IaC**: Terraform

## ローカル開発

### 前提条件

- Go 1.24+
- Docker
- Google Cloud CLI
- ngrok (Webhook テスト用)

### セットアップ

1. **認証設定**
   ```bash
   # gcloudコンテナで認証
   docker-compose run gcloud gcloud auth application-default login
   ```

2. **環境変数設定**
   ```bash
   cp .env.example .env
   # .env ファイルを編集して各種キーを設定
   ```

3. **依存関係のインストール**
   ```bash
   go mod tidy
   ```

### Docker Compose で実行（推奨）

```bash
# アプリケーションを起動
docker-compose up app
```

### Go で直接実行

```bash
go run ./cmd/server
```

### Docker で実行

```bash
# イメージをビルド
docker build -t line-todo-bot .

# コンテナを起動
docker run -p 8080:8080 \
  --env-file .env \
  -v ~/.config/gcloud:/root/.config/gcloud:ro \
  line-todo-bot
```

### Webhook設定

```bash
# ngrokでローカルサーバーを公開
ngrok http 8080

# 出力されたURLをLINE Developersコンソールに設定
# 例: https://xxxxx.ngrok-free.app/webhook
```

**重要**: ngrokを再起動すると新しいURLが生成されるため、その都度LINE Developers コンソールでWebhook URLを更新する必要があります。

## デプロイ

### 1. Terraformでインフラ構築

```bash
cd terraform

# terraform.tfvarsファイルを作成
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvarsを編集

# インフラを構築
terraform init
terraform plan
terraform apply
```

### 2. Docker イメージをCloud Runにデプロイ

```bash
# Container Registryにイメージを push
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/line-todo-bot

# Cloud Run サービスを更新（Terraformで既に作成済み）
gcloud run services update line-todo-bot \
  --image gcr.io/YOUR_PROJECT_ID/line-todo-bot:latest \
  --region asia-northeast1
```

## コマンド仕様

| コマンド | 動作 |
|----------|------|
| `todo 買い物` | TODOを登録し、期限選択のクイックリプライを表示 |
| `TODO一覧` | 未完了タスクをFlex Messageで表示（各タスクに完了ボタン付き） |
| YYYY-MM-DD HH:MM | 日時指定時のフォーマット（例: 2025-06-30 15:00） |

## データモデル

```go
Todo {
  ID        string     // ドキュメントID(UUID)
  UserID    string     // LINE userId
  Title     string     // タスク内容
  IsDone    bool       // 完了フラグ
  DueAt     *time.Time // 締切日時（任意）
  CreatedAt time.Time  // 作成日時
}
```

## API エンドポイント

- `POST /webhook` - LINE Webhook
- `GET /health` - ヘルスチェック

## 料金

GCPの無料枠内で運用可能：
- Cloud Run: リクエスト課金、最小インスタンス0
- Firestore: 1日あたり50,000回の読み取り/20,000回の書き込みまで無料
- Cloud Scheduler: 月3ジョブまで無料
- Secret Manager: 月10,000回のアクセスまで無料
