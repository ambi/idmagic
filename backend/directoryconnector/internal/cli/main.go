package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/api_http"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ldap_goldap"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/state_file"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/usecases"
)

const usage = `usage:
  idmagic-directory-connector check -config <file>    read ldap and report what a full sync would send, sending nothing
  idmagic-directory-connector enroll -config <file>   read the enrollment token from stdin and enroll
  idmagic-directory-connector run -config <file> [-once]
`

// Main はサブコマンドを実行し、終了コードを返す。ctx が終わると常駐の同期を止める。
func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, usage)
		return 2
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "path to the connector configuration file")
	once := flags.Bool("once", false, "sync once and exit (run only)")
	if err := flags.Parse(args[1:]); err != nil || *configPath == "" || (args[0] != "run" && *once) {
		_, _ = io.WriteString(stderr, usage)
		return 2
	}
	config, err := LoadConfig(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
		return 2
	}
	transport := &http.Transport{TLSClientConfig: config.IdMagicTLS}
	if config.Proxy != nil {
		transport.Proxy = http.ProxyURL(config.Proxy)
	}
	httpClient := &http.Client{Timeout: time.Minute, Transport: transport}
	switch args[0] {
	case "check":
		return check(ctx, config, stdout, stderr)
	case "enroll":
		return enroll(ctx, config, httpClient, stdin, stdout, stderr)
	case "run":
		return run(ctx, config, httpClient, *once, stderr)
	default:
		_, _ = io.WriteString(stderr, usage)
		return 2
	}
}

// check は IdMagic へ接続せずに検索の範囲を読み、送るはずの項目の数と送らないエントリを出す
// (REQ-DIRECTORYCONNECTOR-006)。
func check(ctx context.Context, config Config, stdout, stderr io.Writer) int {
	directory, err := ldap_goldap.New(config.LDAP)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check: %v\n", err)
		return 2
	}
	result, err := usecases.Check(ctx, directory, config.Dialect, config.Scope, config.Mapping)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%d users would be sent, %d entries would be skipped\n", result.Items, len(result.Skipped))
	for _, skipped := range result.Skipped {
		_, _ = fmt.Fprintf(stdout, "skipped %s: missing %s\n", skipped.DN, skipped.Attribute)
	}
	return 0
}

func enroll(ctx context.Context, config Config, httpClient *http.Client, stdin io.Reader, stdout, stderr io.Writer) int {
	line, err := bufio.NewReader(stdin).ReadString('\n')
	token := strings.TrimSpace(line)
	if token == "" || (err != nil && !errors.Is(err, io.EOF)) {
		_, _ = io.WriteString(stderr, "enroll: no enrollment token on stdin\n")
		return 2
	}
	enrollment, err := usecases.Enroll(ctx, api_http.NewEnrollmentClient(config.RealmURL, httpClient), state_file.New(config.StateDir), token)
	if err != nil {
		if api, ok := errors.AsType[*ports.APIError](err); ok {
			_, _ = fmt.Fprintf(stderr, "enroll: refused with status %d %s\n", api.Status, api.Type)
		} else {
			_, _ = fmt.Fprintf(stderr, "enroll: %v\n", err)
		}
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "enrolled connector %s with key %s\n", enrollment.ConnectorID, enrollment.KeyID)
	return 0
}

func run(ctx context.Context, config Config, httpClient *http.Client, once bool, stderr io.Writer) int {
	log := slog.New(slog.NewJSONHandler(stderr, nil))
	store := state_file.New(config.StateDir)
	key, enrollment, err := store.LoadCredentials()
	if err != nil {
		log.Error("cannot start", "error", err.Error())
		return 1
	}
	directory, err := ldap_goldap.New(config.LDAP)
	if err != nil {
		log.Error("cannot start", "error", err.Error())
		return 2
	}
	syncer := usecases.Syncer{
		Directory: directory, API: api_http.NewClient(config.RealmURL, httpClient, key, enrollment, time.Now), Cursors: store,
		Dialect: config.Dialect, Scope: config.Scope, Mapping: config.Mapping, FullInterval: config.FullInterval,
		Now: time.Now, Sleep: sleep, Log: log,
	}
	if !once {
		syncer.Run(ctx, config.Interval)
		return 0
	}
	if _, err := syncer.SyncOnce(ctx); err != nil {
		log.Error("sync failed", "error", err.Error())
		return 1
	}
	return 0
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
