package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/ytakahashi/line-to-do-bot/internal/handlers"
	"github.com/ytakahashi/line-to-do-bot/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable is required")
	}

	channelToken := os.Getenv("LINE_CHANNEL_TOKEN")
	if channelToken == "" {
		log.Fatal("LINE_CHANNEL_TOKEN environment variable is required")
	}

	bot, err := messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		log.Fatalf("Failed to create LINE bot client: %v", err)
	}

	firestoreService, err := services.NewFirestoreService(projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore service: %v", err)
	}
	defer firestoreService.Close()

	webhookHandler := handlers.NewWebhookHandler(bot, firestoreService)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/webhook", webhookHandler.HandleWebhook)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}