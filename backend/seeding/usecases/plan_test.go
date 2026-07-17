package usecases

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/seeding/domain"
)

type testContributor struct {
	plan    domain.Plan
	applied bool
	fail    bool
}

type serialContributor struct {
	mu      sync.Mutex
	applied bool
	running int
	maxRun  int
}

func (c *serialContributor) Plan(context.Context, domain.Request) (domain.Plan, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.applied {
		return domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationNoop}}}, nil
	}
	return domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationCreate}}}, nil
}

func (c *serialContributor) Apply(context.Context, domain.Request) error {
	c.mu.Lock()
	c.running++
	if c.running > c.maxRun {
		c.maxRun = c.running
	}
	c.mu.Unlock()
	time.Sleep(10 * time.Millisecond)
	c.mu.Lock()
	c.running--
	c.applied = true
	c.mu.Unlock()
	return nil
}

func (c *testContributor) Plan(context.Context, domain.Request) (domain.Plan, error) {
	if c.applied {
		return domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationNoop}}}, nil
	}
	return c.plan, nil
}

func (c *testContributor) Apply(context.Context, domain.Request) error {
	if c.fail {
		c.fail = false
		return fmt.Errorf("injected partial failure")
	}
	c.applied = true
	return nil
}

func TestRunCanBeRetriedAfterApplyFailure(t *testing.T) {
	contributor := &testContributor{plan: domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationCreate}}}, fail: true}
	request := domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileBootstrap, Mode: domain.ModeApply}
	if _, err := Run(context.Background(), request, contributor); err == nil {
		t.Fatal("first Run() error = nil, want injected failure")
	}
	if _, err := Run(context.Background(), request, contributor); err != nil {
		t.Fatalf("retry Run() error = %v", err)
	}
}

func TestPlanValidatesBeforeProducingPlan(t *testing.T) {
	_, err := Plan(domain.Request{Environment: domain.EnvironmentProduction, Profile: domain.ProfileTest, Mode: domain.ModeApply})
	if err == nil {
		t.Fatal("Plan() error = nil, want production policy rejection")
	}
}

func TestRunDryRunDoesNotApply(t *testing.T) {
	contributor := &testContributor{plan: domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationCreate}}}}
	_, err := Run(context.Background(), domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileBootstrap, Mode: domain.ModeDryRun}, contributor)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if contributor.applied {
		t.Fatal("dry-run applied a contributor")
	}
}

func TestRunAppliesThenConverges(t *testing.T) {
	contributor := &testContributor{plan: domain.Plan{Operations: []domain.Operation{{LogicalKey: "test", Kind: domain.OperationCreate}}}}
	plan, err := Run(context.Background(), domain.Request{Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileBootstrap, Mode: domain.ModeApply}, contributor)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !contributor.applied || plan.Count(domain.OperationNoop) != 1 {
		t.Fatalf("Run() = %+v, applied=%v", plan, contributor.applied)
	}
}

func TestRunSerializesConcurrentApplyForSameRequest(t *testing.T) {
	contributor := &serialContributor{}
	request := domain.Request{Environment: domain.EnvironmentTest, Profile: domain.ProfileTest, Mode: domain.ModeApply}
	errs := make(chan error, 2)
	for range 2 {
		go func() { _, err := Run(context.Background(), request, contributor); errs <- err }()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	if contributor.maxRun != 1 {
		t.Fatalf("concurrent Apply calls = %d, want 1", contributor.maxRun)
	}
}
