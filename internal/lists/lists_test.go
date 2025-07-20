// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package lists

import (
	"testing"
	"time"

	"github.com/oliverandrich/shopping-list-server/internal/models"
	"github.com/oliverandrich/shopping-list-server/internal/testutils"
)

func TestNewService(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.DB != db {
		t.Error("Service DB should match provided database")
	}
}

func TestService_CreateList(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create a test user first
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("create valid list", func(t *testing.T) {
		listName := testutils.TestListName()

		list, err := service.CreateList(user.ID, listName)
		if err != nil {
			t.Fatalf("Failed to create list: %v", err)
		}

		if list.Name != listName {
			t.Errorf("Expected list name to be '%s', got '%s'", listName, list.Name)
		}

		if list.OwnerID != user.ID {
			t.Errorf("Expected owner ID to be '%s', got '%s'", user.ID, list.OwnerID)
		}

		if list.ID == "" {
			t.Error("List ID should not be empty")
		}

		// Verify list was saved to database
		var dbList models.ShoppingList
		err = db.Where("id = ?", list.ID).First(&dbList).Error
		if err != nil {
			t.Fatal("List should be saved in database")
		}

		// Verify owner was added as member
		var member models.ListMember
		err = db.Where("list_id = ? AND user_id = ?", list.ID, user.ID).First(&member).Error
		if err != nil {
			t.Fatal("Owner should be added as list member")
		}

		if member.Role != "owner" {
			t.Errorf("Expected owner role to be 'owner', got '%s'", member.Role)
		}
	})

	t.Run("create list with empty name", func(t *testing.T) {
		_, err := service.CreateList(user.ID, "")
		if err == nil {
			t.Error("Expected error when creating list with empty name")
		}
	})

	t.Run("create list with non-existent user", func(t *testing.T) {
		_, err := service.CreateList("non-existent-user", testutils.TestListName())
		if err == nil {
			t.Error("Expected error when creating list with non-existent user")
		}
	})
}

func TestService_GetUserLists(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test users
	user1 := models.User{
		ID:        "user1-id",
		Email:     "user1@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	user2 := models.User{
		ID:        "user2-id",
		Email:     "user2@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user1).Error
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	err = db.Create(&user2).Error
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create lists for user1
	list1, err := service.CreateList(user1.ID, "User1 List 1")
	if err != nil {
		t.Fatalf("Failed to create list1: %v", err)
	}

	list2, err := service.CreateList(user1.ID, "User1 List 2")
	if err != nil {
		t.Fatalf("Failed to create list2: %v", err)
	}

	// Create list for user2
	_, err = service.CreateList(user2.ID, "User2 List")
	if err != nil {
		t.Fatalf("Failed to create user2 list: %v", err)
	}

	t.Run("get lists for user1", func(t *testing.T) {
		lists, err := service.GetUserLists(user1.ID)
		if err != nil {
			t.Fatalf("Failed to get user lists: %v", err)
		}

		if len(lists) != 2 {
			t.Errorf("Expected 2 lists for user1, got %d", len(lists))
		}

		// Verify list IDs are correct
		foundList1, foundList2 := false, false
		for _, list := range lists {
			switch list.ID {
			case list1.ID:
				foundList1 = true
			case list2.ID:
				foundList2 = true
			}
		}

		if !foundList1 {
			t.Error("List1 should be in user1's lists")
		}
		if !foundList2 {
			t.Error("List2 should be in user1's lists")
		}
	})

	t.Run("get lists for user2", func(t *testing.T) {
		lists, err := service.GetUserLists(user2.ID)
		if err != nil {
			t.Fatalf("Failed to get user lists: %v", err)
		}

		if len(lists) != 1 {
			t.Errorf("Expected 1 list for user2, got %d", len(lists))
		}
	})

	t.Run("get lists for non-existent user", func(t *testing.T) {
		lists, err := service.GetUserLists("non-existent-user")
		if err != nil {
			t.Fatalf("Failed to get lists for non-existent user: %v", err)
		}

		if len(lists) != 0 {
			t.Errorf("Expected 0 lists for non-existent user, got %d", len(lists))
		}
	})
}

func TestService_GetListByID(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test user and list
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	list, err := service.CreateList(user.ID, testutils.TestListName())
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	t.Run("get existing list", func(t *testing.T) {
		retrievedList, err := service.GetListByID(list.ID, user.ID)
		if err != nil {
			t.Fatalf("Failed to get list by ID: %v", err)
		}

		if retrievedList.ID != list.ID {
			t.Error("Retrieved list should have correct ID")
		}

		if retrievedList.Name != list.Name {
			t.Error("Retrieved list should have correct name")
		}

		if retrievedList.OwnerID != list.OwnerID {
			t.Error("Retrieved list should have correct owner ID")
		}
	})

	t.Run("get list with wrong user", func(t *testing.T) {
		_, err := service.GetListByID(list.ID, "wrong-user-id")
		if err == nil {
			t.Error("Expected error when getting list with wrong user ID")
		}
	})

	t.Run("get non-existent list", func(t *testing.T) {
		_, err := service.GetListByID("non-existent-list", user.ID)
		if err == nil {
			t.Error("Expected error when getting non-existent list")
		}
	})
}

func TestService_UpdateList(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test user and list
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	list, err := service.CreateList(user.ID, "Original Name")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	t.Run("update list name as owner", func(t *testing.T) {
		newName := "Updated Name"

		updatedList, err := service.UpdateList(list.ID, user.ID, newName)
		if err != nil {
			t.Fatalf("Failed to update list: %v", err)
		}

		if updatedList.Name != newName {
			t.Errorf("Expected updated name to be '%s', got '%s'", newName, updatedList.Name)
		}

		// Verify update was saved to database
		var dbList models.ShoppingList
		err = db.Where("id = ?", list.ID).First(&dbList).Error
		if err != nil {
			t.Fatal("Failed to retrieve updated list from database")
		}

		if dbList.Name != newName {
			t.Error("List name should be updated in database")
		}
	})

	t.Run("update list as non-owner", func(t *testing.T) {
		_, err := service.UpdateList(list.ID, "other-user-id", "Hacked Name")
		if err == nil {
			t.Error("Expected error when updating list as non-owner")
		}
	})

	t.Run("update non-existent list", func(t *testing.T) {
		_, err := service.UpdateList("non-existent-list", user.ID, "New Name")
		if err == nil {
			t.Error("Expected error when updating non-existent list")
		}
	})
}

func TestService_DeleteList(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test user and list
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	list, err := service.CreateList(user.ID, testutils.TestListName())
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	t.Run("delete list as owner", func(t *testing.T) {
		err := service.DeleteList(list.ID, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete list: %v", err)
		}

		// Verify list was deleted from database
		var count int64
		err = db.Model(&models.ShoppingList{}).Where("id = ?", list.ID).Count(&count).Error
		if err != nil {
			t.Fatal("Failed to count lists after deletion")
		}

		if count != 0 {
			t.Error("List should be deleted from database")
		}

		// Verify list members were also deleted
		err = db.Model(&models.ListMember{}).Where("list_id = ?", list.ID).Count(&count).Error
		if err != nil {
			t.Fatal("Failed to count list members after deletion")
		}

		if count != 0 {
			t.Error("List members should be deleted when list is deleted")
		}
	})

	t.Run("delete list as non-owner", func(t *testing.T) {
		// Create another list to test with
		anotherList, err := service.CreateList(user.ID, "Another List")
		if err != nil {
			t.Fatalf("Failed to create another test list: %v", err)
		}

		err = service.DeleteList(anotherList.ID, "other-user-id")
		if err == nil {
			t.Error("Expected error when deleting list as non-owner")
		}
	})
}

func TestService_HasListAccess(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test users
	owner := models.User{
		ID:        "owner-id",
		Email:     "owner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	member := models.User{
		ID:        "member-id",
		Email:     "member@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	outsider := models.User{
		ID:        "outsider-id",
		Email:     "outsider@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = db.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}
	err = db.Create(&outsider).Error
	if err != nil {
		t.Fatalf("Failed to create outsider: %v", err)
	}

	// Create a list
	list, err := service.CreateList(owner.ID, testutils.TestListName())
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Add member to list
	err = service.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to add member to list: %v", err)
	}

	t.Run("owner has access", func(t *testing.T) {
		hasAccess := service.HasListAccess(list.ID, owner.ID)
		if !hasAccess {
			t.Error("Owner should have access to their list")
		}
	})

	t.Run("member has access", func(t *testing.T) {
		hasAccess := service.HasListAccess(list.ID, member.ID)
		if !hasAccess {
			t.Error("Member should have access to list")
		}
	})

	t.Run("outsider has no access", func(t *testing.T) {
		hasAccess := service.HasListAccess(list.ID, outsider.ID)
		if hasAccess {
			t.Error("Outsider should not have access to list")
		}
	})

	t.Run("non-existent list", func(t *testing.T) {
		hasAccess := service.HasListAccess("non-existent-list", owner.ID)
		if hasAccess {
			t.Error("Should not have access to non-existent list")
		}
	})
}

func TestService_AddMemberToList(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test users
	owner := models.User{
		ID:        "owner-id",
		Email:     "owner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	newMember := models.User{
		ID:        "new-member-id",
		Email:     "newmember@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = db.Create(&newMember).Error
	if err != nil {
		t.Fatalf("Failed to create new member: %v", err)
	}

	// Create a list
	list, err := service.CreateList(owner.ID, testutils.TestListName())
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	t.Run("add member as owner", func(t *testing.T) {
		err := service.AddMemberToList(list.ID, owner.ID, newMember.ID)
		if err != nil {
			t.Fatalf("Failed to add member to list: %v", err)
		}

		// Verify member was added
		var member models.ListMember
		err = db.Where("list_id = ? AND user_id = ?", list.ID, newMember.ID).First(&member).Error
		if err != nil {
			t.Fatal("Member should be added to list")
		}

		if member.Role != "member" {
			t.Errorf("Expected role to be 'member', got '%s'", member.Role)
		}
	})

	t.Run("add member as non-owner", func(t *testing.T) {
		// Create another user to test with
		anotherUser := models.User{
			ID:        "another-user-id",
			Email:     "another@example.com",
			JoinedAt:  time.Now(),
			CreatedAt: time.Now(),
		}
		err := db.Create(&anotherUser).Error
		if err != nil {
			t.Fatalf("Failed to create another user: %v", err)
		}

		err = service.AddMemberToList(list.ID, newMember.ID, anotherUser.ID)
		if err == nil {
			t.Error("Expected error when adding member as non-owner")
		}
	})
}

func TestService_CreateDefaultListForUser(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test user
	user := models.User{
		ID:        "test-user-id",
		Email:     testutils.TestEmailAddress(),
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	list, err := service.CreateDefaultListForUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to create default list: %v", err)
	}

	if list.Name != "My Shopping List" {
		t.Errorf("Expected default list name to be 'My Shopping List', got '%s'", list.Name)
	}

	if list.OwnerID != user.ID {
		t.Error("Default list should be owned by the user")
	}
}

func TestService_GetListMembers(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test users
	owner := models.User{
		ID:        "owner-id",
		Email:     "owner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	member := models.User{
		ID:        "member-id",
		Email:     "member@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	outsider := models.User{
		ID:        "outsider-id",
		Email:     "outsider@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = db.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}
	err = db.Create(&outsider).Error
	if err != nil {
		t.Fatalf("Failed to create outsider: %v", err)
	}

	// Create a test list
	list, err := service.CreateList(owner.ID, "Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Add member to the list
	err = service.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to add member to list: %v", err)
	}

	t.Run("get members as owner", func(t *testing.T) {
		members, err := service.GetListMembers(list.ID, owner.ID)
		if err != nil {
			t.Fatalf("Failed to get list members: %v", err)
		}

		if len(members) != 2 { // owner + member
			t.Errorf("Expected 2 members, got %d", len(members))
		}

		// Verify both users are in the list
		foundOwner, foundMember := false, false
		for _, m := range members {
			switch m.ID {
			case owner.ID:
				foundOwner = true
			case member.ID:
				foundMember = true
			}
		}

		if !foundOwner || !foundMember {
			t.Error("Expected to find both owner and member in the list")
		}
	})

	t.Run("get members as member", func(t *testing.T) {
		members, err := service.GetListMembers(list.ID, member.ID)
		if err != nil {
			t.Fatalf("Failed to get list members: %v", err)
		}

		if len(members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(members))
		}
	})

	t.Run("get members as outsider", func(t *testing.T) {
		_, err := service.GetListMembers(list.ID, outsider.ID)
		if err == nil {
			t.Error("Expected error when getting members as outsider")
		}
	})

	t.Run("get members with invalid list ID", func(t *testing.T) {
		_, err := service.GetListMembers("non-existent-list", owner.ID)
		if err == nil {
			t.Error("Expected error when getting members for non-existent list")
		}
	})

	t.Run("get members with empty list ID", func(t *testing.T) {
		_, err := service.GetListMembers("", owner.ID)
		if err == nil {
			t.Error("Expected error when getting members with empty list ID")
		}
	})

	t.Run("get members with empty user ID", func(t *testing.T) {
		_, err := service.GetListMembers(list.ID, "")
		if err == nil {
			t.Error("Expected error when getting members with empty user ID")
		}
	})
}

func TestService_RemoveMemberFromList(t *testing.T) {
	db := testutils.SetupTestDB(t)
	service := NewService(db)

	// Create test users
	owner := models.User{
		ID:        "owner-id",
		Email:     "owner@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	member := models.User{
		ID:        "member-id",
		Email:     "member@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}
	outsider := models.User{
		ID:        "outsider-id",
		Email:     "outsider@example.com",
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := db.Create(&owner).Error
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}
	err = db.Create(&member).Error
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}
	err = db.Create(&outsider).Error
	if err != nil {
		t.Fatalf("Failed to create outsider: %v", err)
	}

	// Create a test list
	list, err := service.CreateList(owner.ID, "Test List")
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Add member to the list
	err = service.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to add member to list: %v", err)
	}

	t.Run("remove member as owner", func(t *testing.T) {
		err := service.RemoveMemberFromList(list.ID, owner.ID, member.ID)
		if err != nil {
			t.Fatalf("Failed to remove member from list: %v", err)
		}

		// Verify member was removed
		members, err := service.GetListMembers(list.ID, owner.ID)
		if err != nil {
			t.Fatalf("Failed to get list members: %v", err)
		}

		if len(members) != 1 { // only owner should remain
			t.Errorf("Expected 1 member after removal, got %d", len(members))
		}

		if members[0].ID != owner.ID {
			t.Error("Only owner should remain in the list")
		}
	})

	// Re-add member for next test
	err = service.AddMemberToList(list.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to re-add member to list: %v", err)
	}

	t.Run("member removes themselves", func(t *testing.T) {
		err := service.RemoveMemberFromList(list.ID, member.ID, member.ID)
		if err != nil {
			t.Fatalf("Failed for member to remove themselves: %v", err)
		}

		// Verify member was removed
		members, err := service.GetListMembers(list.ID, owner.ID)
		if err != nil {
			t.Fatalf("Failed to get list members: %v", err)
		}

		if len(members) != 1 {
			t.Errorf("Expected 1 member after self-removal, got %d", len(members))
		}
	})

	t.Run("outsider tries to remove member", func(t *testing.T) {
		// Re-add member first
		err := service.AddMemberToList(list.ID, owner.ID, member.ID)
		if err != nil {
			t.Fatalf("Failed to re-add member to list: %v", err)
		}

		err = service.RemoveMemberFromList(list.ID, outsider.ID, member.ID)
		if err == nil {
			t.Error("Expected error when outsider tries to remove member")
		}
	})

	t.Run("remove non-existent member", func(t *testing.T) {
		err := service.RemoveMemberFromList(list.ID, owner.ID, "non-existent-member")
		if err == nil {
			t.Error("Expected error when removing non-existent member")
		}
	})

	t.Run("owner tries to remove themselves as last owner", func(t *testing.T) {
		// Remove the regular member first, leaving only the owner
		err := service.RemoveMemberFromList(list.ID, owner.ID, member.ID)
		if err != nil {
			t.Fatalf("Failed to remove member: %v", err)
		}

		// Now try to remove the owner (should fail as they're the last owner)
		err = service.RemoveMemberFromList(list.ID, owner.ID, owner.ID)
		if err == nil {
			t.Error("Expected error when owner tries to remove themselves as last owner")
		}
	})

	t.Run("remove with empty list ID", func(t *testing.T) {
		err := service.RemoveMemberFromList("", owner.ID, member.ID)
		if err == nil {
			t.Error("Expected error when removing member with empty list ID")
		}
	})

	t.Run("remove with empty user ID", func(t *testing.T) {
		err := service.RemoveMemberFromList(list.ID, "", member.ID)
		if err == nil {
			t.Error("Expected error when removing member with empty user ID")
		}
	})

	t.Run("remove with empty member ID", func(t *testing.T) {
		err := service.RemoveMemberFromList(list.ID, owner.ID, "")
		if err == nil {
			t.Error("Expected error when removing member with empty member ID")
		}
	})
}
