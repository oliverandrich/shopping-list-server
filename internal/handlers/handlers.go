package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/oliverandrich/shopping-list-server/internal/auth"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type Server struct {
	DB   *gorm.DB
	Auth *auth.Service
}

func NewServer(db *gorm.DB, jwtSecret []byte, mailer *gomail.Dialer) *Server {
	return &Server{
		DB:   db,
		Auth: auth.NewService(db, jwtSecret, mailer),
	}
}

// Health check endpoint
func (s *Server) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "shopping-list-api",
	})
}

// Auth Handlers
func (s *Server) RequestLogin(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	code, err := s.Auth.CreateMagicLink(req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create login code")
	}

	if err := s.Auth.SendMagicLink(req.Email, code); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to send email")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Login code sent to your email",
	})
}

func (s *Server) VerifyLogin(c echo.Context) error {
	var req models.VerifyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	user, err := s.Auth.VerifyMagicLink(req.Email, req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired code")
	}

	token, err := s.Auth.GenerateJWT(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate token")
	}

	return c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// Item Handlers
func (s *Server) GetItems(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var items []models.ShoppingItem
	result := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&items)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) CreateItem(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req models.CreateItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Tags == "" {
		req.Tags = "[]"
	}

	item := models.ShoppingItem{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      req.Name,
		Completed: false,
		Tags:      req.Tags,
	}

	result := s.DB.Create(&item)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	return c.JSON(http.StatusCreated, item)
}

func (s *Server) UpdateItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	id := c.Param("id")

	var item models.ShoppingItem
	result := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&item)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}

	var req models.CreateItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	item.Name = req.Name
	if req.Tags != "" {
		item.Tags = req.Tags
	}

	result = s.DB.Save(&item)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	return c.JSON(http.StatusOK, item)
}

func (s *Server) ToggleItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	id := c.Param("id")

	var item models.ShoppingItem
	result := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&item)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}

	item.Completed = !item.Completed
	result = s.DB.Save(&item)
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	return c.JSON(http.StatusOK, item)
}

func (s *Server) DeleteItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	id := c.Param("id")

	result := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ShoppingItem{})
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}

	return c.NoContent(http.StatusNoContent)
}