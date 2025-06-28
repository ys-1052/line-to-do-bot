# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a LINE TODO Bot built with Go that runs on Google Cloud Platform. The bot allows users to manage TODO items through LINE messaging interface with natural language commands.

## Key Architecture

### Application Structure
- **Entry Point**: `cmd/server/main.go` - Echo web server with LINE Bot SDK v8
- **Webhook Handler**: `internal/handlers/webhook.go` - Processes LINE messages and postback events
- **Firestore Service**: `internal/services/firestore.go` - Database operations for TODO management
- **Data Model**: `internal/models/todo.go` - Todo struct definition

### LINE Bot Integration
- Uses LINE Bot SDK v8 with pointer-based message objects (`&messaging_api.TextMessage{}`)
- Case-insensitive command parsing with regex normalization: `regexp.MustCompile(`(?i)[tT][oO][dD][oO]`)`
- Quick Reply for deadline selection and confirmation dialogs
- Unicode emojis in responses (not LINE emojis)

### Infrastructure (Google Cloud)
- **Cloud Run**: Containerized application deployment
- **Firestore Native**: Document database for TODO storage
- **Secret Manager**: Secure credential storage
- **Artifact Registry**: Docker image storage

## Common Development Commands

### Local Development (Container-based)
```bash
# Start application with Docker Compose (recommended)
docker-compose up app

# Access gcloud CLI
docker-compose run --rm gcloud gcloud auth application-default login

# Access Terraform
docker-compose run --rm terraform terraform plan
```

### Direct Go Commands
```bash
# Run locally
go run ./cmd/server

# Install dependencies
go mod tidy

# Build binary
go build -o bin/server ./cmd/server
```

### Docker Commands
```bash
# Build production image
docker build -t line-todo-bot .

# Build development image
docker build -f Dockerfile.dev -t line-todo-bot-dev .
```

### Cloud Deployment
```bash
# Deploy infrastructure
docker-compose run --rm terraform terraform apply

# Build and push Docker image
docker-compose run --rm gcloud gcloud builds submit --tag asia-northeast1-docker.pkg.dev/PROJECT_ID/line-todo-bot/line-todo-bot:latest

# Update Cloud Run service (if needed manually)
docker-compose run --rm gcloud gcloud run services update line-todo-bot --image asia-northeast1-docker.pkg.dev/PROJECT_ID/line-todo-bot/line-todo-bot:latest --region asia-northeast1
```

## Important Development Notes

### LINE Bot Response Format
- Always use pointer types for LINE message objects: `&messaging_api.TextMessage{Text: text}`
- Message responses must use `h.bot.ReplyMessage()` with proper request structure

### Command Pattern Matching
- TODO commands are case-insensitive and handled through regex normalization
- Direct patterns: `TODO テスト` (without action keywords)
- Action patterns: `TODO追加 テスト`, `削除 テスト`, `一覧`, `全削除`
- Confirmation flows use postback actions with structured data format

### Firestore Operations
- Collection: `todos` with composite queries requiring indexes
- Field names: `userId`, `isDone`, `createdAt`, `dueAt`, `title`
- UUIDs for document IDs via `github.com/google/uuid`

### Environment Configuration
- Required: `GOOGLE_CLOUD_PROJECT`, `LINE_CHANNEL_TOKEN`, `LINE_CHANNEL_SECRET`
- Local: `.env` file (never commit)
- Production: Google Secret Manager integration

### Infrastructure as Code
- Terraform configuration in `terraform/` directory
- Variables in `terraform.tfvars` (never commit - use `.example` for templates)
- State management handled by Terraform Cloud or local state files

## Security Considerations
- Credentials stored in Secret Manager, not in code
- `.gitignore` excludes `.env`, `terraform.tfvars`, and `*.json` files
- Docker containers run with read-only credential mounts
- LINE webhook signature verification implemented

## Webhook URL Management
- Local development: Use ngrok for public webhook URL
- Production: Cloud Run provides HTTPS endpoint automatically
- URL format: `https://SERVICE-URL/webhook`
- Must be updated in LINE Developers Console when service URL changes