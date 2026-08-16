---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-16
change_kind: bugfix
priority: p2
initial_context:
  specification:
    - spec/SPECIFICATION.md
  typespec:
    - Product.Saml.SamlServiceProvider
    - Product.WsFederation.WsFedRelyingParty
  source:
    - backend/shared/spec/length.go
    - backend/shared/http/support_http/error_handler.go
    - backend/saml/domain/service_provider.go
    - backend/saml/domain/authnrequest.go
    - backend/saml/handlers_http/admin_service_provider_handler.go
    - backend/wsfederation/domain/relying_party.go
    - backend/wsfederation/handlers_http/admin_relying_party_handler.go
    - backend/application/domain/application.go
    - backend/sourcing/scim/usecases/users.go
    - backend/signingkeys/keys_jose/rsa_jwk.go
    - infra/schema/postgres.sql
  tests:
    - backend/shared/spec/length_test.go
  stop_before_reading:
    - spec/generated
affected_spec:
  - { path: spec/contexts/saml/models.tsp, symbol: Product.Saml.SamlServiceProvider }
  - { path: spec/contexts/ws-federation/models.tsp, symbol: Product.WsFederation.WsFedRelyingParty }
---

# 外部が値を決める識別子に資源の上限を与える

## Motivation
`wi-128-string-length-limits-policy` は、上限を持つ文字列の単位と適用点を揃えた。一方、外部が値を決める識別子（`entity_id`、`wtrealm`、`scim_id`、`kid` など）には「外部仕様が長さを定めていない値を DB 都合で短く切らない」という理由で上限を置かなかった。

しかし「上限なし」もまた、検討された選択ではなく既定値でしかない。これらは連携相手が値を決めるため、こちらの都合で短くはできないが、無制限に受けてよい理由もない。次の 2 点が具体的な問題である。

- **すでに壊れている**。`entity_id`、`wtrealm`、`scim_id` は主キーの構成要素であり、`kid` は主キーそのものである。PostgreSQL の btree v4 は索引行 1 件を 2704 バイトに制限するので、これを超える値は挿入時点で失敗する。実測（`postgres:17`、非圧縮データ）では次のとおり。

  | 長さ | 結果 |
  |---|---|
  | 2000 バイト | 成功 |
  | 2704 バイト | `ERROR: index row size 2720 exceeds btree version 4 maximum 2704 for index "..._pkey" (SQLSTATE 54000)` |
  | 8192 バイト | `ERROR: index row requires 8208 bytes, maximum size is 8191 (SQLSTATE 54000)` |

  つまり上限は事実上すでに存在し、ただしそれが宣言されておらず、超過は 500 として低水準のエラー文言で返る。`wi-128` が長さ違反に与えた「422 とフィールド名つきの `detail`」という扱いから漏れている。値が圧縮しやすいかどうかで閾値が動くため、再現しにくい形でもある。
- **標準が上限を定めている値がある**。少なくとも SAML 2.0 の entity identifier は「1024 文字以下の URI」と規定されている（`saml-core-2.0-os` の Entity Identifier）。`wi-128` はこれを「外部仕様で長さが明確でない識別子」に分類したが、これは誤りだった。実装前に原典で確認する。

## Scope
- 外部が値を決める識別子ごとに、上限の根拠を「標準が定めている」「資源上限として置く」のどちらかに分類する。対象は、btree の鍵の成分になっている外部由来の列すべて。
- 標準が定める値は原典を引いて記録する。定めていない値には、btree の索引行上限より内側で、実運用の実例を拒否しない上限を置く。
- `spec/SPECIFICATION.md` の String length limits に、資源上限の区分とその根拠を追記する。「上限を置かない値もある」という現在の記述を改める。
- Go の domain / usecase で上限を強制する。管理 API は `wi-128` が用意した `spec.LengthError` 経由で 422 として返し、プロトコル接点はそれぞれのプロトコルのエラー形で返す。DB には `CHECK` を最後の防壁として置く。
- 鍵の成分ではない外部由来の文字列（token hash、`tls_client_auth_subject_dn`、`quarantine_reason` など）についても、上限を置くか置かないかを判断して記録する。

## Out of Scope
- 235 個ある `TEXT` 列すべてへ機械的に上限を設定すること。
- 既存データの棚卸しと切り詰め。未リリースのため対象データが存在しない。
- `JSONB` の payload、配列要素数、ネスト深さ。これらは文字列長ではなく `HTTP_MAX_BODY_BYTES` と別の手段が扱う。
- 鍵の設計変更（代理キー + ダイジェストの一意索引）。btree の索引行上限自体は回避できるが、`ORDER BY entity_id` を支える索引が同じ 2704 バイト制限を持つため問題は解けず、変更範囲だけが大きい。

## Design

### 上限を 2 段にする

上限を 1 つの数では表せない。契約の単位は `wi-128` が定めたとおり Unicode コードポイントだが、btree が制限しているのはバイトである。SAML の entity identifier は標準が 1024 **文字**を上限と定めており、これがすべて 3 バイト文字なら 3072 バイトになって btree の 2704 バイトを超える。コードポイントの上限だけでは、標準が許す値のまま 500 が残る。

そこで、鍵の成分になる列には次の 2 つを重ねる。

| 上限 | 単位 | 役割 |
|---|---|---|
| 契約の上限 | コードポイント | 公開契約が示す数。TypeSpec の `@maxLength` と Go の `spec.Chars` が持つ |
| 資源の上限 | バイト | btree の索引行に収まることの保証。Go の `spec.KeyString` と SQL の `octet_length` が持つ |

単位がコードポイントである原則に対する既存の例外は「標準自身がオクテットで上限を定めている値」だけだった。ここに「資源そのものがバイトで測られる場合」を加える。資源をその資源自身の単位で測るのは、換算に伴う推測を避ける唯一の方法である。

### 鍵ごとにバイト予算を持つ

複合鍵では列ごとではなく、鍵 1 件の合計が btree の 2704 バイトに収まらなければならない。合計を **2400 バイト以下**に抑える（索引タプル自身が使う領域と、将来列を足すための余白）。この条件はこの文書ではなくテストで守る。

### 標準の数をそのまま上限にはしない

`saml-schema-metadata-2.0.xsd` の `entityIDType` は `<restriction base="anyURI"><maxLength value="1024"/></restriction>` である。しかし標準を超える entity ID を出す SP は実在しうるので、標準の数は「上限が実運用で拘束的でないことの根拠」として記録し、上限そのものは URI 区分の 2048 に置く。これらは安全のために置く資源の境界であって、業務上望ましい長さではないので、迷ったら広く取る。

### 違反の返し方は接点によって違う

管理 API は `wi-128` どおり 422 / `field_length_exceeded`。プロトコル接点（SAML、WS-Federation、OAuth 2.0、WebAuthn）は、そのプロトコルが定めるエラー形に写像する。SAML の AuthnRequest を送ってきた相手に 422 の Problem Details を返しても読めない。

### 検討して採らなかった案

- **コードポイントの上限だけを、鍵予算の 1/3〜1/4 に置く**。数は 1 つで済むが、標準が許す 1024 文字の entity ID を拒否することになり、相互運用性の側を削る。
- **標準の数（1024）をそのまま契約の上限にする**。極端な多バイト値が btree で落ちる窓が残り、報告された不具合を半分しか直さない。
- **鍵の設計を変える**（代理キー + `sha256(entity_id)` の一意索引）。長さ制限自体は消えるが、`ORDER BY entity_id` を支える索引が同じ btree なので同じ制限を持ち、問題は移動するだけである。

## Plan
- 仕様 → 共通ヘルパー → domain / usecase → Postgres → UI の順に進める。
- 鍵の成分は書き込み側でだけ上限を課す。`GET /Users/{scim_id}` のような検索は長い値でも btree に触れないので、404 のままにする。長い path param を 422 にすると SCIM クライアントの期待を壊す。

## Tasks
- [x] T001 [Inventory] 対象列と、外部標準が定める上限の有無を調査して報告する。
- [x] T002 [Spec] 資源上限の区分と根拠を String length limits に追記し、TypeSpec に `@maxLength` を足す。
- [x] T003 [Shared] `backend/shared/spec/length.go` にバイトの上限と `KeyString` / `CheckMaxBytes` / `CheckKeyString` / `TruncateChars` を足す。
- [x] T004 [Domain] 上限を Go 側で強制し、管理 API では 422、プロトコル接点ではそのプロトコルのエラーになることを確認する。
  - RED: `TestAdminServiceProvider_RejectsEntityIDOverTheCeiling` / `TestAdminServiceProvider_RejectsMultibyteEntityIDOverTheByteCeiling` はいずれも 201 を返していた（超過値をそのまま受理し、PostgreSQL へ渡していた）。
- [x] T005 [Postgres] `CHECK` を追加し、psqldef が収束することを確認する。
- [x] T006 [UI] `lengthLimits.ts` の URI 区分を entity ID と wtrealm の入力欄に配線する。
- [x] T007 [Tests] 標準が許す実例を拒否しないこと、上限 ±1、btree 上限に達する前に止まること、鍵のバイト予算が 2704 の内側であることを検証する。

### T001 の結果

着手時の調査で、この work item を filed した時点の前提が 4 か所で誤っていることが分かった。

| 記述 | 実際 |
|---|---|
| `saml_sp_trusts` / `wsfed_rp_trusts` | テーブル名は `saml_service_providers` / `wsfed_relying_parties` |
| `scim_id` は外部が値を決める | IdMagic が採番する。`backend/sourcing/scim/usecases/users.go` が 16 バイト乱数の hex 32 文字を生成する。RFC 7643 §3.1 の `id` は service provider が割り当てる値で、IdMagic がその service provider である |
| `kid` は外部が値を決める | IdMagic が生成する。`backend/signingkeys/keys_jose/rsa_jwk.go` の RFC 7638 JWK thumbprint（base64url 43 文字）。OpenBao 経路も同じ関数を通る |
| 対象は 6 列 | 同じ不具合を持つ btree 鍵成分が他に 6 列ある |

`scim_id` と `kid` を外から受け取るのは検索経路だけで、`SELECT` は btree の索引行上限に触れない。書き込み側は常に IdMagic が生成する。

**外部由来の btree 鍵成分**

| 列 | コードポイント | バイト | 根拠 |
|---|---|---|---|
| `saml_service_providers.entity_id` | 2048 | 2048 | URI 区分。標準は 1024 文字（`saml-schema-metadata-2.0.xsd` の `entityIDType`）だが、非準拠の SP を拒否しないため 2 倍を取る |
| `saml_authnrequest_replays.entity_id` | 2048 | 2048 | 上に同じ。登録済み SP の entity ID を写す |
| `saml_authnrequest_replays.request_id` | 256 | 256 | 資源上限。AuthnRequest の `ID`（`xs:ID`）に標準の長さ規定はない |
| `wsfed_relying_parties.wtrealm` | 2048 | 2048 | URI 区分。WS-Federation は `wtrealm` を URI としか定めない |
| `federated_identities.external_subject` | 512 | 1024 | 資源上限。OpenID Connect Core 1.0 §2 は `sub` を「MUST NOT exceed 255 ASCII characters in length」と定めるが、SAML の NameID には規定がないので広く取る |
| `federated_response_replays.response_id` | 256 | 256 | 資源上限。SAML `Response` の `ID` |
| `oauth2_replay_jtis.jti` | 256 | 256 | 資源上限。DPoP proof と client assertion の `jti`。RFC 7519 に長さ規定はない |
| `webauthn_credentials.credential_id` | 2048 | 2048 | 資源上限。WebAuthn Level 3 は credential ID を「At most 1023 bytes long」と定めるので base64url で 1364 文字。2048 は余白を残す |

**IdMagic が採番する鍵成分**（安全網としての上限）

| 列 | 上限 | 根拠 |
|---|---|---|
| `scim_user_refs.scim_id`, `scim_group_refs.scim_id` | 64 | Handle 区分。実値は hex 32 文字 |
| `signing_keys.kid` | 64 | Handle 区分。実値は base64url 43 文字 |
| `oauth2_access_token_denylist.jti` | 64 | Handle 区分。IdMagic が発行した token の `jti` |

いずれも ASCII しか取らないので、コードポイントの上限とバイトの上限が同じ数になる。

**鍵の成分ではない外部由来の値**

| 値 | 上限 | 扱い |
|---|---|---|
| `oauth2_clients.tls_client_auth_subject_dn` | 2048 | URI 区分。超過は 422 |
| `provisioning_connections.quarantine_reason` | 500 | Description 区分。連携先のエラー文であり利用者入力ではないので、拒否せず書き込み側で切り詰める |
| 各種 `*_hash`（`token_hash`、`code_hash`、`key_hash`、`device_hash`、`identifier_hash`） | 置かない | 外部入力を通らない固定長のダイジェスト |

**鍵ごとのバイト予算**

| 鍵 | 合計 |
|---|---|
| `saml_service_providers (tenant_id, entity_id)` | 16 + 2048 = 2064 |
| `saml_authnrequest_replays (tenant_id, entity_id, request_id)` | 16 + 2048 + 256 = 2320 |
| `wsfed_relying_parties (tenant_id, wtrealm)` | 16 + 2048 = 2064 |
| `federated_identities (tenant_id, provider_id, external_subject)` | 16 + 128 + 1024 = 1168 |
| `webauthn_credentials (credential_id)` | 2048 |

## Verification
- `just check-spec`
- `just check-api-compat`
- `just check-schema`
- `just verify`
- 手動確認: 各識別子について、上限の根拠が標準の引用か資源上限の説明として記録されている。
- 手動確認: 2704 バイトに達する前に、すべての経路が宣言された上限で止まる。

## Risk Notes
外部プロトコル識別子に短すぎる上限を置くと相互運用性を壊す。これらは安全のために置く資源の境界であって、業務上望ましい長さではないので、迷ったら広く取る。標準が数を定めている値でも、その数をそのまま上限にはしない。標準を超える値を出す実装が存在しうる以上、標準の数は「上限が拘束的でないことの根拠」に留める。

## Completion
- **Completed At**: 2026-08-16
- **Summary**:
  外部が値を決める識別子の「上限なし」を、宣言された上限に置き換えた。意味上の差分は 3 点である。
  - **暗黙の上限を宣言された上限にした**。これらの列は主キーや一意索引の成分であり、PostgreSQL の btree v4 が索引行 1 件を 2704 バイトに制限する。つまり上限は既に存在し、宣言されておらず、超過は `SQLSTATE 54000` として 500 で返っていた。`wi-128` が長さ違反に与えた 422 の扱いから漏れていた。上限を宣言し、超過が btree に到達する前に止まるようにした。回帰テストは `TestAdminServiceProvider_RejectsMultibyteEntityIDOverTheByteCeiling`（着手時は 201 を返し、PostgreSQL へ渡していた）。
  - **上限を 2 段にした**。契約の上限はコードポイント、索引の鍵の成分にはバイトの資源上限を重ねる。コードポイントだけでは足りない。標準が 1024 文字と定める entity ID でも、すべて 3 バイト文字なら 3072 バイトになり、宣言した契約を満たす値のまま btree で落ちる。資源をバイトで測るのは、btree が制限しているものがバイトだからである。`spec/SPECIFICATION.md` の「単位はコードポイント」という原則の例外に、「標準自身がオクテットで定めている値」に加えて「資源そのものがバイトで測られる場合」を足した。
  - **違反の伝え方を接点ごとに分けた**。管理 API は `wi-128` どおり 422 / `field_length_exceeded`。SAML・WS-Federation・OAuth 2.0・WebAuthn の接点は、そのプロトコルが定めるエラー形で返す。AuthnRequest を送ってきた相手に Problem Details を返しても読めない。プロトコル接点では型付きの `*spec.LengthError` を返さないことを `TestParseAuthnRequestLengthErrorIsNotAFieldLengthViolation` が守る。
- **着手時の前提の訂正**:
  - `scim_id` と `kid` は外部が決めない。`scim_id` は `backend/sourcing/scim/usecases/users.go` が採番する 16 バイト乱数の hex 32 文字で、RFC 7643 §3.1 の `id` は service provider が割り当てる値である（IdMagic がその service provider）。`kid` は `backend/signingkeys/keys_jose/rsa_jwk.go` が計算する RFC 7638 の JWK thumbprint（base64url 43 文字）で、OpenBao 経路も同じ関数を通る。これらを外から受け取るのは検索経路だけで、`SELECT` は btree の索引行上限に触れない。
  - work item が挙げたテーブル名（`saml_sp_trusts` / `wsfed_rp_trusts`）は存在せず、実際は `saml_service_providers` / `wsfed_relying_parties` である。
  - 同じ不具合を持つ列が他に 6 つあった。`saml_authnrequest_replays.request_id`、`federated_identities.external_subject`、`federated_response_replays.response_id`、`oauth2_replay_jtis.jti`、`oauth2_access_token_denylist.jti`、`webauthn_credentials.credential_id`。いずれも相手が決めた値がそのまま鍵の成分になる。
- **標準の扱い**:
  原典を確認した。`saml-schema-metadata-2.0.xsd` の `entityIDType` は `<restriction base="anyURI"><maxLength value="1024"/></restriction>`、OpenID Connect Core 1.0 §2 の `sub` は "It MUST NOT exceed 255 ASCII characters in length"、WebAuthn の credential ID は "At most 1023 bytes long"。ただしこれらの数をそのまま上限にはしていない。標準を超える entity ID を出す SP は実在しうるため、上限は URI 区分の 2048 に置き、標準の数は「上限が実運用で拘束的でないことの根拠」として記録した。
- **鍵ごとのバイト予算**:
  複合鍵の索引行は成分ごとではなく 1 件として制限されるので、上限は鍵の合計で守らなければならない。合計を 2400 バイト以下（btree の 2704 の内側）に収め、この条件をこの文書ではなく `TestIndexKeysFitInBtreeRowLimit` が守る。上限を後から広げて予算を割ったときに、そこで分かる。
- **IdMagic が採番する鍵の成分**:
  `scim_id`、`kid`、発行した token の `jti` には `CHECK` だけを置き、Go 側の検査は置かなかった。拒否すべき外部の入力が無く、上限を超えるのは採番の実装が壊れたときだけである。それは利用者の誤りではないので 422 にしない。この切り分けは仕様に明記した。
- **`CHECK` の書き方**:
  1 列につき名前つきの `CHECK` を 1 つ置き、`char_length` と `octet_length` を `AND` で結んだ。`wi-128` が記録したとおり、同じ列に `CHECK` を 2 つ並べると psqldef の差分が収束しない。
- **意図的に見送った点**:
  - 鍵の設計変更（代理キー + ダイジェストの一意索引）。btree の索引行上限自体は回避できるが、`ORDER BY entity_id` を支える索引が同じ btree なので同じ制限を持ち、問題は移動するだけである。
  - 各種 `*_hash`（`token_hash`、`code_hash`、`key_hash`、`device_hash`、`identifier_hash`）。外部入力を通らない固定長のダイジェストなので上限を置かない。判断として仕様に記録した。
- **Verification Results**:
  - `just verify` - passed（check / check-api-compat / lint-go / test-go / lint-ui / format-check-ui / test-ui-unit / build-ui / test-tools / typecheck-tools）
  - `just check-spec` - passed
  - `just check-api-compat` - passed（`@maxLength` の追加は既存の受理範囲を狭めるが、破壊的変更としては検出されない）
  - `just check-schema` - passed（空 DB へ適用 → プレビュー空 → 再適用 → プレビュー空。1 列 1 `CHECK` にまとめたので psqldef が収束する）
  - `just check-boundaries` - passed
  - `just test-go-race` - passed（DATA RACE なし）
  - `just verify-ui` - passed
  - `just spec-render` - passed（826 ページ再生成。生成物は未追跡）
  - `just spec-diff` - `no normative specification change against main`（規範シナリオの追加・削除・変更なし。変更は Design と TypeSpec の制約）
  - `just check-work-items` / `just check-ids` - passed
  - 新規テスト: `backend/shared/spec/key_length_test.go`（鍵ごとのバイト予算が btree 上限の内側であること、`KeyString` が 2 つの上限を課すこと、違反した上限の単位で報告すること、標準が許す実例を拒否しないこと、`TruncateChars` がコードポイント境界で切ること）、`backend/saml/handlers_http/admin_service_provider_length_test.go`（標準の 1024 文字と上限ちょうどを受理、上限 +1 と多バイト超過を 422 とフィールド名つき `detail`）、`backend/wsfederation/handlers_http/admin_relying_party_length_test.go`（同上）、`backend/saml/domain/authnrequest_length_test.go`（AuthnRequest の `ID` の上限 ±1、プロトコル接点が型付きの長さエラーを返さないこと）、`backend/saml/db_postgres/service_provider_length_test.go`（domain が受ける値は DB も受ける / domain を迂回した超過を `saml_service_providers_entity_id_length` が止める / 多バイト超過が btree ではなく `CHECK` で止まる）。
  - 手動確認: 各識別子について、上限の根拠が標準の引用か資源上限の説明として `spec/SPECIFICATION.md` の String length limits に記録されている。
  - 手動確認: 2704 バイトに達する前に、管理 API・プロトコル接点・DB のいずれの経路も宣言された上限で止まる。
