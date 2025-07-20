// go.mod dependencies needed:
// module shopping-list
// go 1.21
// require (
//     github.com/labstack/echo/v4 v4.11.4
//     github.com/google/uuid v1.6.0
//     gorm.io/gorm v1.25.7
//     gorm.io/driver/sqlite v1.5.4
//     github.com/golang-jwt/jwt/v5 v5.2.0
//     gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
// )

package main

import (
    "crypto/rand"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "gopkg.in/gomail.v2"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// Models
type User struct {
    ID        string    `gorm:"primarykey" json:"id"`
    Email     string    `gorm:"unique;not null" json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

type MagicLink struct {
    Code      string    `gorm:"primarykey" json:"code"`
    Email     string    `gorm:"not null" json:"email"`
    ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
    Used      bool      `gorm:"default:false" json:"used"`
}

type ShoppingItem struct {
    ID        string `gorm:"primarykey" json:"id"`
    UserID    string `gorm:"not null;index" json:"user_id"`
    Name      string `json:"name"`
    Completed bool   `json:"completed" gorm:"default:false"`
    Tags      string `json:"tags" gorm:"default:'[]'"` // JSON string for simplicity
    CreatedAt time.Time `json:"created_at"`
}

// Request/Response types
type LoginRequest struct {
    Email string `json:"email" validate:"required,email"`
}

type VerifyRequest struct {
    Email string `json:"email" validate:"required,email"`
    Code  string `json:"code" validate:"required"`
}

type CreateItemRequest struct {
    Name string `json:"name" validate:"required"`
    Tags string `json:"tags"`
}

type LoginResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

// JWT Claims
type JWTClaims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

type Server struct {
    db        *gorm.DB
    jwtSecret []byte
    mailer    *gomail.Dialer
}

// Generate random 6-digit code
func generateCode() string {
    bytes := make([]byte, 3)
    rand.Read(bytes)
    return fmt.Sprintf("%06d", int(bytes[0])<<16|int(bytes[1])<<8|int(bytes[2]))[:6]
}

// Send magic link email
func (s *Server) sendMagicLink(email, code string) error {
    m := gomail.NewMessage()
    m.SetHeader("From", os.Getenv("SMTP_FROM"))
    m.SetHeader("To", email)
    m.SetHeader("Subject", "Your Shopping List Login Code")
    
    body := fmt.Sprintf(`
Your login code is: %s

This code will expire in 15 minutes.

If you didn't request this, please ignore this email.
    `, code)
    
    m.SetBody("text/plain", body)
    
    return s.mailer.DialAndSend(m)
}

// Auth Handlers
func (s *Server) requestLogin(c echo.Context) error {
    var req LoginRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    // Generate magic link code
    code := generateCode()
    expiresAt := time.Now().Add(15 * time.Minute)

    // Clean up old codes for this email
    s.db.Where("email = ?", req.Email).Delete(&MagicLink{})

    // Store magic link
    magicLink := MagicLink{
        Code:      code,
        Email:     req.Email,
        ExpiresAt: expiresAt,
    }

    if err := s.db.Create(&magicLink).Error; err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to create login code",
        })
    }

    // Send email
    if err := s.sendMagicLink(req.Email, code); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to send email",
        })
    }

    return c.JSON(http.StatusOK, map[string]string{
        "message": "Login code sent to your email",
    })
}

func (s *Server) verifyLogin(c echo.Context) error {
    var req VerifyRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    // Find and validate magic link
    var magicLink MagicLink
    result := s.db.Where("code = ? AND email = ? AND used = false AND expires_at > ?", 
        req.Code, req.Email, time.Now()).First(&magicLink)
    
    if result.Error != nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "Invalid or expired code",
        })
    }

    // Mark code as used
    magicLink.Used = true
    s.db.Save(&magicLink)

    // Find or create user
    var user User
    result = s.db.Where("email = ?", req.Email).First(&user)
    if result.Error != nil {
        // Create new user
        user = User{
            ID:    uuid.New().String(),
            Email: req.Email,
        }
        if err := s.db.Create(&user).Error; err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{
                "error": "Failed to create user",
            })
        }
    }

    // Generate JWT token
    claims := &JWTClaims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30 days
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(s.jwtSecret)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to generate token",
        })
    }

    return c.JSON(http.StatusOK, LoginResponse{
        Token: tokenString,
        User:  user,
    })
}

// JWT Middleware
func (s *Server) jwtMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            authHeader := c.Request().Header.Get("Authorization")
            if authHeader == "" {
                return c.JSON(http.StatusUnauthorized, map[string]string{
                    "error": "Missing authorization header",
                })
            }

            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                return c.JSON(http.StatusUnauthorized, map[string]string{
                    "error": "Invalid authorization format",
                })
            }

            token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
                return s.jwtSecret, nil
            })

            if err != nil || !token.Valid {
                return c.JSON(http.StatusUnauthorized, map[string]string{
                    "error": "Invalid token",
                })
            }

            claims, ok := token.Claims.(*JWTClaims)
            if !ok {
                return c.JSON(http.StatusUnauthorized, map[string]string{
                    "error": "Invalid token claims",
                })
            }

            // Add user info to context
            c.Set("user_id", claims.UserID)
            c.Set("user_email", claims.Email)

            return next(c)
        }
    }
}

// Protected Item Handlers
func (s *Server) getItems(c echo.Context) error {
    userID := c.Get("user_id").(string)
    
    var items []ShoppingItem
    result := s.db.Where("user_id = ?", userID).Find(&items)
    if result.Error != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": result.Error.Error(),
        })
    }
    return c.JSON(http.StatusOK, items)
}

func (s *Server) createItem(c echo.Context) error {
    userID := c.Get("user_id").(string)
    
    var req CreateItemRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    if req.Tags == "" {
        req.Tags = "[]"
    }

    item := ShoppingItem{
        ID:        uuid.New().String(),
        UserID:    userID,
        Name:      req.Name,
        Completed: false,
        Tags:      req.Tags,
    }

    result := s.db.Create(&item)
    if result.Error != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": result.Error.Error(),
        })
    }

    return c.JSON(http.StatusCreated, item)
}

func (s *Server) toggleItem(c echo.Context) error {
    userID := c.Get("user_id").(string)
    id := c.Param("id")
    
    var item ShoppingItem
    result := s.db.Where("id = ? AND user_id = ?", id, userID).First(&item)
    if result.Error != nil {
        return c.JSON(http.StatusNotFound, map[string]string{
            "error": "Item not found",
        })
    }

    item.Completed = !item.Completed
    result = s.db.Save(&item)
    if result.Error != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": result.Error.Error(),
        })
    }

    return c.JSON(http.StatusOK, item)
}

func (s *Server) deleteItem(c echo.Context) error {
    userID := c.Get("user_id").(string)
    id := c.Param("id")
    
    result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&ShoppingItem{})
    if result.Error != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": result.Error.Error(),
        })
    }

    if result.RowsAffected == 0 {
        return c.JSON(http.StatusNotFound, map[string]string{
            "error": "Item not found",
        })
    }

    return c.NoContent(http.StatusNoContent)
}

func (s *Server) updateItem(c echo.Context) error {
    userID := c.Get("user_id").(string)
    id := c.Param("id")
    
    var item ShoppingItem
    result := s.db.Where("id = ? AND user_id = ?", id, userID).First(&item)
    if result.Error != nil {
        return c.JSON(http.StatusNotFound, map[string]string{
            "error": "Item not found",
        })
    }

    var req CreateItemRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    item.Name = req.Name
    if req.Tags != "" {
        item.Tags = req.Tags
    }

    result = s.db.Save(&item)
    if result.Error != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": result.Error.Error(),
        })
    }

    return c.JSON(http.StatusOK, item)
}

// Health check endpoint
func (s *Server) health(c echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{
        "status": "healthy",
        "service": "shopping-list-api",
    })
}

func main() {
    // Initialize database
    db, err := gorm.Open(sqlite.Open("shopping.db"), &gorm.Config{})
    if err != nil {
        panic("Failed to connect to database")
    }

    // Auto-migrate the schema
    db.AutoMigrate(&User{}, &MagicLink{}, &ShoppingItem{})

    // Initialize SMTP
    smtpHost := os.Getenv("SMTP_HOST")
    smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
    smtpUser := os.Getenv("SMTP_USER")
    smtpPass := os.Getenv("SMTP_PASS")
    
    if smtpHost == "" {
        // Default to Gmail SMTP for development
        smtpHost = "smtp.gmail.com"
        smtpPort = 587
    }

    mailer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

    // JWT Secret
    jwtSecret := []byte(os.Getenv("JWT_SECRET"))
    if len(jwtSecret) == 0 {
        jwtSecret = []byte("your-secret-key-change-this-in-production")
    }

    // Initialize server
    server := &Server{
        db:        db,
        jwtSecret: jwtSecret,
        mailer:    mailer,
    }

    // Initialize Echo
    e := echo.New()

    // Middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())

    // Public routes
    api := e.Group("/api/v1")
    api.GET("/health", server.health)
    api.POST("/auth/login", server.requestLogin)
    api.POST("/auth/verify", server.verifyLogin)

    // Protected routes
    protected := api.Group("")
    protected.Use(server.jwtMiddleware())
    protected.GET("/items", server.getItems)
    protected.POST("/items", server.createItem)
    protected.PUT("/items/:id", server.updateItem)
    protected.POST("/items/:id/toggle", server.toggleItem)
    protected.DELETE("/items/:id", server.deleteItem)

    // Start server
    e.Logger.Info("Starting server on :3000")
    e.Start(":3000")
}
