package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/usecases"
)

func TestHandlerRegistry_LookupReturnsRegisteredHandler(t *testing.T) {
	r := usecases.NewHandlerRegistry()
	called := false
	r.Register(domain.KindNoopEcho, func(ctx context.Context, job *domain.Job) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`{}`), nil
	})

	h, err := r.Lookup(domain.KindNoopEcho)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if _, err := h(context.Background(), &domain.Job{}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !called {
		t.Error("registered handler was not invoked")
	}
}

func TestHandlerRegistry_LookupUnregisteredReturnsErr(t *testing.T) {
	r := usecases.NewHandlerRegistry()
	_, err := r.Lookup(domain.KindNoopEcho)
	if !errors.Is(err, usecases.ErrHandlerNotRegistered) {
		t.Errorf("Lookup() error = %v, want ErrHandlerNotRegistered", err)
	}
}

func TestHandlerRegistry_RegisterPanicsOnInvalidKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Register() with an invalid JobKind should panic")
		}
	}()
	r := usecases.NewHandlerRegistry()
	r.Register(domain.JobKind("not_a_registered_kind"), func(ctx context.Context, job *domain.Job) (json.RawMessage, error) {
		return nil, nil
	})
}
