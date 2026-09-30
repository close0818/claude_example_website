package models

import "time"

type Book struct {
	ID        int       `json:"id"`
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Member struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BorrowRecord struct {
	ID          int        `json:"id"`
	BookID      int        `json:"book_id"`
	MemberID    int        `json:"member_id"`
	BorrowedAt  time.Time  `json:"borrowed_at"`
	DueAt       time.Time  `json:"due_at"`
	ReturnedAt  *time.Time `json:"returned_at,omitempty"`
	Status      string     `json:"status"`
	BookTitle   string     `json:"book_title"`
	MemberName  string     `json:"member_name"`
	ISBN        string     `json:"isbn"`
}

type BookInput struct {
	ISBN     string `json:"isbn"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

type MemberInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type BorrowInput struct {
	BookID   int `json:"book_id"`
	MemberID int `json:"member_id"`
}
