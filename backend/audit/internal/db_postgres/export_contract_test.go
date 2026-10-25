package db_postgres

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/internal/testing_contract"
	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestExportArtifactContract(t *testing.T) {
	db := pgtest.Require(t)
	a := pgfixtures.SeedTenant(t, db)
	b := pgfixtures.SeedTenant(t, db)
	now := pgtest.Now()
	job, _, err := jobs.NewPostgresModule(db).Repo.Enqueue(context.Background(), jobsports.EnqueueInput{TenantID: a.ID, Kind: "audit_export", Lane: jobsdomain.LaneBulk, Params: []byte(`{}`), MaxAttempts: 1, RunAt: now, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	testing_contract.RunExport(t, &ExportArtifactStore{Pool: db}, a.ID, b.ID, job.ID, now)
	var remaining int
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM audit_export_chunks").Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("orphan chunks=%d %v", remaining, err)
	}
	store := &ExportArtifactStore{Pool: db}
	baseline := db.Stat().AcquiredConns()
	_, err = store.Write(context.Background(), a.ID, job.ID, func(w io.Writer) (ports.ExportCompletion, error) {
		if db.Stat().AcquiredConns() != baseline {
			t.Error("generation holds a database connection")
		}
		_, err := w.Write(make([]byte, 2<<20))
		return ports.ExportCompletion{ExpiresAt: now.Add(time.Hour), BeforePublish: func() error {
			if db.Stat().AcquiredConns() != baseline {
				t.Error("chunk writes hold a database connection")
			}
			return nil
		}}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	reader, _, err := store.Open(context.Background(), a.ID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	if db.Stat().AcquiredConns() != baseline {
		t.Error("a paused download holds a database connection")
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
