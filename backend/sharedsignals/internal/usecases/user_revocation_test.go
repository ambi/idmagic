package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
)

func userRevocation(at time.Time) ssdomain.UserRevocation {
	return ssdomain.UserRevocation{
		TenantID: "tenant-a", UserID: "user-1", Reason: ssdomain.RevocationReasonSessionRevoked,
		Initiator: ssdomain.InitiatingEntityAdmin, At: at,
	}
}

//spec:covers REQ-SHAREDSIGNALS-012: 同じか前の時刻の前進がエポックを変えずイベントも発行しないこと
func TestAdvanceUserRevocationEpochIsMonotonic(t *testing.T) {
	now := time.Now().UTC()
	repo := ssmemory.NewUserRevocationEpochRepository()
	var events []spec.DomainEvent
	sink := func(event spec.DomainEvent) error { events = append(events, event); return nil }
	for _, at := range []time.Time{now, now, now.Add(-time.Hour)} {
		if err := ssusecases.AdvanceUserRevocationEpoch(context.Background(), repo, sink, userRevocation(at)); err != nil {
			t.Fatal(err)
		}
	}
	if len(events) != 1 {
		t.Fatalf("events=%d, want 1", len(events))
	}
	epoch, err := repo.FindByUser(context.Background(), "tenant-a", "user-1")
	if err != nil || !epoch.Epoch.Equal(now) {
		t.Fatalf("epoch=%+v err=%v", epoch, err)
	}
	if epoch.Supersedes(now) || !epoch.Supersedes(now.Add(-time.Nanosecond)) {
		t.Fatal("境界の判定が違う")
	}
	other, err := repo.FindByUser(context.Background(), "tenant-b", "user-1")
	if err != nil || other != nil {
		t.Fatalf("別テナント epoch=%+v err=%v", other, err)
	}
}

//spec:covers REQ-SHAREDSIGNALS-012: 物理削除はエポックを記録せず失効を発行すること
func TestAdvanceUserRevocationEpochRecordsNoRowForDeletion(t *testing.T) {
	now := time.Now().UTC()
	repo := ssmemory.NewUserRevocationEpochRepository()
	var events []spec.DomainEvent
	revocation := userRevocation(now)
	revocation.Reason = ssdomain.RevocationReasonUserDeleted
	if err := ssusecases.AdvanceUserRevocationEpoch(context.Background(), repo, func(event spec.DomainEvent) error { events = append(events, event); return nil }, revocation); err != nil {
		t.Fatal(err)
	}
	if epoch, _ := repo.FindByUser(context.Background(), "tenant-a", "user-1"); epoch != nil || len(events) != 1 {
		t.Fatalf("epoch=%+v events=%v", epoch, events)
	}
}

type failingUserEpochs struct {
	*ssmemory.UserRevocationEpochRepository
	err error
}

func (f failingUserEpochs) Advance(context.Context, ssdomain.UserRevocationEpoch) error { return f.err }

//spec:covers REQ-SHAREDSIGNALS-012: User の停止の前進の失敗を呼び出し元へ返し、投影しないこと
func TestReactorPropagatesUserEpochFailure(t *testing.T) {
	boom := errors.New("down")
	reactor := &ssusecases.AgentRevocationReactor{
		EpochRepo:     ssmemory.NewAgentRevocationEpochRepository(),
		UserEpochRepo: failingUserEpochs{UserRevocationEpochRepository: ssmemory.NewUserRevocationEpochRepository(), err: boom},
	}
	err := reactor.React(context.Background(), &idmdomain.UserDisabled{At: time.Now().UTC(), TenantID: "tenant-a", TargetUserID: "user-1"})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}

//spec:covers REQ-SHAREDSIGNALS-012: 失効を購読するストリームへ User を主体とする配送を作ること
func TestReactorProjectsUserRevocation(t *testing.T) {
	deps, streamRepo, configRepo, deliveryRepo := newProjectorDeps(t)
	seedTransmitStream(t, streamRepo, configRepo, "stream-1", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})
	reactor := ssusecases.NewAgentRevocationReactor(ssusecases.RevocationReactorDeps{
		EpochRepo: ssmemory.NewAgentRevocationEpochRepository(), UserEpochRepo: ssmemory.NewUserRevocationEpochRepository(), Projector: deps,
	})
	if err := reactor.RevokeUser(context.Background(), userRevocation(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	deliveries, err := deliveryRepo.ListByStream(context.Background(), "tenant-a", "stream-1")
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("deliveries=%v err=%v", deliveries, err)
	}
}
