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
	saved   *[]ssdomain.ReceivedSecurityEvent
	ctx     context.Context
	example testing_httprecord.Exchange
}

// savingReceivedEvents は保存した受信の記録を観測できるようにする。
type savingReceivedEvents struct {
	*ssmemory.ReceivedSecurityEventRepository
	saved *[]ssdomain.ReceivedSecurityEvent
}

func (s savingReceivedEvents) Save(ctx context.Context, event *ssdomain.ReceivedSecurityEvent) error {
	*s.saved = append(*s.saved, *event)
	return s.ReceivedSecurityEventRepository.Save(ctx, event)
}

// newCAEPReceiver は、例の iss を信頼し eventTypes を購読する受信側のストリームと、例の主体の識別子を持つ User を置く。
func newCAEPReceiver(t *testing.T, transmitter caepTransmitter, example testing_httprecord.Exchange, eventTypes []ssdomain.CaepEventType, userIDs ...string) caepReceiver {
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
		EventTypes: eventTypes, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: time.Now().UTC(),
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
	r := caepReceiver{epochs: ssmemory.NewUserRevocationEpochRepository(), ended: &[]string{}, events: &[]spec.DomainEvent{}, saved: &[]ssdomain.ReceivedSecurityEvent{}, example: example}
	r.deps = ssusecases.ReceiveDeps{
		StreamRepo: streams, ReceiverConfigRepo: configs,
		ReceivedEventRepo: savingReceivedEvents{ReceivedSecurityEventRepository: ssmemory.NewReceivedSecurityEventRepository(), saved: r.saved},
		EpochRepo:         ssmemory.NewAgentRevocationEpochRepository(), AgentRepo: idmanagement.NewMemoryModule().AgentRepo,
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

// assertLeftUntouched は、受理した SET が User の失効エポック、セッション、受信を知らせるイベントを変えなかったことを確かめる。
func (r caepReceiver) assertLeftUntouched(t *testing.T, err error, userIDs ...string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: ReceiveSecurityEvent: %v, want accepted", r.example.Name, err)
	}
	if len(*r.ended) != 0 {
		t.Fatalf("%s: ended the sessions of %v", r.example.Name, *r.ended)
	}
	for _, id := range userIDs {
		if epoch, _ := r.epochs.FindByUser(context.Background(), caepTenant, id); epoch != nil {
			t.Fatalf("%s: advanced the revocation epoch of %s", r.example.Name, id)
		}
	}
	if len(*r.events) != 0 {
		t.Fatalf("%s: events %v, want none", r.example.Name, *r.events)
	}
}

var (
	sessionRevokedOnly = []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}
	everyKnownType     = []ssdomain.CaepEventType{
		ssdomain.CaepEventTypeSessionRevoked, ssdomain.CaepEventTypeTokenClaimsChange,
		ssdomain.CaepEventTypeCredentialChange, ssdomain.CaepEventTypeAssuranceLevelChange,
	}
)

// 期待値は CAEP 1.0 の例の意味と、IdMagic が宣言した採用の範囲から書く。
// session-revoked の opaque の主体はセッションの失効を意味し（CAEP 1.0 §3.1）、IdMagic は User を失効させてセッションを終わらせる。
// credential-change と assurance-level-change は失効を意味しないので、購読しなければ無視し、購読すれば受理だけを記録する（REQ-SHAREDSIGNALS-019）。
// IdMagic の CaepEventType にない種類は、主体の形式によらず無視する。
// complex、jwt_id、saml_assertion_id の主体は IdMagic が解釈しない形式である（SSF10-RECEIVE-SUB-ID、RFC9493-SUBID-FORMAT）。
// 署名は仕様の例が示さないのでテストの鍵で付け、検証は本物の verifier が行う。
//
//spec:covers SSF10-RECEIVE-SUB-ID: CAEP 1.0 の例の最上位の opaque の sub_id で、イベントに subject がなくても User を解決して失効させ、complex の sub_id を拒否すること,
//spec:covers RFC9493-SUBID-FORMAT: 購読する種類の CAEP 1.0 の例の jwt_id、saml_assertion_id、complex の主体を、主体を解決できないものとして拒否し、失効を反映しないこと,
//spec:covers RFC8417-SET-VERIFY: 例の claims を署名した SET を本物の verifier で検証してから反映すること,
//spec:covers REQ-SHAREDSIGNALS-019: CAEP 1.0 の例を種類と購読ごとに、無視、記録だけ、失効のどれかで反映すること,
//spec:covers EX-SHAREDSIGNALS-019-01: session-revoked だけを購読するストリームで credential-change と assurance-level-change の例を受けても、記録も失効も残さないこと,
//spec:covers EX-SHAREDSIGNALS-019-03: 購読する assurance-level-change と credential-change の例は、reflected_at のない受理の記録だけを残し、User を失効させないこと
func TestReceiveSecurityEvent_CAEPExamples(t *testing.T) {
	rec := loadCAEPExamples(t)
	transmitter := newCAEPTransmitter(t)
	const sessionSubject = "dMTlD|1600802906337.16|16008.16"
	// 例の主体が指しうる識別子を、どれも受信側のテナントの User として置く。拒否が解決できる主体の不在によるものでないことを確かめるためである。
	everyone := []string{sessionSubject, "99beb27c-c1c2-4955-882a-e0dc4996fcbc", "jane.smith@example.com", "jane.doe@example.com", "e9297990-14d2-42ec-a4a9-4036db86509a"}
	replayed := map[string]bool{}

	t.Run("session-revoked-opaque", func(t *testing.T) {
		replayed["session-revoked-opaque"] = true
		r := newCAEPReceiver(t, transmitter, rec.Exchange(t, "session-revoked-opaque"), sessionRevokedOnly, everyone...)
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
		"session-revoked-complex-user-iss-sub", "session-revoked-complex-user-device-tenant",
		"token-claims-change-oidc", "token-claims-change-oidc-optional", "token-claims-change-saml",
	} {
		replayed[name] = true
		t.Run("subscribed/"+name, func(t *testing.T) {
			r := newCAEPReceiver(t, transmitter, rec.Exchange(t, name), everyKnownType, everyone...)
			r.assertNotReflected(t, r.receive(t, transmitter), everyone...)
		})
	}

	for _, name := range []string{"credential-change-fido2-create", "assurance-level-change-increase", "assurance-level-change-custom"} {
		replayed[name] = true
		t.Run("unsubscribed/"+name, func(t *testing.T) {
			r := newCAEPReceiver(t, transmitter, rec.Exchange(t, name), sessionRevokedOnly, everyone...)
			r.assertLeftUntouched(t, r.receive(t, transmitter), everyone...)
			if len(*r.saved) != 0 {
				t.Fatalf("records = %+v, want none for an unsubscribed type", *r.saved)
			}
		})
		t.Run("subscribed/"+name, func(t *testing.T) {
			r := newCAEPReceiver(t, transmitter, rec.Exchange(t, name), everyKnownType, everyone...)
			r.assertLeftUntouched(t, r.receive(t, transmitter), everyone...)
			if len(*r.saved) != 1 {
				t.Fatalf("records = %+v, want one accepted record", *r.saved)
			}
			record := (*r.saved)[0]
			if record.VerificationResult != ssdomain.SecurityEventVerificationAccepted || record.ReflectedAt != nil ||
				record.Subject.SubjectType != ssdomain.SsfSubjectTypeUser || record.Subject.PrincipalID != "jane.smith@example.com" {
				t.Fatalf("record = %+v, want an accepted record for jane.smith@example.com without reflected_at", record)
			}
		})
	}

	// 主体の形式（complex、email、sub_id の iss の食い違い）は、種類を無視するので問わない。
	for _, name := range []string{"device-compliance-change", "session-established", "session-presented", "risk-level-change"} {
		replayed[name] = true
		t.Run("unknown/"+name, func(t *testing.T) {
			r := newCAEPReceiver(t, transmitter, rec.Exchange(t, name), everyKnownType, everyone...)
			r.assertLeftUntouched(t, r.receive(t, transmitter), everyone...)
			if len(*r.saved) != 0 {
				t.Fatalf("records = %+v, want none for an unknown type", *r.saved)
			}
		})
	}

	for _, ex := range rec.Exchanges {
		if !replayed[ex.Name] {
			t.Errorf("example %q is not replayed", ex.Name)
		}
	}
}
