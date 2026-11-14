package oauth2

import (
	"context"
	"errors"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

// PostgresClientDeletionParticipants は、単独のクライアントの削除の Tx に参加する、ほかのモジュールの操作である。
// 組み立て地点が渡し、OAuth2 はほかのモジュールのテーブルへ直接書かない。
type PostgresClientDeletionParticipants struct {
	// DeleteBindingsInTx は、クライアントへの Agent の資格情報の束縛を消し、Agent 本体を残す。
	DeleteBindingsInTx func(ctx context.Context, tx pgx.Tx, tenantID, clientID string) error
	// QuotaInTx は、削除と同じ Tx で使用量を減らす保存先を返す。
	QuotaInTx func(pgx.Tx) tenantports.QuotaRepository
}

// clientDeletionWiring は、組み立て地点が選んだ単独の削除の参加者である。
// メモリ実行の保存先は組み立ての後に差し替わり得るので、削除の操作は使うときの Module から作る。
type clientDeletionWiring struct {
	postgres *postgresClientDeletion
	memory   *memoryClientDeletionParticipants
}

type memoryClientDeletionParticipants struct {
	deleteBindings func(ctx context.Context, tenantID, clientID string) error
	quota          tenantports.QuotaRepository
}

// WithPostgresClientDeletion は、単独のクライアントの削除を一つの PostgreSQL の Tx で確定する Module を返す。
func (m Module) WithPostgresClientDeletion(db postgres.DB, participants PostgresClientDeletionParticipants) Module {
	m.clientDeletion = clientDeletionWiring{postgres: &postgresClientDeletion{db: db, participants: participants}}
	return m
}

// WithMemoryClientDeletion は、メモリ実行の単独のクライアントの削除を組み立てた Module を返す。
// deleteBindings と quota が nil なら、その参加者を省く。
func (m Module) WithMemoryClientDeletion(
	deleteBindings func(ctx context.Context, tenantID, clientID string) error, quota tenantports.QuotaRepository,
) Module {
	m.clientDeletion = clientDeletionWiring{memory: &memoryClientDeletionParticipants{deleteBindings: deleteBindings, quota: quota}}
	return m
}

// StandaloneClientDeletion は、管理 API が使う単独のクライアントの削除を返す。
// 組み立て地点が参加者を設定していなければ nil を返し、管理 API の削除は失敗する。
func (m Module) StandaloneClientDeletion() clientports.ClientDeletion {
	switch {
	case m.clientDeletion.postgres != nil:
		return *m.clientDeletion.postgres
	case m.clientDeletion.memory != nil:
		return memoryClientDeletion{module: m, participants: *m.clientDeletion.memory}
	}
	return nil
}

type postgresClientDeletion struct {
	db           postgres.DB
	participants PostgresClientDeletionParticipants
}

func (d postgresClientDeletion) DeleteStandalone(ctx context.Context, tenantID, clientID string) error {
	if d.participants.DeleteBindingsInTx == nil || d.participants.QuotaInTx == nil {
		return errors.New("client deletion is not wired to its participants")
	}
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db_postgres.New(tx)
	locked, err := q.LockClientForStandaloneDeletion(ctx, db_postgres.LockClientForStandaloneDeletionParams{TenantID: tenantID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return clientports.ErrCatalogClientNotFound
	}
	if err != nil {
		return err
	}
	if err := clientports.StandaloneDeletionRefusal(locked.FirstParty, locked.ApplicationOwned); err != nil {
		return err
	}
	consents, err := deleteClientStateInTx(ctx, q, clientID)
	if err != nil {
		return err
	}
	if err := d.participants.DeleteBindingsInTx(ctx, tx, tenantID, clientID); err != nil {
		return err
	}
	if err := q.DeleteClient(ctx, db_postgres.DeleteClientParams{TenantID: tenantID, ClientID: clientID}); err != nil {
		return err
	}
	if err := releaseStandaloneClientQuota(ctx, d.participants.QuotaInTx(tx), tenantID, consents); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// memoryClientDeletion は、保存先が検証の後に失敗しないメモリ実行で、PostgreSQL と同じ参加者を同じ順に消す。
type memoryClientDeletion struct {
	module       Module
	participants memoryClientDeletionParticipants
}

func (d memoryClientDeletion) DeleteStandalone(ctx context.Context, tenantID, clientID string) error {
	client, err := d.module.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if client == nil {
		return clientports.ErrCatalogClientNotFound
	}
	if err := clientports.StandaloneDeletionRefusal(client.FirstParty, client.ApplicationID != ""); err != nil {
		return err
	}
	consents, err := d.module.deleteApplicationClientState(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if d.participants.deleteBindings != nil {
		if err := d.participants.deleteBindings(ctx, tenantID, clientID); err != nil {
			return err
		}
	}
	if err := d.module.ClientRepo.Delete(ctx, tenantID, clientID); err != nil {
		return err
	}
	if d.participants.quota == nil {
		return nil
	}
	return releaseStandaloneClientQuota(ctx, d.participants.quota, tenantID, consents)
}

func releaseStandaloneClientQuota(ctx context.Context, quota tenantports.QuotaRepository, tenantID string, consents int) error {
	if err := quota.Decrement(ctx, tenantID, tenantdomain.ResourceOAuth2Clients, 1); err != nil {
		return err
	}
	if consents == 0 {
		return nil
	}
	return quota.Decrement(ctx, tenantID, tenantdomain.ResourceConsents, consents)
}
