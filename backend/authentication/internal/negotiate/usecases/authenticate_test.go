package usecases_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	negotiatedomain "github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	negotiateports "github.com/ambi/idmagic/backend/authentication/internal/negotiate/ports"
	negotiateusecases "github.com/ambi/idmagic/backend/authentication/internal/negotiate/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type verifier struct {
	value negotiateports.Verified
	err   error
}

func (v verifier) Verify(context.Context, negotiatedomain.Config, []byte, string) (negotiateports.Verified, error) {
	return v.value, v.err
}

type users struct{ user *userdomain.User }

func (u users) FindBySub(context.Context, string) (*userdomain.User, error) { return u.user, nil }

type failedReplay struct{}

func (failedReplay) Reserve(context.Context, string, string, time.Time, time.Duration) (bool, error) {
	return false, errors.New("store unavailable")
}

//spec:covers REQ-AUTHENTICATION-042: 共有ストア障害を再送なしと誤認せず、User を成功として返さない。
func TestReplayStoreFailureFailsClosed(t *testing.T) {
	config := negotiatedomain.Config{TenantID: "tenant", Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: "keytab", CertificateFile: "tls", Bindings: []negotiatedomain.Binding{{Principal: "alice", UserID: "id"}}}
	v := verifier{value: negotiateports.Verified{Principal: negotiatedomain.Principal{Realm: "EXAMPLE.COM", Name: "alice"}, ReplayID: "authenticator"}}
	u := users{&userdomain.User{ID: "id", TenantID: "tenant", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}}}
	s, err := negotiateusecases.New([]negotiatedomain.Config{config}, v, failedReplay{}, u)
	if err != nil {
		t.Fatal(err)
	}
	var events []string
	user, _, err := s.Authenticate(context.Background(), "tenant", []byte("ticket"), "", time.Now(), func(event spec.DomainEvent) { events = append(events, event.EventType()) })
	if user != nil || err == nil || len(events) != 1 || events[0] != "NegotiateAuthRejected" {
		t.Fatalf("user=%+v err=%v events=%v", user, err, events)
	}
}

//spec:covers REQ-AUTHENTICATION-042: 別プロセス相当の Service から共有ストアへ同じ authenticator を予約しても一度だけ受理する。
func TestSharedReplayReservationSurvivesServiceReconstruction(t *testing.T) {
	config := negotiatedomain.Config{TenantID: "tenant", Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: "keytab", CertificateFile: "tls", Bindings: []negotiatedomain.Binding{{Principal: "alice", UserID: "id"}}}
	v := verifier{value: negotiateports.Verified{Principal: negotiatedomain.Principal{Realm: "EXAMPLE.COM", Name: "alice"}, ReplayID: "authenticator"}}
	u := users{&userdomain.User{ID: "id", TenantID: "tenant", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}}}
	replay := memory.NewRepositories().Replay
	var events []string
	emit := func(event spec.DomainEvent) { events = append(events, event.EventType()) }
	first, err := negotiateusecases.New([]negotiatedomain.Config{config}, v, replay, u)
	if err != nil {
		t.Fatal(err)
	}
	accepted, _, err := first.Authenticate(context.Background(), "tenant", []byte("token"), "192.0.2.1", time.Now(), emit)
	if err != nil || accepted == nil {
		t.Fatalf("user=%+v err=%v", accepted, err)
	}
	second, err := negotiateusecases.New([]negotiatedomain.Config{config}, v, replay, u)
	if err != nil {
		t.Fatal(err)
	}
	if accepted, _, err := second.Authenticate(context.Background(), "tenant", []byte("token"), "192.0.2.1", time.Now(), emit); !errors.Is(err, negotiateusecases.ErrRejected) || accepted != nil {
		t.Fatalf("replay user=%+v err=%v", accepted, err)
	}
	if len(events) != 1 || events[0] != "NegotiateAuthRejected" {
		t.Fatalf("events=%v", events)
	}
}

//spec:covers REQ-AUTHENTICATION-042: 検証失敗、未登録の名前、別 realm、無効と別テナントの User を拒否し、成功として返さない。
func TestRefusedPrincipalNeverResolvesAUser(t *testing.T) {
	for _, tc := range []struct {
		name, principal, realm, tenant string
		status                         idmdomain.UserStatus
		verifyErr                      error
	}{
		{"unmapped", "bob", "EXAMPLE.COM", "tenant", idmdomain.UserStatusActive, nil},
		{"wrong-realm", "alice", "OTHER.COM", "tenant", idmdomain.UserStatusActive, nil},
		{"disabled", "alice", "EXAMPLE.COM", "tenant", idmdomain.UserStatusDisabled, nil},
		{"cross-tenant", "alice", "EXAMPLE.COM", "other", idmdomain.UserStatusActive, nil},
		{"bad-ticket", "alice", "EXAMPLE.COM", "tenant", idmdomain.UserStatusActive, errors.New("invalid ticket")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := negotiatedomain.Config{TenantID: "tenant", Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: "keytab", CertificateFile: "tls", Bindings: []negotiatedomain.Binding{{Principal: "alice", UserID: "id"}}}
			v := verifier{value: negotiateports.Verified{Principal: negotiatedomain.Principal{Realm: tc.realm, Name: tc.principal}, ReplayID: "authenticator"}, err: tc.verifyErr}
			u := users{&userdomain.User{ID: "id", TenantID: tc.tenant, Lifecycle: userdomain.UserLifecycle{Status: tc.status}}}
			s, err := negotiateusecases.New([]negotiatedomain.Config{config}, v, memory.NewRepositories().Replay, u)
			if err != nil {
				t.Fatal(err)
			}
			var events []string
			user, _, err := s.Authenticate(context.Background(), "tenant", []byte("ticket"), "192.0.2.1", time.Now(), func(event spec.DomainEvent) { events = append(events, event.EventType()) })
			if user != nil || !errors.Is(err, negotiateusecases.ErrRejected) || len(events) != 1 || events[0] != "NegotiateAuthRejected" {
				t.Fatalf("user=%+v err=%v events=%v", user, err, events)
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-042: AES でないチケットの拒否を、一般のチケットの失敗と区別した reason の encryption で監査に残す。
func TestAuthenticateReportsUnsupportedEncryption(t *testing.T) {
	config := negotiatedomain.Config{TenantID: "tenant", Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: "keytab", CertificateFile: "tls", Bindings: []negotiatedomain.Binding{{Principal: "alice", UserID: "id"}}}
	u := users{&userdomain.User{ID: "id", TenantID: "tenant", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}}}
	for _, tc := range []struct {
		name   string
		err    error
		reason string
	}{
		{"rc4", fmt.Errorf("etype 23: %w", negotiateports.ErrUnsupportedEncryption), "encryption"},
		{"other", errors.New("negotiate: AP_REQ validation failed"), "ticket"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := negotiateusecases.New([]negotiatedomain.Config{config}, verifier{err: tc.err}, memory.NewRepositories().Replay, u)
			if err != nil {
				t.Fatal(err)
			}
			var events []spec.DomainEvent
			user, _, err := s.Authenticate(context.Background(), "tenant", []byte("ticket"), "", time.Now(), func(event spec.DomainEvent) { events = append(events, event) })
			if user != nil || !errors.Is(err, negotiateusecases.ErrRejected) || len(events) != 1 {
				t.Fatalf("user=%+v err=%v events=%v", user, err, events)
			}
			rejected, ok := events[0].(*negotiateusecases.Rejected)
			if !ok || rejected.Reason != tc.reason {
				t.Fatalf("event=%+v, want reason %s", events[0], tc.reason)
			}
		})
	}
}
