package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var ErrRejected = errors.New("negotiate authentication rejected")

type Users interface {
	FindBySub(context.Context, string) (*userdomain.User, error)
}

// Service は検証と対応付けを編成し、セッションと HTTP の継続は呼び出し側に残す。
type Service struct {
	configs  map[string]domain.Config
	verifier ports.Verifier
	replay   ports.ReplayStore
	users    Users
}

func New(configs []domain.Config, verifier ports.Verifier, replay ports.ReplayStore, users Users) (*Service, error) {
	if verifier == nil || replay == nil || users == nil {
		return nil, fmt.Errorf("negotiate: verifier, replay store and users are required")
	}
	s := &Service{configs: make(map[string]domain.Config), verifier: verifier, replay: replay, users: users}
	services := make(map[string]bool)
	for _, config := range configs {
		if err := config.Validate(); err != nil {
			return nil, err
		}
		serviceKey := config.Realm + "\x00" + config.ServicePrincipal
		if _, exists := s.configs[config.TenantID]; exists || services[serviceKey] {
			return nil, fmt.Errorf("negotiate: duplicate tenant or service principal")
		}
		services[serviceKey] = true
		config.Bindings = slices.Clone(config.Bindings)
		config.IntranetCIDRs = slices.Clone(config.IntranetCIDRs)
		s.configs[config.TenantID] = config
	}
	return s, nil
}

func (s *Service) Enabled(tenantID string) bool {
	if s == nil {
		return false
	}
	_, exists := s.configs[tenantID]
	return exists
}

func (s *Service) AcceptsHostname(tenantID, hostname string) bool {
	return s.Enabled(tenantID) && strings.EqualFold(s.configs[tenantID].ServicePrincipal, "HTTP/"+hostname)
}

// Automatic は自動試行だけを出口の公開 IP へ限定する。手動の認証の信頼根拠にはしない。
func (s *Service) Automatic(tenantID, clientIP string) bool {
	return s.Enabled(tenantID) && s.configs[tenantID].AllowsIP(clientIP)
}

func (s *Service) Authenticate(ctx context.Context, tenantID string, token []byte, clientIP string, now time.Time, emit func(spec.DomainEvent)) (*userdomain.User, []byte, error) {
	reject := func(reason string) (*userdomain.User, []byte, error) {
		emit(&Rejected{At: now, TenantID: tenantID, Reason: reason})
		return nil, nil, ErrRejected
	}
	if !s.Enabled(tenantID) {
		return reject("policy")
	}
	config := s.configs[tenantID]
	verified, err := s.verifier.Verify(ctx, config, token, clientIP)
	if err != nil || verified.ReplayID == "" {
		return reject("ticket")
	}
	reserved, err := s.replay.Reserve(ctx, tenantID, "krb5:"+verified.ReplayID, now.Add(10*time.Minute))
	if err != nil {
		emit(&Rejected{At: now, TenantID: tenantID, Reason: "replay_store"})
		return nil, nil, fmt.Errorf("negotiate: reserve replay: %w", err)
	}
	if !reserved {
		return reject("replay")
	}
	userID := config.UserID(verified.Principal)
	if userID == "" {
		return reject("mapping")
	}
	user, err := s.users.FindBySub(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || user.TenantID != tenantID || !user.IsActive() {
		return reject("user")
	}
	return user, verified.Reply, nil
}

type Succeeded struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
}

func (e *Succeeded) EventType() string     { return "NegotiateAuthSucceeded" }
func (e *Succeeded) OccurredAt() time.Time { return e.At }

type Rejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Reason   string    `json:"reason"`
}

func (e *Rejected) EventType() string     { return "NegotiateAuthRejected" }
func (e *Rejected) OccurredAt() time.Time { return e.At }
