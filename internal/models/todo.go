package models

import (
	"time"
)

// Todo represents a todo item
type Todo struct {
	ID        string     `firestore:"id" json:"id"`
	UserID    string     `firestore:"userId" json:"userId"`
	Title     string     `firestore:"title" json:"title"`
	IsDone    bool       `firestore:"isDone" json:"isDone"`
	DueAt     *time.Time `firestore:"dueAt,omitempty" json:"dueAt,omitempty"`
	CreatedAt time.Time  `firestore:"createdAt" json:"createdAt"`
}