package usecases_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/usecases"
)

//spec:covers REQ-WORKLOADIDENTITY-013: 更新は無効化状態を保ち、正規化と一意性、テナント境界とイベントを守る。
func TestUpdateBindingPreservesStateAndRefusesForeignReferences(t *testing.T) {
	deps := newAdminBindingDeps(t)
	ctx := withTenant("tenant-a")
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps.AdminWorkloadIdentityDeps, validRegisterInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	agent := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-a")
	replacement := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-a")
	foreign := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-b")
	binding, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{SubjectPattern: "old", AgentID: agent}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{SubjectPattern: "reserved", AgentID: agent}, now); err != nil {
		t.Fatal(err)
	}
	disabled, err := usecases.DisableAgentWorkloadBinding(ctx, deps, binding.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	events := []spec.DomainEvent{}
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }
	pattern, target := " new-* ", " "+replacement+" "
	updated, err := usecases.UpdateAgentWorkloadBinding(ctx, deps, binding.ID, usecases.UpdateAgentWorkloadBindingInput{SubjectPattern: &pattern, AgentID: &target}, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != binding.ID || updated.SubjectPattern != "new-*" || updated.AgentID != replacement || updated.Status != "disabled" || !reflect.DeepEqual(updated.DisabledAt, disabled.DisabledAt) || !updated.CreatedAt.Equal(now) {
		t.Fatalf("updated=%+v", updated)
	}
	if len(events) != 1 {
		t.Fatalf("events=%v", events)
	}
	payload, err := spec.MarshalDomainEvent(events[0])
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"tenantId": "tenant-a", "trustBundleId": bundle.ID, "bindingId": binding.ID, "agentId": replacement} {
		if wire[key] != want {
			t.Fatalf("event %s=%v want %s", key, wire[key], want)
		}
	}
	unchanged, err := usecases.UpdateAgentWorkloadBinding(ctx, deps, binding.ID, usecases.UpdateAgentWorkloadBindingInput{}, now.Add(2*time.Hour))
	if err != nil || !reflect.DeepEqual(unchanged, updated) || len(events) != 1 {
		t.Fatalf("empty update=%+v err=%v events=%d", unchanged, err, len(events))
	}
	for _, tc := range []struct {
		pattern, agent string
		want           error
	}{{"reserved", replacement, usecases.ErrBindingSubjectPatternExists}, {"new-*", foreign, usecases.ErrBindingAgentNotFound}, {"", replacement, usecases.ErrBindingSubjectPatternEmpty}, {"[", replacement, usecases.ErrBindingInvalidInput}} {
		_, err := usecases.UpdateAgentWorkloadBinding(ctx, deps, binding.ID, usecases.UpdateAgentWorkloadBindingInput{SubjectPattern: &tc.pattern, AgentID: &tc.agent}, now)
		if !errors.Is(err, tc.want) {
			t.Fatalf("err=%v want %v", err, tc.want)
		}
		saved, err := deps.BindingRepo.FindByID(ctx, "tenant-a", binding.ID)
		if err != nil || !reflect.DeepEqual(saved, updated) || len(events) != 1 {
			t.Fatalf("refusal changed state: %+v err=%v events=%d", saved, err, len(events))
		}
	}
	if _, err := usecases.UpdateAgentWorkloadBinding(withTenant("tenant-b"), deps, binding.ID, usecases.UpdateAgentWorkloadBindingInput{SubjectPattern: &pattern}, now); !errors.Is(err, usecases.ErrBindingNotFound) {
		t.Fatalf("foreign update=%v", err)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-014: 試行は有効な関連付けだけを数え、重なりから自分を除き、保存とイベントを行わない。
func TestPreviewBindingsUsesEnabledTenantLocalPatterns(t *testing.T) {
	deps := newAdminBindingDeps(t)
	ctx := withTenant("tenant-a")
	now := time.Now().UTC()
	bundle, err := usecases.RegisterWorkloadTrustBundle(ctx, deps.AdminWorkloadIdentityDeps, validRegisterInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	agent := seedActiveAgent(ctx, t, deps.AgentRepo, "tenant-a")
	ids := []string{}
	for _, pattern := range []string{"ns/*", "ns/worker", "ns/disabled"} {
		b, err := usecases.CreateAgentWorkloadBinding(ctx, deps, bundle.ID, usecases.CreateAgentWorkloadBindingInput{SubjectPattern: pattern, AgentID: agent}, now)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, b.ID)
	}
	if _, err := usecases.DisableAgentWorkloadBinding(ctx, deps, ids[2], now); err != nil {
		t.Fatal(err)
	}
	before, err := deps.BindingRepo.ListByTrustBundle(ctx, "tenant-a", bundle.ID)
	if err != nil {
		t.Fatal(err)
	}
	deps.Emit = func(spec.DomainEvent) { t.Fatal("preview emitted an event") }
	for _, tc := range []struct {
		subject string
		count   int
	}{{"outside", 0}, {"ns/other", 1}, {"ns/worker", 2}, {"ns/worker/child", 0}, {"ns/disabled", 1}} {
		pattern := "ns/worker"
		result, err := usecases.PreviewAgentWorkloadBindings(ctx, deps, bundle.ID, usecases.PreviewAgentWorkloadBindingsInput{Subject: &tc.subject, SubjectPattern: &pattern, ExcludeBindingID: ids[1]})
		if err != nil || len(result.Matches) != tc.count || len(result.PossibleOverlaps) != 1 || result.PossibleOverlaps[0].ID != ids[0] {
			t.Fatalf("preview=%+v err=%v", result, err)
		}
		if *result.CandidateMatches != (tc.subject == "ns/worker") {
			t.Fatalf("candidate=%v", *result.CandidateMatches)
		}
	}
	subject := "ns/worker"
	if _, err := usecases.PreviewAgentWorkloadBindings(withTenant("tenant-b"), deps, bundle.ID, usecases.PreviewAgentWorkloadBindingsInput{Subject: &subject}); !errors.Is(err, usecases.ErrTrustBundleNotFound) {
		t.Fatalf("foreign preview=%v", err)
	}
	after, err := deps.BindingRepo.ListByTrustBundle(ctx, "tenant-a", bundle.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("preview saved changes: %+v err=%v", after, err)
	}
	// 交換も同じ二件を曖昧な一致として拒否する。
	values := make([]domain.AgentWorkloadBinding, len(before))
	for i, b := range before {
		values[i] = *b
	}
	if _, err := domain.MatchAgent(values, subject); !errors.Is(err, domain.ErrAmbiguousBindingMatch) {
		t.Fatalf("exchange match=%v", err)
	}
}
