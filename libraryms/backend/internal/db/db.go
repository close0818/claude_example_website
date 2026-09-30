package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func Open(dsn string) (*sql.DB, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse MySQL DSN: %w", err)
	}
	if config.DBName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	databaseName := config.DBName
	serverConfig := *config
	serverConfig.DBName = ""
	server, err := sql.Open("mysql", serverConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open MySQL server connection: %w", err)
	}
	if err := server.Ping(); err != nil {
		server.Close()
		return nil, fmt.Errorf("connect to MySQL server: %w", err)
	}

	quotedDatabaseName := strings.ReplaceAll(databaseName, "`", "``")
	_, createErr := server.Exec("CREATE DATABASE IF NOT EXISTS `" + quotedDatabaseName + "` CHARACTER SET utf8mb4")
	server.Close()
	if createErr != nil {
		return nil, fmt.Errorf("create database %q: %w", databaseName, createErr)
	}

	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("connect to database %q: %w", databaseName, err)
	}

	schema := []string{
		`
		CREATE TABLE IF NOT EXISTS books (
			id INT AUTO_INCREMENT PRIMARY KEY,
			isbn VARCHAR(50) NOT NULL,
			title VARCHAR(255) NOT NULL,
			author VARCHAR(255) NOT NULL,
			category VARCHAR(100) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'available',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS members (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			phone VARCHAR(50) NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS borrow_records (
			id INT AUTO_INCREMENT PRIMARY KEY,
			book_id INT NOT NULL,
			member_id INT NOT NULL,
			borrowed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			due_at DATETIME NOT NULL,
			returned_at DATETIME NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'borrowed',
			FOREIGN KEY (book_id) REFERENCES books(id),
			FOREIGN KEY (member_id) REFERENCES members(id)
		)`,
	}
	for i, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			database.Close()
			return nil, fmt.Errorf("initialize schema statement %d: %w", i+1, err)
		}
	}

	return database, nil
}
