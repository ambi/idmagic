package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
)

// CredentialFixture は資格情報の保存先と、記録を持てる二人のユーザーを渡す。
// PostgreSQL の user_id は users への外部キーなので、ユーザーは保存先ごとに用意する。
type CredentialFixture struct {
	Repo    ports.WebAuthnCredentialRepository
	Context func(context.Context) context.Context
	Owner   string
	Other   string
	// Now は PostgreSQL の分解能に切り捨てた時刻である。
	Now time.Time
}

type NewCredentialFixture func(t *testing.T) CredentialFixture

// RunCredentialRepository は、保存済みの credential ID を本人と他人のどちらが保存しても
// ErrWebAuthnCredentialExists で拒否し、既存の記録を変えないことを検査する。
// 一意性を保存の制約に置くのは、登録の確定が事前に照会しても同時の確定を防げないためである。
func RunCredentialRepository(t *testing.T, newFixture NewCredentialFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := f.Context(context.Background())
	label := "Owner key"
	original := &domain.WebAuthnCredential{
		CredentialID: "Y3JlZC0x",
		UserID:       f.Owner,
		PublicKey:    "b3duZXIta2V5",
		SignCount:    7,
		Transports:   []string{"usb"},
		Label:        &label,
		CreatedAt:    f.Now,
	}
	if err := f.Repo.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	for name, userID := range map[string]string{"another user": f.Other, "the owner": f.Owner} {
		otherLabel := "Claimed key"
		claim := &domain.WebAuthnCredential{
			CredentialID: original.CredentialID,
			UserID:       userID,
			PublicKey:    "Y2xhaW1lZC1rZXk",
			SignCount:    1000,
			Transports:   []string{"internal"},
			Label:        &otherLabel,
			CreatedAt:    f.Now.Add(time.Minute),
		}
		if err := f.Repo.Save(ctx, claim); !errors.Is(err, ports.ErrWebAuthnCredentialExists) {
			t.Fatalf("Save by %s: err=%v, want ErrWebAuthnCredentialExists", name, err)
		}
		stored, err := f.Repo.FindByCredentialID(ctx, original.CredentialID)
		if err != nil || stored == nil {
			t.Fatalf("after Save by %s: credential=%v err=%v", name, stored, err)
		}
		if stored.UserID != f.Owner || stored.PublicKey != original.PublicKey || stored.SignCount != 7 ||
			stored.Label == nil || *stored.Label != label || stored.LastUsedAt != nil {
			t.Fatalf("Save by %s overwrote the record: %+v", name, stored)
		}
	}
	if claimed, err := f.Repo.ListBySub(ctx, f.Other); err != nil || len(claimed) != 0 {
		t.Fatalf("other user's credentials=%v err=%v, want none", claimed, err)
	}
}
