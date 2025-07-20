package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oliverandrich/shopping-list-server/internal/config"
	"github.com/oliverandrich/shopping-list-server/internal/db"
	"github.com/oliverandrich/shopping-list-server/internal/handlers"
	"github.com/oliverandrich/shopping-list-server/internal/setup"
	"gopkg.in/gomail.v2"
)

func main() {
	// Check command line arguments
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		runSetup()
		return
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Check if system needs setup
	setupService := setup.NewService(database)
	isSetup, err := setupService.IsSystemSetup()
	if err != nil {
		log.Fatal("Failed to check system setup:", err)
	}

	if !isSetup {
		// Try to migrate existing data
		if err := setupService.MigrateExistingData(); err != nil {
			log.Fatal("Failed to migrate existing data:", err)
		}

		// Check again if migration completed setup
		isSetup, err = setupService.IsSystemSetup()
		if err != nil {
			log.Fatal("Failed to check system setup after migration:", err)
		}

		if !isSetup {
			fmt.Println("System is not setup. Please run 'shopping-list-server setup' first.")
			os.Exit(1)
		}
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

func runSetup() {
	fmt.Println("Shopping List Server Setup")
	fmt.Println("=========================")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	setupService := setup.NewService(database)

	// Check if already setup
	isSetup, err := setupService.IsSystemSetup()
	if err != nil {
		log.Fatal("Failed to check system setup:", err)
	}

	if isSetup {
		fmt.Println("System is already setup!")
		return
	}

	// Get email from user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter admin email address: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Failed to read input:", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		log.Fatal("Email address is required")
	}

	// Setup system
	user, err := setupService.SetupSystem(email)
	if err != nil {
		log.Fatal("Failed to setup system:", err)
	}

	fmt.Printf("System setup completed!\n")
	fmt.Printf("Admin user created: %s\n", user.Email)
	fmt.Printf("You can now start the server with: shopping-list-server\n")
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

	// Lists
	protected.GET("/lists", server.GetLists)
	protected.POST("/lists", server.CreateList)
	protected.GET("/lists/:id", server.GetList)
	protected.PUT("/lists/:id", server.UpdateList)
	protected.DELETE("/lists/:id", server.DeleteList)
	protected.GET("/lists/:id/members", server.GetListMembers)
	protected.DELETE("/lists/:id/members/:userId", server.RemoveListMember)

	// List Items
	protected.GET("/lists/:id/items", server.GetListItems)
	protected.POST("/lists/:id/items", server.CreateListItem)
	protected.PUT("/lists/:id/items/:itemId", server.UpdateListItem)
	protected.POST("/lists/:id/items/:itemId/toggle", server.ToggleListItem)
	protected.DELETE("/lists/:id/items/:itemId", server.DeleteListItem)

	// Invitations
	protected.POST("/invitations", server.CreateInvitation)
	protected.GET("/invitations", server.GetInvitations)
	protected.DELETE("/invitations/:id", server.RevokeInvitation)
}