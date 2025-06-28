package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/ytakahashi/line-to-do-bot/internal/services"
)

type WebhookHandler struct {
	bot       *messaging_api.MessagingApiAPI
	firestore *services.FirestoreService
}

func NewWebhookHandler(bot *messaging_api.MessagingApiAPI, firestore *services.FirestoreService) *WebhookHandler {
	return &WebhookHandler{
		bot:       bot,
		firestore: firestore,
	}
}

func getUserID(source webhook.SourceInterface) string {
	switch s := source.(type) {
	case webhook.UserSource:
		return s.UserId
	case webhook.GroupSource:
		return s.UserId
	case webhook.RoomSource:
		return s.UserId
	default:
		return ""
	}
}

func (h *WebhookHandler) HandleWebhook(c echo.Context) error {
	cb, err := webhook.ParseRequest(os.Getenv("LINE_CHANNEL_SECRET"), c.Request())
	if err != nil {
		if err == webhook.ErrInvalidSignature {
			log.Println("Invalid signature")
			return c.NoContent(http.StatusBadRequest)
		}
		log.Printf("Parse request error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, event := range cb.Events {
		switch e := event.(type) {
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				userID := getUserID(e.Source)
				if err := h.handleTextMessage(e.ReplyToken, userID, message.Text); err != nil {
					log.Printf("Error handling text message: %v", err)
				}
			}
		case webhook.PostbackEvent:
			userID := getUserID(e.Source)
			if err := h.handlePostback(e.ReplyToken, userID, e.Postback.Data); err != nil {
				log.Printf("Error handling postback: %v", err)
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *WebhookHandler) handleTextMessage(replyToken, userID, text string) error {
	// デバッグログ
	log.Printf("Received text: '%s'", text)

	// 大文字小文字を区別しないように変換
	lowerText := strings.ToLower(text)
	log.Printf("Lower text: '%s'", lowerText)

	// まず大文字小文字を統一して処理しやすくする
	normalizedText := regexp.MustCompile(`(?i)[tT][oO][dD][oO]`).ReplaceAllString(text, "TODO")

	// TODO追加のパターンマッチング
	// パターン1: "TODO テスト" のような形式（追加、削除、一覧、全、ヘルプが含まれていない場合）
	todoDirectPattern := regexp.MustCompile(`^TODO[\s　]+(.+)$`)
	if matches := todoDirectPattern.FindStringSubmatch(normalizedText); matches != nil {
		title := strings.TrimSpace(matches[1])
		// 特定のキーワードが含まれていないかチェック
		if !regexp.MustCompile(`(追加|削除|一覧|全|ヘルプ)`).MatchString(title) {
			return h.askForDeadline(replyToken, userID, title)
		}
	}

	// パターン2: "TODO追加 テスト" または "追加 テスト"
	addPattern := regexp.MustCompile(`^(TODO[\s　]*)?追加[\s　]+[\""]?([^\"\"]+)[\""]?$`)
	if matches := addPattern.FindStringSubmatch(normalizedText); matches != nil {
		title := strings.TrimSpace(matches[2])
		if title == "" {
			return h.replyMessage(replyToken, "TODOのタイトルを入力してください。\n例: TODO テスト")
		}
		return h.askForDeadline(replyToken, userID, title)
	}

	// TODO一覧表示
	todoListPattern := regexp.MustCompile(`^(TODO[\s　]*)?一覧$`)
	if todoListPattern.MatchString(normalizedText) || lowerText == "一覧" {
		return h.showTodoList(replyToken, userID)
	}

	// TODO全削除のパターンマッチング
	deleteAllPattern := regexp.MustCompile(`^(TODO[\s　]*)?全削除$`)
	if deleteAllPattern.MatchString(normalizedText) || lowerText == "全削除" {
		return h.askDeleteAllConfirmation(replyToken, userID)
	}

	// TODO削除のパターンマッチング
	deletePattern := regexp.MustCompile(`^(TODO[\s　]*)?削除[\s　]+[\""]?([^\"\"]+)[\""]?$`)
	if matches := deletePattern.FindStringSubmatch(normalizedText); matches != nil {
		title := strings.TrimSpace(matches[2])
		if title == "" {
			return h.replyMessage(replyToken, "削除するTODOのタイトルを入力してください。\n例: 削除 \"買い物\"")
		}
		return h.deleteTodoByTitle(replyToken, userID, title)
	}

	// ヘルプ表示
	if lowerText == "ヘルプ" {
		return h.showHelp(replyToken)
	}

	// 認識できないメッセージには応答しない
	return nil
}

func (h *WebhookHandler) askForDeadline(replyToken, userID, title string) error {
	quickReply := &messaging_api.QuickReply{
		Items: []messaging_api.QuickReplyItem{
			{
				Action: &messaging_api.PostbackAction{
					Label:       "今日中",
					Data:        fmt.Sprintf("deadline:today:%s:%s", userID, title),
					DisplayText: "今日中",
				},
			},
			{
				Action: &messaging_api.PostbackAction{
					Label:       "明日まで",
					Data:        fmt.Sprintf("deadline:tomorrow:%s:%s", userID, title),
					DisplayText: "明日まで",
				},
			},
			{
				Action: &messaging_api.PostbackAction{
					Label:       "今週中",
					Data:        fmt.Sprintf("deadline:this_week:%s:%s", userID, title),
					DisplayText: "今週中",
				},
			},
			{
				Action: &messaging_api.PostbackAction{
					Label:       "今月中",
					Data:        fmt.Sprintf("deadline:this_month:%s:%s", userID, title),
					DisplayText: "今月中",
				},
			},
			{
				Action: &messaging_api.PostbackAction{
					Label:       "期限なし",
					Data:        fmt.Sprintf("deadline:none:%s:%s", userID, title),
					DisplayText: "期限なし",
				},
			},
		},
	}

	message := &messaging_api.TextMessage{
		Text:       "期限はありますか？",
		QuickReply: quickReply,
	}

	_, err := h.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages:   []messaging_api.MessageInterface{message},
		},
	)

	return err
}

func (h *WebhookHandler) handlePostback(replyToken, userID, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "deadline":
		if len(parts) != 4 {
			return nil
		}
		deadlineType := parts[1]
		todoUserID := parts[2]
		title := parts[3]

		return h.createTodoWithDeadline(replyToken, todoUserID, title, deadlineType)

	case "complete":
		if len(parts) != 2 {
			return nil
		}
		todoID := parts[1]
		return h.completeTodo(replyToken, todoID)

	case "delete_all":
		if len(parts) != 3 {
			return nil
		}
		confirmation := parts[1]
		todoUserID := parts[2]
		return h.handleDeleteAllConfirmation(replyToken, todoUserID, confirmation)
	}

	return nil
}

func (h *WebhookHandler) createTodoWithDeadline(replyToken, userID, title, deadlineType string) error {
	ctx := context.Background()
	var dueAt *time.Time

	now := time.Now()
	switch deadlineType {
	case "today":
		endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		dueAt = &endOfDay
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		endOfDay := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, tomorrow.Location())
		dueAt = &endOfDay
	case "this_week":
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		endOfWeek := now.AddDate(0, 0, daysUntilSunday)
		endOfDay := time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(), 23, 59, 59, 0, endOfWeek.Location())
		dueAt = &endOfDay
	case "this_month":
		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		lastOfMonth := firstOfNextMonth.AddDate(0, 0, -1)
		endOfDay := time.Date(lastOfMonth.Year(), lastOfMonth.Month(), lastOfMonth.Day(), 23, 59, 59, 0, lastOfMonth.Location())
		dueAt = &endOfDay
	case "none":
		dueAt = nil
	}

	_, err := h.firestore.CreateTodo(ctx, userID, title, dueAt)
	if err != nil {
		return h.replyMessage(replyToken, "TODOの作成に失敗しました。")
	}

	var deadlineText string
	if dueAt != nil {
		deadlineText = fmt.Sprintf("（期限: %s）", dueAt.Format("2006-01-02 15:04"))
	} else {
		deadlineText = "（期限なし）"
	}

	return h.replyMessage(replyToken, fmt.Sprintf("✅ TODO「%s」を追加しました。%s", title, deadlineText))
}

func (h *WebhookHandler) showTodoList(replyToken, userID string) error {
	ctx := context.Background()
	todos, err := h.firestore.GetIncompleteTodos(ctx, userID)
	if err != nil {
		log.Printf("Failed to get incomplete todos for user %s: %v", userID, err)
		return h.replyMessage(replyToken, "TODO一覧の取得に失敗しました。")
	}

	if len(todos) == 0 {
		return h.replyMessage(replyToken, "未完了のTODOはありません。")
	}

	// 一時的にシンプルなテキストメッセージで応答
	var todoItems []string
	for i, todo := range todos {
		deadlineText := "期限なし"
		if todo.DueAt != nil {
			deadlineText = todo.DueAt.Format("2006-01-02 15:04")
		}
		todoItems = append(todoItems, fmt.Sprintf("%d. %s (%s)", i+1, todo.Title, deadlineText))
	}

	todoText := strings.Join(todoItems, "\n")
	return h.replyMessage(replyToken, fmt.Sprintf("📝 TODO一覧 (%d件)\n\n%s", len(todos), todoText))
}

func (h *WebhookHandler) createTodoListFlexMessage(todos []*services.Todo) *messaging_api.FlexMessage {
	var contents []messaging_api.FlexComponentInterface

	for i, todo := range todos {
		var deadlineText string
		if todo.DueAt != nil {
			deadlineText = todo.DueAt.Format("2006-01-02 15:04")
		} else {
			deadlineText = "期限なし"
		}

		box := &messaging_api.FlexBox{
			Layout: "vertical",
			Contents: []messaging_api.FlexComponentInterface{
				&messaging_api.FlexText{
					Text:   todo.Title,
					Weight: "bold",
					Size:   "md",
				},
				&messaging_api.FlexText{
					Text:  deadlineText,
					Size:  "sm",
					Color: "#999999",
				},
				&messaging_api.FlexButton{
					Action: &messaging_api.PostbackAction{
						Label: "完了",
						Data:  fmt.Sprintf("complete:%s", todo.ID),
					},
					Style: "primary",
					Color: "#1DB446",
				},
			},
			Margin:  "md",
			Spacing: "sm",
		}

		if i > 0 {
			box.PaddingTop = "md"
		}

		contents = append(contents, box)
	}

	return &messaging_api.FlexMessage{
		AltText: "TODO一覧",
		Contents: &messaging_api.FlexBubble{
			Header: &messaging_api.FlexBox{
				Layout: "vertical",
				Contents: []messaging_api.FlexComponentInterface{
					messaging_api.FlexText{
						Text:   "TODO一覧",
						Weight: "bold",
						Size:   "xl",
					},
				},
				PaddingAll: "md",
			},
			Body: &messaging_api.FlexBox{
				Layout:   "vertical",
				Contents: contents,
				Spacing:  "md",
			},
		},
	}
}

func (h *WebhookHandler) completeTodo(replyToken, todoID string) error {
	ctx := context.Background()
	err := h.firestore.CompleteTodo(ctx, todoID)
	if err != nil {
		return h.replyMessage(replyToken, "TODOの完了処理に失敗しました。")
	}

	return h.replyMessage(replyToken, "🎉 TODOを完了しました！")
}

func (h *WebhookHandler) deleteTodoByTitle(replyToken, userID, title string) error {
	ctx := context.Background()
	err := h.firestore.DeleteTodoByTitle(ctx, userID, title)
	if err != nil {
		if strings.Contains(err.Error(), "no todo found") {
			return h.replyMessage(replyToken, fmt.Sprintf("「%s」というTODOが見つかりませんでした。", title))
		}
		log.Printf("Failed to delete todo by title for user %s: %v", userID, err)
		return h.replyMessage(replyToken, "TODOの削除に失敗しました。")
	}

	return h.replyMessage(replyToken, fmt.Sprintf("🗑️ TODO「%s」を削除しました。", title))
}

func (h *WebhookHandler) askDeleteAllConfirmation(replyToken, userID string) error {
	quickReply := &messaging_api.QuickReply{
		Items: []messaging_api.QuickReplyItem{
			{
				Action: &messaging_api.PostbackAction{
					Label:       "はい",
					Data:        fmt.Sprintf("delete_all:yes:%s", userID),
					DisplayText: "はい",
				},
			},
			{
				Action: &messaging_api.PostbackAction{
					Label:       "いいえ",
					Data:        fmt.Sprintf("delete_all:no:%s", userID),
					DisplayText: "いいえ",
				},
			},
		},
	}

	message := &messaging_api.TextMessage{
		Text:       "⚠️ 本当にすべてのTODOを削除しますか？",
		QuickReply: quickReply,
	}

	_, err := h.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages:   []messaging_api.MessageInterface{message},
		},
	)

	return err
}

func (h *WebhookHandler) handleDeleteAllConfirmation(replyToken, userID, confirmation string) error {
	if confirmation == "yes" {
		ctx := context.Background()
		count, err := h.firestore.DeleteAllTodos(ctx, userID)
		if err != nil {
			log.Printf("Failed to delete all todos for user %s: %v", userID, err)
			return h.replyMessage(replyToken, "TODOの全削除に失敗しました。")
		}

		if count == 0 {
			return h.replyMessage(replyToken, "削除するTODOがありませんでした。")
		}

		return h.replyMessage(replyToken, fmt.Sprintf("🗑️ すべてのTODO（%d件）を削除しました。", count))
	} else {
		return h.replyMessage(replyToken, "全削除をキャンセルしました。")
	}
}

func (h *WebhookHandler) showHelp(replyToken string) error {
	helpText := `📝 TODO Bot 使い方

🆕 TODOを追加:
・TODO追加 "<タイトル>"
・追加 "<タイトル>"
・例: 追加 "買い物"

📋 TODO一覧を表示:
・一覧
・TODO一覧

🗑️ TODOを削除:
・TODO削除 "<タイトル>"
・削除 "<タイトル>"
・例: 削除 "買い物"

🗑️ 全TODOを削除:
・TODO全削除
・全削除

❓ ヘルプ表示:
・ヘルプ

💡 その他:
・期限は追加時に選択できます
・TODOの大文字小文字は区別しません（todo、Todo、TODO等すべて可）`

	return h.replyMessage(replyToken, helpText)
}

func (h *WebhookHandler) replyMessage(replyToken, text string) error {
	log.Printf("Sending reply: '%s'", text)

	message := &messaging_api.TextMessage{
		Text: text,
	}

	_, err := h.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages:   []messaging_api.MessageInterface{message},
		},
	)

	if err != nil {
		log.Printf("Failed to send reply message: %v", err)
	} else {
		log.Printf("Reply message sent successfully")
	}

	return err
}
