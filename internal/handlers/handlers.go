// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
func (s *Server) Health(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "healthy",
		"service": "shopping-list-api",
	})
}

// Auth Handlers
func (s *Server) RequestLogin(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	code, err := s.Auth.CreateMagicLink(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create login code",
		})
	}

	if err := s.Auth.SendMagicLink(req.Email, code); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send email",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login code sent to your email",
	})
}

func (s *Server) VerifyLogin(c *fiber.Ctx) error {
	var req models.VerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, invitation, err := s.Auth.VerifyMagicLinkWithInvitation(req.Email, req.Code)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired code",
		})
	}

	// Handle invitation acceptance if present
	if invitation != nil {
		_, err := s.Invitations.AcceptInvitation(req.Email, invitation.Code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to accept invitation",
			})
		}

		// For new users with server invitation, create default list
		if invitation.Type == "server" {
			_, err := s.Lists.CreateDefaultListForUser(user.ID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create default list",
				})
			}
		}

		// For list invitations, add user to the list
		if invitation.Type == "list" && invitation.ListID != nil {
			err := s.Lists.AddMemberToList(*invitation.ListID, invitation.InvitedBy, user.ID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to add to list",
				})
			}
		}
	}

	token, err := s.Auth.GenerateJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// List Handlers
func (s *Server) GetLists(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	lists, err := s.Lists.GetUserLists(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(lists)
}

func (s *Server) CreateList(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req models.CreateListRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	list, err := s.Lists.CreateList(userID, req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(list)
}

func (s *Server) GetList(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	list, err := s.Lists.GetListByID(listID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(list)
}

func (s *Server) UpdateList(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	var req models.UpdateListRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	list, err := s.Lists.UpdateList(listID, userID, req.Name)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(list)
}

func (s *Server) DeleteList(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	err := s.Lists.DeleteList(listID, userID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) GetListMembers(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	members, err := s.Lists.GetListMembers(listID, userID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(members)
}

func (s *Server) RemoveListMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")
	memberID := c.Params("userId")

	err := s.Lists.RemoveMemberFromList(listID, userID, memberID)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// List Item Handlers
func (s *Server) GetListItems(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var items []models.ShoppingItem
	err := s.DB.Where("list_id = ?", listID).Order("created_at DESC").Find(&items).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(items)
}

func (s *Server) CreateListItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var req models.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(item)
}

func (s *Server) UpdateListItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")
	itemID := c.Params("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var item models.ShoppingItem
	if err := s.DB.Where("id = ? AND list_id = ?", itemID, listID).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Item not found",
		})
	}

	var req models.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	item.Name = req.Name
	if req.Tags != "" {
		item.Tags = req.Tags
	}

	if err := s.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(item)
}

func (s *Server) ToggleListItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")
	itemID := c.Params("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	var item models.ShoppingItem
	if err := s.DB.Where("id = ? AND list_id = ?", itemID, listID).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Item not found",
		})
	}

	item.Completed = !item.Completed
	if err := s.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(item)
}

func (s *Server) DeleteListItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	listID := c.Params("id")
	itemID := c.Params("itemId")

	// Check list access
	if !s.Lists.HasListAccess(listID, userID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	result := s.DB.Where("id = ? AND list_id = ?", itemID, listID).Delete(&models.ShoppingItem{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Item not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Invitation Handlers
func (s *Server) CreateInvitation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req models.CreateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	invitation, err := s.Invitations.CreateInvitation(userID, req.Email, req.Type, req.ListID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(invitation)
}

func (s *Server) GetInvitations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	invitations, err := s.Invitations.GetUserInvitations(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(invitations)
}

func (s *Server) RevokeInvitation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	invitationID := c.Params("id")

	err := s.Invitations.RevokeInvitation(invitationID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
