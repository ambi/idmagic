// Command testing_transitrecord は、Vault と OpenBao の開発用サーバーを起動し、Transit との往復を記録する。
//
// `mise run record-transit -- <vault|openbao>` から使う。開発用サーバーの実行ファイルは PATH から探すので、
// タスクが `mise exec` で固定したバージョンを PATH に置く。記録はテストだけが読み、製品の経路から使わない。
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	name := flag.String("product", "", "counterpart to record: vault or openbao")
	out := flag.String("out", "", "recording path; defaults to the product's testdata file")
	flag.Parse()
	p, ok := products[*name]
	if !ok {
		return fmt.Errorf("unknown -product %q; want vault or openbao", *name)
	}
	if *out == "" {
		*out = p.output
	}

	token, err := randomToken()
	if err != nil {
		return err
	}
	port, err := freePort(context.Background())
	if err != nil {
		return err
	}
	addr := "127.0.0.1:" + strconv.Itoa(port)
	var serverLog bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//nolint:gosec // 実行ファイルは products の固定の名前であり、引数はこの道具が作ったトークンとアドレスだけである。
	server := exec.CommandContext(ctx, p.binary, "server", "-dev", "-dev-root-token-id="+token, "-dev-listen-address="+addr)
	server.Stdout, server.Stderr = &serverLog, &serverLog
	if err := server.Start(); err != nil {
		return fmt.Errorf("start %s dev server: %w", p.binary, err)
	}
	defer func() {
		cancel()
		_ = server.Wait()
	}()

	r := &recorder{address: "http://" + addr, token: token, client: &http.Client{Timeout: 10 * time.Second}}
	version, err := r.waitHealthy(30 * time.Second)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, serverLog.String())
	}
	if err := r.enableTransit(); err != nil {
		return err
	}
	exchanges, err := p.script(r)
	if err != nil {
		return err
	}
	rec := testing_httprecord.Recording{
		Counterpart:        p.counterpart,
		CounterpartVersion: p.title + " " + version,
		RetrievedAt:        time.Now().Format(time.DateOnly),
		Method:             p.method,
		Exchanges:          exchanges,
	}
	if err := CheckSecrets(rec, token); err != nil {
		return fmt.Errorf("refusing to save the recording: %w", err)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, append(data, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Printf("recorded %d exchanges with %s %s into %s\n", len(exchanges), p.title, version, *out)
	return nil
}

// recorder は開発用サーバーへ要求を送り、要求と応答を記録の往復にする。
type recorder struct {
	address string
	token   string
	client  *http.Client
}

// send は、アダプターと同じ形の要求を送って往復を記録する。
// payload が nil でなければ json.Marshal の本文と Content-Type を付ける。アダプターも同じ規則で本文を作る。
func (r *recorder) send(name, source, method, path string, payload any) (testing_httprecord.Exchange, error) {
	ex := testing_httprecord.Exchange{Name: name, Source: source}
	ex.Request.Method, ex.Request.Path = method, path
	ex.Request.Headers = map[string]string{"X-Vault-Token": RecordedToken}
	var reader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return ex, err
		}
		ex.Request.Headers["Content-Type"] = "application/json"
		ex.Request.Body = body
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, r.address+path, reader)
	if err != nil {
		return ex, err
	}
	req.Header.Set("X-Vault-Token", r.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return ex, fmt.Errorf("%s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ex, fmt.Errorf("%s: read response: %w", name, err)
	}
	ex.Response.Status = resp.StatusCode
	ex.Response.Headers = map[string]string{}
	for _, header := range []string{"Content-Type", "Cache-Control"} {
		if value := resp.Header.Get(header); value != "" {
			ex.Response.Headers[header] = value
		}
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if !json.Valid(body) {
			return ex, fmt.Errorf("%s: response body is not JSON: %s", name, body)
		}
		ex.Response.Body = bytes.TrimSpace(body)
	}
	return ex, nil
}

// expect は往復を送り、状態コードが want でなければ失敗する。記録の手順が想定どおりに進んだことを確かめる。
func (r *recorder) expect(want int, name, source, method, path string, payload any) (testing_httprecord.Exchange, error) {
	ex, err := r.send(name, source, method, path, payload)
	if err == nil && ex.Response.Status != want {
		err = fmt.Errorf("%s: status %d, want %d: %s", name, ex.Response.Status, want, ex.Response.Body)
	}
	return ex, err
}

func (r *recorder) waitHealthy(limit time.Duration) (string, error) {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, r.address+"/v1/sys/health", http.NoBody)
		if err != nil {
			return "", err
		}
		resp, err := r.client.Do(req)
		if err == nil {
			var health struct {
				Version string `json:"version"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&health)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && decodeErr == nil && health.Version != "" {
				return health.Version, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", errors.New("dev server did not become healthy")
}

// enableTransit は Transit を transit に有効にする。開発用サーバーの準備であり、記録しない。
func (r *recorder) enableTransit() error {
	ex, err := r.send("enable-transit", "", http.MethodPost, "/v1/sys/mounts/transit", map[string]any{"type": "transit"})
	if err != nil {
		return err
	}
	if ex.Response.Status != http.StatusNoContent && ex.Response.Status != http.StatusOK {
		return fmt.Errorf("enable transit: status %d: %s", ex.Response.Status, ex.Response.Body)
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "dev-" + hex.EncodeToString(buf), nil
}

// freePort は空いているポートを返す。開発用サーバーは次の番号もクラスターの待受けに使うので、両方が空いている番号を選ぶ。
func freePort(ctx context.Context) (int, error) {
	var lc net.ListenConfig
	for range 20 {
		l, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		addr, ok := l.Addr().(*net.TCPAddr)
		_ = l.Close()
		if !ok {
			return 0, fmt.Errorf("unexpected listener address %v", l.Addr())
		}
		next, err := lc.Listen(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(addr.Port+1))
		if err != nil {
			continue
		}
		_ = next.Close()
		return addr.Port, nil
	}
	return 0, errors.New("no free pair of ports")
}
