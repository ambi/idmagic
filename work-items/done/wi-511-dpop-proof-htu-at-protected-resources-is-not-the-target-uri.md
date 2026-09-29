---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-07
priority: p1
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 保護リソースが絶対 URL の `htu` を受理し、パスだけの `htu` を拒否するようになる。subdomain style のテナントでは `/token` と `/userinfo` の期待値も正規ロケーションの URL へ変わる。DPoP を使うクライアントの読み手には振る舞いの変化として見える。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri.md }
initial_context:
  specification:
    - docs/domain/api-tokens/standards.md#RFC9449-API-TOKEN-DPOP
    - docs/domain/oauth2/internals.md
  typespec:
    - IdMagic.Contract.DpopProofClaims
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/security/tokens_jose/dpop_verifier.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/userinfo_handler.go
  tests:
    - backend/shared/http/server_http/api_token_standards_test.go
    - backend/shared/http/support_http/tenant_middleware_test.go
    - backend/shared/http/server_http/tenant_host_routes_test.go
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: docs/domain/api-tokens/standards.md, requirement: RFC9449-API-TOKEN-DPOP }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.DpopProofClaims }
primary_use_cases:
  - id: dpop-bound-api-token-reaches-protected-resource
    requirement: RFC9449-API-TOKEN-DPOP
    observable_result: 絶対 URL の `htu` を持つ DPoP 証明を添えた送信者制約付き API アクセストークンが、path style と subdomain style のどちらのテナントでも保護された管理 API から利用者の一覧を得る。パスだけの `htu` と別リソースの絶対 URL の `htu` は到達しない。
    unit_test: { path: backend/shared/http/support_http/tenant_middleware_test.go, name: TestRequestHTUIsTheTargetURIAtTheCanonicalLocation, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/api_token_standards_test.go, name: TestDPoPBoundApiTokenAcceptsTheAbsoluteTargetURIAtEitherTenantStyle, task: test-go-race }
    unit_fault_model: '`RequestHTU` が正規 issuer 全体にリクエストのパスを継ぎ、path style で `/realms/{realm}` が二重になる。'
    e2e_fault_model: 保護リソースの検証がテナントの正規ロケーションを期待値へ渡さず、パスだけを `htu` として照合する（配線が外れる）。
---

# 保護リソースが DPoP 証明の `htu` をパスとして照合するので、適合クライアントが到達できない

## Motivation

[[wi-500-back-api-tokens-standards-rows-with-tests]] が `RFC9449-API-TOKEN-DPOP` にテストを対応付けているときに見つけた。

`backend/shared/http/support_http/auth.go:150` は、保護リソースの DPoP 証明を検証するとき `RequestHTU(c, "")` を期待値として渡す。`RequestHTU` は `strings.TrimRight(base, "/") + c.Request().URL.Path` なので、base が空文字なら返るのはパスだけ、たとえば `/realms/default/api/admin/v1/users` である。`verifyDPoP` は証明の `htu` をこれと完全一致で照合する。

同じ関数を、トークンエンドポイント（`backend/oauth2/handlers_http/token_handler.go:84`）と `/userinfo`（`userinfo_handler.go:77`）は `RequestHTU(c, d.Issuer)` で呼ぶ。こちらは絶対 URL になる。**同じ製品の中で、DPoP 証明の `htu` に 2 つの別々の形を要求している。**

RFC 9449 §4.2 は `htu` を「the HTTP target URI, without query and fragment parts」と定めており、`spec/contexts/oauth2/models.tsp:607` も `htu: url` と宣言している。したがって適合クライアントは絶対 URL を送る。それは保護リソースで必ず `htu mismatch` になる。

**送信者制約付きの API アクセストークンは、適合クライアントからは 1 度も使えない。** `/token` で受け取ったトークンを持って保護リソースへ行くと、同じクライアントが同じ規則で作った証明が今度は拒否される。この経路を通せるのは、保護リソースにだけパスを送るように作り分けた非適合クライアントだけである。

`docs/domain/oauth2/internals.md:37` は「Proof の検証はパラメーター化せず、エンドポイントの種類で分ける」と書いているが、分けると書いてあるのは `ath` の要否であって `htu` の形ではない。`htu` がエンドポイントごとに違う形になることは、どの正典文書にも書かれていない。

## Scope

- 保護リソースの `htu` 期待値を、他のエンドポイントと同じ絶対 URL に揃える。
- 絶対 URL を持つ証明が保護リソースで受理されること、およびパスだけの証明が受理されないことを、保護されたエンドポイントから観測する。
- テナントの正規ロケーションが 2 形（`/realms/{realm}` と subdomain）あることを踏まえて期待値を組み立てる。`RequestHTU(c, d.Issuer)` を単純に写すと、path style で prefix が二重になる（`TenantURL` の doc コメントが同じ罠を挙げている）。
- `backend/shared/http/server_http/api_token_standards_test.go` の `TestDPoPBoundApiTokenVerifiesEveryProofElement` が、いまパスを無傷の値として使っている注記を落とす。

## Out of Scope

- `htm`、`iat`、`jti`、`ath`、サムプリントの検証。いずれも期待どおり働いている。
- mTLS による送信者制約。

## Design

**期待値はテナントの正規 issuer から組み、リクエストの `Host` からは組まない。** Risk Notes の 2 案のうち、正規 issuer を採る。正規ロケーションは discovery が広告する URL と同じ出所なので、クライアントが discovery から得た URL へそのまま送れば一致する。`Host` から組むと、TLS 終端プロキシが Host を書き換える構成で期待値がクライアントの URL とずれ、またヘッダーを偽装すれば期待値を動かせる。テナント解決のミドルウェアは Host と正規ロケーションの一致を既に確かめているので、正規 issuer を使っても到達可能な URL の集合は狭まらない。

**正規 issuer 全体ではなく、その origin にリクエストのパスを継ぐ。** path style の issuer は `https://idp/realms/acme` であり、リクエストのパスは `/realms/acme/...` を含むので、issuer 全体へ継ぐと prefix が二重になる。subdomain style の issuer は `https://acme.idp` でパスを持たない。origin（scheme と authority）だけを取り出せば、どちらの形でもクライアントが送った URL を復元できる。

```go
// backend/shared/http/support_http
// RequestHTU はテナントの正規ロケーションの origin にリクエストのパスを継いだ絶対 URL を返す。
// 文脈にテナントが無いときは fallback を issuer として使う。origin を決められなければ "" を返す。
func RequestHTU(c *echo.Context, fallback string) string
```

入力は文脈上のテナント issuer（`tenancy.Issuer`）、`fallback`、リクエストのパスだけであり、時刻も乱数も読まない。保護リソース（`auth.go`）は fallback を持たないので `""` を渡す。保護リソースへは必ずテナント解決を通って到達するので、文脈の issuer がある。

**`/token` と `/userinfo` も同じ関数を通す。** 両者はいま `RequestHTU(c, d.Issuer)` で基底の issuer を前置している。path style ではたまたま正しいが、subdomain style では `https://idp/token` を期待し、クライアントの `https://acme.idp/token` と一致しない。同じ欠陥の別の現れであり、関数の定義を直せば 3 つの呼び出し元が 1 つの規則にそろう。`d.Issuer` は文脈にテナントが無い場合の fallback として残す。

**期待値が空、または証明が `htu` を持たない場合は拒否する。** `verifyDPoP` はいま `payload["htu"]` と期待値を文字列比較するだけなので、期待値が `""` のとき `htu` を持たない証明が一致する。`RequestHTU` が origin を決められず `""` を返す経路を fail-closed にするため、`tokens_jose` の照合で両方が空でないことを求める。

**却下した案**: 保護リソースだけに別関数を置き、`/token` と `/userinfo` はそのままにする。subdomain style の `/token` の欠陥が残り、`htu` の形がエンドポイントごとに違う状態が続く。これは Motivation が問題にした状態そのものである。

## Plan

1. `docs/domain/oauth2/internals.md` に `htu` の期待値の規則を書き、`mise run check-spec` を通す。
2. E2E RED を先に固定する。`api_token_standards_test.go` の無傷の証明を絶対 URL へ変え、subdomain style のテナントとパスだけの `htu` を扱う新しいテストを足す。
3. `RequestHTU` の Unit RED → GREEN。`tokens_jose` の空 `htu` 拒否の Unit RED → GREEN。
4. `auth.go` の呼び出しを直して E2E を GREEN にし、既存の `/token` と `/userinfo` のテストが GREEN のままであることを確かめる。
5. リリースノートを書き、変異テストと `mise run verify`。

## Tasks

- [x] T001 [Spec] `docs/domain/oauth2/internals.md` に `htu` の期待値の規則を書く。
  recipe: `mise run check-spec`
- [x] T002 [Acceptance] E2E RED を観測する。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http 'TestDPoPBoundApiToken|TestApiTokenSenderConstraintIsChosenAtIssuance'`
- [x] T003 [Adapters] `RequestHTU` を正規ロケーションの origin から組む。
  recipe: `mise run test-go-test -- ./backend/shared/http/support_http TestRequestHTUIsTheTargetURIAtTheCanonicalLocation`
- [x] T004 [Adapters] `verifyDPoP` が空の `htu` を拒否する。
  recipe: `mise run test-go-test -- ./backend/shared/security/tokens_jose TestVerifyDPoPRejectsAMissingHTUEvenWhenNoTargetIsExpected`
- [x] T005 [Adapters] 保護リソースの検証が `RequestHTU` の新しい規則を通って E2E が GREEN になることを確かめる。`auth.go` は既に `RequestHTU(c, "")` を呼んでいたので呼び出しは変えず、パスだけの `htu` を送っていた `auth_test.go` の証明を絶対 URL へ直す。
  recipe: `mise run test-go-changed -- main`
- [x] T006 [Docs] `docs/releases/changes/wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri.md` を書く。
- [x] T007 [Verify] `mise run test-go-mutation -- backend/shared/http/support_http`、`mise run verify`。

## Risk Notes

- **期待値を絶対 URL にすると、prefix の二重化で全経路が落ちる。** テナントの 2 形それぞれについて、クライアントが送ったままの URL を復元できていることを観測する。
- **リバースプロキシ越しのスキームとホスト。** 期待値をリクエストの `Host` から組むか、テナントの正規 issuer から組むかで、プロキシ配下の振る舞いが変わる。どちらを採るかを Design で決める。
- **互換性とロールバック。** パスだけの `htu` を送る非適合クライアントは保護リソースで拒否されるようになる。製品は未リリースなので移行は要らない。永続データも公開契約の形も変えないので、ロールバックはコードを戻すだけで済む。
- **セキュリティ。** 照合を緩める変更ではない。期待値の形を 1 つに固定し、空の期待値で `htu` の無い証明が一致する経路を閉じる。

## Verification

- 保護リソースへ絶対 URL の `htu` を持つ DPoP 証明を提示した送信者制約付き API アクセストークンが、
  保護されたエンドポイントへ到達する。
- 別のリソースの絶対 URL を持つ証明は到達しない。
- テナントの正規ロケーションが path style と subdomain style のどちらでも、同じことが成り立つ。
- `mise run verify`

## Completion

- **Completed At**: 2026-09-30
- **Summary**:
  保護リソースが、DPoP 証明の `htu` をパスではなくリクエストの絶対 URL と照合するようになった。
  RFC 9449 に従うクライアントが、`/token` で得た送信者制約付きのトークンをそのまま保護リソースで使える。
  `mise run spec-diff` は規範差分なしを返す。変えたのは `docs/domain/oauth2/internals.md` の仕組みの記述だけであり、
  `RFC9449-API-TOKEN-DPOP` と `DpopProofClaims.htu: url` が既に求めていた形を実装で満たしにいった記録である。
  直したのは `RequestHTU` の定義である。テナントの正規 issuer の origin にリクエストのパスを継ぐので、
  path style で prefix が二重にならず、subdomain style でテナントの host が落ちない。
  `/token`、`/userinfo`、保護リソースの 3 つの呼び出し元が同じ関数を通るので、subdomain style の
  `/token` と `/userinfo` が基底の host を期待していた同じ欠陥も一緒に直った。
  期待値を決められない場合に `""` と `htu` の無い証明が一致する経路は、`verifyDPoP` が空の `htu` を拒否することで閉じた。
  保護リソースの呼び出し元（`auth.go`）は既に `RequestHTU(c, "")` を呼んでいたので、変更は関数の定義だけで済んだ。
- **Primary Use Case Evidence**:
  - id: dpop-bound-api-token-reaches-protected-resource
    unit_red: 'TestRequestHTUIsTheTargetURIAtTheCanonicalLocation が 4 つの subtest で落ちた。path style で RequestHTU = "/realms/acme/api/admin/v1/users"、subdomain style で "https://idp.test/token"（want "https://globex.idp.test:8443/token"）、issuer が無いときに "/realms/default/token"、host の無い issuer で "idp.test/token"（いずれも want ""）。'
    e2e_red: 'TestDPoPBoundApiTokenAcceptsTheAbsoluteTargetURIAtEitherTenantStyle が path style と subdomain style の両方で「絶対 URL の htu を持つ証明で管理 API へ到達しない: status=401 invalid_token」で落ちた。同時に、無傷の証明を絶対 URL に直した TestDPoPBoundApiTokenVerifiesEveryProofElement と TestApiTokenSenderConstraintIsChosenAtIssuance も前提の段で落ちた。'
    unit_fault_injection: 'RequestHTU の戻り値を strings.TrimRight(issuer.String(), "/") + パス（issuer 全体に継ぐ）へ差し替えると、同テストが RequestHTU = "https://idp.test/realms/acme/realms/acme/api/admin/v1/users" で落ち、E2E も両スタイルで 401 になった。'
    e2e_fault_injection: 'auth.go の期待値を c.Request().URL.Path（パスだけ）へ差し替えると、TestDPoPBoundApiTokenAcceptsTheAbsoluteTargetURIAtEitherTenantStyle が両スタイルで 401 になって落ちた。RequestHTU が文脈のテナント issuer を読まず fallback だけを使う故障も、同じテストが両スタイルで検出した。'
- **Unit RED Evidence**:
  - **Test**: `TestVerifyDPoPRejectsAMissingHTUEvenWhenNoTargetIsExpected`
  - **Requirement**: N/A: 対応する要件は standards.md の `RFC9449-API-TOKEN-DPOP` であり、この欄が受け付ける `REQ-*` の形を持たない。主要ユースケースの証拠とは別の補助の証拠である。
  - **Observed Failure**: `err = <nil>, want an htu rejection`。期待値 `""` と `htu` を持たない証明が一致として受理された。
  - **Detection Reason**: 空どうしを一致とみなす照合は、`RequestHTU` が origin を決められなかった経路で `htu` を持たない証明を通す。このテストはその組み合わせだけを提示するので、照合の文字列比較だけの実装と区別できる。
- **Change-Resistance Results**:
  - `mise run test-go-mutation -- backend/shared/http/support_http`：変更した `RequestHTU` の行の変異 3 件（条件の否定）はすべて検出された。`+` を `-` にする 3 件はコンパイルできず not viable。パッケージ全体の生存 65 件はいずれも本項目で変更していない行にある。
  - `mise run test-go-mutation -- backend/shared/security/tokens_jose`：`verifyDPoP` の `htm` と `htu` の照合行の変異 3 件はすべて検出された。
  - 手で加えた故障は Primary Use Case Evidence の 3 件（issuer 全体に継ぐ、保護リソースがパスだけを期待する、テナント issuer を読まない）であり、いずれも検出された。変異器はこれらを表現できない。
  - 限界: 基底の issuer の host を前置して subdomain を落とす故障は、保護リソースが fallback を持たないため保護リソースの経路では組めない。単体テストの subdomain style の事例がこの形を固定している。
- **Verification Results**:
  - `mise run test-go-test -- ./backend/shared/http/server_http 'TestDPoPBoundApiToken|TestApiTokenSenderConstraintIsChosenAtIssuance'` - passed
  - `mise run test-go-package -- ./backend/shared/http/support_http` - passed
  - `mise run test-go-package -- ./backend/shared/security/tokens_jose` - passed
  - `mise run test-go-changed -- main` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - `N/A: TypeSpec を変更していない。`
  - `mise run spec-diff` - passed（規範差分なし）
  - `mise run test-ui-e2e` - `N/A: frontend は dpop_bound_access_tokens の設定値を扱うだけで、DPoP 証明を送らない。`
  - `mise run verify` - passed（終了コード 0 を直接確認）
