package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"libraryms/internal/models"
)

type Handler struct {
	DB *sql.DB
}

func New(db *sql.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "library-management-system",
	})
}

func (h *Handler) ListBooks(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, isbn, title, author, category, status, created_at, updated_at
		FROM books
		ORDER BY id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := make([]models.Book, 0)
	for rows.Next() {
		var book models.Book
		if err := rows.Scan(&book.ID, &book.ISBN, &book.Title, &book.Author, &book.Category, &book.Status, &book.CreatedAt, &book.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, books)
}

func (h *Handler) GetBook(c *gin.Context) {
	id := c.Param("id")
	var book models.Book
	err := h.DB.QueryRow(`
		SELECT id, isbn, title, author, category, status, created_at, updated_at
		FROM books WHERE id = ?
	`, id).Scan(&book.ID, &book.ISBN, &book.Title, &book.Author, &book.Category, &book.Status, &book.CreatedAt, &book.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) CreateBook(c *gin.Context) {
	var input models.BookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ISBN = strings.TrimSpace(input.ISBN)
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.Category = strings.TrimSpace(input.Category)
	if input.ISBN == "" || input.Title == "" || input.Author == "" || input.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "isbn, title, author and category are required"})
		return
	}
	if input.Status == "" {
		input.Status = "available"
	}

	result, err := h.DB.Exec(`
		INSERT INTO books (isbn, title, author, category, status) VALUES (?, ?, ?, ?, ?)
	`, input.ISBN, input.Title, input.Author, input.Category, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	bookID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var book models.Book
	if err := h.DB.QueryRow(`
		SELECT id, isbn, title, author, category, status, created_at, updated_at
		FROM books WHERE id = ?
	`, bookID).Scan(&book.ID, &book.ISBN, &book.Title, &book.Author, &book.Category, &book.Status, &book.CreatedAt, &book.UpdatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, book)
}

func (h *Handler) UpdateBook(c *gin.Context) {
	bookID := c.Param("id")
	var input models.BookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ISBN = strings.TrimSpace(input.ISBN)
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.Category = strings.TrimSpace(input.Category)
	input.Status = strings.TrimSpace(input.Status)
	if input.ISBN == "" || input.Title == "" || input.Author == "" || input.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "isbn, title, author and category are required"})
		return
	}
	if input.Status != "available" && input.Status != "borrowed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be available or borrowed"})
		return
	}

	_, err := h.DB.Exec(`
		UPDATE books
		SET isbn = ?, title = ?, author = ?, category = ?, status = ?, updated_at = NOW()
		WHERE id = ?
	`, input.ISBN, input.Title, input.Author, input.Category, input.Status, bookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var book models.Book
	err = h.DB.QueryRow(`
		SELECT id, isbn, title, author, category, status, created_at, updated_at FROM books WHERE id = ?
	`, bookID).Scan(&book.ID, &book.ISBN, &book.Title, &book.Author, &book.Category, &book.Status, &book.CreatedAt, &book.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, book)
}

func (h *Handler) DeleteBook(c *gin.Context) {
	bookID := c.Param("id")
	result, err := h.DB.Exec(`DELETE FROM books WHERE id = ?`, bookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "book deleted successfully"})
}

func (h *Handler) ListMembers(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id, name, email, phone, created_at, updated_at FROM members ORDER BY id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	members := make([]models.Member, 0)
	for rows.Next() {
		var member models.Member
		if err := rows.Scan(&member.ID, &member.Name, &member.Email, &member.Phone, &member.CreatedAt, &member.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

func (h *Handler) CreateMember(c *gin.Context) {
	var input models.MemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	if input.Name == "" || input.Email == "" || input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email, and phone are required"})
		return
	}

	result, err := h.DB.Exec(`
		INSERT INTO members (name, email, phone) VALUES (?, ?, ?)
	`, input.Name, input.Email, input.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	memberID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var member models.Member
	if err := h.DB.QueryRow(`SELECT id, name, email, phone, created_at, updated_at FROM members WHERE id = ?`, memberID).Scan(&member.ID, &member.Name, &member.Email, &member.Phone, &member.CreatedAt, &member.UpdatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, member)
}

func (h *Handler) BorrowBook(c *gin.Context) {
	var input models.BorrowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.BookID == 0 || input.MemberID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book_id and member_id are required"})
		return
	}

	var book models.Book
	err := h.DB.QueryRow(`SELECT id, status FROM books WHERE id = ?`, input.BookID).Scan(&book.ID, &book.Status)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if book.Status != "available" {
		c.JSON(http.StatusConflict, gin.H{"error": "book is not available for borrowing"})
		return
	}

	var member models.Member
	err = h.DB.QueryRow(`SELECT id FROM members WHERE id = ?`, input.MemberID).Scan(&member.ID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE books
		SET status = 'borrowed', updated_at = NOW()
		WHERE id = ? AND status = 'available'
	`, input.BookID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updatedRows, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if updatedRows == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "book is not available for borrowing"})
		return
	}

	dueAt := time.Now().Add(14 * 24 * time.Hour)
	borrowResult, err := tx.Exec(`
		INSERT INTO borrow_records (book_id, member_id, due_at, status)
		VALUES (?, ?, ?, 'borrowed')
	`, input.BookID, input.MemberID, dueAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	borrowID, err := borrowResult.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": borrowID, "book_id": input.BookID, "member_id": input.MemberID, "due_at": dueAt})
}

func (h *Handler) ReturnBook(c *gin.Context) {
	recordID := c.Param("id")
	var record models.BorrowRecord
	err := h.DB.QueryRow(`SELECT id, book_id, member_id, status FROM borrow_records WHERE id = ? AND returned_at IS NULL`, recordID).Scan(&record.ID, &record.BookID, &record.MemberID, &record.Status)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "active borrow record not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE borrow_records
		SET returned_at = NOW(), status = 'returned'
		WHERE id = ? AND returned_at IS NULL
	`, recordID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updatedRows, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if updatedRows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "active borrow record not found"})
		return
	}

	if _, err := tx.Exec(`UPDATE books SET status = 'available', updated_at = NOW() WHERE id = ?`, record.BookID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "book returned successfully"})
}

func (h *Handler) ListBorrowRecords(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT br.id, br.book_id, br.member_id, br.borrowed_at, br.due_at, br.returned_at, br.status,
			b.title AS book_title, b.isbn,
			m.name AS member_name
		FROM borrow_records br
		LEFT JOIN books b ON b.id = br.book_id
		LEFT JOIN members m ON m.id = br.member_id
		ORDER BY br.borrowed_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	records := make([]models.BorrowRecord, 0)
	for rows.Next() {
		var record models.BorrowRecord
		if err := rows.Scan(&record.ID, &record.BookID, &record.MemberID, &record.BorrowedAt, &record.DueAt, &record.ReturnedAt, &record.Status, &record.BookTitle, &record.ISBN, &record.MemberName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) GetStats(c *gin.Context) {
	var totalBooks, availableBooks, borrowedBooks, totalMembers, activeLoans int64

	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&totalBooks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM books WHERE status = 'available'`).Scan(&availableBooks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM books WHERE status = 'borrowed'`).Scan(&borrowedBooks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM members`).Scan(&totalMembers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM borrow_records WHERE returned_at IS NULL`).Scan(&activeLoans); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_books":     totalBooks,
		"available_books": availableBooks,
		"borrowed_books":  borrowedBooks,
		"total_members":   totalMembers,
		"active_loans":    activeLoans,
	})
}

func (h *Handler) SetupRoutes(router *gin.Engine) {
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api := router.Group("/api")
	api.GET("/health", h.Health)
	api.GET("/books", h.ListBooks)
	api.GET("/books/:id", h.GetBook)
	api.POST("/books", h.CreateBook)
	api.PUT("/books/:id", h.UpdateBook)
	api.DELETE("/books/:id", h.DeleteBook)
	api.GET("/members", h.ListMembers)
	api.POST("/members", h.CreateMember)
	api.GET("/borrow-records", h.ListBorrowRecords)
	api.POST("/borrow", h.BorrowBook)
	api.POST("/returns/:id", h.ReturnBook)
	api.GET("/stats", h.GetStats)

}
