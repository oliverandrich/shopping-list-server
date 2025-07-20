// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// Routes
	setupRoutes(app, server)

	// Start server
	log.Printf("Starting server on %s", cfg.ServerPort)
	if err := app.Listen(cfg.ServerPort); err != nil {
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

func setupRoutes(app *fiber.App, server *handlers.Server) {
	// API v1 group
	api := app.Group("/api/v1")

	// Public routes
	api.Get("/health", server.Health)
	api.Post("/auth/login", server.RequestLogin)
	api.Post("/auth/verify", server.VerifyLogin)

	// Protected routes
	protected := api.Group("", server.Auth.JWTMiddleware())

	// Lists
	protected.Get("/lists", server.GetLists)
	protected.Post("/lists", server.CreateList)
	protected.Get("/lists/:id", server.GetList)
	protected.Put("/lists/:id", server.UpdateList)
	protected.Delete("/lists/:id", server.DeleteList)
	protected.Get("/lists/:id/members", server.GetListMembers)
	protected.Delete("/lists/:id/members/:userId", server.RemoveListMember)

	// List Items
	protected.Get("/lists/:id/items", server.GetListItems)
	protected.Post("/lists/:id/items", server.CreateListItem)
	protected.Put("/lists/:id/items/:itemId", server.UpdateListItem)
	protected.Post("/lists/:id/items/:itemId/toggle", server.ToggleListItem)
	protected.Delete("/lists/:id/items/:itemId", server.DeleteListItem)

	// Invitations
	protected.Post("/invitations", server.CreateInvitation)
	protected.Get("/invitations", server.GetInvitations)
	protected.Delete("/invitations/:id", server.RevokeInvitation)
}
