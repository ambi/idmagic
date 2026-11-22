package usecases

// 主要ユースケース追跡: REQ-SEEDING-010。

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
)

type testContributor struct {
	plan    seedingdomain.Plan
	applied bool
	fail    bool
}

type partialFailureContributor struct {
	applied  map[string]bool
	failOnce bool
	applyLog []string
}

type fakeSecretResolver map[string]string

func (r fakeSecretResolver) Resolve(reference seedingdomain.SecretReference) (string, error) {
	value := r[reference.Locator]
	if value == "" {
		return "", fmt.Errorf("unavailable")
	}
	return value, nil
}

type serialContributor struct {
	mu      sync.Mutex
	applied bool
	running int
	maxRun  int
}

func (c *serialContributor) Plan(context.Context, seedingdomain.Request) (seedingdomain.Plan, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.applied {
		return seedingdomain.Plan{Operations: []seedingdomain.Operation{{LogicalKey: "test", Kind: seedingdomain.OperationNoop}}}, nil
	}
	return seedingdomain.Plan{Operations: []seedingdomain.Operation{{LogicalKey: "test", Kind: seedingdomain.OperationCreate}}}, nil
}

func (c *serialContributor) Apply(context.Context, seedingdomain.Request) error {
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

func (c *testContributor) Plan(context.Context, seedingdomain.Request) (seedingdomain.Plan, error) {
	if c.applied {
		return seedingdomain.Plan{Operations: []seedingdomain.Operation{{LogicalKey: "test", Kind: seedingdomain.OperationNoop}}}, nil
	}
	return c.plan, nil
}

func (c *testContributor) Apply(context.Context, seedingdomain.Request) error {
	if c.fail {
		c.fail = false
		return fmt.Errorf("injected partial failure")
	}
	c.applied = true
	return nil
}

func (c *partialFailureContributor) Plan(context.Context, seedingdomain.Request) (seedingdomain.Plan, error) {
	operations := make([]seedingdomain.Operation, 0, 2)
	for _, logicalKey := range []string{"completed", "remaining"} {
		kind := seedingdomain.OperationCreate
		if c.applied[logicalKey] {
			kind = seedingdomain.OperationNoop
		}
		operations = append(operations, seedingdomain.Operation{LogicalKey: logicalKey, Kind: kind})
	}
	return seedingdomain.Plan{Operations: operations}, nil
}

func (c *partialFailureContributor) Apply(context.Context, seedingdomain.Request) error {
	for _, logicalKey := range []string{"completed", "remaining"} {
		if c.applied[logicalKey] {
			continue
		}
		c.applied[logicalKey] = true
		c.applyLog = append(c.applyLog, logicalKey)
		if c.failOnce {
			c.failOnce = false
			return fmt.Errorf("injected partial failure")
		}
	}
	return nil
}

// 未完了のキーだけを一度適用して目的の状態へ収束させる。
//
//spec:covers EX-SEEDING-010-01: 一部の論理キーを適用した後に失敗しても、再試行ではそのキーを noop とし、
func TestRunCanBeRetriedAfterApplyFailure(t *testing.T) {
	contributor := &partialFailureContributor{applied: map[string]bool{}, failOnce: true}
	request := seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileBootstrap, Mode: seedingdomain.ModeApply}
	failedPlan, err := Run(context.Background(), request, contributor)
	if err == nil {
		t.Fatal("first Run() error = nil, want injected failure")
	}
	if failedPlan.Count(seedingdomain.OperationCreate) != 2 || !contributor.applied["completed"] || contributor.applied["remaining"] {
		t.Fatalf("failed Run() plan = %+v, applied = %#v; want only completed applied", failedPlan, contributor.applied)
	}
	converged, err := Run(context.Background(), request, contributor)
	if err != nil {
		t.Fatalf("retry Run() error = %v", err)
	}
	if converged.Count(seedingdomain.OperationNoop) != 2 || !reflect.DeepEqual(contributor.applyLog, []string{"completed", "remaining"}) {
		t.Fatalf("retry plan = %+v, apply log = %v; want two no-ops and one apply per logical key", converged, contributor.applyLog)
	}
}

func TestPlanValidatesBeforeProducingPlan(t *testing.T) {
	_, err := Plan(seedingdomain.Request{Environment: seedingdomain.EnvironmentProduction, Profile: seedingdomain.ProfileTest, Mode: seedingdomain.ModeApply})
	if err == nil {
		t.Fatal("Plan() error = nil, want production policy rejection")
	}
}

func TestRunDryRunDoesNotApply(t *testing.T) {
	contributor := &testContributor{plan: seedingdomain.Plan{Operations: []seedingdomain.Operation{{LogicalKey: "test", Kind: seedingdomain.OperationCreate}}}}
	_, err := Run(context.Background(), seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileBootstrap, Mode: seedingdomain.ModeDryRun}, contributor)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if contributor.applied {
		t.Fatal("dry-run applied a contributor")
	}
}

func TestRunAppliesThenConverges(t *testing.T) {
	contributor := &testContributor{plan: seedingdomain.Plan{Operations: []seedingdomain.Operation{{LogicalKey: "test", Kind: seedingdomain.OperationCreate}}}}
	plan, err := Run(context.Background(), seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileBootstrap, Mode: seedingdomain.ModeApply}, contributor)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !contributor.applied || plan.Count(seedingdomain.OperationNoop) != 1 {
		t.Fatalf("Run() = %+v, applied=%v", plan, contributor.applied)
	}
}

func TestRunSerializesConcurrentApplyForSameRequest(t *testing.T) {
	contributor := &serialContributor{}
	request := seedingdomain.Request{Environment: seedingdomain.EnvironmentTest, Profile: seedingdomain.ProfileTest, Mode: seedingdomain.ModeApply}
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

func TestMaterializeManifestValidatesBeforeResolving(t *testing.T) {
	manifest := seedingdomain.Manifest{
		SchemaVersion: seedingdomain.CurrentManifestSchemaVersion,
		Profile:       seedingdomain.ProfileDevelopment,
		Resources: []seedingdomain.Resource{{
			Kind:       seedingdomain.ResourceKindDevelopmentDemo,
			LogicalKey: "development-demo",
			Demo:       &seedingdomain.DevelopmentDemoSeed{ClientID: "demo", Users: []seedingdomain.DemoUserSeed{{ID: "user"}}},
			Secrets: map[string]seedingdomain.SecretReference{
				"user_password": {Provider: seedingdomain.SecretProviderEnv, Locator: "DEMO_USER_PASSWORD", Version: "v1"},
			},
		}},
	}
	request := seedingdomain.Request{Environment: seedingdomain.EnvironmentDevelopment, Profile: seedingdomain.ProfileDevelopment, Mode: seedingdomain.ModeDryRun}
	resolved, err := MaterializeManifest(request, manifest, fakeSecretResolver{"DEMO_USER_PASSWORD": "do-not-print"})
	if err != nil {
		t.Fatalf("MaterializeManifest() error = %v", err)
	}
	if got := resolved.Secret("development-demo", "user_password"); got != "do-not-print" {
		t.Fatalf("resolved secret = %q", got)
	}

	request.Profile = seedingdomain.ProfileBootstrap
	if _, err := MaterializeManifest(request, manifest, fakeSecretResolver{"DEMO_USER_PASSWORD": "do-not-print"}); err == nil {
		t.Fatal("profile mismatch succeeded, want error")
	}
}
