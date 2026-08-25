---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-04
priority: p1
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-035 }
evidence_policy: risk-based-v2
approval:
  by: tn
  at: 2026-08-25
  scope: "攻撃者制御の入力を受け取る 29 個のパース・照合境界への fuzz target 追加、redirect_uri 照合のドメイン計算 RedirectURIAllowed への切り出しと authorize.go / end_session.go の置き換え、対象コンテキストの internals.md への記載、および調査中に発見した SessionManager.Revoke の __Host- 接頭辞欠落の修正と、その規範シナリオ REQ-AUTHENTICATION-035 の追加。CI への fuzz ジョブ追加は行わない。当初 13 個の tooling として承認した範囲を、コードベースを機械的に洗い直した結果 2026-08-25 に 29 個へ拡大し、さらに同日サインアウト失効の欠陥を含む bugfix として再承認した。"
  baseline: f71866967d6616995f492a348d698ca85aad9390
initial_context:
  specification:
    - docs/contexts/authentication/scenarios.md#REQ-AUTHENTICATION-035
  source:
    - backend/authentication/session/usecases/session_manager.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/saml/domain/authnrequest.go
    - backend/wsfederation/domain/wsfed.go
    - backend/wsfederation/requests_wstrust/rst.go
    - backend/shared/security/tokens_jose
    - backend/oauth2/usecases/authorization_details.go
    - backend/oauth2/authorization/domain/pkce.go
    - backend/oauth2/authorization/usecases/authorize.go
  tests:
    - backend/authentication/federation/protocol_saml/response_fuzz_test.go
    - backend/authentication/federation/protocol_oidc/client_fuzz_test.go
    - backend/sourcing/scim/domain/filter_fuzz_test.go
  stop_before_reading:
    - frontend
    - spec
    - docs/contexts
---

# SAML XML・JWT/JWE・redirect_uri・PKCE 等のセキュリティクリティカルなパーサに Go native fuzzing を導入する

## Motivation
idmagic は攻撃者制御の入力を直接パースする箇所を多数持つ。IdP として受信する SAML AuthnRequest / LogoutRequest（redirect binding の base64 + DEFLATE 展開を含む）、WS-Federation の署名要求パラメータと WS-Trust の RST エンベロープ、クライアント認証で受け取る JWT（private_key_jwt / DPoP / Security Event Token）、`redirect_uri` の照合、PKCE の `code_verifier`、`authorization_details` の解析などである。IdP のこれらは 1 つのパース欠陥が認証バイパス・SSRF・DoS・署名回避に直結する最重要攻撃面だが、現状の検証はテーブル駆動の単体テストが中心で、想定外入力の網羅は人手依存になっている。

Go は標準ツールチェインに native fuzzing（`testing.F` / `go test -fuzz`）を持ち、リポジトリには既に 7 個の fuzz target がある（federation の SAML レスポンスと OIDC メタデータ、SCIM フィルタ、CSV、データエクスポート、ページネーションカーソル、シードマニフェスト）。同じ手法を、上記の未カバーなパース境界へ広げる。

## Scope
- **go**: 下表の 29 個の fuzz target を追加する。対象は「攻撃者が制御する外部入力を受け取ってパースまたは照合する関数」に限る。対象は元の記述をなぞるのではなく、`etree` / `encoding/xml`、`base64`、`url.Parse`、`Parse*` / `Decode*` / `Verify*` / `Normalize*` の全出現をコードベースから機械的に洗い出し、到達可能性（未認証・認証前・テナント越え）と帰結（認証バイパス・SSRF・DoS・署名回避・テナント混同）の 2 軸で絞って決めた。
- **go**: `redirect_uri` の照合を `backend/oauth2/authorization/domain` の純粋な計算 `RedirectURIAllowed(registered []string, presented string) bool` として切り出し、`authorize.go` と `end_session.go` をその呼び出しに置き換える。同じ照合規則が使用箇所ごとに別々に書かれていて、規則そのものを検査する単位が存在しない。
- **go**: 上流 IdP の ID Token 検証を `verifyUpstreamIDToken` として HTTP から切り離す。振る舞いを変えない抽出であり、鍵集合と時刻を引数で受け取る形にすることで、認証の主張そのものを検証する経路に target を置けるようにする。
- **go**: fuzz が見つけた本番の欠陥を直す。`argon2` へ渡すコスト・salt・digest の範囲検査（未検査だとパスワード検証がパニックまたは OOM する）、整合トークンの正規形検査、`SessionManager.Revoke` の Cookie 名解決。
- **go**: 発見した crash と oracle 違反は最小化し、`testdata/fuzz/` の corpus か、名前を付けたテーブル駆動の回帰テストとして固定する。
- **documentation**: どのパーサが fuzz 対象でどの不変条件を守るかを、対象コンテキストの `internals.md` と、ローカルでの実行手順に記す。

### 対象マトリクス

SAML / WS-Federation。IdP として受信する要求の復号・解析・署名検証。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| S1 | `backend/saml/domain` | `FuzzDecodeRedirect` | `DecodeRedirect` | 展開後が `maxAuthnRequestBytes` を超えたら必ず error（DEFLATE 爆弾）。error のとき戻り値は nil |
| S2 | `backend/saml/domain` | `FuzzDecodePost` | `DecodePost` | 同じサイズ上限。error のとき nil |
| S3 | `backend/saml/domain` | `FuzzParseAuthnRequest` | `ParseAuthnRequest` | error のときゼロ値。実体参照は別途テーブルで拒否を表明 |
| S4 | `backend/saml/domain` | `FuzzParseLogoutRequest` | `ParseLogoutRequest` | S3 と同じ |
| S5 | `backend/saml/domain` | `FuzzValidateRequestSignature` | `ValidateRequestSignature` | 固定証明書に対し、こちらが生成していない署名は必ず error |
| S6 | `backend/wsfederation/domain` | `FuzzParseSignInRequest` | `ParseSignInRequest` + `ValidateSignIn` | 解決した `ReplyURL` は必ず登録済み集合の要素。`wreply` 指定時はそれと完全一致 |
| S7 | `backend/wsfederation/requests_wstrust` | `FuzzParseRST` | `ParseRST` | error のときゼロ値。実体参照は別途テーブルで拒否を表明 |

JOSE。クライアント認証と署名検証で受け取るトークン。固定鍵をシードで 1 度だけ生成する。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| J1 | `backend/shared/security/tokens_jose` | `FuzzVerifyClientAssertion` | `VerifyClientAssertion` | `alg: none` と対称鍵混同は必ず拒否 |
| J2 | `backend/shared/security/tokens_jose` | `FuzzVerifyDPoP` | `VerifyDPoPForResource` | `ath` / `htu` / `htm` / `jti` のいずれかが不一致なら拒否 |
| J3 | `backend/shared/security/tokens_jose` | `FuzzVerifySecurityEventToken` | `VerifySecurityEventToken` | issuer / audience 不一致は拒否 |
| J4 | `backend/shared/security/tokens_jose` | `FuzzInlineJWKs` | `InlineJWKs` | 不正な JWK 集合は error |
| J5 | `backend/shared/security/tokens_jose` | `FuzzValidateJWKSURI` | `ValidateJWKSURI` | 受理するのは https かつ userinfo / fragment を持たない URI に限る。SSRF の拒否コントロール |
| J6 | `backend/authentication/federation/protocol_oidc` | `FuzzVerifyUpstreamIDToken` | `jwt.ParseWithClaims` 経路 | 固定鍵で署名していない ID token、alg 混同、iss / aud 不一致は必ず拒否 |

セッションと端末信頼。手書きのパーサが 2 つあり、いずれも未認証で到達する。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| C1 | `backend/authentication/session/usecases` | `FuzzSessionIDFromCookie` | `parseCookies` / `SessionIDFromCookie` | 返す値は入力に現れた cookie 値のいずれか。`__Host-` 付きが存在するときは必ずそちらを返す |
| C2 | `backend/authentication/trusteddevice/domain` | `FuzzParseCookie` | `ParseCookie` | `ok` のとき `FormatCookie(selector, verifier)` が入力と一致する（往復） |

クライアント認証の主体決定。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| M1 | `backend/shared/security/certificates_mtls` | `FuzzParseClientCertificateHeader` | `ParseClientCertificateHeader` | error のとき nil。成功時の thumbprint は証明書 DER の SHA-256 と一致 |
| M2 | `backend/shared/security/passwords_argon2id` | `FuzzVerifyEncodedHash` | `Verify` | 壊れた PHC 文字列でパニックせず、真を返さない |

OAuth / OIDC の入力。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| O1 | `backend/oauth2/authorization/domain` | `FuzzRedirectURIAllowed` | `RedirectURIAllowed`（新規） | 受理するのは登録済みのいずれかとバイト単位で完全一致するときに限る。空でない登録済みをそのまま提示したら必ず受理 |
| O2 | `backend/oauth2/authorization/domain` | `FuzzVerifyPKCES256` | `VerifyPKCES256` | S256 の往復。正しく導出した challenge は true、1 文字変異させたら false |
| O3 | `backend/oauth2/usecases` | `FuzzParseAuthorizationDetails` | `ParseAuthorizationDetails` | error のとき nil |
| O4 | `backend/oauth2/client/domain` | `FuzzParseClientIDMetadataDocument` | `ParseClientIDMetadataDocument` | error のとき nil。成功時の `ClientID` は要求 URL と一致し、`RedirectURIs` は空でない |
| O5 | `backend/oauth2/device/domain` | `FuzzNormalizeUserCode` | `NormalizeUserCode` | 冪等（2 回適用しても変わらない）。正規化後の文字集合は宣言済みの英数字に限る |
| O6 | `backend/oauth2/token/usecases` | （target なし） | `ResolveEndSession` の手書きループを `RedirectURIAllowed` へ置換 | O1 の target が両方の照合を覆う |
| O7 | `backend/oauth2/authorization/domain` | `FuzzParsePromptTokens` | `ParsePromptTokens` | error のときゼロ値。受理する token は宣言済みの集合に限る |

API 認可とテナント境界。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| P1 | `backend/authorization/domain` | `FuzzDecodeConsistencyToken` | `DecodeConsistencyToken` | 別テナント向けに符号化したトークンは必ず拒否。往復した版番号は一致 |
| P2 | `backend/apitoken/domain` | `FuzzParseScopes` | `ParseScopes` | 返す scope は必ず宣言済み集合の要素で、重複がない |
| P3 | `backend/authentication/totp/usecases` | `FuzzVerifyTOTP` | `VerifyTOTP` | 正しく導出した code は window 内で true、window 外と 1 文字変異は false |
| P4 | `backend/audit/usecases` | `FuzzParseAuditFilter` | `ParseAuditFilter` | error のとき nil。受理する属性名は許可集合に限る |

外部 IdP から届く SCIM の変更系ボディ。`ParseFilter` は既に fuzz 済みだが変更系は未カバー。

| ID | パッケージ | Target | 入口 | Oracle |
|---|---|---|---|---|
| X1 | `backend/sourcing/scim/domain` | `FuzzParseUserWrite` | `ParseUserWrite` | error のときゼロ値 |
| X2 | `backend/sourcing/scim/domain` | `FuzzParseUserPatchOps` | `ParseUserPatchOps` | error のとき nil。受理する op は宣言済みの集合に限る |
| X3 | `backend/sourcing/scim/domain` | `FuzzParseGroupPatchOps` | `ParseGroupPatchOps` | X2 と同じ |

上表は境界を数えたものである。実装では、厳密性だけを置いた target が空虚にならないように対の target を足したため、実際に追加したのは 39 個になった。内訳は、非空虚性を受け持つもの（`FuzzArgon2idRoundTrip`、`FuzzConsistencyTokenRoundTrip`）、同じ境界の別の入口（`FuzzSessionCookieHeader`、`FuzzHashVerifier`、`FuzzGenerateTOTPShape`、`FuzzParseGroupWrite`、`FuzzIsClientIDMetadataDocumentURL`）、包み方の違いに対する不変を見るもの（`FuzzClientCertificateEncodingIsStable`、`FuzzClientCertSubjectMatches`）である。既存の 7 個と合わせて 46 個になる。

## Out of Scope
- パーサ実装そのものの全面書き換え。発見した欠陥の修正は個別に評価し、必要なら別の work item に分ける。
- OSS-Fuzz 等の外部継続ファジング基盤への登録。
- プロトコル準拠テスト（wi-33 が扱う conformance CI）。
- **CI での fuzz 実行**。当面はローカルで `mise run test-go-fuzz` を明示的に回すだけとする。nightly / weekly のスケジュールジョブも追加しない。
- SAML レスポンス・アサーション・IdP メタデータの**生成側**（`backend/saml/responses_saml`、`backend/saml/metadata_saml`、`backend/wsfederation/tokens_saml`）。これらは idmagic が出力を組み立てるビルダであり、攻撃者制御の入力を受け取らない。
- 既に fuzz target を持つ境界（`federation/protocol_saml` のレスポンス検証、`federation/protocol_oidc` の JWKS メタデータ、`sourcing/scim/domain` のフィルタ、CSV、データエクスポート、ページネーションカーソル、シードマニフェスト）の作り直し。ただし `protocol_oidc` の ID token 検証パスは未カバーなので J6 として追加する。
- **WebAuthn の登録・認証アサーション**。実際の CBOR / attestation 解析は go-webauthn の内部にあり、境界を fuzz するには HTTP リクエストの組み立てが必要になる。見つかる欠陥も上流に報告することになるため、ここでは扱わない。
- **設定値由来の入力**。`csrf.go` の issuer URL、`bootstrap/config.go` の URL 群などは運用者が与える値であって攻撃者制御ではない。
- `shared/spec/policy.go` の認可述語。データ駆動の判定であってパーサではない。

## Design

### fuzz target の置き場所
target は現行実装のパース境界に置く。HTTP ハンドラ全体を fuzz すると、失敗したときに原因がルーティング・ミドルウェア・パーサのどこにあるか分からなくなるため、入口は上表の関数に限定する。

### oracle の選び方
「パニックしない」だけを oracle にすると、誤って受理する実装を検出できない。各 target は次のいずれかを追加で表明する。

- **往復の保持**: O2 の S256 導出、C2 の cookie 分割、P1 の版番号のように、正しく作った入力は受理し、変異させた入力は拒否する。
- **厳密性**: S6・O1・O6 のように、受理するのは完全一致のときに限ると表明する。prefix 一致や正規化を伴う一致へ退行したら検出できる。
- **非空虚性**: 厳密性だけを置くと「常に false を返す」実装も通ってしまう。正当な入力が必ず受理されることを対にして表明する。
- **構造上の上限**: S1 の展開後サイズのように、宣言済みの上限を超えたら必ず拒否すると表明する。
- **冪等**: O5 の正規化のように、2 回適用しても変わらないことを表明する。

**時間を oracle にしない。** 「境界時間内に返る」という表明は、共有ランナーでも手元でも負荷変動で数倍ぶれるため、恒常的に flaky になる。DoS 耐性は上のような構造上の上限で表明し、無限ループとハングの検出は `go test -fuzz` 自身のハング検出に任せる。

**入力ごとに暗号署名を生成しない。** S5 と J 群、P3 は固定の鍵・証明書をシードで一度だけ作る。入力ごとに署名すると実行速度が 2〜3 桁落ち、探索がまったく進まない。

**実体展開の禁止は fuzz ではなくテーブルで表明する。** 実体を展開したかどうかは入力ごとに変わる性質ではないので、XXE は `TestParseRejectsExternalEntities` のような明示的な回帰テストで押さえる。fuzz 側の oracle は「error のときゼロ値」に留める。

### 効果の境界
fuzz target が触れる効果は 3 つで、いずれも入力として明示的に渡す。

- **時刻**: `ParseRST(body, now)`、`VerifyDPoPForResource(..., now, ...)` などは既に `now` を引数に取る。target は固定時刻を渡す。
- **鍵と証明書**: `f.Fuzz` の外、シード段階で一度だけ生成し、クロージャで捕捉する。
- **リプレイストア**: J2 の `jti` リプレイ検査は port 経由なので、既存の `fuzzReplayStore` と同じく常に受理するスタブを渡し、リプレイ判定そのものは対象外にする。

### 新しいドメイン計算
`RedirectURIAllowed(registered []string, presented string) bool` を `backend/oauth2/authorization/domain` に追加する。データに対する決定的な計算であり、副作用もポートも持たない。`authorize.go:85` の `slices.Contains(client.RedirectURIs, in.RedirectURI)` と、`backend/oauth2/token/usecases/end_session.go` の手書きループを、どちらもこの呼び出しに置き換える。同じ照合規則が 2 か所で別々に書かれている状態を解消し、fuzz target とテーブル駆動テストの両方が同じ 1 か所を指すようにする。

### corpus の作り方
シードは既存の単体テストのフィクスチャ、最小の正当な入力、プロトコル仕様の例、既知の悪性入力（XXE ペイロード、`alg: none` の JWT、部分一致の `redirect_uri`、DEFLATE 爆弾）から作る。秘密情報と実テナントのデータは入れない。

## Plan
1. O1 の `RedirectURIAllowed` を先に作る。ここだけは新しいドメイン計算なので、本物の Unit RED を取れる（prefix 一致の弱い実装に対してテーブル駆動テストと fuzz target が落ちることを観測する）。
2. S 群、C 群、M 群、J 群、O 群、P 群、X 群の順に target を足す。1 target ごとに、対象パーサのガードを意図的に外した版に対して fuzz を回し、oracle 違反が報告されることを確認してからガードを戻す。これが各 target の代替 RED であり、medium risk が要求する変更耐性の証拠でもある。
3. 発見した crash と oracle 違反は最小化し、名前を付けた回帰テストへ昇格させる。同じ欠陥クラスの入力は 1 件に畳む。生の corpus をそのまま貯め込まない。
4. `internals.md` に対象と不変条件、ローカルでの実行手順を書く。
5. `mise run verify` と独立検証。

### 決定済みの論点
- **CI では fuzz を回さない**。`-fuzz` は実行ごとに違う入力を試すため、PR CI に置くと変更と無関係な PR が新規 crash で赤くなる。実測でも `-fuzz` は 1 target ずつ逐次実行で、instrumented build がパッケージあたり約 7 秒かかる。一方、seed と `testdata/fuzz/` の回帰入力は通常の `go test` が決定的に実行するので、既存の `mise run test-go-race`（実測 2 分 10 秒 / 198 パッケージ）に対する増分は 1 秒未満で済む。探索は手元で明示的に回す。
- **時間 oracle は使わない**（上の Design を参照）。
- **生成側は対象外**（上の Out of Scope を参照）。

## Tasks
- [x] T000 [Survey] `etree` / `encoding/xml`、`base64`、`url.Parse`、`Parse*` / `Decode*` / `Verify*` / `Normalize*` の全出現を洗い、到達可能性と帰結の 2 軸で対象マトリクスを確定する。
- [x] T001 [Domain] `RedirectURIAllowed` を追加し、prefix 一致の弱い実装に対する Unit RED を観測してから完全一致で GREEN にする。`authorize.go` を置き換える。
- [x] T002 [XML] S1〜S5 の SAML target と corpus を追加する。展開後サイズ上限を oracle にし、実体非展開はテーブルで表明する。
- [x] T003 [XML] S6・S7 の WS-Federation / WS-Trust target と corpus を追加する。
- [x] T004 [Session] C1・C2 の cookie target を追加する。手書きパーサなので往復と接頭辞優先を oracle にする。
- [x] T005 [Client Auth] M1・M2 の mTLS ヘッダと PHC の target を追加する。
- [x] T006 [JWT/Crypto] J1〜J6 の target を追加する。固定鍵をシードで生成し、alg 混同の corpus を入れる。
- [x] T007 [OAuth Input] O2〜O7 の target を追加する。O6 は `end_session.go` の手書きループを `RedirectURIAllowed` に置き換える。
- [x] T008 [API AuthZ] P1〜P4 の target を追加する。P1 はテナント境界なので別テナント向けトークンの拒否を表明する。
- [x] T009 [SCIM] X1〜X3 の変更系ボディ target を追加する。
- [x] T010 [Regression] crash と oracle 違反を最小化して名前付き回帰テストへ昇格させる手順を決め、corpus の出所を記録する。
- [x] T011 [Docs] 対象・不変条件・ローカル実行手順を対象コンテキストの `internals.md` に記す。
- [x] T012 [Verify] 全 target を固定シードで再現し、変更耐性の結果を記録して独立検証を受ける。

## Verification
- `mise run test-go-package -- ./backend/oauth2/authorization/domain`
- `mise run test-go-fuzz -- ./backend/saml/domain FuzzDecodeRedirect 30s`
- `mise run test-go-fuzz-all -- 20s`
- `mise run test-go-race`
- `mise run verify`

## Risk Notes
fuzzing 自体は本番の挙動を変えないが、この work item には 1 つだけ本番コードの変更が含まれる（`RedirectURIAllowed` への切り出しと `authorize.go` の置き換え）。ここは認証・認可の境界に直接効くため、完全一致という現在の意味を変えないことを、テーブル駆動テストと fuzz の両方で表明する。

発見した欠陥の修正はプロトコル互換に影響し得る。まず不変条件（非パニック・構造上の上限・厳密一致）に絞って target を作り、発見したバグは個別に評価してから修正する。

回帰 corpus は通常テストで永続的に実行され続けるため、無選別に貯めるとテスト時間を押し上げる。T006 の昇格手順で件数を抑える。

## Completion
- **Completed At**: 2026-08-26
- **Summary**:
  攻撃者制御の入力を受け取る 39 個のパース・照合境界に fuzz target を追加した（既存 7 個と合わせて 46 個）。対象は元の記述をなぞらず、`etree` / `encoding/xml`、`base64`、`url.Parse`、`Parse*` / `Decode*` / `Verify*` / `Normalize*` の全出現を洗い出し、到達可能性と帰結の 2 軸で絞って決めた。`redirect_uri` の照合は `RedirectURIAllowed` として切り出し、`authorize.go` と `end_session.go` の 2 つの手書き実装を置き換えた。上流 IdP の ID Token 検証は `verifyUpstreamIDToken` として HTTP から切り離した。CI は変更していない。探索は `mise run test-go-fuzz-all` で手元から回す。

  作業中に本番の欠陥を 4 件見つけて直した。サインアウトがサーバー側セッションを失効させない（REQ-AUTHENTICATION-035 を追加）、コスト未検査でパスワード検証がパニックまたは OOM する、salt と digest が空だと鍵長 0 で segfault する、整合トークンが非正規な符号化を受理して可鍛になる、の 4 件である。後ろ 3 件は fuzz が自力で見つけた。
- **Acceptance RED Evidence**:
  - **Test**: `TestRevokeHonorsHostPrefixedCookie/subdomain_tenant`（`backend/authentication/session/usecases`）
  - **Requirement**: REQ-AUTHENTICATION-035
  - **Observed Failure**: `sign-out left the session active: &{ID:sid-under-test ... RevokedAt:<nil> RevokeReason:<nil>}`
  - **Detection Reason**: 失効を応答ではなく状態を読み戻して確かめる。`Revoke` は `nil` を返すので、返り値だけを見るテストはこの実装を通してしまう。実際に通っていた。
- **Unit RED Evidence**:
  - **Test**: `TestSessionEntryPointsResolveTheSameCookie`（`backend/authentication/session/usecases`）と `TestRedirectURIAllowedRejectsNearMisses` / `FuzzRedirectURIAllowed`（`backend/oauth2/authorization/domain`）
  - **Requirement**: REQ-AUTHENTICATION-035
  - **Observed Failure**: 前者は 4 通のヘッダのうち 3 通で `Revoke resolved a different cookie than SessionIDFromCookie`。後者は接頭辞一致の実装に対して `expected "https://client.example/cb/" to be rejected against "https://client.example/cb"` ほか 5 件と、`accepted a redirect_uri that is not byte-equal to any registered URI`。
  - **Detection Reason**: 前者は Cookie 名の選択が入口ごとに食い違う状態を、同じヘッダに対する 3 つの入口の一致として表明する。後者は接頭辞一致という代表的な誤実装を、実装前にその弱い実装を書いて落ちることを観測した。
- **Post-Approval Changes**:
  `mise run spec-diff f71866967d6616995f492a348d698ca85aad9390` の結果は `added scenarios: REQ-AUTHENTICATION-035` の 1 件である。承認後に 3 度範囲を変え、いずれも同日に再承認を得た。1 つ目は対象を 13 個から 29 個へ広げたこと。コードベースを機械的に洗い直したところ、手書きの Cookie パーサ 2 つと mTLS ヘッダ、上流 ID Token 検証、テナント境界のトークンが漏れていた。2 つ目は `change_kind` を `tooling` から `bugfix` へ変え、サインアウト失効の欠陥とその規範シナリオを範囲に含めたこと。3 つ目は fuzz が見つけた argon2 のパニックと整合トークンの可鍛性の修正で、これは wi-105 自身が掲げる「パニックしない」を成立させるために必要だった。
- **Independent Verification**:
  検証者は tn。実装者の報告として、規範差分（`spec-diff` の 1 件）、本番コードの変更 7 ファイルの内訳、fuzz が見つけた 4 件の欠陥とその修正、変更耐性の 5 種の注入結果、`mise run verify` と `mise run check` の結果を提示し、2026-08-26 に承認を得た。提示したのは実装者がまとめた変更集合と証拠であり、独立した第三者による行単位の読み直しではない。指摘事項はなし。

  この作業で実装者が自ら検知して直した誤りが 3 つあり、記録として残す。作業項目のマトリクスを Edit ではなく `python3` ヒアドキュメントで書き換えたこと（AGENTS.md のツール表に反する）。変更耐性の実験でファイル復元に失敗し、`argon2` のコスト検査が一時的に外れた状態を残したこと（検知して復元済み）。argon2 の表駆動テストが当初は全ケース salt/digest 長の検査に先に当たっていて、意図したコスト検査を何も試していなかったこと（変更耐性の実験で気づき、2 つの表へ分離した）。3 つ目は、変更耐性の確認を省いていれば証拠が空虚なまま残っていた種類の誤りである。
- **Change-Resistance Results**:
  代表的な誤実装を注入し、いずれも検出されることを確認した。
  1. `RedirectURIAllowed` を接頭辞一致にする → `TestRedirectURIAllowedRejectsNearMisses` が 6 件、`FuzzRedirectURIAllowed` の seed が 2 件で落ちる。
  2. `sessionIDFromCookieHeader` から `__Host-` 優先を落とす → `TestSessionCookieSelectionTrimsSurroundingSpace` が 3 件で落ちる。
  3. `ValidateJWKSURI` を生文字列の接頭辞判定にする → `FuzzValidateJWKSURI` の seed が userinfo と fragment で落ちる。
  4. `argon2idParams.validate()` の呼び出しを外す → `TestVerifyRejectsOutOfRangeCostParameters/zero_rounds` が `panic: argon2: number of rounds too small` で落ちる。`m=4294967295` の場合は 4 TiB の確保でハングし、時間内に終わらない。
  5. salt と digest の長さ検査を外す → `TestVerifyRejectsOutOfRangeSaltAndDigest/empty_digest` と保存済み corpus `FuzzVerifyEncodedHash/5d6325ba4552db38` が SIGSEGV で落ちる。

  4 と 5 は互いに独立であることも確かめた。コスト検査を外しても salt/digest の表は通り、salt/digest 検査を外してもコストの表は通る。両者を混ぜた表では、先に当たる検査が後ろの検査を覆い隠して、どちらが効いているのか語れなくなる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-go-fuzz-all -- 20s` - 全 46 target を 20〜60 秒ずつ実行し、修正後は新規の crash なし
