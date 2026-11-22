package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/internal/usecases"
)

func TestHandlerRegistry_LookupReturnsRegisteredHandler(t *testing.T) {
	r := jobsusecases.NewHandlerRegistry()
	called := false
	r.Register(jobsdomain.KindNoopEcho, func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`{}`), nil
	})

	h, err := r.Lookup(jobsdomain.KindNoopEcho)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if _, err := h(context.Background(), &jobsdomain.Job{}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !called {
		t.Error("registered handler was not invoked")
	}
}

func TestHandlerRegistry_LookupUnregisteredReturnsErr(t *testing.T) {
	r := jobsusecases.NewHandlerRegistry()
	_, err := r.Lookup(jobsdomain.KindNoopEcho)
	if !errors.Is(err, jobsusecases.ErrHandlerNotRegistered) {
		t.Errorf("Lookup() error = %v, want ErrHandlerNotRegistered", err)
	}
}

// worker はハンドラーの一覧を起動処理の中で登録するので、この panic が起動を失敗させる。
//
//spec:covers EX-JOBS-010-01: レーンを登録していない JobKind のハンドラーを登録しようとすると、起動処理は panic で止まる。
func TestHandlerRegistry_RegisterPanicsOnInvalidKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Register() with an invalid JobKind should panic")
		}
	}()
	r := jobsusecases.NewHandlerRegistry()
	r.Register(jobsdomain.JobKind("not_a_registered_kind"), func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		return nil, nil
	})
}
