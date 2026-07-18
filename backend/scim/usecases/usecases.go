package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var ErrNotFound = errors.New("SCIM resource not found")

// ErrDuplicate signals a uniqueness conflict (userName/displayName already
// used within the tenant). Wrapped with errors.Is-compatible context by
// callers; handlers map it to HTTP 409 with scimType "uniqueness".
var ErrDuplicate = errors.New("SCIM resource already exists")

type Usecases struct {
	ScimRepo  ports.ScimRepository
	UserRepo  idmports.UserRepository
	GroupRepo idmports.GroupRepository
	Emit      func(spec.DomainEvent)
}

func NewUsecases(
	scimRepo ports.ScimRepository,
	userRepo idmports.UserRepository,
	groupRepo idmports.GroupRepository,
	emit func(spec.DomainEvent),
) *Usecases {
	return &Usecases{
		ScimRepo:  scimRepo,
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
		Emit:      emit,
	}
}

// Token Management
func (u *Usecases) GenerateToken(ctx context.Context, tenantID, description string, expiryDays int) (string, *ports.ScimToken, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, err
	}
	tokenStr := hex.EncodeToString(rawBytes)

	hash := sha256.Sum256([]byte(tokenStr))
	hashStr := hex.EncodeToString(hash[:])

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, err
	}
	id := hex.EncodeToString(idBytes)

	var expiresAt *time.Time
	if expiryDays > 0 {
		exp := time.Now().AddDate(0, 0, expiryDays)
		expiresAt = &exp
	}

	tok := &ports.ScimToken{
		ID:          id,
		TenantID:    tenantID,
		TokenHash:   hashStr,
		Description: description,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	if err := u.ScimRepo.SaveToken(ctx, tok); err != nil {
		return "", nil, err
	}

	return tokenStr, tok, nil
}

func (u *Usecases) AuthenticateToken(ctx context.Context, tokenStr string) (string, error) {
	hash := sha256.Sum256([]byte(tokenStr))
	hashStr := hex.EncodeToString(hash[:])

	tok, err := u.ScimRepo.FindToken(ctx, hashStr)
	if err != nil {
		return "", err
	}
	if tok == nil {
		return "", errors.New("invalid token")
	}

	if tok.ExpiresAt != nil && tok.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}

	return tok.TenantID, nil
}

func (u *Usecases) ListTokens(ctx context.Context, tenantID string) ([]*ports.ScimToken, error) {
	return u.ScimRepo.ListTokens(ctx, tenantID)
}

func (u *Usecases) RevokeToken(ctx context.Context, tenantID, id string) error {
	return u.ScimRepo.DeleteToken(ctx, tenantID, id)
}

// ListQuery is the normalized input to ListUsers/ListGroups (SCL
// interfaces.ListScimUsers / ListScimGroups). StartIndex/Count are nil when
// the caller omitted the corresponding query parameter; HasCount
// distinguishes an omitted count from an explicit 0.
type ListQuery struct {
	Filter     string
	StartIndex *int
	Count      *int
	HasCount   bool
}

// ListResult is a filtered, paginated SCIM collection page.
type ListResult struct {
	Total        int
	Items        []map[string]any
	StartIndex   int
	ItemsPerPage int
}
