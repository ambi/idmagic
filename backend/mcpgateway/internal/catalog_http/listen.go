package catalog_http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

// 間隔と上限は REQ-MCPGATEWAY-012 が定める。
const (
	sendCatalogInterval = 10 * time.Second
	defaultAckTimeout   = 10 * time.Second
	initialBackoff      = time.Second
	maxBackoff          = 5 * time.Minute
	maxListenLineBytes  = 1 << 20
	// listenRequestID は subscriptions/listen の要求の id で、通知の subscriptionId と照合する。
	listenRequestID = 1
)

// Listener は上流の subscriptions/listen で変更通知を待ち、通知のたびにカタログを再送する。
type Listener struct {
	Publisher Publisher
	// Client は上流へ待ち受けを送る。応答は開いたままなので、全体の期限を持たないクライアントにする。
	Client *http.Client
	// Log には件数、状態、理由だけを書き、ツールの説明とシークレットを書かない。
	Log io.Writer
	// 以下は省略すると既定の値と実時間を使う。テストは待たずに間隔を観測するために差し替える。
	AckTimeout      time.Duration
	Now             func() time.Time
	WaitToReconnect func(context.Context, time.Duration) error
	WaitToSend      func(context.Context, time.Duration) error
}

// Run は ctx が終わるか、上流が変更通知に対応しないと分かるまで、待ち受けとカタログの送信を続ける。
// 戻る前に、受けた契機による送信を済ませる。
func (l Listener) Run(ctx context.Context) {
	scheduler := newCatalogScheduler(l.now, l.waitToSend(), func(ctx context.Context) { l.sendCatalog(ctx) })
	var done sync.WaitGroup
	done.Go(func() { scheduler.run(ctx) })
	defer done.Wait()
	defer scheduler.close()

	backoff := initialBackoff
	for attempt := 0; ; attempt++ {
		outcome := l.listen(ctx, scheduler.trigger)
		if ctx.Err() != nil {
			return
		}
		// 起動時の送信は、待ち受けに対応しない上流でも行う (REQ-MCPGATEWAY-011)。
		if attempt == 0 && !outcome.acknowledged {
			scheduler.trigger()
		}
		if outcome.stop {
			l.logf("tool change listener stopped: %v", outcome.reason)
			return
		}
		if outcome.acknowledged {
			backoff = initialBackoff
		}
		l.logf("tool change listener reconnecting in %s: %v", backoff, outcome.reason)
		if err := l.waitToReconnect()(ctx, backoff); err != nil {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

func nextBackoff(current time.Duration) time.Duration {
	return min(current*2, maxBackoff)
}

func (l Listener) sendCatalog(ctx context.Context) {
	result, err := l.Publisher.SendCatalog(ctx)
	if err != nil {
		l.logf("tool catalog failed: %v", err)
		return
	}
	l.logf("sent %d tools (complete=%t)", result.Tools, result.Complete)
}

// listenOutcome は一回の待ち受けの終わり方である。stop は開き直しても変わらない失敗を表す。
type listenOutcome struct {
	acknowledged bool
	stop         bool
	reason       error
}

func (l Listener) listen(ctx context.Context, trigger func()) listenOutcome {
	stream, cancel := context.WithCancel(ctx)
	defer cancel()
	ackTimer := time.AfterFunc(l.ackTimeout(), cancel)
	defer ackTimer.Stop()

	request, err := l.listenRequest(stream)
	if err != nil {
		return listenOutcome{stop: true, reason: err}
	}
	response, err := l.Client.Do(request)
	if err != nil {
		if stream.Err() != nil && ctx.Err() == nil {
			err = errors.New("no acknowledgment within the time limit")
		}
		return listenOutcome{reason: err}
	}
	defer func() { _ = response.Body.Close() }()
	if outcome, ok := classifyListenStatus(response.StatusCode); !ok {
		return outcome
	}

	acknowledged := false
	events := newEventReader(response.Body)
	for {
		data, err := events.next()
		if err != nil {
			if !acknowledged && stream.Err() != nil && ctx.Err() == nil {
				err = errors.New("no acknowledgment within the time limit")
			} else if errors.Is(err, io.EOF) {
				err = errors.New("the stream closed")
			}
			return listenOutcome{acknowledged: acknowledged, reason: err}
		}
		switch event := classifyListenMessage(data); event {
		case listenAcknowledged:
			if !acknowledged {
				acknowledged = true
				ackTimer.Stop()
				trigger()
			}
		case listenUnsupported:
			return listenOutcome{stop: true, reason: errors.New("the upstream does not send tools/list_changed")}
		case listenToolsChanged:
			if acknowledged {
				trigger()
			}
		case listenEnded:
			return listenOutcome{acknowledged: acknowledged, reason: errors.New("the upstream ended the subscription")}
		case listenFailed:
			return listenOutcome{acknowledged: acknowledged, stop: true, reason: errors.New("subscriptions/listen returned an error")}
		}
	}
}

// listenRequestBody は subscriptions/listen の要求で、tools/list_changed だけを求める。
type listenRequestBody struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  struct {
		Meta struct {
			ProtocolVersion    string   `json:"io.modelcontextprotocol/protocolVersion"`
			ClientCapabilities struct{} `json:"io.modelcontextprotocol/clientCapabilities"`
		} `json:"_meta"`
		Notifications struct {
			ToolsListChanged bool `json:"toolsListChanged"`
		} `json:"notifications"`
	} `json:"params"`
}

func (l Listener) listenRequest(ctx context.Context) (*http.Request, error) {
	message := listenRequestBody{JSONRPC: "2.0", ID: listenRequestID, Method: "subscriptions/listen"}
	message.Params.Meta.ProtocolVersion = domain.ProtocolVersion
	message.Params.Notifications.ToolsListChanged = true
	body, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, l.Publisher.Upstream, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", domain.ProtocolVersion)
	request.Header.Set("Mcp-Method", "subscriptions/listen")
	return request, nil
}

// classifyListenStatus は、200 なら読み進め、それ以外は開き直すかやめるかを返す。
func classifyListenStatus(status int) (listenOutcome, bool) {
	switch {
	case status == http.StatusOK:
		return listenOutcome{}, true
	case status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500:
		return listenOutcome{reason: fmt.Errorf("subscriptions/listen returned HTTP %d", status)}, false
	case status >= 400:
		return listenOutcome{stop: true, reason: fmt.Errorf("subscriptions/listen returned HTTP %d", status)}, false
	default:
		return listenOutcome{reason: fmt.Errorf("subscriptions/listen returned HTTP %d", status)}, false
	}
}

type listenEvent int

const (
	listenIgnored listenEvent = iota
	listenAcknowledged
	listenUnsupported
	listenToolsChanged
	listenEnded
	listenFailed
)

// classifyListenMessage は一つの JSON-RPC の通知か応答を読む。別の購読の通知と読めない値は無視する。
func classifyListenMessage(data []byte) listenEvent {
	var message struct {
		ID     *json.Number    `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
		Params struct {
			Meta struct {
				SubscriptionID *json.Number `json:"io.modelcontextprotocol/subscriptionId"`
			} `json:"_meta"`
			Notifications struct {
				ToolsListChanged bool `json:"toolsListChanged"`
			} `json:"notifications"`
		} `json:"params"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&message); err != nil {
		return listenIgnored
	}
	if message.ID != nil {
		if message.ID.String() != fmt.Sprint(listenRequestID) {
			return listenIgnored
		}
		if message.Error != nil {
			return listenFailed
		}
		return listenEnded
	}
	if subscription := message.Params.Meta.SubscriptionID; subscription == nil || subscription.String() != fmt.Sprint(listenRequestID) {
		return listenIgnored
	}
	switch message.Method {
	case "notifications/subscriptions/acknowledged":
		if message.Params.Notifications.ToolsListChanged {
			return listenAcknowledged
		}
		return listenUnsupported
	case "notifications/tools/list_changed":
		return listenToolsChanged
	default:
		return listenIgnored
	}
}

// eventReader は SSE の data 行を空行までつなげて一つのメッセージとして返す。
// JSON の一つの応答で返す上流のために、data 行のない本文はそのまま一つのメッセージとして返す。
type eventReader struct {
	lines *bufio.Scanner
}

func newEventReader(body io.Reader) eventReader {
	lines := bufio.NewScanner(body)
	lines.Buffer(make([]byte, 0, 4096), maxListenLineBytes)
	return eventReader{lines: lines}
}

func (r eventReader) next() ([]byte, error) {
	var data []string
	for r.lines.Scan() {
		line := r.lines.Text()
		if line == "" {
			if len(data) > 0 {
				return []byte(strings.Join(data, "\n")), nil
			}
			continue
		}
		if value, ok := strings.CutPrefix(line, "data:"); ok {
			data = append(data, strings.TrimPrefix(value, " "))
			continue
		}
		if strings.HasPrefix(line, "{") && len(data) == 0 {
			return []byte(line), nil
		}
	}
	if len(data) > 0 {
		return []byte(strings.Join(data, "\n")), nil
	}
	if err := r.lines.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (l Listener) logf(format string, args ...any) {
	if l.Log != nil {
		_, _ = fmt.Fprintf(l.Log, format+"\n", args...)
	}
}

func (l Listener) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

func (l Listener) ackTimeout() time.Duration {
	if l.AckTimeout > 0 {
		return l.AckTimeout
	}
	return defaultAckTimeout
}

func (l Listener) waitToReconnect() func(context.Context, time.Duration) error {
	if l.WaitToReconnect != nil {
		return l.WaitToReconnect
	}
	return sleep
}

func (l Listener) waitToSend() func(context.Context, time.Duration) error {
	if l.WaitToSend != nil {
		return l.WaitToSend
	}
	return sleep
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// catalogScheduler は送信の契機をまとめ、送信の開始の間隔を sendCatalogInterval 以上空ける。
// 契機は一件だけ保留し、待つ間の契機は次の一回の送信にまとめる。
type catalogScheduler struct {
	pending     chan struct{}
	closing     chan struct{}
	once        *sync.Once
	now         func() time.Time
	wait        func(context.Context, time.Duration) error
	sendCatalog func(context.Context)
}

func newCatalogScheduler(now func() time.Time, wait func(context.Context, time.Duration) error, sendCatalog func(context.Context)) catalogScheduler {
	return catalogScheduler{pending: make(chan struct{}, 1), closing: make(chan struct{}), once: &sync.Once{}, now: now, wait: wait, sendCatalog: sendCatalog}
}

// trigger は待たずに戻る。保留中の契機があれば何もしない。
func (s catalogScheduler) trigger() {
	select {
	case s.pending <- struct{}{}:
	default:
	}
}

// close は保留中の契機による送信を済ませてから run を終えさせる。
func (s catalogScheduler) close() { s.once.Do(func() { close(s.closing) }) }

func (s catalogScheduler) run(ctx context.Context) {
	var last time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.pending:
		case <-s.closing:
			select {
			case <-s.pending:
			default:
				return
			}
		}
		if !last.IsZero() {
			if remaining := sendCatalogInterval - s.now().Sub(last); remaining > 0 {
				if err := s.wait(ctx, remaining); err != nil {
					return
				}
			}
		}
		// 待つ間の契機は、これから始める送信が反映する。
		select {
		case <-s.pending:
		default:
		}
		last = s.now()
		s.sendCatalog(ctx)
	}
}
