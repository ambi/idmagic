---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: docs
affected_spec:
  - { path: docs/domain/application/assignment/README.md, requirement: REQ-APPLICATION-014, impact: modifies }
  - { path: docs/domain/authorization/model/README.md, requirement: REQ-AUTHORIZATION-010, impact: modifies }
  - { path: docs/domain/claim-mapping/issuance/README.md, requirement: REQ-CLAIMMAPPING-001, impact: modifies }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-005, impact: modifies }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-012, impact: modifies }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-089, impact: modifies }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-002, impact: modifies }
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-001, impact: modifies }
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-002, impact: modifies }
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003, impact: modifies }
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-004, impact: modifies }
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-005, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientSecretCredentialStatus, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientSecretIssued, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientSecretLimitExceededError, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientSecretRevoked, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientType, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.FapiProfile, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.GrantType, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ResponseType, impact: modifies }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.TokenEndpointAuthMethod, impact: modifies }
  - { path: spec/contexts/audit/models.tsp, symbol: IdMagic.Contract.AdminAuditEventResponse, impact: modifies }
  - { path: spec/contexts/audit/models.tsp, symbol: IdMagic.Contract.AuditEventQuery, impact: modifies }
  - { path: spec/contexts/audit/models.tsp, symbol: IdMagic.Contract.AuditEventSearchAttribute, impact: modifies }
  - { path: spec/contexts/authorization/models.tsp, symbol: IdMagic.Contract.FgaCheckEvaluated, impact: modifies }
  - { path: spec/contexts/claim-mapping/models.tsp, symbol: IdMagic.Contract.AttrVisibility, impact: modifies }
  - { path: spec/contexts/claim-mapping/models.tsp, symbol: IdMagic.Contract.UserAttributeDef, impact: modifies }
  - { path: spec/contexts/data-keys/models.tsp, symbol: IdMagic.Contract.DataKeyStillReferencedError, impact: modifies }
  - { path: spec/contexts/data-keys/models.tsp, symbol: IdMagic.Contract.EncryptedSecret, impact: modifies }
  - { path: spec/contexts/identity-governance/models.tsp, symbol: IdMagic.Contract.AssignmentVisibility, impact: modifies }
  - { path: spec/contexts/identity-governance/models.tsp, symbol: IdMagic.Contract.RequiredAction, impact: modifies }
  - { path: spec/contexts/identity-governance/models.tsp, symbol: IdMagic.Contract.UserStatus, impact: modifies }
  - { path: spec/contexts/identity-governance/models.tsp, symbol: IdMagic.Contract.WorkflowActionDef, impact: modifies }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.CreateAdminUser, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AdminUserCreateRequest, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AttributeType, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupImportJob, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupImportJobRef, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipImportJob, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipImportJobRef, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UserImportJob, impact: modifies }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UserImportJobRef, impact: modifies }
  - { path: spec/contexts/jobs/models.tsp, symbol: IdMagic.Contract.AdminJobResponse, impact: modifies }
  - { path: spec/contexts/jobs/models.tsp, symbol: IdMagic.Contract.JobRef, impact: modifies }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ResumeFederatedAuthorization, impact: modifies }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AuthorizationRequest, impact: modifies }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.LogoutNotification, impact: modifies }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.Application, impact: modifies }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ApplicationAssignment, impact: modifies }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.AssignmentSubjectType, impact: modifies }
  - { path: spec/contexts/saml/models.tsp, symbol: IdMagic.Contract.ClaimMappingPolicy, impact: modifies }
  - { path: spec/contexts/seeding/models.tsp, symbol: IdMagic.Contract.SeedManifest, impact: modifies }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.AccessDeniedError, impact: modifies }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.InvalidRequestError, impact: modifies }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.QuotaExceededError, impact: modifies }
  - { path: spec/contexts/system/models.tsp, symbol: IdMagic.Contract.DomainEventEnvelope, impact: modifies }
  - { path: spec/contexts/system/models.tsp, symbol: IdMagic.Contract.DomainEventPayload, impact: modifies }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetAdminSettings, impact: modifies }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateAdminSettings, impact: modifies }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.AdminSettingsResponse, impact: modifies }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.PasswordPolicyDefaults, impact: modifies }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.PasswordPolicyOverride, impact: modifies }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.Tenant, impact: modifies }
  - { path: spec/contexts/workloadidentity/models.tsp, symbol: IdMagic.Contract.AgentRef, impact: modifies }
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: リポジトリ内の設計文書、仕様の散文、コードのコメント、用語の検査だけを変え、製品の利用者と運用者が観測する振る舞い、設定、API は変わらない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/terminology.ts
    - tools/check/src/check-terminology.ts
    - DOCUMENTATION_GUIDE.md
    - docs/domain/README.md
  tests:
    - tools/check/src/terminology.test.ts
  stop_before_reading: [frontend]
---

# 残りの文書の Context をモジュールへ改名し、用語の検査で固定する

## 動機

モジュール設計の項目は、概念を定義する文書と、その項目で変えた検査の識別子と出力だけを改名した。
方針の変更と機械的な置換を同じ差分に混ぜると、方針の変更をレビューで読み分けられなくなるからである。
その結果、ほかの文書、コードのコメント、各モジュールの仕様の散文には「Bounded Context」「Context Map」「Context 間」「公開言語」などの語が残り、「モジュール」と混在している。
語が混在すると、読み手は二つの語が同じものを指すのかを毎回判断しなければならない。

## 対象範囲

- 着手時に `rg` で残りの語を検索し、文書、仕様の散文、TypeSpec の説明、コードのコメント、検査の出力文言を改名する。
  語の対応は[モジュール設計の項目](wi-92970-adopt-information-hiding-modular-design.md)の「用語の改名」の表に従う。
- 廃止した区分の判断を記した節を、各モジュールの判断の文書から削除する。
- 改名の完了後に、`mise run check-terminology` へ設計の単位としての「Context」と「公開言語」「Published Language」を採らない表記として追加する。

## 対象外

- 外部文献のタイトル、C4 の System Context、Go の `context.Context`、DNS のサブドメインなど区分ではない意味の語。
- 固定したパスと規範 ID（`docs/domain/<名前>/`、`spec/contexts/<名前>/`、`REQ-<名前>-NNN`）と TypeSpec の名前空間。
- 完了済みの作業記録。
- 未完了の作業記録。用語の検査は `work-items/` を書かれた時点の記録として対象にしないので、着手時に各項目が改名する。
- 旧形式の文書を読む互換処理と、そのテストの入力（`boundary-fitness.ts`、`boundary-debt-ratchet.test.ts`、`schema-tables.ts` の旧い見出し）。基準 revision の旧い文書を読むための固定の文字列である。
- API 互換の基準として凍結した `spec/idmagic.openapi.baseline.json`。

## 設計

### 着手時の棚卸し

着手時の revision は `cd85b4295` である。
`docs/`、root 直下の文書、`backend/`、`spec/`、`tools/` で、改名の対象の語は約 300 ファイル、約 1000 行に残っていた。
大半は `docs/domain/<モジュール>/` の定型の表現（「この Context が提供する」「この Context での語義」「該当なし：Context の設計に従う」など）である。
コードのコメントには小文字の「bounded context」「published language」「context_map」が約 100 箇所あった。

| 見つかったもの | 扱い |
| --- | --- |
| 各モジュールの `design/decisions.md` の「この Context を Core／Supporting／Generic に分類する」の節（21 個） | 改名ではなく削除する。区分は廃止したので、節が述べる決定そのものが無効である。四つのモジュールの `design/architecture.md` がこの節を根拠として引いていたので、表の理由を自己完結させ、委譲の根拠は設計ガイドラインの「確立した実装への委譲」へ付け替える |
| `SPECIFICATION_FORMAT.md` の、ルートの索引で区分を付ける *(checked)* の規則 | 区分の検査はすでに撤去したので、一覧を責務表に一度だけ書く規則へ書き換える |
| TypeSpec の説明の「公表された言語として写したもの」 | 「公開契約として写したもの」へ改名する。生成する API リファレンスの説明文に現れる |
| Go のコメントの「Context Map の向き」を理由にした説明（6 箇所） | 現在の規則（モジュール単位の非循環、公開パッケージ経由の依存）で理由を書き直す |
| 仕様の木の階層としての「コンテキスト」（`DOCUMENTATION_GUIDE.md` の第 3 節など） | 改名しない。方法論の文書が、モデルと用語体系が通用する範囲として定義した語であり、実装の単位を指さない |
| DNS のホスト名の意味の「サブドメイン」、テナントの解決方式の `Subdomain` | 改名しない |
| C4 の System Context、Go の `context.Context` と `echo.Context`、パス、規範 ID、TypeSpec の名前空間、検査のコード中の識別子（`CONTEXTS` など） | 改名しない |

### 用語の検査の規則

規則は単独の「Context」と「公開言語」「Published Language」とする。
棚卸しでは、検査の対象の文書に残る正当な「Context」は `context.Context` と C4 の System Context の二つだけだった。
そこで「Bounded Context」「Context Map」「Context 間」を個別に並べず、単独の「Context」を対象にし、二つを `allow` の literal で通す。
個別に並べると、「各 Context」「所有 Context」のような未知の組み合わせを見逃す。

区分の意味の「Subdomain」「サブドメイン」は規則にしない。
区分の節と列を削除した後に残る用法は、すべて DNS とテナントの解決方式の意味であり、literal では区分の意味と区別できない。
規則にすると、正当な用法をすべて `allow` に並べることになる。

### 規範の文面への影響

改名は規範の要件の文面と TypeSpec の説明にも及ぶ。
語を差し替えるだけで、主体、条件、応答、状態、イベントの意味は変えない。
`affected_spec` に、文面が変わった規範シナリオと TypeSpec の宣言を `modifies` として列挙する。
標準の行のうち `docs/domain/standards.md` の GDPR-CONSENT-WITHDRAWAL、GDPR-PROCESSING-RECORDS と、`docs/domain/sourcing/standards.md` の RFC7644-BEARER-AUTHORIZATION は、担当の名前に「Context」を含むが、文面を変えない。
標準の行を変える作業項目には主要ユースケースの証拠が必要になるが、語の差し替えには固定する振る舞いがなく、RED を観測できないからである。
用語の検査は、この 3 行が使う「ApiTokens Context」「OAuth2 Context」「Audit Context」を `allow` の literal で通す。
3 行の改名は、その行に触れる次の作業項目が行い、同時に `allow` を外す。

## タスク

- [x] T001 [Inventory] 残りの語を検索し、改名の対象と除外を分ける。
- [x] T002 [Docs] 文書、仕様の散文、TypeSpec の説明、コメントを改名し、区分の節を削除する。
- [x] T003 [Tooling] 用語の検査に規則を追加し、除外が誤検出されないことを確かめる。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-tools-file -- check/src/terminology.test.ts`
- `mise run check-terminology`
- `mise run check-links`
- `mise run verify`

## リスク

- 機械的な置換が、区分ではない意味の語や固定した識別子まで変えるおそれがある。
  置換は文脈を確かめながら行い、検査の規則には除外を明示する。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` が main に対して示した規範の差分は、規範シナリオ 12 件（REQ-APPLICATION-014、REQ-AUTHORIZATION-010、REQ-CLAIMMAPPING-001、REQ-DATAKEYS-005、REQ-IDMANAGEMENT-012、REQ-IDMANAGEMENT-089、REQ-JOBS-002、REQ-PLATFORM-001〜005）と TypeSpec の宣言 51 件の文面である。
  どれも「Context」「公表された言語」を「モジュール」「公開契約」へ差し替えただけで、主体、条件、応答、状態、イベントの意味は変えていない。
  文書、仕様の散文、TypeSpec の説明、Go と TypeScript のコメントの旧い語を改名し、各モジュールの判断の文書から廃止した区分の節 21 個を削除した。
  `SPECIFICATION_FORMAT.md` の区分を付ける規則は、一覧を責務表に一度だけ書く規則へ書き換えた。
  `mise run check-terminology` は、設計の単位としての「Context」と「公開言語」「Published Language」を拒否する。
  通すのは `context.Context`、C4 の System Context と、文面を変えなかった標準の 3 行の語である。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-terminology`。
  - **要件**: N/A: 製品の振る舞いを変えない用語の検査と文書の変更であり、規範となる製品要件はない。
  - **観測した失敗**: 規則を足した直後、改名前の文書に対して 690 件の指摘で失敗した。
  - **検出できる理由**: 指摘は残った語の位置ごとに出るので、改名の漏れを文書と行で特定できる。改名後は 0 件になった。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/terminology.test.ts` の「設計の単位としての Context と公開言語を指摘し、Go の型と C4 のビュー名は通す」。
  - **要件**: N/A: 製品要件のない検査の変更である。
  - **観測した失敗**: 規則を足す前は、指摘が空で期待した 6 件と一致せずに失敗した。
  - **検出できる理由**: 期待値は「Bounded Context」「Context Map」「Context 間」を含む各位置の指摘と、`context.Context` と System Context を通すことの両方を表明する。
- **変更耐性の結果**:
  規則と許容を一つずつ外す故障を注入し、単体テストを実行した。
  「Context」の規則の対象語を変えると 2 件、`context.Context` の許容を外すと 1 件が失敗した。
- **検証結果**:
  - `mise run check-terminology` - 成功
  - `mise run check-links` - 成功
  - `mise run verify` - 成功
