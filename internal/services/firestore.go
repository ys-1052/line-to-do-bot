package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/ytakahashi/line-to-do-bot/internal/models"
	"google.golang.org/api/iterator"
)

type Todo = models.Todo

type FirestoreService struct {
	client *firestore.Client
}

func NewFirestoreService(projectID string) (*FirestoreService, error) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %v", err)
	}

	return &FirestoreService{
		client: client,
	}, nil
}

func (fs *FirestoreService) Close() error {
	return fs.client.Close()
}

func (fs *FirestoreService) CreateTodo(ctx context.Context, userID, title string, dueAt *time.Time) (*models.Todo, error) {
	todo := &models.Todo{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		IsDone:    false,
		DueAt:     dueAt,
		CreatedAt: time.Now(),
	}

	_, err := fs.client.Collection("todos").Doc(todo.ID).Set(ctx, todo)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %v", err)
	}

	return todo, nil
}

func (fs *FirestoreService) GetIncompleteTodos(ctx context.Context, userID string) ([]*models.Todo, error) {
	iter := fs.client.Collection("todos").
		Where("userId", "==", userID).
		Where("isDone", "==", false).
		OrderBy("createdAt", firestore.Asc).
		Documents(ctx)

	var todos []*models.Todo
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate todos: %v", err)
		}

		var todo models.Todo
		if err := doc.DataTo(&todo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal todo: %v", err)
		}

		todos = append(todos, &todo)
	}

	return todos, nil
}

func (fs *FirestoreService) CompleteTodo(ctx context.Context, todoID string) error {
	_, err := fs.client.Collection("todos").Doc(todoID).Update(ctx, []firestore.Update{
		{Path: "isDone", Value: true},
	})
	if err != nil {
		return fmt.Errorf("failed to complete todo: %v", err)
	}

	return nil
}

func (fs *FirestoreService) DeleteTodoByTitle(ctx context.Context, userID, title string) error {
	iter := fs.client.Collection("todos").
		Where("userId", "==", userID).
		Where("title", "==", title).
		Where("isDone", "==", false).
		Documents(ctx)

	var deletedCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate todos for deletion: %v", err)
		}

		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete todo: %v", err)
		}
		deletedCount++
	}

	if deletedCount == 0 {
		return fmt.Errorf("no todo found with title: %s", title)
	}

	return nil
}

func (fs *FirestoreService) DeleteAllTodos(ctx context.Context, userID string) (int, error) {
	iter := fs.client.Collection("todos").
		Where("userId", "==", userID).
		Documents(ctx)

	var deletedCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to iterate todos for deletion: %v", err)
		}

		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to delete todo: %v", err)
		}
		deletedCount++
	}

	return deletedCount, nil
}

