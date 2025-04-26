package main

import (
	"fmt"
	"hospital-middleware/internal/api"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/services"
	"log"
	"os"
)

func main() {
	// Set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Starting Hospital Middleware Service...")

	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Could not load configuration: %v", err)
		os.Exit(1) // Ensure exit on fatal error
	}
	log.Println("Configuration loaded successfully.")

	// 2. Initialize Database Connection
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("FATAL: Could not connect to database: %v", err)
		os.Exit(1)
	}
	log.Println("Database connection pool initialized.")
	// Get DB instance for potential use (though often accessed via database.GetDB())
	// db := database.GetDB()
	// defer func() {
	// 	sqlDB, err := db.DB()
	// 	if err == nil {
	// 		log.Println("Closing database connection...")
	// 		sqlDB.Close()
	// 	}
	// }() // GORM handles connection pooling, explicit closing might not be needed here.

	// 3. Initialize Services (like Auth Service)
	services.InitializeAuthService(cfg)
	log.Println("Services initialized.")

	// 4. Setup Gin Router
	router := api.SetupRouter()
	log.Println("HTTP router setup complete.")

	// 5. Start HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("FATAL: Could not start server: %v", err)
		os.Exit(1)
	}
}
