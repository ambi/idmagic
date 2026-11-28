package verify_jose_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/verify_jose"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	caepTenant = "tenant-caep"
	caepStream = "stream-caep"
)

// caepTransmitter は CAEP の例の claims をテストの鍵で署名する送信側である。
type caepTransmitter struct {
	key  *ecdsa.PrivateKey
	jwks map[string]any
}

func newCAEPTransmitter(t *testing.T) caepTransmitter {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	coordinate := func(n *big.Int) string { return base64.RawURLEncoding.EncodeToString(n.FillBytes(make([]byte, 32))) }
	jwk := map[string]any{"kty": "EC", "crv": "P-256", "kid": "caep-example", "alg": "ES256", "use": "sig", "x": coordinate(key.X), "y": coordinate(key.Y)}
	return caepTransmitter{key: key, jwks: map[string]any{"keys": []any{jwk}}}
}

func (c caepTransmitter) sign(t *testing.T, claims []byte) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","kid":"caep-example","typ":"secevent+jwt"}`))
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func loadCAEPExamples(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/caep-1_0-examples.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

type caepReceiver struct {
	deps    ssusecases.ReceiveDeps
	epochs  *ssmemory.UserRevocationEpochRepository
	ended   *[]string
	events  *[]spec.DomainEvent
	ctx     context.Context
	example testing_httprecord.Exchange
}

// newCAEPReceiver は、例の iss を信頼する受信側のストリームと、例の主体の識別子を持つ User を置く。
func newCAEPReceiver(t *testing.T, transmitter caepTransmitter, example testing_httprecord.Exchange, userIDs ...string) caepReceiver {
	t.Helper()
	var claims struct {
		Iss string `json:"iss"`
		Aud any    `json:"aud"`
	}
	if err := json.Unmarshal(example.Request.Body, &claims); err != nil {
		t.Fatal(err)
	}
	audience, _ := claims.Aud.(string)
	ctx := context.Background()
	streams, configs := ssmemory.NewSsfStreamRepository(), ssmemory.NewSsfReceiverConfigRepository()
	if err := streams.Save(ctx, &ssdomain.SsfStream{
		ID: caepStream, TenantID: caepTenant, Direction: ssdomain.SsfStreamDirectionReceive,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := configs.Save(ctx, caepTenant, &ssdomain.SsfReceiverConfig{
		StreamID: caepStream, TrustedIssuer: claims.Iss, JWKS: transmitter.jwks, AcceptedAudiences: []string{audience},
	}); err != nil {
		t.Fatal(err)
	}
	users := idmanagement.NewMemoryModule().UserRepo
	now := time.Now().UTC()
	for _, id := range userIDs {
		if err := users.Save(ctx, &userdomain.User{ID: id, TenantID: caepTenant, PreferredUsername: id, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	r := caepReceiver{epochs: ssmemory.NewUserRevocationEpochRepository(), ended: &[]string{}, events: &[]spec.DomainEvent{}, example: example}
	r.deps = ssusecases.ReceiveDeps{
		StreamRepo: streams, ReceiverConfigRepo: configs, ReceivedEventRepo: ssmemory.NewReceivedSecurityEventRepository(),
		EpochRepo: ssmemory.NewAgentRevocationEpochRepository(), AgentRepo: idmanagement.NewMemoryModule().AgentRepo,
		Users: users, UserEpochRepo: r.epochs,
		EndUserSessions: func(_ context.Context, userID string) error { *r.ended = append(*r.ended, userID); return nil },
		Verifier:        &verify_jose.Verifier{JWKResolver: tokens_jose.NewJWKResolver()},
		Emit:            func(event spec.DomainEvent) error { *r.events = append(*r.events, event); return nil },
	}
	r.ctx = tenantports.WithTenant(ctx, &tenancydomain.Tenant{ID: caepTenant}, "", "")
	return r
}

func (r caepReceiver) receive(t *testing.T, transmitter caepTransmitter) error {
	t.Helper()
	return ssusecases.ReceiveSecurityEvent(r.ctx, r.deps, caepStream, transmitter.sign(t, r.example.Request.Body), time.Now().UTC())
}

func (r caepReceiver) assertNotReflected(t *testing.T, err error, userIDs ...string) {
	t.Helper()
	if !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
		t.Fatalf("%s: err = %v, want ErrSecurityEventRejected", r.example.Name, err)
	}
	if len(*r.ended) != 0 {
		t.Fatalf("%s: a rejected SET ended the sessions of %v", r.example.Name, *r.ended)
	}
	for _, id := range userIDs {
		if epoch, _ := r.epochs.FindByUser(context.Background(), caepTenant, id); epoch != nil {
			t.Fatalf("%s: a rejected SET advanced the revocation epoch of %s", r.example.Name, id)
		}
	}
	rejected := false
	for _, event := range *r.events {
		if e, ok := event.(*ssdomain.SecurityEventRejected); ok && e.VerificationResult == ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
			rejected = true
		}
	}
	if !rejected {
		t.Fatalf("%s: events %v, want SecurityEventRejected with rejected_subject_unresolved", r.example.Name, *r.events)
	}
}

// caepNotReplayed は、CAEP の例のうち再生しない往復と、その理由である。
var caepNotReplayed = map[string]string{
	"credential-change-fido2-create":  "受け付けた SET はイベントの種類を問わず主体を失効させる。credential-change の作成の通知で失効させないよう、種類ごとの反映を wi-32902 が決める",
	"assurance-level-change-increase": "同上（保証レベルが上がった通知）",
	"assurance-level-change-custom":   "同上（独自の保証レベルの通知）",
}

// 期待値は CAEP 1.0 の例の意味と、IdMagic が宣言した採用の範囲から書く。
// session-revoked の opaque の主体はセッションの失効を意味し（CAEP 1.0 §3.1）、IdMagic は User を失効させてセッションを終わらせる。
// complex、email、jwt_id、saml_assertion_id の主体は IdMagic が解釈しない形式であり（SSF10-RECEIVE-SUB-ID、RFC9493-SUBID-FORMAT）、
// sub_id の iss が SET の iss と食い違う iss_sub は信頼する発行者と一致しない（RFC9493-SUBID-ISS-SUB）。
// 署名は仕様の例が示さないのでテストの鍵で付け、検証は本物の verifier が行う。
//
//spec:covers SSF10-RECEIVE-SUB-ID: CAEP 1.0 の例の最上位の opaque の sub_id で、イベントに subject がなくても User を解決して失効させ、complex の sub_id を拒否すること,
//spec:covers RFC9493-SUBID-FORMAT: CAEP 1.0 の例の email、jwt_id、saml_assertion_id、complex の主体を、主体を解決できないものとして拒否し、失効を反映しないこと,
//spec:covers RFC9493-SUBID-ISS-SUB: CAEP 1.0 の risk-level-change の例のように sub_id の iss が信頼する発行者と食い違う iss_sub を拒否すること,
//spec:covers RFC8417-SET-VERIFY: 例の claims を署名した SET を本物の verifier で検証してから反映すること
func TestReceiveSecurityEvent_CAEPExamples(t *testing.T) {
	rec := loadCAEPExamples(t)
	transmitter := newCAEPTransmitter(t)
	const sessionSubject = "dMTlD|1600802906337.16|16008.16"
	// 例の主体が指しうる識別子を、どれも受信側のテナントの User として置く。拒否が解決できる主体の不在によるものでないことを確かめるためである。
	everyone := []string{sessionSubject, "99beb27c-c1c2-4955-882a-e0dc4996fcbc", "jane.smith@example.com", "jane.doe@example.com", "e9297990-14d2-42ec-a4a9-4036db86509a"}

	t.Run("session-revoked-opaque", func(t *testing.T) {
		r := newCAEPReceiver(t, transmitter, rec.Exchange(t, "session-revoked-opaque"), everyone...)
		if err := r.receive(t, transmitter); err != nil {
			t.Fatalf("ReceiveSecurityEvent: %v", err)
		}
		if len(*r.ended) != 1 || (*r.ended)[0] != sessionSubject {
			t.Fatalf("ended sessions = %v, want only %s", *r.ended, sessionSubject)
		}
		if epoch, _ := r.epochs.FindByUser(context.Background(), caepTenant, sessionSubject); epoch == nil || epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
			t.Fatalf("epoch = %+v, want an InboundSecurityEvent revocation", epoch)
		}
	})

	for _, name := range []string{
		"session-revoked-complex-user-iss-sub", "session-revoked-complex-user-device-tenant", "device-compliance-change",
		"token-claims-change-oidc", "token-claims-change-oidc-optional", "token-claims-change-saml",
		"session-established", "session-presented", "risk-level-change",
	} {
		t.Run(name, func(t *testing.T) {
			r := newCAEPReceiver(t, transmitter, rec.Exchange(t, name), everyone...)
			r.assertNotReflected(t, r.receive(t, transmitter), everyone...)
		})
	}

	replayed := map[string]bool{"session-revoked-opaque": true}
	for _, name := range []string{
		"session-revoked-complex-user-iss-sub", "session-revoked-complex-user-device-tenant", "device-compliance-change",
		"token-claims-change-oidc", "token-claims-change-oidc-optional", "token-claims-change-saml",
		"session-established", "session-presented", "risk-level-change",
	} {
		replayed[name] = true
	}
	for _, ex := range rec.Exchanges {
		if _, skipped := caepNotReplayed[ex.Name]; replayed[ex.Name] == skipped {
			t.Errorf("example %q must be either replayed or listed with a reason, not both or neither", ex.Name)
		}
	}
}
