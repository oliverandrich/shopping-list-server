package setup

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

func (s *Service) IsSystemSetup() (bool, error) {
	var settings models.SystemSettings
	err := s.DB.First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return settings.IsSetup, nil
}

func (s *Service) SetupSystem(email string) (*models.User, error) {
	// Check if system is already setup
	isSetup, err := s.IsSystemSetup()
	if err != nil {
		return nil, err
	}
	if isSetup {
		return nil, errors.New("system is already setup")
	}

	// Create initial admin user
	user := models.User{
		ID:        uuid.New().String(),
		Email:     email,
		InvitedBy: nil, // This is the initial admin
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	// Create default shopping list for the admin
	defaultList := models.ShoppingList{
		ID:        uuid.New().String(),
		Name:      "My Shopping List",
		OwnerID:   user.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.DB.Create(&defaultList).Error; err != nil {
		return nil, err
	}

	// Add admin as owner of the default list
	listMember := models.ListMember{
		ListID:   defaultList.ID,
		UserID:   user.ID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}

	if err := s.DB.Create(&listMember).Error; err != nil {
		return nil, err
	}

	// Mark system as setup
	settings := models.SystemSettings{
		ID:           "system",
		IsSetup:      true,
		SetupAt:      time.Now(),
		InitialAdmin: user.ID,
	}

	if err := s.DB.Create(&settings).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) MigrateExistingData() error {
	// Check if we have existing users without the system being setup
	var userCount int64
	s.DB.Model(&models.User{}).Count(&userCount)

	if userCount == 0 {
		return nil // No existing data to migrate
	}

	// Check if system is already setup
	isSetup, err := s.IsSystemSetup()
	if err != nil {
		return err
	}
	if isSetup {
		return nil // Already migrated
	}

	// Get the first user as the initial admin
	var firstUser models.User
	if err := s.DB.First(&firstUser).Error; err != nil {
		return err
	}

	// Create default lists for all existing users
	var users []models.User
	if err := s.DB.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		// Create default list
		defaultList := models.ShoppingList{
			ID:        uuid.New().String(),
			Name:      "My Shopping List",
			OwnerID:   user.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.DB.Create(&defaultList).Error; err != nil {
			return err
		}

		// Add user as owner
		listMember := models.ListMember{
			ListID:   defaultList.ID,
			UserID:   user.ID,
			Role:     "owner",
			JoinedAt: time.Now(),
		}

		if err := s.DB.Create(&listMember).Error; err != nil {
			return err
		}

		// Update existing items to belong to this list
		if err := s.DB.Model(&models.ShoppingItem{}).
			Where("user_id = ?", user.ID).
			Update("list_id", defaultList.ID).Error; err != nil {
			return err
		}
	}

	// Mark system as setup
	settings := models.SystemSettings{
		ID:           "system",
		IsSetup:      true,
		SetupAt:      time.Now(),
		InitialAdmin: firstUser.ID,
	}

	return s.DB.Create(&settings).Error
}
