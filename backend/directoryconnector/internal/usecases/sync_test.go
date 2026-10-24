package usecases

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// fakeDirectory は検索の範囲のエントリを持ち、受けたフィルターを記録する。
// byBase があれば起点ごとのエントリを、なければどの起点にも entries を返す。
type fakeDirectory struct {
	entries    []domain.Entry
	byBase     map[string][]domain.Entry
	root       domain.RootInfo
	connectErr error
	filters    []string
	bases      []string
}

func (d *fakeDirectory) Connect(context.Context) (ports.DirectorySession, error) {
	if d.connectErr != nil {
		return nil, d.connectErr
	}
	return d, nil
}

func (d *fakeDirectory) RootInfo(context.Context) (domain.RootInfo, error) { return d.root, nil }

func (d *fakeDirectory) Search(_ context.Context, baseDN, filter string, _ []string, page func([]domain.Entry) error) error {
	d.filters = append(d.filters, filter)
	d.bases = append(d.bases, baseDN)
	entries := d.entries
	if d.byBase != nil {
		entries = d.byBase[baseDN]
	}
	for chunk := range slices.Chunk(entries, 30) {
		if err := page(chunk); err != nil {
			return err
		}
	}
	return nil
}

func (d *fakeDirectory) Close() error { return nil }

// recordingAPI は Connector API へ出した要求を順に記録し、予定した失敗を返す。
type recordingAPI struct {
	calls     []string
	batches   map[string][]domain.Item
	computers map[string][]domain.ComputerItem
	failures  map[string][]error // 要求の名前ごとに、先頭から一つずつ返す失敗
	runs      int
}

func (a *recordingAPI) fail(call string) error {
	a.calls = append(a.calls, call)
	if queue := a.failures[call]; len(queue) > 0 {
		a.failures[call] = queue[1:]
		return queue[0]
	}
	return nil
}

func (a *recordingAPI) Heartbeat(context.Context) error { return a.fail("heartbeat") }

func (a *recordingAPI) StartRun(_ context.Context, mode domain.Mode) (string, error) {
	if err := a.fail("start " + string(mode)); err != nil {
		return "", err
	}
	a.runs++
	return fmt.Sprintf("run%d", a.runs), nil
}

func (a *recordingAPI) SendBatch(_ context.Context, runID, batchID string, items []domain.Item) error {
	if err := a.fail("batch " + batchID); err != nil {
		return err
	}
	if a.batches == nil {
		a.batches = map[string][]domain.Item{}
	}
	a.batches[batchID] = items
	return nil
}

func (a *recordingAPI) CompleteRun(_ context.Context, runID string, itemCount int) error {
	return a.fail(fmt.Sprintf("complete %s %d", runID, itemCount))
}

type memoryCursors struct{ cursor *domain.Cursor }

func (m *memoryCursors) Load(context.Context) (*domain.Cursor, error) { return m.cursor, nil }

func (m *memoryCursors) Save(_ context.Context, cursor domain.Cursor) error {
	m.cursor = &cursor
	return nil
}

func people(n int) []domain.Entry { return peopleFrom(0, n) }

func peopleFrom(start, n int) []domain.Entry {
	entries := make([]domain.Entry, n)
	for j := range entries {
		i := start + j
		entries[j] = domain.Entry{DN: fmt.Sprintf("uid=u%d,ou=people", i), Attributes: []domain.Attribute{
			{Name: "entryUUID", Values: [][]byte{fmt.Appendf(nil, "uuid-%d", i)}},
			{Name: "uid", Values: [][]byte{fmt.Appendf(nil, "u%d", i)}},
			{Name: "mail", Values: [][]byte{fmt.Appendf(nil, "secret-mail-%d@example.test", i)}},
		}}
	}
	return entries
}

type harness struct {
	syncer  Syncer
	dir     *fakeDirectory
	api     *recordingAPI
	cursors *memoryCursors
	sleeps  []time.Duration
	log     *bytes.Buffer
	now     time.Time
}

func newHarness(t *testing.T, entries []domain.Entry) *harness {
	t.Helper()
	mapping, err := domain.EffectiveMapping(domain.OpenLDAP, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{
		dir: &fakeDirectory{entries: entries}, api: &recordingAPI{failures: map[string][]error{}},
		cursors: &memoryCursors{}, log: &bytes.Buffer{}, now: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
	}
	h.syncer = Syncer{
		Directory: h.dir, API: h.api, Cursors: h.cursors,
		Dialect: domain.OpenLDAP, Scope: domain.Scope{BaseDNs: []string{"ou=people"}, Filter: domain.DefaultUserFilter(domain.OpenLDAP)},
		Mapping: mapping, FullInterval: 24 * time.Hour,
		Now: func() time.Time { return h.now },
		Sleep: func(_ context.Context, d time.Duration) error {
			h.sleeps = append(h.sleeps, d)
			return nil
		},
		Log: slog.New(slog.NewJSONHandler(h.log, nil)),
	}
	return h
}

//spec:covers REQ-DIRECTORYCONNECTOR-003, REQ-DIRECTORYCONNECTOR-004: 接続を報告してから full の実行を始め、項目を読んだ順に 100 件ずつ連番のバッチで送り、送った数で完了し、欠けたエントリを属性値を出さずにログへ出して飛ばし、完了の後に位置を保存する。
func TestSyncSendsAFullRunInBatchesOf100(t *testing.T) {
	entries := people(250)
	entries = slices.Insert(entries, 120, domain.Entry{DN: "uid=nouuid,ou=people", Attributes: []domain.Attribute{{Name: "uid", Values: [][]byte{[]byte("nouuid")}}}})
	h := newHarness(t, entries)

	result, err := h.syncer.SyncOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{"heartbeat", "start full", "batch run1-1", "batch run1-2", "batch run1-3", "complete run1 250"}
	if !slices.Equal(h.api.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", h.api.calls, wantCalls)
	}
	if got := []int{len(h.api.batches["run1-1"]), len(h.api.batches["run1-2"]), len(h.api.batches["run1-3"])}; !slices.Equal(got, []int{100, 100, 50}) {
		t.Fatalf("batch sizes = %v", got)
	}
	if first, last := h.api.batches["run1-1"][0], h.api.batches["run1-3"][49]; first.ExternalID != "uuid-0" || last.ExternalID != "uuid-249" || !first.Enabled {
		t.Fatalf("first=%+v last=%+v, want the order read", first, last)
	}
	if result.Mode != domain.ModeFull || result.Items != 250 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if h.cursors.cursor == nil || !h.cursors.cursor.LastFullAt.Equal(h.now) {
		t.Fatalf("cursor = %+v, want saved after the completion", h.cursors.cursor)
	}
	log := h.log.String()
	if !strings.Contains(log, "uid=nouuid,ou=people") || !strings.Contains(log, "entryUUID") {
		t.Fatalf("log does not name the skipped entry: %s", log)
	}
	if strings.Contains(log, "secret-mail") {
		t.Fatalf("log contains an attribute value: %s", log)
	}

	h.now = h.now.Add(time.Hour)
	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := "(&(objectClass=inetOrgPerson)(modifyTimestamp>=20261024115500Z))"; h.dir.filters[1] != want {
		t.Fatalf("second filter = %s, want %s", h.dir.filters[1], want)
	}
	if h.api.calls[7] != "start incremental" {
		t.Fatalf("second run = %s, want incremental", h.api.calls[7])
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-005: 一時的な失敗のバッチを同じバッチ ID で倍の間隔で送り直し、完了を受理されなければ位置を進めず、次の回が同じ位置から新しい実行を始める。
func TestSyncRetriesAndResumesFromTheCommittedCursor(t *testing.T) {
	h := newHarness(t, people(150))
	unavailable := &ports.APIError{Status: 503, Type: "temporarily_unavailable"}
	h.api.failures["batch run1-2"] = []error{unavailable, errors.New("connection reset")}
	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"heartbeat", "start full", "batch run1-1", "batch run1-2", "batch run1-2", "batch run1-2", "complete run1 150"}; !slices.Equal(h.api.calls, want) {
		t.Fatalf("calls = %v, want %v", h.api.calls, want)
	}
	if !slices.Equal(h.sleeps, []time.Duration{time.Second, 2 * time.Second}) {
		t.Fatalf("sleeps = %v", h.sleeps)
	}
	committed := *h.cursors.cursor

	h.now = h.now.Add(time.Hour)
	h.api.calls, h.sleeps = nil, nil
	h.api.failures["complete run2 150"] = []error{unavailable, unavailable, unavailable, unavailable, unavailable}
	if _, err := h.syncer.SyncOnce(context.Background()); !errors.Is(err, unavailable) {
		t.Fatalf("error = %v, want the last 503", err)
	}
	if len(h.api.calls) != 4+5 || !slices.Equal(h.sleeps, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
		t.Fatalf("calls=%v sleeps=%v, want five attempts", h.api.calls, h.sleeps)
	}
	if !reflect.DeepEqual(*h.cursors.cursor, committed) {
		t.Fatalf("cursor = %+v, want the committed %+v", *h.cursors.cursor, committed)
	}

	h.now = h.now.Add(time.Hour)
	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if h.dir.filters[1] != h.dir.filters[2] {
		t.Fatalf("resumed filter = %s, want the same position %s", h.dir.filters[2], h.dir.filters[1])
	}
	if !slices.Contains(h.api.calls, "start incremental") || h.api.runs != 3 {
		t.Fatalf("calls = %v, want a new run", h.api.calls)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-005: 408 と 429 以外の 4xx は送り直さずに同期を中断し、位置を変えず、408、429、500 は送り直す。
func TestSyncAbortsOnAClientError(t *testing.T) {
	h := newHarness(t, people(10))
	closed := &ports.APIError{Status: 409, Type: "ingestion_run_closed"}
	h.api.failures["batch run1-1"] = []error{closed}
	if _, err := h.syncer.SyncOnce(context.Background()); !errors.Is(err, closed) {
		t.Fatalf("error = %v, want the 409", err)
	}
	if want := []string{"heartbeat", "start full", "batch run1-1"}; !slices.Equal(h.api.calls, want) || len(h.sleeps) != 0 {
		t.Fatalf("calls=%v sleeps=%v, want no retry", h.api.calls, h.sleeps)
	}
	if h.cursors.cursor != nil {
		t.Fatalf("cursor = %+v, want none", h.cursors.cursor)
	}

	for _, status := range []int{408, 429, 500} {
		h := newHarness(t, people(1))
		h.api.failures["start full"] = []error{&ports.APIError{Status: status}}
		if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
			t.Fatalf("status %d: %v, want a retry", status, err)
		}
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-002, REQ-DIRECTORYCONNECTOR-003: LDAP を読む前に接続を報告し、LDAP への接続に失敗したら実行を始めず、位置を変えない。
func TestSyncReportsTheHeartbeatBeforeReadingLDAP(t *testing.T) {
	h := newHarness(t, people(1))
	h.dir.connectErr = errors.New("x509: certificate signed by unknown authority")
	if _, err := h.syncer.SyncOnce(context.Background()); !errors.Is(err, h.dir.connectErr) {
		t.Fatalf("error = %v", err)
	}
	if !slices.Equal(h.api.calls, []string{"heartbeat"}) || h.cursors.cursor != nil {
		t.Fatalf("calls=%v cursor=%+v", h.api.calls, h.cursors.cursor)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-005: 常駐の間、失敗した回の後も間隔を待って次の回を始め、停止の合図で終わる。
func TestRunRepeatsAfterAFailedSyncUntilStopped(t *testing.T) {
	h := newHarness(t, people(1))
	h.api.failures["heartbeat"] = []error{&ports.APIError{Status: 401, Type: "invalid_token"}}
	ctx, stop := context.WithCancel(context.Background())
	waits := 0
	h.syncer.Sleep = func(_ context.Context, d time.Duration) error {
		if d != 5*time.Minute {
			t.Fatalf("wait = %s, want the interval", d)
		}
		waits++
		if waits == 2 {
			stop()
			return ctx.Err()
		}
		return nil
	}
	h.syncer.Run(ctx, 5*time.Minute)
	if want := []string{"heartbeat", "heartbeat", "start full", "batch run1-1", "complete run1 1"}; !slices.Equal(h.api.calls, want) {
		t.Fatalf("calls = %v, want %v", h.api.calls, want)
	}
	if !strings.Contains(h.log.String(), "sync failed") {
		t.Fatalf("log = %s, want the failed sync", h.log.String())
	}
}

func (a *recordingAPI) SendComputerBatch(_ context.Context, runID, batchID string, items []domain.ComputerItem) error {
	if err := a.fail("computers " + batchID); err != nil {
		return err
	}
	if a.computers == nil {
		a.computers = map[string][]domain.ComputerItem{}
	}
	a.computers[batchID] = items
	return nil
}

func adEntry(dn string, attributes map[string][]byte) domain.Entry {
	entry := domain.Entry{DN: dn}
	for _, name := range slices.Sorted(maps.Keys(attributes)) {
		entry.Attributes = append(entry.Attributes, domain.Attribute{Name: name, Values: [][]byte{attributes[name]}})
	}
	return entry
}

func computerEntries(n int) []domain.Entry {
	entries := make([]domain.Entry, n)
	for i := range entries {
		guid := make([]byte, 16)
		guid[0] = byte(i)
		sid := []byte{1, 2, 0, 0, 0, 0, 0, 5, 21, 0, 0, 0, byte(i), 0, 0, 0}
		entries[i] = adEntry(fmt.Sprintf("CN=WS%d,CN=Computers,DC=corp,DC=example", i), map[string][]byte{
			"objectGUID": guid, "objectSid": sid, "sAMAccountName": fmt.Appendf(nil, "WS%d$", i), "userAccountControl": []byte("4096"),
		})
	}
	return entries
}

// 100 件ずつ <実行 ID>-computers-<連番> で送り、欠けたエントリを飛ばし、User とコンピューターの和で完了し、次の回は両方に uSNChanged の条件を付ける。
//
//spec:covers REQ-DIRECTORYCONNECTOR-004, REQ-DIRECTORYCONNECTOR-007: computers の節を持つ AD の同期は、User を送った後にコンピューターの起点を読み、
func TestSyncSendsComputersAfterUsers(t *testing.T) {
	h := newHarness(t, nil)
	mapping, err := domain.EffectiveMapping(domain.ActiveDirectory, nil)
	if err != nil {
		t.Fatal(err)
	}
	user := adEntry("CN=Alice,OU=Staff,DC=corp,DC=example", map[string][]byte{
		"objectGUID": make([]byte, 16), "sAMAccountName": []byte("alice"), "userAccountControl": []byte("512"),
	})
	computers := slices.Insert(computerEntries(120), 50, adEntry("CN=Broken,CN=Computers,DC=corp,DC=example", map[string][]byte{
		"objectGUID": make([]byte, 16), "sAMAccountName": []byte("BROKEN$"), "userAccountControl": []byte("4096"),
	}))
	h.dir.byBase = map[string][]domain.Entry{"OU=Staff,DC=corp,DC=example": {user}, "CN=Computers,DC=corp,DC=example": computers}
	h.dir.root = domain.RootInfo{Server: "CN=DC1", HighestUSN: 70}
	h.syncer.Dialect, h.syncer.Mapping = domain.ActiveDirectory, mapping
	h.syncer.Scope = domain.Scope{BaseDNs: []string{"OU=Staff,DC=corp,DC=example"}, Filter: domain.DefaultUserFilter(domain.ActiveDirectory)}
	h.syncer.Computers = &domain.Scope{BaseDNs: []string{"CN=Computers,DC=corp,DC=example"}, Filter: domain.DefaultComputerFilter}

	result, err := h.syncer.SyncOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{"heartbeat", "start full", "batch run1-1", "computers run1-computers-1", "computers run1-computers-2", "complete run1 121"}
	if !slices.Equal(h.api.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", h.api.calls, wantCalls)
	}
	first, last := h.api.computers["run1-computers-1"][0], h.api.computers["run1-computers-2"][19]
	if len(h.api.computers["run1-computers-1"]) != 100 || first.SAMAccountName != "WS0$" || first.Domain != "corp.example" ||
		first.ObjectSID != "S-1-5-21-0" || last.SAMAccountName != "WS119$" || !first.Enabled {
		t.Fatalf("first=%+v last=%+v", first, last)
	}
	if result.Items != 121 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(h.log.String(), "CN=Broken,CN=Computers,DC=corp,DC=example") {
		t.Fatalf("log does not name the skipped computer: %s", h.log.String())
	}
	if h.cursors.cursor == nil || h.cursors.cursor.Computers == nil {
		t.Fatalf("cursor = %+v, want the computer scope", h.cursors.cursor)
	}

	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := "(&" + domain.DefaultComputerFilter + "(uSNChanged>=71))"; h.dir.filters[len(h.dir.filters)-1] != want {
		t.Fatalf("second computer filter = %s, want %s", h.dir.filters[len(h.dir.filters)-1], want)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-007: computers の節がない同期は、コンピューターを検索も送信もしない。
func TestSyncWithoutComputersReadsOnlyUsers(t *testing.T) {
	h := newHarness(t, people(3))
	if _, err := h.syncer.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(h.dir.filters) != 1 || len(h.api.computers) != 0 {
		t.Fatalf("filters = %v, computer batches = %v", h.dir.filters, h.api.computers)
	}
}
