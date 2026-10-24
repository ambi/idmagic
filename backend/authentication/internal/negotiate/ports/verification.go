package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
)

type Verified struct {
	Principal domain.Principal
	ReplayID  string
	Reply     []byte
}

// Verifier は鍵、ASN.1、channel binding を呼び出し側へ漏らさない。
type Verifier interface {
	Verify(context.Context, domain.Config, []byte, string) (Verified, error)
}

// ReplayStore は全レプリカで同じ認証応答を一度だけ予約する。
type ReplayStore interface {
	Reserve(context.Context, string, string, time.Time) (bool, error)
}
