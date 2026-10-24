// Package usecases は、Directory Connector の一回の同期と登録を編成する。
package usecases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// Syncer は一回の同期を行う。Now と Sleep は時刻と待ちの作用である。
type Syncer struct {
	Directory ports.Directory
	API       ports.IngestionAPI
	Cursors   ports.CursorStore
	Dialect   domain.Dialect
	Scope     domain.Scope
	// Computers は AD のコンピューターの検索の範囲である。nil ならコンピューターを読まない。
	Computers    *domain.Scope
	Mapping      domain.Mapping
	FullInterval time.Duration
	Now          func() time.Time
	Sleep        func(context.Context, time.Duration) error
	Log          *slog.Logger
}

type Result struct {
	Mode    domain.Mode
	RunID   string
	Items   int
	Skipped int
}

// SyncOnce は接続を報告し、LDAP を読んで一つの実行を送り、完了が受理されたら位置を保存する
// (REQ-DIRECTORYCONNECTOR-003〜005)。失敗したら位置を変えずに返し、次の回が同じ位置から読み直す。
func (s Syncer) SyncOnce(ctx context.Context) (Result, error) {
	if err := s.retry(ctx, func() error { return s.API.Heartbeat(ctx) }); err != nil {
		return Result{}, fmt.Errorf("report heartbeat: %w", err)
	}
	session, err := s.Directory.Connect(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("connect to ldap: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			s.Log.Warn("closing the ldap connection failed", "error", err.Error())
		}
	}()

	prev, err := s.Cursors.Load(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("load cursor: %w", err)
	}
	root, err := session.RootInfo(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("read root dse: %w", err)
	}
	plan := domain.PlanRun(s.Dialect, s.Scope, s.Computers, prev, root, s.Now(), s.FullInterval)

	result := Result{Mode: plan.Mode}
	if err := s.retry(ctx, func() (err error) {
		result.RunID, err = s.API.StartRun(ctx, plan.Mode)
		return err
	}); err != nil {
		return result, fmt.Errorf("start %s run: %w", plan.Mode, err)
	}

	skip := func(incomplete *domain.IncompleteEntryError) {
		result.Skipped++
		s.Log.Warn("skipped an ldap entry", "dn", incomplete.DN, "missing_attribute", incomplete.Attribute)
	}
	users := batchSender[domain.Item]{syncer: s, ctx: ctx, runID: result.RunID, prefix: result.RunID + "-", send: s.API.SendBatch}
	err = readItems(ctx, session, s.Dialect, s.Scope.BaseDNs, plan.Filter, s.Mapping, users.add, skip)
	if err == nil {
		err = users.flush()
	}
	computers := batchSender[domain.ComputerItem]{
		syncer: s, ctx: ctx, runID: result.RunID, prefix: result.RunID + "-computers-", send: s.API.SendComputerBatch,
	}
	if err == nil && s.Computers != nil {
		if err = readComputers(ctx, session, s.Computers.BaseDNs, plan.ComputerFilter, computers.add, skip); err == nil {
			err = computers.flush()
		}
	}
	result.Items = users.sent + computers.sent
	if err != nil {
		return result, fmt.Errorf("send run %s: %w", result.RunID, err)
	}

	if err := s.retry(ctx, func() error { return s.API.CompleteRun(ctx, result.RunID, result.Items) }); err != nil {
		return result, fmt.Errorf("complete run %s: %w", result.RunID, err)
	}
	if err := s.Cursors.Save(ctx, plan.Next); err != nil {
		return result, fmt.Errorf("save cursor: %w", err)
	}
	s.Log.Info("completed a sync", "mode", plan.Mode, "run_id", result.RunID, "items", result.Items, "skipped", result.Skipped)
	return result, nil
}

// readItems は起点を並べた順に検索し、写せたエントリを emit へ、写せないエントリを skip へ渡す
// (REQ-DIRECTORYCONNECTOR-004)。
func readItems(ctx context.Context, session ports.DirectorySession, dialect domain.Dialect, baseDNs []string, filter string,
	mapping domain.Mapping, emit func(domain.Item) error, skip func(*domain.IncompleteEntryError),
) error {
	mapEntry := func(entry domain.Entry) (domain.Item, error) { return domain.MapEntry(dialect, mapping, entry) }
	return readEntries(ctx, session, baseDNs, filter, domain.RequestedAttributes(dialect, mapping), mapEntry, emit, skip)
}

// readComputers は起点を並べた順にコンピューターを検索し、写せたエントリを emit へ、写せないエントリを
// skip へ渡す (REQ-DIRECTORYCONNECTOR-007)。
func readComputers(ctx context.Context, session ports.DirectorySession, baseDNs []string, filter string,
	emit func(domain.ComputerItem) error, skip func(*domain.IncompleteEntryError),
) error {
	return readEntries(ctx, session, baseDNs, filter, domain.ComputerAttributes(), domain.MapComputer, emit, skip)
}

func readEntries[T any](ctx context.Context, session ports.DirectorySession, baseDNs []string, filter string, attributes []string,
	mapEntry func(domain.Entry) (T, error), emit func(T) error, skip func(*domain.IncompleteEntryError),
) error {
	for _, baseDN := range baseDNs {
		err := session.Search(ctx, baseDN, filter, attributes, func(entries []domain.Entry) error {
			for _, entry := range entries {
				item, err := mapEntry(entry)
				if incomplete, ok := errors.AsType[*domain.IncompleteEntryError](err); ok {
					skip(incomplete)
					continue
				}
				if err != nil {
					return err
				}
				if err := emit(item); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Run は、前の回が終わってから interval を待って同期を繰り返す。失敗した回はログへ出して次の回へ
// 進み、ctx が終わると待ちも送信中の要求も打ち切って返す (REQ-DIRECTORYCONNECTOR-005)。
func (s Syncer) Run(ctx context.Context, interval time.Duration) {
	for {
		if _, err := s.SyncOnce(ctx); err != nil && ctx.Err() == nil {
			s.Log.Error("sync failed", "error", err.Error())
		}
		if s.Sleep(ctx, interval) != nil {
			return
		}
	}
}

// batchSender は項目を読んだ順に 100 件ずつ、prefix に連番を続けたバッチの ID で送る。
type batchSender[T any] struct {
	syncer  Syncer
	ctx     context.Context
	runID   string
	prefix  string
	send    func(ctx context.Context, runID, batchID string, items []T) error
	pending []T
	batches int
	sent    int
}

func (b *batchSender[T]) add(item T) error {
	b.pending = append(b.pending, item)
	if len(b.pending) < domain.MaxBatchItems {
		return nil
	}
	return b.flush()
}

func (b *batchSender[T]) flush() error {
	if len(b.pending) == 0 {
		return nil
	}
	b.batches++
	batchID := fmt.Sprintf("%s%d", b.prefix, b.batches)
	items := b.pending
	if err := b.syncer.retry(b.ctx, func() error { return b.send(b.ctx, b.runID, batchID, items) }); err != nil {
		return fmt.Errorf("batch %s: %w", batchID, err)
	}
	b.sent += len(items)
	b.pending = nil
	return nil
}

// 一つの要求を送る回数と、最初の待ちの長さ。待ちは失敗ごとに倍にする (REQ-DIRECTORYCONNECTOR-005)。
const (
	maxAttempts  = 5
	initialDelay = time.Second
)

func (s Syncer) retry(ctx context.Context, call func() error) error {
	delay := initialDelay
	for attempt := 1; ; attempt++ {
		err := call()
		if err == nil || attempt == maxAttempts || !retryable(err) {
			return err
		}
		s.Log.Warn("retrying a connector api call", "attempt", attempt, "error", err.Error())
		if err := s.Sleep(ctx, delay); err != nil {
			return err
		}
		delay *= 2
	}
}

// retryable は一時的な失敗かどうかである。応答のない接続の失敗と、408、429、5xx を送り直す。
func retryable(err error) bool {
	var api *ports.APIError
	if !errors.As(err, &api) {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	return api.Status == http.StatusRequestTimeout || api.Status == http.StatusTooManyRequests || api.Status >= 500
}
