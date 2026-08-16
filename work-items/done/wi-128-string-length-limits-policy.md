---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-05
change_kind: bugfix
initial_context:
  typespec:
    - Product.IdManagement.User
    - Product.IdManagement.Group
    - Product.Authorization.RelationTuple
  source:
    - backend/shared/spec/validation.go
    - backend/shared/http/support_http/error_handler.go
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/agent/domain/agents.go
    - backend/tenancy/domain/tenancy.go
    - backend/oauth2/client/domain/client.go
    - backend/workloadidentity/domain
    - backend/authorization/domain/tuple.go
    - infra/schema/postgres.sql
  tests:
    - backend/idmanagement/group/handlers_http
    - backend/shared/http/support_http/error_handler_test.go
  stop_before_reading:
    - frontend/src/features
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: Product.IdManagement.User }
  - { path: spec/contexts/identity-management/models.tsp, symbol: Product.IdManagement.Group }
  - { path: spec/contexts/authorization/models.tsp, symbol: Product.Authorization.RelationTuple }
---

# 文字列入力値の最大文字数ルールを定義し必要な DB 制約へ反映する

## Motivation
PostgreSQL では `TEXT` と制約なし `varchar` に実質的な性能差はなく、`varchar(n)` を使う主な理由は最大文字数制約を表現することにある。現時点では、表示名、説明、URL、メール、ラベル、外部プロトコル識別子などの最大文字数が業務ルールとして明確に決まっていない。

最大文字数を決めないまま DB 型だけを `varchar(n)` に変えると、根拠のない上限で外部互換性や将来の拡張を壊す可能性がある。一方で、アプリケーション側の validation 漏れにより過大な文字列が DB に保存されることも避けたい。まず値カテゴリごとの上限を業務・仕様・UI・運用の観点から決め、必要な箇所だけアプリケーション validation と SQL 制約の両方に反映する。

着手時の調査で、上限の「数値」より先に**単位と適用点**が壊れていることが分かった。次の 2 つは実害のある不具合である。

- **単位の不一致**：TypeSpec の `@maxLength`、PostgreSQL の `char_length()`、JSON Schema の `maxLength` はいずれも Unicode コードポイントを数えるのに対し、Go 側の検証に使っている zog の `String().Max(n)` は `len(string)`、すなわち **UTF-8 バイト数**を数える。ドキュメントコメントは "at most n characters long" と書いてあるが実装はバイト数である。このため、上限 100 文字と宣言したグループ名に日本語を入れると **34 文字で拒否される**。同じ値を DB は 100 文字まで受け入れるので、公開契約・DB・実装の 3 者がそれぞれ別の上限を持っている。
- **違反の表現**：長さ違反は domain の `Validate()` から素の `error` として返り、各 handler のエラー写像の `default:` を素通りして **HTTP 500** になる。利用者には `{"message":"Internal Server Error"}` しか届かず、どのフィールドがどの上限を超えたのか分からない。

## Scope
- **policy / documentation**:
  - 文字列値カテゴリごとの最大文字数ポリシーを定義する。対象候補:
    - tenant / user / group / agent / application / client / category の表示名・名前・説明。
    - メールアドレス、URL、URI、SAML entity_id、WS-Fed realm、SCIM id、OIDC client_id など外部プロトコルと接する識別子。
    - token description、key id、object key、content type、audit type、outbox topic / event_type / published_to、エラー文字列。
    - tenant id、user id、group id など domain id。
  - RFC・外部仕様・主要 IdP の慣行・UI 表示上限・検索 index サイズ・ログ/監査保存量を参照し、上限を置く値と置かない値を分類する。
  - 上限を DB に置く場合、`varchar(n)` と `TEXT CHECK (char_length(column) <= n)` のどちらを採用するかを `wi-127-postgres-column-type-policy` の型ポリシーと整合させる。
- **spec**:
  - 最大文字数が公開 contract、管理 UI 入力制約、または保証義務に関わる場合は、specification-first で `spec/SPECIFICATION.md` を最小限更新し、derived artifacts を再生成する。
- **implementation**:
  - 決定した上限を、HTTP request validation、domain/service validation、UI form validation、OpenAPI/JSON Schema など該当する境界に反映する。
  - DB 側の最後の防衛線が必要な列には、`infra/schema/postgres.sql` と migration / seed / test fixture を更新する。
  - 制約違反時のエラーが API / UI で利用者に理解できる表現になるようにする。
- **tests**:
  - 境界値ちょうど、1 文字超過、空文字/空白のみ、マルチバイト文字を含む入力を確認する。
  - 外部プロトコル識別子は、仕様上許される実例を誤って拒否しないことを確認する。

## Out of Scope
- `TEXT` / `varchar` / `JSONB` / `UUID` / enum などの列型一般の選定ポリシー策定。これは `wi-127-postgres-column-type-policy` で扱う。
- 文字数上限を根拠なく全列に機械的に設定すること。現に上限を持たない列（audit payload、token hash、`entity_id` / `wtrealm` / `scim_id` などの外部プロトコル識別子）へ、この work item で新たに上限を導入することはしない。
- 外部仕様で長さが明確でない識別子を、DB 都合だけで短く切ること。
- 表示上の省略や折り返しだけで十分な値に、永続化上限を過剰に導入すること。
- 既存の上限値そのものの引き下げ。公開契約を狭める変更は互換性を壊すため、この work item では単位・適用点・不足分の追加だけを行う。
- HTTP body 全体のサイズ上限、配列要素数、ネスト深さ。`wi-110-http-server-hardening-timeouts-and-body-limits`（完了済み、`HTTP_MAX_BODY_BYTES`）が所有する。

## Design

### 単位を 1 つに固定する

上限の単位は **Unicode コードポイント**とする。この単位を選ぶのは、境界のうち 3 つがすでにそれを使っているからである。

| 境界 | 数えるもの |
|---|---|
| TypeSpec `@maxLength` → OpenAPI / JSON Schema | コードポイント |
| PostgreSQL `char_length(col)` | コードポイント |
| Go `utf8.RuneCountInString` | コードポイント |
| Go zog `String().Max(n)` | **UTF-8 バイト**（不一致） |

したがって zog の `Max` / `Min` は文字列フィールドに使わない。`backend/shared/spec` にコードポイントを数える `Chars` / `CharsAtMost` を置き、全 domain schema をそちらへ移す。

例外は、準拠する標準自身がオクテットで上限を定めている値に限る。メールアドレス（RFC 5321 §4.5.3.1 の 254 オクテット）と DNS ラベル（63 オクテット）がそれで、これらは単位をバイトと明示する。いずれも書式が ASCII に制限されるため、実際にはコードポイント数と一致する。

### 上限を 4 つの境界へ同じ数で置く

| 境界 | 役割 |
|---|---|
| TypeSpec `@maxLength` | 公開契約。OpenAPI と生成ドキュメントの表示元。 |
| Go domain schema (`spec.Chars`) | 唯一の強制点。ここを通らない書き込み経路を作らない。 |
| PostgreSQL `CHECK (char_length(col) <= n)` | 最後の防壁。ここで落ちたら実装の不具合であり、利用者向けエラーの発生源にしてはならない。 |
| UI の `maxLength` 属性 | 入力補助。保証ではない。 |

### 違反を 422 で返す

長さ違反は解析できた内容が業務規則に違反する場合なので、`spec/SPECIFICATION.md` の HTTP error responses が定める 422 に当たる。`spec.Chars` の検査失敗だけを型付きの `*spec.LengthError` として返し、`support_http.ErrorHandler` が既存の `quotaExceeded` と同じ構造的インターフェースで受けて `422` の Problem Details に写像する。context ごとの handler を 1 つずつ直さずに全経路が同時に直り、かつ長さ以外の検証失敗の現在の挙動は変えない。

### 検討して採らなかった案

- **zog の `Max` を残し、バイト換算した値を渡す**：`@maxLength(100)` に対して Go 側だけ 400 を渡す形になる。3 つの境界の数値が一致しなくなり、契約を読んだだけでは実際の上限が分からない。
- **すべての検証失敗を 422 にする**：`spec.Validate` は DB から読んだ集約の再検証にも使われる。保存済みデータの破損まで 422 になると、サーバ側の不具合が利用者入力の誤りに見える。長さ違反だけを型付きにする。
- **上限値の統一（80 → 100 など）へ寄せる**：branding の 80 や realm の 63 には表示面と DNS という別々の根拠がある。値の種類を減らすこと自体は目的ではないので、根拠を明記したうえで残す。
- **golangci-lint の forbidigo で `.Max(` を禁止する**：`z.Int()` など文字列以外の `Max` も巻き込み、誤検知の抑制コメントが増える。代わりに `just check` へ TypeSpec と Go の突き合わせを追加する。

## Plan
- `infra/schema/postgres.sql` の現行policy（unconstrained varchar禁止、limitはTEXT+CHECKまたはvarchar）を基礎に、specification model fieldごとにprotocol limit、security/resource limit、UI usability limitを分類したregistryを作る。一律255文字にはしない。
- 文字数の単位はfieldごとにUTF-8 bytes、Unicode code points、protocol-defined bytesを明示する。Goの`len`とPostgreSQL`char_length`の差を放置せず、正規化が必要なidentifier/email/URIはnormalize後に測る。
- validationはdomain/value constructorまたはusecase commandで正本化し、HTTP/UIは同じlimit metadata/error codeを表示する。DB CHECKはrace/bypassへの最後の防壁で、DB errorを500にしない。
- 既存schema/data/API inputをinventoryし、現存最大値と違反行をreportしてからconstraintを追加する。自動truncateはせず、互換が必要なexternal protocol fieldはより広い上限かmigrationを選ぶ。
- unbounded body/JSON/array/mapは文字列fieldとは別にHTTP body limit、element count、nesting depthで制限し、wi-110のbody limitsと重複実装しない。

## Tasks
- [x] T001 [Inventory] specification fields、Go structs/validators、HTTP forms、frontend inputs、Postgres text columnsを対応付け、現行/外部仕様/実data最大値をreportする。
- [x] T002 [Policy/specification] field別limit/unit/normalization/error codeを定義し、models/interfaces/constraints/contractsへ反映して再生成する。
- [x] T003 [Validation Core] code-point/byte/normalized length helpersとtyped errorsを追加し、各contextのvalue/commandへowner単位で適用する。
- [x] T004 [HTTP/UI] typed error→400/SCIM/OAuth protocol error mapping、OpenAPI maxLength（単位が一致するfieldのみ）、form max/remaining表示を追加する。
- [x] T005 [Postgres] data audit queryを通した後にCHECK/varchar制約をcontextごとに追加し、constraint error mappingとindex size影響を検証する。
- [x] T006 [Protocol Tests] OAuth URI/client metadata、SAML/WS-Fed identifiers、SCIM attributes、user/group/application fieldのlimit±1とoversize body/collectionを検証する。
- [x] T007 [Unicode/Compatibility] multibyte、combining、normalization、legacy最大値、DB/API/UIの同一判定と非truncateを検証する。

### T001 の結果

現に上限を持つフィールドの棚卸し。「Go」列の値は zog の `Max`（バイト）、「TypeSpec」と「DB」はコードポイント。

| 値 | TypeSpec | Go | DB CHECK | 根拠 |
|---|---|---|---|---|
| `Group.id` / `Agent.id` / `WorkloadTrustBundle.id` / `AgentWorkloadBinding.id` / `LifecycleWorkflowDefinition.id` | 64 | 64 | なし | 内部採番 id |
| `User.preferred_username` / `Group.name` / `Agent.name` / `WorkloadTrustBundle.name` / workflow `name` | 100 | 100 | なし | 一行の名前 |
| `User.name` | なし | 200 | なし | 表示名 |
| `User.given_name` / `User.family_name` | なし | 100 | なし | 名前の部分 |
| `User.email` / `Group.email` | なし | なし | なし | **未設定** |
| `Tenant.display_name` / `OAuth2Client.client_name` / `McpResourceServer.name` / role policy `name` | 200 | 200 | なし | 表示名 |
| `Group.description` / `Agent.description` / workflow `description` / `AuthorizationDetailType.description` / `display_template` / `AgentWorkloadBinding.subject_pattern` | 500 | 500 | なし | 説明文 |
| `TenantFooterLink.url` | 2048 | 2048 | なし | URL |
| `DynamicGroupRule.expression` | 4096 | なし | 4096 | CEL 式 |
| `NotificationTemplate.subject` | 200 | なし | 200 | 件名 |
| `NotificationTemplate.body_text` / `body_html` | 8000 / 20000 | なし | 8000 / 20000 | 本文 |
| `TenantBranding.product_name` / `from_display_name` / `TenantFooterLink.label` | 80 | 80 | 80（footer label のみ） | 固定枠の表示面 |
| `TenantBranding.footer_text` | 280 | 280 | なし | 固定枠の表示面 |
| `Tenant.realm` | 63 | 63 | 正規表現で 63 | DNS ラベル |
| `WorkloadTrustBundle.trust_domain` | 255 | 255 | なし | DNS 名 |
| `OAuth2Client.client_id` | 128 | 128 | なし | 内部採番 |
| `PasswordChangeRequest.new_password` | 128 | 128（コードポイント） | 該当なし | `PasswordPolicyMaxLength` |
| `RelationTuple` の `resource_type` / `relation` / `subject_type` / `subject_relation` | 64 | 正規表現のみ | 64 | 語彙的な名前 |
| `RelationTuple.resource_id` / `subject_id` | 256 | 256（バイトと明記） | 256 | 外部が決める識別子 |
| `CibaRequest.binding_message` | なし | 64（usecase はコードポイント、domain はバイト） | なし | 表示用の短文 |

上限を持たない主な列（この work item では触れない）：`entity_id`、`wtrealm`、`scim_id`、`kid`、各種 token hash、audit / outbox の payload、`scope`、`tls_client_auth_subject_dn`。

## Verification
- `just check-work-items`
- `just check-ids`
- `just check-spec`
- `just spec-render`
- `just check-schema`
- `just verify-go`
- `just verify-ui`
- `just verify`
- 手動確認: 文字列値カテゴリごとに、最大文字数を「置く / 置かない」とその根拠がドキュメント、specification、または ADR に残っている。
- 手動確認: 上限を置いた値について、API / UI / DB の各境界で同じ制限が適用され、違反時のエラーが利用者に理解できる。

## Risk Notes
文字数上限は一度公開 contract や DB 制約に入ると、外部連携・既存データ・UI 操作に影響する。特に SAML entity_id、OIDC client_id、URL/URI、SCIM id など外部プロトコルと接する値は、短すぎる上限で相互運用性を壊しやすい。実装時は、内部表示名のように閉じた値から先に制約を入れ、外部識別子は仕様根拠と実データ例を確認してから上限を決める。

単位をバイトからコードポイントへ変えることは、既存の受理範囲を**広げる**方向にしか働かないので、これまで通っていた入力が通らなくなることはない。逆に、これまで拒否されていたマルチバイト入力が保存されるようになるため、DB CHECK を同時に入れないと「Go は通すが DB が落ちる」区間が生まれる。CHECK と Go の緩和は同じ変更で入れる。

## Completion
- **Completed At**: 2026-08-16
- **Summary**:
  文字列長の上限に、単位（Unicode コードポイント）、既定の区分、4 つの境界それぞれの役割を与え、境界ごとにばらばらだった実際の上限を 1 つに揃えた。数値そのものはほぼ据え置きで、意味上の差分は「同じ数がどこでも同じ意味を持つようになったこと」と「違反が利用者に伝わるようになったこと」の 2 点である。着手時の調査で見つかった 2 つの不具合を実際に直した。
  - **マルチバイト入力が上限の 1/3 で拒否されていた**。zog の `String().Max(n)` は `len(string)`、すなわち UTF-8 バイト数を数える（ドキュメントコメントは "at most n characters long" と書いてある）。上限 100 文字と宣言したグループ名に日本語を入れると 34 文字で拒否されていた。`backend/shared/spec/length.go` に `Chars` / `CharsAtMost` を追加し、`utf8.RuneCountInString` で数えるようにしたうえで、10 ファイル・30 か所の `.Max(` を置き換えた。`authorization/domain/tuple.go` の「256 bytes」判定と、CIBA `binding_message` の usecase（コードポイント）と domain（バイト）の食い違いも同じ数え方に寄せた。
  - **上限超過が HTTP 500 になっていた**。domain の `Validate()` が返す素の error が各 handler のエラー写像の `default:` を素通りし、`{"message":"Internal Server Error"}` だけが返っていた。長さ違反だけを型付きの `*spec.LengthError` にし、`support_http.ErrorHandler` が既存の `quotaExceeded` と同じ構造的インターフェースで受けて `422` / `field_length_exceeded` の Problem Details に写像する。`detail` には違反したフィールドの wire 名と上限が載る（`name: must be at most 100 characters`）。長さ以外の検証失敗は素の error のまま残し、保存済みデータの破損がサーバの不具合として扱われる余地を保った。
  - このエラー写像は `backend/cmd/idmagic/server.go` だけで組み立てられていたため、ハンドラのテストは本番と別のエラー経路を通っていた。既定版を `server_http.Register` へ移し、ログとメトリクスを持つ版は起動時に cmd が差し替える形にした。分類漏れがテストから見えるようになる。
  - **仕様**：`spec/SPECIFICATION.md` の Cross-cutting Concerns に `String length limits` を追加。単位、既定の 9 区分、外部の標準や固定の表示面から決まる 7 つの例外、4 つの境界の役割、上限を置かない値、違反時の 422 を記述した。Database design policy §1 の「`TEXT` + `CHECK` か `varchar(N)` のいずれか」という未決を `TEXT` + `CHECK` に確定した。
  - **契約の穴埋め**：`User.name` / `given_name` / `family_name` / `email`、`Group.email`、`GroupAttributeDef.label`、CIBA `binding_message` に `@maxLength` を追加した。メールは RFC 5321 の 254。いずれも従来 Go 側だけが持っていた上限か、どこにも無かった上限である。
  - **DB の最後の防壁**：`users`、`groups`、`agents`、`tenants`、`tenant_brandings`、`oauth2_clients`、`mcp_resource_servers`、`authorization_detail_types`、`workload_trust_bundles`、`agent_workload_bindings` に `CHECK (char_length(col) ...)` を追加した。通知テンプレートは DB と TypeSpec に上限があるのに Go 側の検査が無く、超過が制約違反として返って 500 になっていたので、`template.ValidateDefinition` に同じ上限を入れた。
  - **UI**：`frontend/src/lib/lengthLimits.ts` に同じ区分を置き、group / user / agent の作成・編集、テナント設定、通知テンプレート、ライフサイクルワークフローの各入力欄の `maxLength` をそこから引くようにした。既存の直書きの数値も同じ表に寄せた。パスワード欄には付けない（貼り付けを黙って切り詰めるため）。
  - **上限値の変更**：`UserAttributeDef.OIDCScope` の 60 を Handle 区分の 64 へ広げた 1 件のみ。他はすべて据え置きで、引き下げは行っていない。
- **`CHECK` に置く規則の切り分け**:
  レビューで、`tenant_brandings.footer_link_{1,2}_url` のスキーム allowlist（`~ '^https://'`）を SQL に置くのが適切かを問われた。置かないのが正しい。上の表が `CHECK` に与えた役割は「実装の不具合だけが落ちる最後の防壁」であり、管理者が `http://` と打ったという通常の入力誤りで落ちる規則はその定義に反する。加えてスキームの集合は `mailto:` やオンプレの `http://` へ広がりうる可変な製品ポリシーで、DDL に置くと変更のたびに全配備でスキーマ移行が要る。長さは安定した資源境界なので SQL に残す資格がある。
  そこで `_url_format` を廃し、`_url_length`（2048）だけを置いた。スキーム規則は Go の `TenantBranding.Validate()` が引き続き強制し、`manage_branding_test.go` が `javascript:` / `http://` / `data:` の拒否を、`branding_handler_test.go` が HTTP 境界を検証している。この切り分けは `spec/SPECIFICATION.md` の String length limits に明文化した。
  なお psqldef が同一列に `CHECK` を 2 つ置くと収束しない件（当初これを理由に長さ CHECK を見送っていた）は、1 つの `CHECK` にまとめれば回避できることを確認した。今回は上の設計判断により、そもそもまとめる必要がなくなっている。
- **意図的に見送った点**:
  - `entity_id`、`wtrealm`、`scim_id`、`kid`、token hash など、現に上限を持たない外部由来の値。Out of Scope のとおり根拠なく上限を導入していないが、「上限なし」もまた検討されていない既定でしかない、というレビュー指摘は妥当である。実測では、これらが主キー成分になっている列は btree v4 の索引行上限 2704 バイトで既に破綻し、`SQLSTATE 54000`（`index row size ... exceeds btree version 4 maximum 2704`）という低水準のエラーを返す。資源上限としての明示的な天井を与える作業を `wi-380` に分離した。既存データの棚卸しを挟む必要があるため、この work item では扱わない。
  - TypeSpec の `UserAttributeDef` は `key` と `visibility` しか宣言していないが、handler は `userdomain.UserAttributeDef` をそのまま JSON 化しており、実際には `label` / `type` / `multi_valued` / `required` / `editable_by_user` / `claim_name` / `oidc_scope` / `pii` を含む 10 フィールドが線上に出ている。契約が 8 フィールド不足しているのは長さとは独立した仕様の欠落なので、`wi-381` に分離した。この work item では Go 側の数え方だけを直した。
- **Verification Results**:
  - `just verify` - passed（check / check-api-compat / lint-go / test-go / lint-ui / format-check-ui / test-ui-unit / build-ui / test-tools / typecheck-tools）
  - `just verify-go` - passed
  - `just test-go-race` - passed（DATA RACE なし）
  - `just verify-ui` - passed（653 pass / 0 fail）
  - `just check-spec` - passed
  - `just check-api-compat` - passed（破壊的変更なし。`@maxLength` の追加は既存の受理範囲を広げないが、既存の Go 側上限と同値なので実挙動は不変）
  - `just check-schema` - passed（空 DB へ適用 → プレビュー空 → 再適用 → プレビュー空）
  - `just spec-render` - passed（826 ページ再生成。生成物は未追跡）
  - `just spec-diff` - `no normative specification change against main`（規範シナリオの追加・削除・変更なし。変更は Design と TypeSpec の制約）
  - `just check-work-items` / `just check-ids` - passed
  - 新規テスト: `backend/shared/spec/length_test.go`（境界ちょうど・1 超過を ASCII / 日本語 / 絵文字で、結合文字は書記素ではなくコードポイントで数えること、型付きエラー、wire 名、`snakeCase`）、`backend/idmanagement/group/handlers_http/admin_group_length_test.go`（100 文字の日本語名が 201、101 文字が 422 と Problem Details の `detail`、説明の境界 ±1）、`backend/idmanagement/group/db_postgres/groups_length_test.go`（domain が受ける値は DB も受ける / domain を迂回した超過は `groups_name_length` が止める）、`frontend/src/lib/lengthLimits.test.ts`（UI の表がサーバと一致すること）。
  - 手動確認: 上限を置く値と置かない値、およびその根拠を `spec/SPECIFICATION.md` の `String length limits` に記載した。
  - 手動確認: グループ名で API / DB / UI が同じ 100 という数を同じ単位で適用し、超過時に `422 field_length_exceeded` とフィールド名・上限を含む `detail` が返ることを確認した。
