---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-05
---

# 文字列入力値の最大文字数ルールを定義し必要な DB 制約へ反映する

## Motivation
PostgreSQL では `TEXT` と制約なし `varchar` に実質的な性能差はなく、`varchar(n)` を
使う主な理由は最大文字数制約を表現することにある。現時点では、表示名、説明、URL、
メール、ラベル、外部プロトコル識別子などの最大文字数が業務ルールとして明確に決まって
いない。

最大文字数を決めないまま DB 型だけを `varchar(n)` に変えると、根拠のない上限で外部互換性や
将来の拡張を壊す可能性がある。一方で、アプリケーション側の validation 漏れにより過大な
文字列が DB に保存されることも避けたい。まず値カテゴリごとの上限を業務・仕様・UI・運用の
観点から決め、必要な箇所だけアプリケーション validation と SQL 制約の両方に反映する。

## Scope
- **policy / documentation**:
  - 文字列値カテゴリごとの最大文字数ポリシーを定義する。対象候補:
    - tenant / user / group / agent / application / client / category の表示名・名前・説明。
    - メールアドレス、URL、URI、SAML entity_id、WS-Fed realm、SCIM id、OIDC client_id など
      外部プロトコルと接する識別子。
    - token description、key id、object key、content type、audit type、outbox topic /
      event_type / published_to、エラー文字列。
    - tenant id、user id、group id など domain id。
  - RFC・外部仕様・主要 IdP の慣行・UI 表示上限・検索 index サイズ・ログ/監査保存量を
    参照し、上限を置く値と置かない値を分類する。
  - 上限を DB に置く場合、`varchar(n)` と `TEXT CHECK (char_length(column) <= n)` のどちらを
    採用するかを `wi-127-postgres-column-type-policy` の型ポリシーと整合させる。
- **spec**:
  - 最大文字数が公開 contract、管理 UI 入力制約、または保証義務に関わる場合は、
    SCL-first で `spec/scl.yaml` を最小限更新し、derived artifacts を再生成する。
- **implementation**:
  - 決定した上限を、HTTP request validation、domain/service validation、UI form validation、
    OpenAPI/JSON Schema など該当する境界に反映する。
  - DB 側の最後の防衛線が必要な列には、`deploy/schema/postgres.sql` と migration / seed /
    test fixture を更新する。
  - 制約違反時のエラーが API / UI で利用者に理解できる表現になるようにする。
- **tests**:
  - 境界値ちょうど、1 文字超過、空文字/空白のみ、マルチバイト文字を含む入力を確認する。
  - 外部プロトコル識別子は、仕様上許される実例を誤って拒否しないことを確認する。

## Out of Scope
- `TEXT` / `varchar` / `JSONB` / `UUID` / enum などの列型一般の選定ポリシー策定。
  これは `wi-127-postgres-column-type-policy` で扱う。
- 文字数上限を根拠なく全列に機械的に設定すること。
- 外部仕様で長さが明確でない識別子を、DB 都合だけで短く切ること。
- 表示上の省略や折り返しだけで十分な値に、永続化上限を過剰に導入すること。

## Plan
- `deploy/schema/postgres.sql` の現行policy（unconstrained varchar禁止、limitはTEXT+CHECKまたはvarchar）を基礎に、SCL model fieldごとにprotocol limit、security/resource limit、UI usability limitを分類したregistryを作る。一律255文字にはしない。
- 文字数の単位はfieldごとにUTF-8 bytes、Unicode code points、protocol-defined bytesを明示する。Goの`len`とPostgreSQL`char_length`の差を放置せず、正規化が必要なidentifier/email/URIはnormalize後に測る。
- validationはdomain/value constructorまたはusecase commandで正本化し、HTTP/UIは同じlimit metadata/error codeを表示する。DB CHECKはrace/bypassへの最後の防壁で、DB errorを500にしない。
- 既存schema/data/API inputをinventoryし、現存最大値と違反行をreportしてからconstraintを追加する。自動truncateはせず、互換が必要なexternal protocol fieldはより広い上限かmigrationを選ぶ。
- unbounded body/JSON/array/mapは文字列fieldとは別にHTTP body limit、element count、nesting depthで制限し、wi-110のbody limitsと重複実装しない。

## Tasks
- [ ] T001 [Inventory] SCL fields、Go structs/validators、HTTP forms、frontend inputs、Postgres text columnsを対応付け、現行/外部仕様/実data最大値をreportする。
- [ ] T002 [Policy/SCL] field別limit/unit/normalization/error codeを定義し、models/interfaces/invariantsへ反映して再生成する。
- [ ] T003 [Validation Core] code-point/byte/normalized length helpersとtyped errorsを追加し、各contextのvalue/commandへowner単位で適用する。
- [ ] T004 [HTTP/UI] typed error→400/SCIM/OAuth protocol error mapping、OpenAPI maxLength（単位が一致するfieldのみ）、form max/remaining表示を追加する。
- [ ] T005 [Postgres] data audit queryを通した後にCHECK/varchar制約をcontextごとに追加し、constraint error mappingとindex size影響を検証する。
- [ ] T006 [Protocol Tests] OAuth URI/client metadata、SAML/WS-Fed identifiers、SCIM attributes、user/group/application fieldのlimit±1とoversize body/collectionを検証する。
- [ ] T007 [Unicode/Compatibility] multibyte、combining、normalization、legacy最大値、DB/API/UIの同一判定と非truncateを検証する。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`（SCL を変更した場合）
- `just scl-render`（SCL を変更した場合）
- `just verify-go`
- `just verify-ui`（UI validation / 表示を変更した場合）
- `just verify`
- 手動確認: 文字列値カテゴリごとに、最大文字数を「置く / 置かない」とその根拠が
  ドキュメント、SCL、または ADR に残っている。
- 手動確認: 上限を置いた値について、API / UI / DB の各境界で同じ制限が適用され、
  違反時のエラーが利用者に理解できる。

## Risk Notes
文字数上限は一度公開 contract や DB 制約に入ると、外部連携・既存データ・UI 操作に影響する。
特に SAML entity_id、OIDC client_id、URL/URI、SCIM id など外部プロトコルと接する値は、
短すぎる上限で相互運用性を壊しやすい。実装時は、内部表示名のように閉じた値から先に制約を
入れ、外部識別子は仕様根拠と実データ例を確認してから上限を決める。
