package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/oliverandrich/shopping-list-server/internal/auth"
	"github.com/oliverandrich/shopping-list-server/internal/invitations"
	"github.com/oliverandrich/shopping-list-server/internal/lists"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type Server struct {
	DB          *gorm.DB
	Auth        *auth.Service
	Lists       *lists.Service
	Invitations *invitations.Service
}

func NewServer(db *gorm.DB, jwtSecret []byte, mailer *gomail.Dialer) *Server {
	return &Server{
		DB:          db,
		Auth:        auth.NewService(db, jwtSecret, mailer),
		Lists:       lists.NewService(db),
		Invitations: invitations.NewService(db, mailer),
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

	user, invitation, err := s.Auth.VerifyMagicLinkWithInvitation(req.Email, req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired code")
	}

	// Handle invitation acceptance if present
	if invitation != nil {
		_, err := s.Invitations.AcceptInvitation(req.Email, invitation.Code)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to accept invitation")
		}

		// For new users with server invitation, create default list
		if invitation.Type == "server" {
			_, err := s.Lists.CreateDefaultListForUser(user.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create default list")
			}
		}

		// For list invitations, add user to the list
		if invitation.Type == "list" && invitation.ListID != nil {
			err := s.Lists.AddMemberToList(*invitation.ListID, invitation.InvitedBy, user.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to add to list")
			}
		}
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

// List Handlers
func (s *Server) GetLists(c echo.Context) error {
	userID := c.Get("user_id").(string)

	lists, err := s.Lists.GetUserLists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, lists)
}

func (s *Server) CreateList(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req models.CreateListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	list, err := s.Lists.CreateList(userID, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, list)
}

func (s *Server) GetList(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	list, err := s.Lists.GetListByID(listID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, list)
}

func (s *Server) UpdateList(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	var req models.UpdateListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	list, err := s.Lists.UpdateList(listID, userID, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	return c.JSON(http.StatusOK, list)
}

func (s *Server) DeleteList(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	err := s.Lists.DeleteList(listID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) GetListMembers(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	members, err := s.Lists.GetListMembers(listID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	return c.JSON(http.StatusOK, members)
}

func (s *Server) RemoveListMember(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")
	memberID := c.Param("userId")

	err := s.Lists.RemoveMemberFromList(listID, userID, memberID)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// List Item Handlers
func (s *Server) GetListItems(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var items []models.ShoppingItem
	err := s.DB.Where("list_id = ?", listID).Order("created_at DESC").Find(&items).Error
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, items)
}

func (s *Server) CreateListItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var req models.CreateItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Tags == "" {
		req.Tags = "[]"
	}

	item := models.ShoppingItem{
		ID:        uuid.New().String(),
		ListID:    listID,
		Name:      req.Name,
		Completed: false,
		Tags:      req.Tags,
	}

	if err := s.DB.Create(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, item)
}

func (s *Server) UpdateListItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")
	itemID := c.Param("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var item models.ShoppingItem
	if err := s.DB.Where("id = ? AND list_id = ?", itemID, listID).First(&item).Error; err != nil {
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

	if err := s.DB.Save(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, item)
}

func (s *Server) ToggleListItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")
	itemID := c.Param("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	var item models.ShoppingItem
	if err := s.DB.Where("id = ? AND list_id = ?", itemID, listID).First(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}

	item.Completed = !item.Completed
	if err := s.DB.Save(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, item)
}

func (s *Server) DeleteListItem(c echo.Context) error {
	userID := c.Get("user_id").(string)
	listID := c.Param("id")
	itemID := c.Param("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	result := s.DB.Where("id = ? AND list_id = ?", itemID, listID).Delete(&models.ShoppingItem{})
	if result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Item not found")
	}

	return c.NoContent(http.StatusNoContent)
}

// Invitation Handlers
func (s *Server) CreateInvitation(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req models.CreateInvitationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	invitation, err := s.Invitations.CreateInvitation(userID, req.Email, req.Type, req.ListID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, invitation)
}

func (s *Server) GetInvitations(c echo.Context) error {
	userID := c.Get("user_id").(string)

	invitations, err := s.Invitations.GetUserInvitations(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, invitations)
}

func (s *Server) RevokeInvitation(c echo.Context) error {
	userID := c.Get("user_id").(string)
	invitationID := c.Param("id")

	err := s.Invitations.RevokeInvitation(invitationID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

