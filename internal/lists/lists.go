package lists

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/oliverandrich/shopping-list-server/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) GetUserLists(userID string) ([]models.ShoppingList, error) {
	var lists []models.ShoppingList
	err := s.DB.Joins("JOIN list_members ON shopping_lists.id = list_members.list_id").
		Where("list_members.user_id = ?", userID).
		Preload("Owner").
		Order("shopping_lists.created_at DESC").
		Find(&lists).Error
	return lists, err
}

func (s *Service) GetListByID(listID, userID string) (*models.ShoppingList, error) {
	var list models.ShoppingList
	err := s.DB.Joins("JOIN list_members ON shopping_lists.id = list_members.list_id").
		Where("shopping_lists.id = ? AND list_members.user_id = ?", listID, userID).
		Preload("Owner").
		First(&list).Error
	if err != nil {
		return nil, errors.New("list not found or access denied")
	}
	return &list, nil
}

func (s *Service) CreateList(userID, name string) (*models.ShoppingList, error) {
	list := models.ShoppingList{
		ID:        uuid.New().String(),
		Name:      name,
		OwnerID:   userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.DB.Create(&list).Error; err != nil {
		return nil, err
	}

	// Add creator as owner
	member := models.ListMember{
		ListID:   list.ID,
		UserID:   userID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}

	if err := s.DB.Create(&member).Error; err != nil {
		return nil, err
	}

	// Load the owner information
	s.DB.Preload("Owner").First(&list, "id = ?", list.ID)

	return &list, nil
}

func (s *Service) UpdateList(listID, userID, name string) (*models.ShoppingList, error) {
	// Check if user is owner
	if !s.IsListOwner(listID, userID) {
		return nil, errors.New("only list owners can update lists")
	}

	var list models.ShoppingList
	if err := s.DB.First(&list, "id = ?", listID).Error; err != nil {
		return nil, errors.New("list not found")
	}

	list.Name = name
	list.UpdatedAt = time.Now()

	if err := s.DB.Save(&list).Error; err != nil {
		return nil, err
	}

	s.DB.Preload("Owner").First(&list, "id = ?", list.ID)
	return &list, nil
}

func (s *Service) DeleteList(listID, userID string) error {
	// Check if user is owner
	if !s.IsListOwner(listID, userID) {
		return errors.New("only list owners can delete lists")
	}

	// Delete list members
	s.DB.Where("list_id = ?", listID).Delete(&models.ListMember{})

	// Delete list items
	s.DB.Where("list_id = ?", listID).Delete(&models.ShoppingItem{})

	// Delete the list
	result := s.DB.Delete(&models.ShoppingList{}, "id = ?", listID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("list not found")
	}

	return nil
}

func (s *Service) GetListMembers(listID, userID string) ([]models.User, error) {
	// Check if user has access to this list
	if !s.HasListAccess(listID, userID) {
		return nil, errors.New("access denied")
	}

	var users []models.User
	err := s.DB.Joins("JOIN list_members ON users.id = list_members.user_id").
		Where("list_members.list_id = ?", listID).
		Find(&users).Error
	return users, err
}

func (s *Service) AddMemberToList(listID, userID, newMemberID string) error {
	// Check if user is owner
	if !s.IsListOwner(listID, userID) {
		return errors.New("only list owners can add members")
	}

	// Check if member already exists
	var existing models.ListMember
	err := s.DB.Where("list_id = ? AND user_id = ?", listID, newMemberID).First(&existing).Error
	if err == nil {
		return errors.New("user is already a member of this list")
	}

	// Add new member
	member := models.ListMember{
		ListID:   listID,
		UserID:   newMemberID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	return s.DB.Create(&member).Error
}

func (s *Service) RemoveMemberFromList(listID, userID, memberID string) error {
	// Check if user is owner or removing themselves
	if !s.IsListOwner(listID, userID) && userID != memberID {
		return errors.New("access denied")
	}

	// Don't allow owner to remove themselves if they're the only owner
	if userID == memberID && s.IsListOwner(listID, userID) {
		var ownerCount int64
		s.DB.Model(&models.ListMember{}).Where("list_id = ? AND role = ?", listID, "owner").Count(&ownerCount)
		if ownerCount <= 1 {
			return errors.New("cannot remove the last owner from the list")
		}
	}

	result := s.DB.Where("list_id = ? AND user_id = ?", listID, memberID).Delete(&models.ListMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}

	return nil
}

func (s *Service) IsListOwner(listID, userID string) bool {
	var member models.ListMember
	err := s.DB.Where("list_id = ? AND user_id = ? AND role = ?", listID, userID, "owner").First(&member).Error
	return err == nil
}

func (s *Service) HasListAccess(listID, userID string) bool {
	var member models.ListMember
	err := s.DB.Where("list_id = ? AND user_id = ?", listID, userID).First(&member).Error
	return err == nil
}

func (s *Service) CreateDefaultListForUser(userID string) (*models.ShoppingList, error) {
	return s.CreateList(userID, "My Shopping List")
}
