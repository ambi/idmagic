package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication/internal/password/testing_contract"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		users := idmanagement.NewMemoryModule().UserRepo
		user := &userdomain.User{ID: "password-user", PreferredUsername: "password-user", PasswordHash: "hash-before", CreatedAt: now, UpdatedAt: now}
		_ = users.Save(context.Background(), user)
		history := NewPasswordHistoryRepository()
		return testing_contract.Fixture{History: history, Reset: NewPasswordResetTokenStore(users, history), User: user, Now: now}
	})
}
