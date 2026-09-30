package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"libraryms/internal/config"
	"libraryms/internal/db"
	"libraryms/internal/handlers"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.MySQLDSN)
	if err != nil {
		log.Fatal("failed to connect to MySQL: ", err)
	}
	defer database.Close()

	router := gin.Default()
	h := handlers.New(database)
	h.SetupRoutes(router)

	log.Printf("server is running on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server: ", err)
	}
}
