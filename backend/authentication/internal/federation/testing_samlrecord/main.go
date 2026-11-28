// Command testing_samlrecord は、Keycloak の開発モードを起動し、上流の SAML IdP が IdMagic の ACS へ届ける応答を記録する。
//
// `mise run record-saml` から使う。タスクが固定したイメージを -image で受け取り、docker で起動する。
// 記録はテストだけが読み、製品の経路から使わない。
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

const defaultOutput = "backend/authentication/internal/federation/protocol_saml/testdata/keycloak-saml.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	image := flag.String("image", "", "Keycloak container image, pinned by the mise task")
	out := flag.String("out", defaultOutput, "recording path")
	flag.Parse()
	if *image == "" {
		return errors.New("-image is required")
	}
	adminPassword, err := randomSecret()
	if err != nil {
		return err
	}
	userPassword, err := randomSecret()
	if err != nil {
		return err
	}

	ctx := context.Background()
	container, err := startKeycloak(ctx, *image, adminPassword)
	if err != nil {
		return err
	}
	defer container.stop(ctx)

	kc := &keycloak{base: container.base, client: &http.Client{Timeout: 30 * time.Second}}
	if err := kc.waitReady(3 * time.Minute); err != nil {
		return fmt.Errorf("%w\n%s", err, container.logs())
	}
	if err := kc.login(adminPassword); err != nil {
		return err
	}
	version, err := kc.serverVersion()
	if err != nil {
		return err
	}
	if err := kc.prepareRealm(userPassword); err != nil {
		return err
	}
	exchanges, err := recordPlacements(kc, userPassword)
	if err != nil {
		return err
	}
	rec := testing_httprecord.Recording{
		Counterpart:        "Keycloak SAML 2.0 identity provider",
		CounterpartVersion: "Keycloak " + version,
		RetrievedAt:        time.Now().Format(time.DateOnly),
		Method:             recordingMethod(*image),
		Exchanges:          exchanges,
	}
	if err := CheckSecrets(rec, adminPassword, userPassword, kc.token); err != nil {
		return fmt.Errorf("refusing to save the recording: %w", err)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, append(data, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Printf("recorded %d responses from Keycloak %s into %s\n", len(exchanges), version, *out)
	return nil
}

func recordingMethod(image string) string {
	return "mise run record-saml が " + image + " を start-dev（インメモリの H2、HTTP、既定のテーマとフロー）で起動し、" +
		"管理 REST API で realm idmagic、パスワードを持つ利用者 alice、SAML クライアント（クライアント ID と ACS は IdMagic の ACS の URL、" +
		"AuthnRequest の署名は求めない、NameID は username、email を Basic の属性 email に対応付ける）を作った。" +
		"製品の BuildAuthnRequest が作った Redirect binding の AuthnRequest でログインを始め、ログイン画面に利用者のパスワードを送り、" +
		"Keycloak が返した自動送信フォームの値を IdMagic の ACS が受け取る POST として記録した。" +
		"署名の置き場所は、クライアントの Sign documents と Sign assertions を往復ごとに切り替えて変えた。" +
		"IdMagic の応答は記録の対象ではないので、往復の応答は空である。setup は接続の設定、realm の SAML メタデータの署名証明書、POST を受け取った時刻を持つ。" +
		"本番の構成（ホスト名の固定、外部の DB、鍵の回転、暗号化した Assertion）での応答は保証しない。"
}

// container は起動した Keycloak のコンテナである。
type container struct {
	id   string
	base string
}

func startKeycloak(ctx context.Context, image, adminPassword string) (*container, error) {
	//nolint:gosec // イメージは mise のタスクが固定した値であり、ほかの引数はこの道具が作った値だけである。
	start := exec.CommandContext(ctx, "docker", "run", "--detach", "--rm",
		"--publish", "127.0.0.1::8080",
		"--env", "KC_BOOTSTRAP_ADMIN_USERNAME=admin",
		"--env", "KC_BOOTSTRAP_ADMIN_PASSWORD="+adminPassword,
		image, "start-dev")
	output, err := start.Output()
	if err != nil {
		return nil, fmt.Errorf("start Keycloak: %w", commandError(err))
	}
	c := &container{id: strings.TrimSpace(string(output))}
	//nolint:gosec // id は docker が返したコンテナの識別子である。
	port, err := exec.CommandContext(ctx, "docker", "port", c.id, "8080/tcp").Output()
	if err != nil {
		c.stop(ctx)
		return nil, fmt.Errorf("read Keycloak port: %w", commandError(err))
	}
	address := strings.TrimSpace(strings.SplitN(string(port), "\n", 2)[0])
	c.base = "http://" + address
	return c, nil
}

func (c *container) stop(ctx context.Context) {
	//nolint:gosec // id は docker が返したコンテナの識別子である。
	_ = exec.CommandContext(context.WithoutCancel(ctx), "docker", "stop", c.id).Run()
}

func (c *container) logs() string {
	//nolint:gosec // id は docker が返したコンテナの識別子である。
	output, _ := exec.CommandContext(context.Background(), "docker", "logs", "--tail", "50", c.id).CombinedOutput()
	return string(output)
}

func commandError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) && len(exit.Stderr) > 0 {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exit.Stderr)))
	}
	return err
}

func randomSecret() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "rec-" + hex.EncodeToString(buf), nil
}
