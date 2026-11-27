package envelope_openbao

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// transitTwin は、transit にマウントした OpenBao の Transit の aes256-gcm96 の鍵の模倣である。
//
// 要求をまたぐ状態（鍵のバージョン、運用者の回転、min_decryption_version）で結果が決まる検査だけに使う。
// 応答の形は OpenBao から取った記録と一致することを TestTransitTwin_MatchesOpenBaoRecording が確かめる。
// 拒否、5xx、不正な本文のように一つの応答で決まる分岐は持たない。それは要求ごとに応答を返す関数で作る。
type transitTwin struct {
	mu       sync.Mutex
	keys     map[string]*twinKey
	requests int
}

type twinKey struct {
	versions      [][]byte // 1 から数えるバージョンごとの AES-256 の鍵
	createdAt     []int64
	minDecryption int
}

func newTransitTwin() *transitTwin {
	return &transitTwin{keys: map[string]*twinKey{}}
}

func (tw *transitTwin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.requests++
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/transit/"), "/")
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	switch {
	case len(parts) == 2 && parts[0] == "keys" && r.Method == http.MethodGet:
		key, ok := tw.keys[parts[1]]
		if !ok {
			tw.writeErrors(w, http.StatusNotFound)
			return
		}
		tw.writeKey(w, parts[1], key)
	case len(parts) == 2 && parts[0] == "keys" && r.Method == http.MethodPost:
		tw.writeKey(w, parts[1], tw.ensure(parts[1]))
	case len(parts) == 3 && parts[0] == "keys" && parts[2] == "rotate" && r.Method == http.MethodPost:
		key := tw.ensure(parts[1])
		key.addVersion()
		tw.writeKey(w, parts[1], key)
	case len(parts) == 3 && parts[0] == "keys" && parts[2] == "config" && r.Method == http.MethodPost:
		key := tw.ensure(parts[1])
		if v, ok := body["min_decryption_version"].(float64); ok {
			key.minDecryption = int(v)
		}
		tw.writeKey(w, parts[1], key)
	case len(parts) == 2 && parts[0] == "encrypt" && r.Method == http.MethodPost:
		tw.encrypt(w, tw.ensure(parts[1]), body)
	case len(parts) == 2 && parts[0] == "decrypt" && r.Method == http.MethodPost:
		key, ok := tw.keys[parts[1]]
		if !ok {
			tw.writeErrors(w, http.StatusBadRequest, "encryption key not found")
			return
		}
		tw.decrypt(w, key, body)
	default:
		tw.writeErrors(w, http.StatusNotFound, "1 error occurred:\n\t* unsupported path\n\n")
	}
}

// ensure は鍵を返し、なければバージョン 1 の鍵を作る。OpenBao の作成と、鍵のない暗号化の upsert に合わせる。
func (tw *transitTwin) ensure(name string) *twinKey {
	key, ok := tw.keys[name]
	if !ok {
		key = &twinKey{minDecryption: 1}
		key.addVersion()
		tw.keys[name] = key
	}
	return key
}

func (k *twinKey) addVersion() {
	material := make([]byte, 32)
	_, _ = rand.Read(material)
	k.versions = append(k.versions, material)
	k.createdAt = append(k.createdAt, time.Now().Unix())
}

func (k *twinKey) aead(version int) cipher.AEAD {
	block, err := aes.NewCipher(k.versions[version-1])
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return gcm
}

func (tw *transitTwin) encrypt(w http.ResponseWriter, key *twinKey, body map[string]any) {
	plaintext, err := base64.StdEncoding.DecodeString(fmt.Sprint(body["plaintext"]))
	if err != nil {
		tw.writeErrors(w, http.StatusBadRequest, "failed to base64-decode plaintext")
		return
	}
	version := len(key.versions)
	gcm := key.aead(version)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = rand.Read(nonce)
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	tw.writeData(w, map[string]any{
		"ciphertext":  "vault:v" + strconv.Itoa(version) + ":" + base64.StdEncoding.EncodeToString(sealed),
		"key_version": version,
	})
}

func (tw *transitTwin) decrypt(w http.ResponseWriter, key *twinKey, body map[string]any) {
	fields := strings.SplitN(fmt.Sprint(body["ciphertext"]), ":", 3)
	if len(fields) != 3 || fields[0] != "vault" || !strings.HasPrefix(fields[1], "v") {
		tw.writeErrors(w, http.StatusBadRequest, "invalid ciphertext: no prefix")
		return
	}
	version, err := strconv.Atoi(strings.TrimPrefix(fields[1], "v"))
	if err != nil || version < 1 || version > len(key.versions) {
		tw.writeErrors(w, http.StatusBadRequest, "invalid ciphertext: version")
		return
	}
	if version < key.minDecryption {
		tw.writeErrors(w, http.StatusBadRequest, "ciphertext or signature version is disallowed by policy (too old)")
		return
	}
	sealed, err := base64.StdEncoding.DecodeString(fields[2])
	gcm := key.aead(version)
	if err != nil || len(sealed) < gcm.NonceSize() {
		tw.writeErrors(w, http.StatusBadRequest, "invalid ciphertext length")
		return
	}
	plaintext, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
	if err != nil {
		tw.writeErrors(w, http.StatusBadRequest, "cipher: message authentication failed")
		return
	}
	tw.writeData(w, map[string]any{"plaintext": base64.StdEncoding.EncodeToString(plaintext)})
}

// writeKey は鍵の読み取り、作成、回転、設定に共通の応答を書く。keys は min_decryption_version 以降のバージョンだけを持つ。
func (tw *transitTwin) writeKey(w http.ResponseWriter, name string, key *twinKey) {
	versions := map[string]any{}
	for v := key.minDecryption; v <= len(key.versions); v++ {
		versions[strconv.Itoa(v)] = key.createdAt[v-1]
	}
	tw.writeData(w, map[string]any{
		"allow_plaintext_backup": false, "auto_rotate_period": 0, "deletion_allowed": false, "derived": false,
		"exportable": false, "imported_key": false, "keys": versions, "latest_version": len(key.versions),
		"min_available_version": 0, "min_decryption_version": key.minDecryption, "min_encryption_version": 0,
		"name": name, "soft_deleted": false, "supports_decryption": true, "supports_derivation": true,
		"supports_encryption": true, "supports_signing": false, "type": "aes256-gcm96",
	})
}

func (tw *transitTwin) writeData(w http.ResponseWriter, data map[string]any) {
	tw.writeJSON(w, http.StatusOK, map[string]any{
		"request_id": "twin-request-" + strconv.Itoa(tw.requests), "lease_id": "", "renewable": false, "lease_duration": 0,
		"data": data, "wrap_info": nil, "warnings": nil, "auth": nil,
	})
}

func (tw *transitTwin) writeErrors(w http.ResponseWriter, status int, messages ...string) {
	if messages == nil {
		messages = []string{}
	}
	tw.writeJSON(w, status, map[string]any{"errors": messages})
}

func (tw *transitTwin) writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// divergeFromRecording は、記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// request_id と暗号文、鍵のバージョンごとの作成時刻は相手が割り当てるので型だけを比べる。
// 暗号化のバージョンは key_version の値で比べる。記録の暗号文は、模倣が同じ往復で返した暗号文へ読み替えて送る。
func divergeFromRecording(twin http.Handler, rec testing_httprecord.Recording) []string {
	ciphertexts := map[string]string{}
	return testing_httprecord.Fidelity{
		VolatileKeys: map[string]bool{"request_id": true, "ciphertext": true, "1": true, "2": true},
		Headers:      []string{"Content-Type"},
		ExactHeaders: []string{"Cache-Control"},
		Rewrite: func(s string) string {
			for recorded, twin := range ciphertexts {
				s = strings.ReplaceAll(s, recorded, twin)
			}
			return s
		},
		Learn: func(want, got any) {
			if recorded, twin := ciphertextIn(want), ciphertextIn(got); recorded != "" && twin != "" {
				ciphertexts[recorded] = twin
			}
		},
	}.Diverge(twin, rec)
}

func ciphertextIn(body any) string {
	object, _ := body.(map[string]any)
	data, _ := object["data"].(map[string]any)
	value, _ := data["ciphertext"].(string)
	return value
}

// 運用者が OpenBao でマスター鍵を回転しても、それまでにラップした DEK は戻せ、新しいラップは新しいバージョンを使う。
// 運用者が min_decryption_version を上げると、それより古いバージョンでラップした DEK は ErrDataKeyUnavailable で戻せなくなる。
// 回転と設定は運用者の操作であり、アダプターは送らないので、OpenBao の HTTP API へ直接送る。
func TestOpenBaoMasterKeyProvider_FollowsOperatorKeyVersions(t *testing.T) {
	server := httptest.NewServer(newTransitTwin())
	defer server.Close()
	operator := func(path, body string) {
		t.Helper()
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/v1/transit/keys/"+DefaultKeyNamePrefix+"-tenant-a"+path, strings.NewReader(body))
		resp, err := server.Client().Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("operator %s: %v %v", path, resp, err)
		}
		_ = resp.Body.Close()
	}
	provider := NewOpenBaoMasterKeyProvider(NewHTTPTransitEngine(server.URL, recordedToken, ""), DefaultKeyNamePrefix)
	ctx := t.Context()
	dek := []byte("0123456789abcdef0123456789abcdef")

	before, masterKeyID, err := provider.WrapDataKey(ctx, "tenant-a", dek)
	if err != nil {
		t.Fatal(err)
	}
	operator("/rotate", "")
	after, _, err := provider.WrapDataKey(ctx, "tenant-a", dek)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(before), "vault:v1:") || !strings.HasPrefix(string(after), "vault:v2:") {
		t.Fatalf("wrapped before = %q, after = %q; want versions 1 and 2", before, after)
	}
	for _, wrapped := range [][]byte{before, after} {
		if got, err := provider.UnwrapDataKey(ctx, "tenant-a", wrapped, masterKeyID); err != nil || !bytes.Equal(got, dek) {
			t.Fatalf("unwrap %q after rotation = %q, %v; want the DEK", wrapped, got, err)
		}
	}

	operator("/config", `{"min_decryption_version":2}`)
	if _, err := provider.UnwrapDataKey(ctx, "tenant-a", before, masterKeyID); !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("unwrap below min_decryption_version = %v, want ErrDataKeyUnavailable", err)
	}
	if got, err := provider.UnwrapDataKey(ctx, "tenant-a", after, masterKeyID); err != nil || !bytes.Equal(got, dek) {
		t.Fatalf("unwrap at min_decryption_version = %q, %v; want the DEK", got, err)
	}
}

// 模倣は、OpenBao から取った記録の全往復（運用者の回転と min_decryption_version の設定を含む）に記録と同じ応答を返す。
func TestTransitTwin_MatchesOpenBaoRecording(t *testing.T) {
	for _, diff := range divergeFromRecording(newTransitTwin(), openBaoRecording(t)) {
		t.Error(diff)
	}
}

// 照合が空虚でないことを確かめる。模倣の応答を記録から外すと、照合が食い違いを報告する。
func TestTransitTwin_DivergeReportsTwinThatLeavesTheRecording(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(w http.ResponseWriter, r *http.Request, twin http.Handler)
		want   string
	}{
		{"min_decryption_version を無視する", func(w http.ResponseWriter, r *http.Request, twin http.Handler) {
			rec := httptest.NewRecorder()
			twin.ServeHTTP(rec, r)
			status := rec.Code
			if status == http.StatusBadRequest && strings.Contains(rec.Body.String(), "too old") {
				status = http.StatusOK
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(status)
			_, _ = w.Write(rec.Body.Bytes())
		}, "decrypt-v1-below-min-version: status = 200, recorded 400"},
		{"Cache-Control を返さない", func(w http.ResponseWriter, r *http.Request, twin http.Handler) {
			rec := httptest.NewRecorder()
			twin.ServeHTTP(rec, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
		}, `create-key: header Cache-Control = "", recorded "no-store"`},
		{"暗号化が key_version を返さない", func(w http.ResponseWriter, r *http.Request, twin http.Handler) {
			rec := httptest.NewRecorder()
			twin.ServeHTTP(rec, r)
			var body map[string]any
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if data, ok := body["data"].(map[string]any); ok {
				delete(data, "key_version")
			}
			data, err := json.Marshal(body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(rec.Code)
			_, _ = w.Write(data)
		}, "encrypt-v1: body.data.key_version is missing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			twin := newTransitTwin()
			mutated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { tc.mutate(w, r, twin) })
			diffs := divergeFromRecording(mutated, openBaoRecording(t))
			if !strings.Contains(strings.Join(diffs, "\n"), tc.want) {
				t.Errorf("divergeFromRecording() = %q, want a line %q", diffs, tc.want)
			}
		})
	}
}
