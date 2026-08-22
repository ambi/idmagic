---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
priority: p2
depends_on: []
change_kind: feature
initial_context:
  source:
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/oauth2/handlers_http/par_handler.go
    - backend/oauth2/handlers_http/validation.go
    - backend/oauth2/client/domain
    - backend/oauth2/authorization/usecases
  tests:
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - backend/oauth2/client/domain
  stop_before_reading:
    - frontend
    - backend/saml
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.Authorize }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.PushAuthorizationRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AuthorizationRequest }
---

# 署名付き認可リクエスト (JAR / RFC 9101) と署名付き認可レスポンス (JARM) に対応する

## Motivation

OAuth2 context の `standards` は FAPI 2.0 Security Profile を宣言し、PAR / DPoP /
PKCE / private_key_jwt / mTLS を実装済みである。しかし **JWT-Secured Authorization
Request (JAR, RFC 9101) と JWT Secured Authorization Response Mode (JARM) が無い**。
specification 全体を grep しても `request_object` / `JARM` / `RFC9101` が存在せず、
`/authorize` の `request` / `request_uri` パラメータ (OIDC Core §6 の request object) も
未対応である (`request_uri` は PAR 発行のものだけで、外部参照の request object ではない)。

これが問題になる場面:

1. **FAPI 2.0 Message Signing プロファイルを満たせない**。オープンバンキング等の規制領域では
   Security Profile に加えて Message Signing (JAR + JARM) が要求される。IdMagic は
   Security Profile まで到達しているのに、その次の段で止まっている。
2. **認可リクエストの完全性が保証されない**。PAR はリクエストをチャネル外に出さないことで
   改ざんを防ぐが、「クライアントが署名した」ことの証明にはならない。JAR は
   リクエスト内容に対する否認防止 (non-repudiation) を与える。
3. **認可レスポンスの改ざん・混同攻撃**。JARM は認可レスポンス (code / state / iss) を
   署名付き JWT で返し、レスポンス改ざんと IdP 混同 (mix-up) 攻撃に対する防御を与える。
   IdMagic は RFC 9207 (issuer identification) を実装しているが、JARM はより強い保証になる。

競合比較:

- **Keycloak**: request object (JAR) と JARM の両方を実装し、FAPI プロファイルとして提供。
- **Okta**: request object (signed) をサポート。
- **Entra ID**: JAR / JARM は非対応 (ここは IdMagic が優位を取れる領域)。

つまり JAR / JARM は「Keycloak が持っていて IdMagic に無い、規制領域で必須の機能」であり、
FAPI を掲げる以上の整合性の問題でもある。

## Scope

- **decision**:
  - `spec/contexts/oauth2/decisions.md` へ記録する決定 (署名付き認可メッセージ): JAR の受け入れ方式 (`request` パラメータのみ対応し、
    外部 `request_uri` fetch は SSRF 面のため**対応しない**方針を明記する)、署名アルゴリズムの
    許可集合 (`spec/contexts/oauth2/standards.md` の `RFC7518-SIGNATURE-ALGORITHMS` と整合し PS256 / ES256 に限定)、
    request object 内パラメータと素のクエリパラメータが競合した場合の優先規則
    (OIDC Core は request object 優先、FAPI はクエリ側の重複を禁止) 、
    クライアントごとの必須化設定 (`require_signed_request_object`)、
    JARM の `response_mode` (`jwt` / `query.jwt` / `fragment.jwt` / `form_post.jwt`) の対応範囲、
    レスポンス署名鍵 (`spec/contexts/signing-keys/decisions.md` のテナントごとの署名鍵 の tenant key を使う)、
    PAR との組み合わせ (PAR に署名付き request object を POST する経路) を記録する。
- **specification**:
  - `standards` に `RFC9101` (JAR) と JARM の仕様参照を追加し、要件を書き下す。
  - `OAuth2.models.OAuth2Client` に `request_object_signing_alg` /
    `require_signed_request_object` / `authorization_signed_response_alg` を追加する。
  - `AuthorizationRequest` に `request` パラメータを追加し、request object 展開後の
    正規化を記述する。
  - `Authorize` / `PushAuthorizationRequest` の requires に署名検証・`aud` / `iss` /
    `exp` / `nbf` 検証、リプレイ防止 (`jti` 一意性) を追加する。
  - `response_modes_supported` に JARM の値を追加し、discovery
    (`GetOpenidConfiguration` / `GetOauthAuthorizationServer`) に
    `request_object_signing_alg_values_supported` /
    `authorization_signing_alg_values_supported` /
    `request_parameter_supported` / `request_uri_parameter_supported: false` を追加する。
  - `states` / events に RequestObjectRejected を追加する。
  - `scenarios`: 有効な署名付き request object で認可が成立する / 署名不正で拒否 /
    `exp` 超過で拒否 / `aud` 不一致で拒否 / `jti` 再利用で拒否 /
    `require_signed_request_object` のクライアントが素のクエリで来て拒否 /
    JARM の `response_mode=jwt` で署名付きレスポンスが返る /
    外部 `request_uri` が拒否される。
- **go**:
  - request object の検証を `backend/oauth2` に実装する。クライアントの登録済み JWKS
    (または `jwks_uri` の取得済みキャッシュ) で署名検証する。
    `spec/contexts/oauth2/internals.md` の `private_key_jwt` 検証 の検証基盤を再利用する。
  - `jti` のリプレイ防止ストア (短命 / PostgreSQL、`spec/persistence.md` の一時状態の PostgreSQL 統合
    に従う) を追加する。
  - request object 展開後のパラメータ正規化を `/authorize` と `/par` の共通経路に置き、
    両方で同一の検証が効くようにする。
  - JARM のレスポンス生成 (SigningKeys の tenant key で署名、`iss` / `aud` / `exp` /
    認可パラメータを含む JWT) と `response_mode` 別の返し方 (query / fragment / form_post) を実装する。
  - エラーレスポンスも JARM 対象になる (`response_mode=jwt` 指定時はエラーも署名 JWT で返す)。
- **http**:
  - `/authorize` と `/par` で `request` パラメータを受け付ける。外部 `request_uri` は
    `request_uri_not_supported` で明示的に拒否する。
- **ui**:
  - 管理コンソールのアプリケーション編集 (OIDC 詳細設定) に
    `request_object_signing_alg` / `require_signed_request_object` /
    `authorization_signed_response_alg` を追加する
    (`spec/contexts/application/decisions.md` に従い Application 配下に置く)。
- **documentation**:
  - README に JAR / JARM の設定手順と、`request_uri` 非対応の理由を追記する。

## Out of Scope

- 暗号化された request object (JWE, `request_object_encryption_alg`)。署名を必達とし、
  暗号化は需要が出た時点で別 WI にする。
- 外部 `request_uri` の取得 (SSRF 面を作らないため明示的に非対応)。
- ID Token / UserInfo の暗号化 (JWE)。
- FAPI 2.0 Message Signing プロファイルの正式適合試験。
  → [[wi-33-protocol-conformance-ci]] に conformance suite として追加する。
- クライアント側 (RP) としての JAR / JARM 生成。IdMagic が RP になる経路は
  [[wi-30-inbound-federation-and-identity-broker]] の領域。

## Plan

- **`request_uri` を非対応にするのが安全側の設計判断**である。外部 URL を取得する実装は
  SSRF とサービス間の可用性結合を生む。PAR が既にあるため、「事前に POST して短命の
  `request_uri` を得る」という等価な体験は既に提供できている。`spec/contexts/oauth2/decisions.md` にこの等価性を書く。
- **検証は既存の private_key_jwt 検証基盤に載せる**。`spec/contexts/oauth2/internals.md` の `private_key_jwt` 検証 で
  クライアント JWKS の解決と署名検証は既に実装されているため、新しい鍵解決経路を作らない。
- **`/authorize` と `/par` で同一の展開・検証経路を通す**。ここを分けると、片方だけ
  検証が緩いという典型的な脆弱性を作る。展開後の正規化パラメータを 1 つの型にして、
  両ハンドラがそれを受け取る形にする。
- **パラメータ競合の規則を明示的に実装する**。OIDC Core は request object 優先だが、
  FAPI では `response_type` / `client_id` 以外のクエリ重複を禁止する。
  厳格側 (重複を拒否) を既定とし、緩和が必要なら client 設定で開く。ここを最初のテストにする。
- **JARM はエラーレスポンスも含む**。`response_mode=jwt` を指定したクライアントに、
  エラーだけ素のクエリで返すと実装差異による混乱を生む。エラーも署名 JWT で返すことを
  scenario で固定する。
- **リプレイ防止は `jti` + `exp` で行う**。`jti` ストアの保持期間は `exp` の最大許容値に
  揃える。無期限に持つと肥大化するため、既存の短命状態の掃除機構
  (`spec/contexts/audit/decisions.md` の保持期間 の retention sweep) に載せる。
- 未決定: `require_signed_request_object` をテナント既定として持つか、クライアント個別のみか。
  第 1 段はクライアント個別とし、テナント既定は
  [[wi-115-tenant-default-application-login-policy]] の形に倣って必要なら後続で足す。

## Tasks

- [ ] T001 [Spec] `standards` に RFC9101 / JARM、client メタデータ 3 件、`request` パラメータ、
      Authorize / PAR の requires、discovery メタデータ、event、scenario 8 件を追加し
      `mise run check-spec` を通す。
- [ ] T002 [Spec] 署名付き認可メッセージの決定を `spec/contexts/oauth2/decisions.md` に記録する (request_uri 非対応の理由・
      アルゴリズム許可集合・パラメータ競合規則・JARM の範囲・鍵の出所)。
- [ ] T003 [Domain] request object の検証 (署名・`iss` / `aud` / `exp` / `nbf` / `jti`) と
      パラメータ展開・競合検出を実装する。RED: 署名不正 / `exp` 超過 / `aud` 不一致 /
      パラメータ重複が拒否されるテストを先に書く
      (scenario `OAuth2.signed_request_object_rejected_on_invalid_signature`) → GREEN。
- [ ] T004 [Persistence] `jti` リプレイ防止ストアを追加し (memory / postgres)、
      retention sweep に掃除を組み込む。RED: 同一 `jti` の 2 回目が拒否されるテスト → GREEN。
- [ ] T005 [Client] `request_object_signing_alg` / `require_signed_request_object` /
      `authorization_signed_response_alg` を client メタデータに追加する (DCR / 管理 API /
      スキーマ / sqlc)。RED: 不許可アルゴリズムの登録が拒否されるテスト → GREEN。
- [ ] T006 [Authorize/PAR] `/authorize` と `/par` の両方で request object を同一経路で
      展開・検証する。外部 `request_uri` を `request_uri_not_supported` で拒否する。
      RED: 両エンドポイントで同じ検証が効くテスト → GREEN。
- [ ] T007 [JARM] 署名付き認可レスポンス生成と `response_mode` 別の返し方 (query.jwt /
      fragment.jwt / form_post.jwt) を実装する。エラーも JWT で返す。
      RED: 正常系とエラー系の両方の JARM テスト → GREEN。
- [ ] T008 [Discovery] discovery メタデータに JAR / JARM の対応値を追加する
      (`request_uri_parameter_supported: false` を明示)。RED: discovery の contract テスト → GREEN。
- [ ] T009 [UI] Application の OIDC 詳細設定に 3 つのメタデータを追加する。
      RED: presentation logic の unit test → GREEN。
- [ ] T010 [Docs] README に JAR / JARM の設定と `request_uri` 非対応の理由を追記する。
- [ ] T011 [Verify] 下記 Verification を緑にする。`mise run spec-render` を実行する。

## Verification

- `mise run check` / `mise run check-spec` / `mise run check-work-items` / `mise run check-ids`
- `mise run test-go` / `mise run test-go-race` / `mise run verify-go`
- `mise run verify-ui`
- 手動: `mise run dev` で (1) 署名付き request object を使った認可コードフローが完了すること、
  (2) 署名を改変すると拒否されること、(3) `response_mode=jwt` で署名付きレスポンスが返り、
  公開鍵で検証できること、(4) `request_uri=https://...` が明示エラーになること、を確認する。
- 手動: `mise run demo` の OIDC デモが従来どおり動作すること (既存クライアントの回帰確認)。

## Risk Notes

`/authorize` と `/par` は認可の入口であり、request object の展開でパラメータ解釈が
分岐すると**認可判断が意図と異なる値で行われる**危険がある。展開後の正規化を 1 つの型に
閉じ、両エンドポイントが同じ検証を通ることをテストで固定する。
パラメータ競合を緩く扱うと「クエリ側の値で認可され、request object の値で
クライアントは安心している」というズレを生む。厳格側 (重複拒否) を既定にする。
JARM でエラーを署名 JWT にすると、既存クライアントが素のクエリを期待している場合に
壊れる。JARM は `response_mode` の明示指定時のみ有効にし、既定挙動は変えない。
`jti` ストアは書き込み頻度が高くなりうる。保持期間を `exp` 上限に揃え、
retention sweep で確実に掃除する。
