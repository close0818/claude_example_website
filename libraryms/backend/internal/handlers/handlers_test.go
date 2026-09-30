package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func newTestHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() {
		mock.ExpectClose()
		if err := database.Close(); err != nil {
			t.Errorf("close mock database: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectations: %v", err)
		}
	})

	return New(database), mock
}

func performRequest(t *testing.T, method, path, body string, route gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	router := gin.New()
	router.Handle(method, path, route)
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func performRequestWithRoute(t *testing.T, method, routePath, requestPath, body string, route gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	router := gin.New()
	router.Handle(method, routePath, route)
	request := httptest.NewRequest(method, requestPath, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestHealth(t *testing.T) {
	handler, _ := newTestHandler(t)
	response := performRequest(t, http.MethodGet, "/api/health", "", handler.Health)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health response: %s", response.Body.String())
	}
}

func TestCreateBookRejectsMissingFields(t *testing.T) {
	handler, _ := newTestHandler(t)
	response := performRequest(t, http.MethodPost, "/api/books", `{"isbn":"978-1","title":"A title"}`, handler.CreateBook)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateBookTrimsFieldsAndDefaultsStatus(t *testing.T) {
	handler, mock := newTestHandler(t)
	createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)

	mock.ExpectExec("INSERT INTO books").
		WithArgs("978-1", "A title", "An author", "History", "available").
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books WHERE id = \?`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "isbn", "title", "author", "category", "status", "created_at", "updated_at"}).
			AddRow(7, "978-1", "A title", "An author", "History", "available", createdAt, updatedAt))

	response := performRequest(t, http.MethodPost, "/api/books", `{"isbn":" 978-1 ","title":" A title ","author":" An author ","category":" History "}`, handler.CreateBook)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var book struct {
		ID       int    `json:"id"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &book); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if book.ID != 7 || book.Title != "A title" || book.Status != "available" || book.Category != "History" {
		t.Fatalf("unexpected book response: %+v", book)
	}
}

func TestCreateMemberTrimsFields(t *testing.T) {
	handler, mock := newTestHandler(t)
	createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)

	mock.ExpectExec("INSERT INTO members").
		WithArgs("Lee", "lee@example.com", "555-0100").
		WillReturnResult(sqlmock.NewResult(4, 1))
	mock.ExpectQuery(`SELECT id, name, email, phone, created_at, updated_at FROM members WHERE id = \?`).
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "phone", "created_at", "updated_at"}).
			AddRow(4, "Lee", "lee@example.com", "555-0100", createdAt, createdAt))

	response := performRequest(t, http.MethodPost, "/api/members", `{"name":" Lee ","email":" lee@example.com ","phone":" 555-0100 "}`, handler.CreateMember)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"email":"lee@example.com"`) {
		t.Fatalf("member fields were not trimmed: %s", response.Body.String())
	}
}

func TestBorrowBookRejectsUnavailableBook(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT id, status FROM books WHERE id = \?`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(3, "borrowed"))

	response := performRequest(t, http.MethodPost, "/api/borrow", `{"book_id":3,"member_id":2}`, handler.BorrowBook)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestBorrowBookCommitsTransaction(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT id, status FROM books WHERE id = \?`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(3, "available"))
	mock.ExpectQuery(`SELECT id FROM members WHERE id = \?`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE books SET status = 'borrowed'`).
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO borrow_records").
		WithArgs(3, 2, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectCommit()

	response := performRequest(t, http.MethodPost, "/api/borrow", `{"book_id":3,"member_id":2}`, handler.BorrowBook)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var result struct {
		ID       int `json:"id"`
		BookID   int `json:"book_id"`
		MemberID int `json:"member_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.ID != 11 || result.BookID != 3 || result.MemberID != 2 {
		t.Fatalf("unexpected borrow response: %+v", result)
	}
}

func TestBorrowBookRollsBackWhenAtomicUpdateFails(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT id, status FROM books WHERE id = \?`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(3, "available"))
	mock.ExpectQuery(`SELECT id FROM members WHERE id = \?`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE books SET status = 'borrowed'`).
		WithArgs(3).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectRollback()

	response := performRequest(t, http.MethodPost, "/api/borrow", `{"book_id":3,"member_id":2}`, handler.BorrowBook)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func TestBorrowBookReturnsConflictWhenAtomicUpdateAffectsNoRows(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectQuery(`SELECT id, status FROM books WHERE id = \?`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(3, "available"))
	mock.ExpectQuery(`SELECT id FROM members WHERE id = \?`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE books SET status = 'borrowed'`).
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	response := performRequest(t, http.MethodPost, "/api/borrow", `{"book_id":3,"member_id":2}`, handler.BorrowBook)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestReturnBookCommitsTransaction(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, book_id, member_id, status FROM borrow_records WHERE id = ? AND returned_at IS NULL")).
		WithArgs("11").
		WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "member_id", "status"}).AddRow(11, 3, 2, "borrowed"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE borrow_records").
		WithArgs("11").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE books SET status = 'available'").
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	response := performRequestWithRoute(t, http.MethodPost, "/api/returns/:id", "/api/returns/11", "", handler.ReturnBook)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestListBooksReturnsRows(t *testing.T) {
	handler, mock := newTestHandler(t)
	createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books\s+ORDER BY id DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "isbn", "title", "author", "category", "status", "created_at", "updated_at"}).
			AddRow(7, "978-1", "A title", "An author", "History", "available", createdAt, createdAt))

	response := performRequest(t, http.MethodGet, "/api/books", "", handler.ListBooks)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var books []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &books); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(books) != 1 || books[0].ID != 7 || books[0].Title != "A title" {
		t.Fatalf("unexpected books response: %+v", books)
	}
}

func TestGetStatsReturnsCounts(t *testing.T) {
	handler, mock := newTestHandler(t)
	queries := []struct {
		query string
		count int64
	}{
		{"SELECT COUNT(*) FROM books", 8},
		{"SELECT COUNT(*) FROM books WHERE status = 'available'", 5},
		{"SELECT COUNT(*) FROM books WHERE status = 'borrowed'", 3},
		{"SELECT COUNT(*) FROM members", 4},
		{"SELECT COUNT(*) FROM borrow_records WHERE returned_at IS NULL", 2},
	}
	for _, item := range queries {
		mock.ExpectQuery(regexp.QuoteMeta(item.query)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(item.count))
	}

	response := performRequest(t, http.MethodGet, "/api/stats", "", handler.GetStats)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var stats map[string]int64
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if stats["total_books"] != 8 || stats["available_books"] != 5 || stats["borrowed_books"] != 3 || stats["total_members"] != 4 || stats["active_loans"] != 2 {
		t.Fatalf("unexpected stats response: %+v", stats)
	}
}

func TestDeleteBookReturnsNotFoundWhenNoRowsAffected(t *testing.T) {
	handler, mock := newTestHandler(t)
	mock.ExpectExec(`DELETE FROM books WHERE id = \?`).
		WithArgs("99").
		WillReturnResult(sqlmock.NewResult(0, 0))

	response := performRequestWithRoute(t, http.MethodDelete, "/api/books/:id", "/api/books/99", "", handler.DeleteBook)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestGetBookMapsDatabaseErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books WHERE id = \?`).
			WithArgs("42").
			WillReturnError(sql.ErrNoRows)

		response := performRequestWithRoute(t, http.MethodGet, "/api/books/:id", "/api/books/42", "", handler.GetBook)

		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books WHERE id = \?`).
			WithArgs("42").
			WillReturnError(sqlmock.ErrCancelled)

		response := performRequestWithRoute(t, http.MethodGet, "/api/books/:id", "/api/books/42", "", handler.GetBook)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body.String())
		}
	})
}

func TestUpdateBookValidatesInputAndPersistsChanges(t *testing.T) {
	t.Run("missing required field", func(t *testing.T) {
		handler, _ := newTestHandler(t)
		response := performRequestWithRoute(t, http.MethodPut, "/api/books/:id", "/api/books/4", `{"isbn":"978-1","title":"A title","author":"An author","status":"available"}`, handler.UpdateBook)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		handler, _ := newTestHandler(t)
		response := performRequestWithRoute(t, http.MethodPut, "/api/books/:id", "/api/books/4", `{"isbn":"978-1","title":"A title","author":"An author","category":"History","status":"lost"}`, handler.UpdateBook)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("successful update", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		updatedAt := time.Date(2026, time.September, 30, 10, 0, 0, 0, time.UTC)
		mock.ExpectExec("UPDATE books").
			WithArgs("978-1", "A title", "An author", "History", "borrowed", "4").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books WHERE id = \?`).
			WithArgs("4").
			WillReturnRows(sqlmock.NewRows([]string{"id", "isbn", "title", "author", "category", "status", "created_at", "updated_at"}).
				AddRow(4, "978-1", "A title", "An author", "History", "borrowed", updatedAt, updatedAt))

		response := performRequestWithRoute(t, http.MethodPut, "/api/books/:id", "/api/books/4", `{"isbn":" 978-1 ","title":" A title ","author":" An author ","category":" History ","status":"borrowed"}`, handler.UpdateBook)

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
		}
	})
}

func TestListBooksReturnsServerErrorWhenRowsFail(t *testing.T) {
	handler, mock := newTestHandler(t)
	createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books\s+ORDER BY id DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "isbn", "title", "author", "category", "status", "created_at", "updated_at"}).
			AddRow(7, "978-1", "A title", "An author", "History", "available", createdAt, createdAt).
			RowError(0, sqlmock.ErrCancelled))

	response := performRequest(t, http.MethodGet, "/api/books", "", handler.ListBooks)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func TestListMembersHandlesRowsAndIterationErrors(t *testing.T) {
	t.Run("returns member", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
		mock.ExpectQuery("SELECT id, name, email, phone, created_at, updated_at FROM members ORDER BY id DESC").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "phone", "created_at", "updated_at"}).
				AddRow(4, "Lee", "lee@example.com", "555-0100", createdAt, createdAt))

		response := performRequest(t, http.MethodGet, "/api/members", "", handler.ListMembers)

		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"Lee"`) {
			t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("reports iteration error", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
		mock.ExpectQuery("SELECT id, name, email, phone, created_at, updated_at FROM members ORDER BY id DESC").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "phone", "created_at", "updated_at"}).
				AddRow(4, "Lee", "lee@example.com", "555-0100", createdAt, createdAt).
				RowError(0, sqlmock.ErrCancelled))

		response := performRequest(t, http.MethodGet, "/api/members", "", handler.ListMembers)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
		}
	})
}

func TestListBorrowRecordsHandlesRowsAndIterationErrors(t *testing.T) {
	t.Run("returns record", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		borrowedAt := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
		dueAt := borrowedAt.Add(14 * 24 * time.Hour)
		mock.ExpectQuery(`(?s)SELECT br.id, br.book_id, br.member_id, br.borrowed_at, br.due_at, br.returned_at, br.status,\s+b.title AS book_title, b.isbn,\s+m.name AS member_name\s+FROM borrow_records br\s+LEFT JOIN books b ON b.id = br.book_id\s+LEFT JOIN members m ON m.id = br.member_id\s+ORDER BY br.borrowed_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "member_id", "borrowed_at", "due_at", "returned_at", "status", "book_title", "isbn", "member_name"}).
				AddRow(11, 3, 2, borrowedAt, dueAt, nil, "borrowed", "A title", "978-1", "Lee"))

		response := performRequest(t, http.MethodGet, "/api/borrow-records", "", handler.ListBorrowRecords)

		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"book_title":"A title"`) {
			t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("reports iteration error", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		borrowedAt := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
		dueAt := borrowedAt.Add(14 * 24 * time.Hour)
		mock.ExpectQuery(`(?s)SELECT br.id, br.book_id, br.member_id, br.borrowed_at, br.due_at, br.returned_at, br.status,\s+b.title AS book_title, b.isbn,\s+m.name AS member_name\s+FROM borrow_records br\s+LEFT JOIN books b ON b.id = br.book_id\s+LEFT JOIN members m ON m.id = br.member_id\s+ORDER BY br.borrowed_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "member_id", "borrowed_at", "due_at", "returned_at", "status", "book_title", "isbn", "member_name"}).
				AddRow(11, 3, 2, borrowedAt, dueAt, nil, "borrowed", "A title", "978-1", "Lee").
				RowError(0, sqlmock.ErrCancelled))

		response := performRequest(t, http.MethodGet, "/api/borrow-records", "", handler.ListBorrowRecords)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
		}
	})
}

func TestSetupRoutesRegistersEndpointsAndHandlesPreflight(t *testing.T) {
	handler, _ := newTestHandler(t)
	router := gin.New()
	handler.SetupRoutes(router)

	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /api/health", "GET /api/books", "GET /api/books/:id", "POST /api/books",
		"PUT /api/books/:id", "DELETE /api/books/:id", "GET /api/members", "POST /api/members",
		"GET /api/borrow-records", "POST /api/borrow", "POST /api/returns/:id", "GET /api/stats",
	} {
		if !registered[route] {
			t.Errorf("route %q was not registered", route)
		}
	}

	request := httptest.NewRequest(http.MethodOptions, "/api/books", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("unexpected CORS origin: %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestGetBookReturnsBook(t *testing.T) {
	handler, mock := newTestHandler(t)
	createdAt := time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT id, isbn, title, author, category, status, created_at, updated_at\s+FROM books WHERE id = \?`).
		WithArgs("42").
		WillReturnRows(sqlmock.NewRows([]string{"id", "isbn", "title", "author", "category", "status", "created_at", "updated_at"}).
			AddRow(42, "978-42", "A title", "An author", "History", "available", createdAt, createdAt))

	response := performRequestWithRoute(t, http.MethodGet, "/api/books/:id", "/api/books/42", "", handler.GetBook)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"title":"A title"`) {
		t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestGetStatsReturnsServerErrorForEachQueryFailure(t *testing.T) {
	queries := []string{
		"SELECT COUNT(*) FROM books",
		"SELECT COUNT(*) FROM books WHERE status = 'available'",
		"SELECT COUNT(*) FROM books WHERE status = 'borrowed'",
		"SELECT COUNT(*) FROM members",
		"SELECT COUNT(*) FROM borrow_records WHERE returned_at IS NULL",
	}
	for failedIndex, failedQuery := range queries {
		t.Run(failedQuery, func(t *testing.T) {
			handler, mock := newTestHandler(t)
			for index, query := range queries {
				if index == failedIndex {
					mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(sqlmock.ErrCancelled)
					break
				}
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			}

			response := performRequest(t, http.MethodGet, "/api/stats", "", handler.GetStats)

			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
			}
		})
	}
}

func TestReturnBookHandlesLookupAndDuplicateReturnErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, book_id, member_id, status FROM borrow_records WHERE id = ? AND returned_at IS NULL")).
			WithArgs("11").
			WillReturnError(sql.ErrNoRows)

		response := performRequestWithRoute(t, http.MethodPost, "/api/returns/:id", "/api/returns/11", "", handler.ReturnBook)

		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, book_id, member_id, status FROM borrow_records WHERE id = ? AND returned_at IS NULL")).
			WithArgs("11").
			WillReturnError(sqlmock.ErrCancelled)

		response := performRequestWithRoute(t, http.MethodPost, "/api/returns/:id", "/api/returns/11", "", handler.ReturnBook)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("another request already returned it", func(t *testing.T) {
		handler, mock := newTestHandler(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, book_id, member_id, status FROM borrow_records WHERE id = ? AND returned_at IS NULL")).
			WithArgs("11").
			WillReturnRows(sqlmock.NewRows([]string{"id", "book_id", "member_id", "status"}).AddRow(11, 3, 2, "borrowed"))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE borrow_records`).
			WithArgs("11").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		response := performRequestWithRoute(t, http.MethodPost, "/api/returns/:id", "/api/returns/11", "", handler.ReturnBook)

		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})
}
