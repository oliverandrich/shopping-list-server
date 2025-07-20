package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oliverandrich/shopping-list-server/internal/config"
	"github.com/oliverandrich/shopping-list-server/internal/db"
	"github.com/oliverandrich/shopping-list-server/internal/handlers"
	"gopkg.in/gomail.v2"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize SMTP mailer
	mailer := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	// Initialize server with handlers
	server := handlers.NewServer(database, cfg.JWTSecret, mailer)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	setupRoutes(e, server)

	// Start server
	log.Printf("Starting server on %s", cfg.ServerPort)
	if err := e.Start(cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(e *echo.Echo, server *handlers.Server) {
	// API v1 group
	api := e.Group("/api/v1")

	// Public routes
	api.GET("/health", server.Health)
	api.POST("/auth/login", server.RequestLogin)
	api.POST("/auth/verify", server.VerifyLogin)

	// Protected routes
	protected := api.Group("")
	protected.Use(server.Auth.JWTMiddleware())
	protected.GET("/items", server.GetItems)
	protected.POST("/items", server.CreateItem)
	protected.PUT("/items/:id", server.UpdateItem)
	protected.POST("/items/:id/toggle", server.ToggleItem)
	protected.DELETE("/items/:id", server.DeleteItem)
}