package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/password/testing_contract"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		users := usermemory.NewUserRepository()
		user := &userdomain.User{ID: "password-user", PreferredUsername: "password-user", PasswordHash: "hash-before", CreatedAt: now, UpdatedAt: now}
		users.Seed(user)
		history := NewPasswordHistoryRepository()
		return testing_contract.Fixture{History: history, Reset: NewPasswordResetTokenStore(users, history), User: user, Now: now}
	})
}
