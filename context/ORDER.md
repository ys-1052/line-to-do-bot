# LINE TODO Bot (Go) on Google Cloud

## 目的
Go で個人向けの LINE TODO ボットを Google Cloud 上に簡単に構築・運用する。最低限の機能で構成し、維持コストも抑える。

## ✅ 最小要件

### LINEのチャットで TODO を登録できる
- **書式**: `todo <タイトル>` のみで登録可能（例：`todo 買い物`）
- 送信後、Botが「期限はありますか？」と聞き返し、以下のクイックリプライを提示：
  - 今日中
  - 明日まで
  - 今週中
  - 今月中
  - 期限なし
  - 日時を指定する

- ユーザーが「日時を指定する」を選んだ場合は、自由入力で日時（例：`2025-06-30 15:00`）を送信させる。
- この場合、フォーマットは `YYYY-MM-DD HH:MM` 形式で入力してもらう必要がある。

### TODO 一覧を取得できる（完了前のみ）
- **書式**: `TODO一覧`
- Bot は各タスクを Flex Message で表示し、各タスクに「完了」ボタン（Postback Action）を含める

### TODO を完了扱いにできる
- Bot は Postback Event を受け取って該当タスクを完了状態に変更する

### その他の要件
- すべてのデータは Firestore に保存（Cloud Firestore / Native）
- GCP の無料枠内で完結（Cloud Run, Scheduler, Firestore, Secret Manager）
- LINE Messaging API の Webhook / Reply / Push / Flex Message / Postback を使用

## 🧱 技術構成

| 項目 | 内容 |
|------|------|
| 使用言語 | Go 最新版 |
| Web Framework | Echo |
| DB | Firestore（Nativeモード） |
| LINE SDK | github.com/line/line-bot-sdk-go/v8 |
| コンテナ | Docker（ローカル開発・本番デプロイ共通） |
| デプロイ | Cloud Run（最小インスタンス0） |
| 認証情報管理 | Secret Manager: LINE_CHANNEL_SECRET・LINE_CHANNEL_TOKENを格納。ローカル開発時は .env を使用し、Go アプリは github.com/joho/godotenv を使って .env を読み込む構成とする。 |
| インフラ管理 | Terraform を使用し、Cloud Run、Scheduler、Firestore などを定義 |
| UI補助 | LINEリッチメニュー、クイックリプライ、Flex Message + Postback ボタン |

## 🗃 データモデル（Firestore: todos コレクション）

```go
Todo {
  id: string         // ドキュメントID(UUID)
  userId: string     // LINE userId
  title: string      // タスク内容
  isDone: boolean    // 完了フラグ
  dueAt?: timestamp  // 締切日時（任意）
  createdAt: timestamp
}
```

## 🧾 コマンドパターン

| コマンド | 内容 |
|----------|------|
| `todo 買い物` | Botが期限の有無を聞き、選択肢提示（クイックリプライ）→ 期限を自動設定または自由入力を促す |
| `TODO一覧` | 未完了タスクを Flex Message 形式で表示し、それぞれに「完了」ボタンを含める |

## 🧪 ローカル開発手順

### 従来の Go 実行方法

```bash
gcloud auth application-default login
cp .env.example .env # .env に各種キーを定義

# Go アプリ内で godotenv を使用し、.env を読み込む構成にするため export は不要

go run ./cmd/server
```

### Docker を使用した開発方法

```bash
# 1. 認証と環境設定
gcloud auth application-default login
cp .env.example .env # .env に各種キーを定義

# 2. Docker イメージをビルド
docker build -t line-todo-bot .

# 3. コンテナを起動（.env ファイルと GCP 認証情報をマウント）
docker run -p 8080:8080 \
  --env-file .env \
  -v ~/.config/gcloud:/root/.config/gcloud:ro \
  line-todo-bot

# 4. 開発時の再ビルド（ソースコード変更時）
docker build -t line-todo-bot . && docker run -p 8080:8080 --env-file .env -v ~/.config/gcloud:/root/.config/gcloud:ro line-todo-bot
```

### Webhook 設定
Webhook は ngrok などで公開して LINE Developers に登録（例：`ngrok http 8080` → `https://xxxxx.ngrok.io/webhook` を登録）



