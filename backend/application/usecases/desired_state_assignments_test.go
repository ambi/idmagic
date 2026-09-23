package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// countingAssignments は保存と削除の呼び出し回数を数える。changed=false の経路で
// 保存先へ書き込みが届かないことを、応答ではなく作用の回数で観測するために使う。
type countingAssignments struct {
	*appmemory.ApplicationAssignmentRepository
	saves, deletes int
}

func (r *countingAssignments) Save(ctx context.Context, assignment *domain.ApplicationAssignment) error {
	r.saves++
	return r.ApplicationAssignmentRepository.Save(ctx, assignment)
}

func (r *countingAssignments) Delete(ctx context.Context, tenantID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string) error {
	r.deletes++
	return r.ApplicationAssignmentRepository.Delete(ctx, tenantID, applicationID, subjectType, subjectID)
}

type provisioningCall struct {
	userID  string
	trigger ports.ProvisioningTrigger
}

type recordingProvisioning struct{ calls []provisioningCall }

func (r *recordingProvisioning) NotifyAssignmentMutation(_ context.Context, _, _, userID string, trigger ports.ProvisioningTrigger, _ time.Time) error {
	r.calls = append(r.calls, provisioningCall{userID: userID, trigger: trigger})
	return nil
}

type desiredStateFixture struct {
	ctx          context.Context
	t0           time.Time
	assignments  *countingAssignments
	provisioning *recordingProvisioning
	events       []string
	operations   appusecases.DesiredStateAssignments
}

// newDesiredStateFixture は acme テナントに Application "portal"、User "alice"、Group "engineering" を置く。
// テナントは引数で渡すので、ctx にはテナントを持たせない。
func newDesiredStateFixture(t *testing.T) *desiredStateFixture {
	t.Helper()
	f := &desiredStateFixture{
		ctx:          context.Background(),
		t0:           time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		assignments:  &countingAssignments{ApplicationAssignmentRepository: appmemory.NewApplicationAssignmentRepository()},
		provisioning: &recordingProvisioning{},
	}
	applications := appmemory.NewApplicationRepository()
	if err := applications.Save(f.ctx, &domain.Application{TenantID: "acme", ID: "portal", Name: "Portal", Kind: domain.ApplicationWeblink, Status: domain.ApplicationActive}); err != nil {
		t.Fatalf("seed application: %v", err)
	}
	f.operations = appusecases.DesiredStateAssignments{
		Deps: appusecases.AssignmentDeps{
			Repo: applications, AssignmentRepo: f.assignments,
			SubjectDirectory: subjectDirectoryFake{existing: map[ports.SubjectRef]bool{
				{Type: domain.AssignmentSubjectUser, ID: "alice"}:        true,
				{Type: domain.AssignmentSubjectGroup, ID: "engineering"}: true,
			}},
			Emit:                 func(event spec.DomainEvent) { f.events = append(f.events, event.EventType()) },
			ProvisioningNotifier: f.provisioning,
		},
		ActorUserID: "lifecycle-workflow",
	}
	return f
}

func (f *desiredStateFixture) seed(t *testing.T, assignment domain.ApplicationAssignment) {
	t.Helper()
	if err := f.assignments.ApplicationAssignmentRepository.Save(f.ctx, &assignment); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
}

// row は保存先から subject の割り当てを読み直す。無ければ nil。
func (f *desiredStateFixture) row(t *testing.T, subjectType domain.AssignmentSubjectType, subjectID string) *domain.ApplicationAssignment {
	t.Helper()
	rows, err := f.assignments.ListBySubjects(f.ctx, "acme", []ports.SubjectRef{{Type: subjectType, ID: subjectID}})
	if err != nil {
		t.Fatalf("read assignments: %v", err)
	}
	for _, row := range rows {
		if row.ApplicationID == "portal" {
			return row
		}
	}
	return nil
}

func groupAssignment(at time.Time) domain.ApplicationAssignment {
	return domain.ApplicationAssignment{
		TenantID: "acme", ApplicationID: "portal", SubjectType: domain.AssignmentSubjectGroup, SubjectID: "engineering",
		Visibility: domain.AssignmentVisible, CreatedAt: at, UpdatedAt: at,
	}
}

//spec:covers REQ-APPLICATION-014, EX-APPLICATION-014-01: グループ割り当てを持つ User への AssignApplicationDesiredState が直接割り当てを作り、UnassignApplicationDesiredState が直接割り当てだけを消し、グループ割り当ての行は内容ごと変わらず、グループを介した割り当ての判定が真のまま残ることを固定する。
func TestDesiredStateAssignmentsLeaveTheGroupAssignmentUntouched(t *testing.T) {
	f := newDesiredStateFixture(t)
	f.seed(t, groupAssignment(f.t0))
	before := *f.row(t, domain.AssignmentSubjectGroup, "engineering")
	t1 := f.t0.Add(time.Hour)

	changed, err := f.operations.AssignApplicationDesiredState(f.ctx, "acme", "portal", "alice", domain.AssignmentVisible, t1)
	if err != nil || !changed {
		t.Fatalf("AssignApplicationDesiredState() = %v, %v; want true, nil", changed, err)
	}
	direct := f.row(t, domain.AssignmentSubjectUser, "alice")
	if direct == nil || direct.Visibility != domain.AssignmentVisible || !direct.CreatedAt.Equal(t1) {
		t.Fatalf("direct assignment = %+v, want a visible user assignment created at %v", direct, t1)
	}
	if after := f.row(t, domain.AssignmentSubjectGroup, "engineering"); after == nil || *after != before {
		t.Fatalf("group assignment after assign = %+v, want unchanged %+v", after, before)
	}

	changed, err = f.operations.UnassignApplicationDesiredState(f.ctx, "acme", "portal", "alice", t1)
	if err != nil || !changed {
		t.Fatalf("UnassignApplicationDesiredState() = %v, %v; want true, nil", changed, err)
	}
	if direct := f.row(t, domain.AssignmentSubjectUser, "alice"); direct != nil {
		t.Fatalf("direct assignment after unassign = %+v, want none", direct)
	}
	if after := f.row(t, domain.AssignmentSubjectGroup, "engineering"); after == nil || *after != before {
		t.Fatalf("group assignment after unassign = %+v, want unchanged %+v", after, before)
	}
	assigned, err := appusecases.IsSubjectAssigned(f.ctx, f.assignments, "acme", "portal", []ports.SubjectRef{
		{Type: domain.AssignmentSubjectUser, ID: "alice"}, {Type: domain.AssignmentSubjectGroup, ID: "engineering"},
	})
	if err != nil || !assigned {
		t.Fatalf("IsSubjectAssigned() = %v, %v; want the group assignment to keep alice assigned", assigned, err)
	}
	if want := []string{"ApplicationAssigned", "ApplicationUnassigned"}; !slices.Equal(f.events, want) {
		t.Fatalf("events = %v, want %v", f.events, want)
	}
	wantCalls := []provisioningCall{{"alice", ports.ProvisioningAssignmentAdded}, {"alice", ports.ProvisioningAssignmentRemoved}}
	if !slices.Equal(f.provisioning.calls, wantCalls) {
		t.Fatalf("provisioning calls = %v, want %v", f.provisioning.calls, wantCalls)
	}
}

//spec:covers REQ-APPLICATION-014, EX-APPLICATION-014-02: 指定どおりの visibility の直接割り当てがあるとき AssignApplicationDesiredState が changed=false を返し、保存、ApplicationAssigned、Provisioning への通知のいずれも起こさず、同じ入力を 2 回適用しても作用が 1 回分であることを固定する。
func TestAssignApplicationDesiredStateWithTheSameVisibilityChangesNothing(t *testing.T) {
	f := newDesiredStateFixture(t)
	f.seed(t, groupAssignment(f.t0))

	first, err := f.operations.AssignApplicationDesiredState(f.ctx, "acme", "portal", "alice", domain.AssignmentHidden, f.t0)
	if err != nil || !first {
		t.Fatalf("first AssignApplicationDesiredState() = %v, %v; want true, nil", first, err)
	}
	stored := *f.row(t, domain.AssignmentSubjectUser, "alice")

	second, err := f.operations.AssignApplicationDesiredState(f.ctx, "acme", "portal", "alice", domain.AssignmentHidden, f.t0.Add(time.Hour))
	if err != nil || second {
		t.Fatalf("second AssignApplicationDesiredState() = %v, %v; want false, nil", second, err)
	}
	if f.assignments.saves != 1 {
		t.Fatalf("saves = %d, want 1 for two identical applications", f.assignments.saves)
	}
	if want := []string{"ApplicationAssigned"}; !slices.Equal(f.events, want) {
		t.Fatalf("events = %v, want %v", f.events, want)
	}
	if len(f.provisioning.calls) != 1 {
		t.Fatalf("provisioning calls = %v, want exactly one", f.provisioning.calls)
	}
	if after := f.row(t, domain.AssignmentSubjectUser, "alice"); after == nil || *after != stored {
		t.Fatalf("direct assignment = %+v, want unchanged %+v", after, stored)
	}
}

// visibility だけが異なる直接割り当ては、作成時刻を保って指定どおりに更新する。
// Provisioning が送る属性は visibility に依らないため、通知はしない。
func TestAssignApplicationDesiredStateUpdatesADifferentVisibility(t *testing.T) {
	f := newDesiredStateFixture(t)
	f.seed(t, domain.ApplicationAssignment{
		TenantID: "acme", ApplicationID: "portal", SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: domain.AssignmentVisible, CreatedAt: f.t0, UpdatedAt: f.t0,
	})
	t1 := f.t0.Add(time.Hour)

	changed, err := f.operations.AssignApplicationDesiredState(f.ctx, "acme", "portal", "alice", domain.AssignmentHidden, t1)
	if err != nil || !changed {
		t.Fatalf("AssignApplicationDesiredState() = %v, %v; want true, nil", changed, err)
	}
	direct := f.row(t, domain.AssignmentSubjectUser, "alice")
	if direct == nil || direct.Visibility != domain.AssignmentHidden || !direct.CreatedAt.Equal(f.t0) || !direct.UpdatedAt.Equal(t1) {
		t.Fatalf("direct assignment = %+v, want hidden, created at %v, updated at %v", direct, f.t0, t1)
	}
	if want := []string{"ApplicationAssigned"}; !slices.Equal(f.events, want) {
		t.Fatalf("events = %v, want %v", f.events, want)
	}
	if len(f.provisioning.calls) != 0 {
		t.Fatalf("provisioning calls = %v, want none for a visibility-only change", f.provisioning.calls)
	}
}

// 直接割り当てのない User の解除は、Application の有無を問わず何もせず changed=false で終わる。
func TestUnassignApplicationDesiredStateWithoutADirectAssignmentChangesNothing(t *testing.T) {
	f := newDesiredStateFixture(t)
	f.seed(t, groupAssignment(f.t0))
	before := *f.row(t, domain.AssignmentSubjectGroup, "engineering")

	for _, applicationID := range []string{"portal", "missing"} {
		changed, err := f.operations.UnassignApplicationDesiredState(f.ctx, "acme", applicationID, "alice", f.t0)
		if err != nil || changed {
			t.Fatalf("UnassignApplicationDesiredState(%q) = %v, %v; want false, nil", applicationID, changed, err)
		}
	}
	if f.assignments.deletes != 0 || len(f.events) != 0 || len(f.provisioning.calls) != 0 {
		t.Fatalf("deletes=%d events=%v provisioning=%v, want no effect", f.assignments.deletes, f.events, f.provisioning.calls)
	}
	if after := f.row(t, domain.AssignmentSubjectGroup, "engineering"); after == nil || *after != before {
		t.Fatalf("group assignment = %+v, want unchanged %+v", after, before)
	}
}

// 内部インターフェースでも、引数のテナントの外にある Application と User は受け付けない。
func TestAssignApplicationDesiredStateRefusesIdentifiersOutsideTheTenant(t *testing.T) {
	tests := []struct {
		name                            string
		tenantID, applicationID, userID string
		visibility                      domain.AssignmentVisibility
		wantErr                         error
	}{
		{name: "application of another tenant", tenantID: "globex", applicationID: "portal", userID: "alice", visibility: domain.AssignmentVisible, wantErr: appusecases.ErrApplicationNotFound},
		{name: "user outside the tenant", tenantID: "acme", applicationID: "portal", userID: "mallory", visibility: domain.AssignmentVisible, wantErr: appusecases.ErrSubjectNotFound},
		{name: "unknown visibility", tenantID: "acme", applicationID: "portal", userID: "alice", visibility: "shown", wantErr: appusecases.ErrInvalidVisibility},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newDesiredStateFixture(t)

			changed, err := f.operations.AssignApplicationDesiredState(f.ctx, test.tenantID, test.applicationID, test.userID, test.visibility, f.t0)

			if !errors.Is(err, test.wantErr) || changed {
				t.Fatalf("AssignApplicationDesiredState() = %v, %v; want false, %v", changed, err, test.wantErr)
			}
			if f.assignments.saves != 0 || len(f.events) != 0 {
				t.Fatalf("saves=%d events=%v, want no effect", f.assignments.saves, f.events)
			}
		})
	}
}
