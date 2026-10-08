---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 監査ログに新しいイベント型が現れ、既存のイベントのペイロードに項目が増えることを、監査ログを読む運用者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-470-record-control-plane-state-changes-in-the-audit-log.md }
initial_context:
  specification:
    - docs/domain/tenancy/resolution/README.md#REQ-TENANCY-011
    - docs/domain/tenancy/quota/README.md#REQ-TENANCY-012
    - docs/domain/audit/event-search/README.md#REQ-AUDIT-007
  typespec:
    - IdMagic.Contract.TenantQuotaUpdated
    - IdMagic.Tenancy.Operations.UpdateTenantQuota
    - IdMagic.Tenancy.Operations.SetTenantEndpointStyle
  source:
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/routes.go
    - backend/tenancy/domain/events.go
    - backend/tenancy/usecases/manage_tenants.go
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/audit/handlers_http/admin_audit_event_handler.go
    - backend/audit/usecases/audit_search_extractor.go
    - tools/check/src/check-event-contract.ts
    - tools/check/src/event-contract.ts
    - tools/check/src/check-unspecified-vocabulary.ts
    - tools/check/unspecified-vocabulary-debt.json
  tests:
    - backend/tenancy/handlers_http/admin_tenant_handler_test.go
    - backend/shared/http/server_http/tenant_quota_csrf_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
  stop_before_reading:
    - frontend
    - backend/audit/db_postgres
primary_use_cases:
  - id: quota-update-audited
    requirement: REQ-TENANCY-012
    observable_result: System 管理者がクォータを更新すると、システムの経路の監査イベント検索で、操作者、対象テナント、要求したリソースの名前を載せた TenantQuotaUpdated が読める。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/control_plane_audit_test.go, name: TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog, task: test-go-race }
    fault_model: クォータ更新のハンドラーが保存した後にイベントを発行しないか、操作者や対象テナントを載せずに発行する。
  - id: endpoint-style-switch-audited
    requirement: REQ-TENANCY-011
    observable_result: System 管理者が正規ロケーションを切り替えると、システムの経路の監査イベント検索で、操作者、対象テナント、切り替える前と後の endpoint_style を載せた TenantEndpointStyleChanged が読める。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/control_plane_audit_test.go, name: TestTenantEndpointStyleSwitchIsReadableFromTheSystemAuditLog, task: test-go-race }
    fault_model: 切替のハンドラーがイベントを発行しないか、切り替える前の値として切替後の値を載せる。
  - id: refused-change-not-audited
    requirement: REQ-TENANCY-011
    observable_result: 不正な endpoint_style、tenant_base_domain のないデプロイでの subdomain、存在しない realm で拒否された切替は、イベントを発行しない。
    boundary: adapter
    test: { path: backend/tenancy/handlers_http/control_plane_audit_test.go, name: TestRefusedEndpointStyleSwitchEmitsNoEvent, task: test-go-race }
    fault_model: 切替のハンドラーが use case の結果を確かめる前にイベントを発行する。
  - id: refused-quota-update-not-audited
    requirement: REQ-TENANCY-012
    observable_result: 負の上限による拒否と保存の失敗では、TenantQuotaUpdated を発行しない。
    boundary: adapter
    test: { path: backend/tenancy/handlers_http/control_plane_audit_test.go, name: TestRefusedOrUnsavedQuotaUpdateEmitsNoEvent, task: test-go-race }
    fault_model: クォータ更新のハンドラーが保存の成否を確かめる前にイベントを発行する。
  - id: every-control-plane-change-audited
    requirement: REQ-TENANCY-011
    observable_result: 制御面に登録した状態を変える経路は、成功するたびに操作者と対象テナントを載せたイベントを一つ発行し、棚卸しにない経路を登録するとテストが落ちる。
    boundary: adapter
    test: { path: backend/tenancy/handlers_http/control_plane_audit_test.go, name: TestEveryControlPlaneStateChangeEmitsAnAuditEvent, task: test-go-race }
    fault_model: 制御面に新しい状態変更の経路を足すときに、発行を書き漏らす。
affected_spec:
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-011 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-012 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-001, impact: conforms }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantQuotaUpdated }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantEndpointStyleChanged }
---

# 制御面のクォータ更新と正規ロケーション切替を監査ログへ記録する

## Motivation

Tenancy の制御面ハンドラーのうち、テナントの作成、属性更新、停止、再開は、状態変更が成立した後に監査イベントを発行する。

しかしクォータ更新と正規ロケーション切替は、状態を変えたうえで何も発行しない。

クォータ更新については、仕様が `TenantQuotaUpdated` を公開イベントとして宣言し、実装側にも同名の型と `EventType()` が存在するが、それを発行する呼び出しはリポジトリのどこにもない。

宣言だけがあり発行経路のないイベントは、監査ログを読む側からは「その操作が一度も行われていない」ことと区別できない。

正規ロケーション切替には、対応するイベント定義そのものがない。

この操作は issuer、WebAuthn RP ID、Cookie スコープを作り替え、発行済みトークンの `iss` 検証と既存のパスキーを無効化し、進行中のセッションを切る。

制御面で最も影響範囲の広い 2 つの操作について、誰がいつ何を変えたのかが残らない状態になっている。

Audit の決定は、書き込み経路を公開せず、記録の追加は各 Context の発行経路だけが行うと定めている。

したがってこの欠落を埋められるのは発行側の Tenancy であり、監査側の検索、保持、エクスポートをいくら整えても補えない。

既存の `check-event-contract` は、公開されたイベントペイロードの語彙とそれを読む側の一致を見る検査であり、発行経路を持たないイベント型を落とさない。

そのため今回の欠落は、仕様と実装の両方に型が存在するまま、すべての検査を通過している。

## Scope

- `UpdateTenantQuota` が上限を保存できた場合にだけ `TenantQuotaUpdated` を発行する。
- 正規ロケーション切替のイベントを仕様と実装へ追加し、切替を行った主体、対象テナント、切替前後の `endpoint_style` を記録する。
- `TenantQuotaUpdated` に変更されたリソースキーの一覧を加え、`TenantUpdated` と同じ「何が変わったか」の形にそろえる。
- REQ-TENANCY-011 と REQ-TENANCY-012 に、操作の結果として監査イベントが記録されることを規範として加える。
- 拒否された要求、`tenant_base_domain` 未設定による切替失敗、保存失敗では発行しないことを確認する。
- 記録されたイベントが `ListAdminAuditEvents` の制御面横断参照から実際に読めることを、HTTP 境界で確認する。
- Tenancy の制御面変更ハンドラーを棚卸しし、状態を変えるすべての経路が監査イベントを持つことを回帰テストで固定する。
- 宣言だけがあり発行経路を持たないイベント型を検出する検査を `check-event-contract` へ追加する。

## Out of Scope

- 監査イベントの同期エクスポートの有界化。
  [[wi-464-bound-the-synchronous-audit-event-export]] が扱う。
- システムコンソールのテナント横断読出しとページングの有界化。
  [[wi-469-bound-system-console-cross-tenant-reads]] が扱う。
- クォータ更新の `Origin` と CSRF トークンの検証。
  [[wi-467-enforce-csrf-on-tenant-quota-update]] が扱う。
- システムコンソールに要求する認証強度と再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- 監査レコードの保持期間、検索属性、エクスポート形式を変えること。
- クォータ値の妥当性検証と、クォータ更新経路のテナント識別子解決および応答規約の不統一。
  監査記録の有無とは独立した欠陥であり、別の作業項目として扱う。

## Design

監査イベントは、副作用が確定した後、応答を書く前に発行する。

Tenancy の既存の制御面ハンドラーと同じ位置に置き、発行と保存の順序を操作ごとに変えない。

use case の内部で発行する案は採らない。

主体の解決はハンドラー層の `requireSystemAdmin` が担っており、`ActorUserID` を得るために use case へ認証文脈を持ち込むと、Tenancy の use case が HTTP の関心へ依存するためである。

2xx 応答を一律に記録する共通ミドルウェアを置く案も採らない。

その記録は「経路が呼ばれた」ことしか示さず、どのリソースの上限が動いたのか、どの正規ロケーションからどこへ切り替わったのかを残せない。

監査ログの価値は、変更後の状態を再構成できることにあり、経路の呼出し記録では代替できない。

正規ロケーション切替のイベントは `TenantEndpointStyleChanged` として新設し、`TenantUpdated` の `changedFields` に混ぜない。

通常の属性更新と切替は、影響範囲も復旧手順も異なる操作であり、同じ型で表すと、監査ログから「トークンとパスキーを無効化した操作」だけを取り出せなくなるためである。

切替前後の値を両方持たせるのは、切替後の状態だけでは、失敗した切替の再試行と実際に状態が動いた切替を区別できないためである。

`TenantQuotaUpdated` へ加えるのは変更されたリソースのキーであり、上限値そのものは記録しない。

上限値は秘密ではないが、監査の目的は変更の事実と範囲の特定であり、現在値は対象テナントの読出しから常に取得できる。

現在のクォータ更新ハンドラーは主体を破棄しているため、`requireSystemAdmin` が返す利用者を保持する形へ変える。

正規ロケーション切替のハンドラーも同様に主体を保持する。

### 仕様の差分

| 要件 | 変更 | 要件文 |
| --- | --- | --- |
| REQ-TENANCY-011 | 追加 | System 管理者がテナントの `endpoint_style` を切り替えたとき、Tenancy は、操作者、テナント、切り替える前と後の `endpoint_style` を載せた `TenantEndpointStyleChanged` を発行する。 |
| REQ-TENANCY-011 | 変更 | 不正な値、`tenant_base_domain` のないデプロイでの `subdomain`、存在しない realm の各拒否に「`TenantEndpointStyleChanged` を発行しない」を加える。 |
| REQ-TENANCY-011 | 追加（判断） | 前と同じ値への切替にも発行する。前後の値を比べれば、状態が動いた切替と再送を区別できる。通常の属性更新と型を分ける理由も記す。 |
| REQ-TENANCY-012 | 追加 | System 管理者がテナントの上限を更新したとき、Tenancy は、操作者、テナント、要求に含まれたリソースの名前を `changedFields` に載せた `TenantQuotaUpdated` を発行する。 |
| REQ-TENANCY-012 | 追加 | 上限の上書きを保存できなかった場合、Tenancy は、`TenantQuotaUpdated` を発行しない。 |
| REQ-TENANCY-037 | 変更 | 負の上限の拒否に「`TenantQuotaUpdated` を発行しない」を加える。 |

TypeSpec では `TenantQuotaUpdated` に `changedFields: string[]` を加え、`TenantEndpointStyleChanged`（`occurredAt`、`actorUserId`、`tenantId`、`previousEndpointStyle`、`endpointStyle`）を新設する。

### 主要な型と操作

| 型または操作 | 形 |
| --- | --- |
| `domain.TenantQuotaUpdated` | `At`、`ActorUserID`、`TenantID`、`ChangedFields []string` |
| `domain.TenantEndpointStyleChanged` | `At`、`ActorUserID`、`TenantID`、`PreviousEndpointStyle`、`EndpointStyle domain.TenantEndpointStyle` |
| `Deps.handleUpdateTenantQuota` | 保存の成功後に `d.Emit(&domain.TenantQuotaUpdated{...})` |
| `Deps.handleSetTenantEndpointStyle` | `tenantusecases.SetEndpointStyle` の成功後に、解決済みの対象の切替前の値と use case が返したテナントの値で `d.Emit(&domain.TenantEndpointStyleChanged{...})` |

時刻は既存の兄弟ハンドラーと同じくハンドラーで一度取得し、保存とイベントの両方へ渡す。
発行は `Deps.Emit` という出力ポートだけを通す。

### 観点表による分類

| 観点 | 答え | 分類 |
| --- | --- | --- |
| 記録 | 成功した切替とクォータ更新を記録し、拒否と保存失敗は記録しない | (a) 要件にする |
| 冪等性 | 同じ値への再切替、同じ上書きの再送にも発行する。`TenantDisabled` と `TenantUpdated` の既存の扱いと同じ | (a) 判断として要件に記す |
| 失敗の観測者 | 発行の失敗は操作を巻き戻さず、ログに残る。Audit の既存の方針どおり | (b) この項目では変えない |
| 同じ種類の操作 | 兄弟の制御面経路はすべて成功後に一つ発行する。棚卸しテストで固定する | (a) 既存要件の形にそろえる |
| 監査の分類 | `category=tenant` で新しい二つの型も引けるよう、監査の分類表へ加える | (a) REQ-AUDIT-001 の `category` の語彙に含める |

## Plan

1. クォータ更新と正規ロケーション切替を成功させた後、`ListAdminAuditEvents` に記録が現れないことを HTTP 境界で観測し、RED を確認する。
2. REQ-TENANCY-011 と REQ-TENANCY-012 へ監査記録の規範を加え、`TenantEndpointStyleChanged` と `TenantQuotaUpdated` の変更キーを TypeSpec へ定義する。
3. 生成物を再生成し、公開イベント語彙の差分を確認する。
4. 両ハンドラーで主体を保持し、保存が成立した経路だけで発行する。
5. 拒否、`tenant_base_domain` 未設定、保存失敗の各経路で発行が起きないことを確認する。
6. Tenancy の制御面変更ハンドラーを棚卸しし、結果をテスト名として残す。
7. 発行経路のないイベント型を検出する検査を追加し、今回の欠落と同じ形が再発したときに落ちることを確認する。

## Tasks

- [x] T001 [Acceptance] クォータ更新と正規ロケーション切替の後に監査イベントが読めないことを観測し、RED を確認する。
  `mise run test-go-test -- ./backend/shared/http/server_http 'TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog|TestTenantEndpointStyleSwitchIsReadableFromTheSystemAuditLog'` が、両方とも `audit events = 0, want 1` で失敗した（REQ-TENANCY-011、REQ-TENANCY-012）。
- [x] T002 [Spec] REQ-TENANCY-011 と REQ-TENANCY-012 へ監査記録を規範化し、TypeSpec のイベント定義を更新する。
  REQ-TENANCY-037 の負の上限の拒否にも不発行を加えた。`mise run check-spec` と `mise run check-api-compat` が成功した。
- [x] T003 [App] `UpdateTenantQuota` の保存成功時に `TenantQuotaUpdated` を変更キー付きで発行する。
- [x] T004 [App] `SetTenantEndpointStyle` の切替成功時に `TenantEndpointStyleChanged` を切替前後の値付きで発行する。
  切替前の値は解決済みの対象から、切替後の値は use case が返したテナントから、どちらも `EffectiveEndpointStyle()` で取る。
- [x] T005 [Unit] 拒否および保存失敗の経路で発行が起きないことを確認する。
  `TestRefusedEndpointStyleSwitchEmitsNoEvent`、`TestRefusedOrUnsavedQuotaUpdateEmitsNoEvent`、`TestEndpointStyleSwitchToTheSameStyleStillEmitsAnEvent`（`backend/tenancy/handlers_http/control_plane_audit_test.go`）。
- [x] T006 [Acceptance] 発行されたイベントが制御面の監査一覧から主体と対象テナントで検索できることを確認する。
  システムの経路 `/api/admin/v1/system/audit-events` を `type` と `filter=actor.id:eq:ops` で検索し、`tenant_id` が realm ではなくテナントの ID であることまで確かめる。`category=tenant` へ二つの型を加え、`TestAdminAuditEventsTenantCategoryIncludesQuotaAndEndpointStyleChanges` が分類表の変更前に `category=tenant types = []` で失敗することを確かめた（REQ-AUDIT-001）。
- [x] T007 [Inventory] Tenancy の制御面変更ハンドラーの監査イベント発行を棚卸しし、回帰テストで固定する。
  `TestEveryControlPlaneStateChangeEmitsAnAuditEvent` が、`RegisterControlPlaneRoutes` の POST、PUT、PATCH、DELETE の経路の集合を棚卸しの表と突き合わせ、各経路の成功で操作者とテナントを載せたイベントがちょうど一つ出ることを確かめる。棚卸しの結果は、作成、属性更新、正規ロケーション切替、停止、再開、クォータ更新の 6 経路である。
- [x] T008 [Tooling] 発行経路を持たないイベント型を `check-event-contract` で検出する。
  N/A: 製品要件ではない検査の追加。代替検査として、クォータ更新の発行を外した木で `mise run check-event-contract` が `fail  backend: event TenantQuotaUpdated is declared but no production code emits it` で失敗することを観測した。既存の 4 件（`FederationLinked`、`FederationUnlinked`、`GroupMembershipPushed`、`SigningKeyRetired`）は `tools/check/event-emission-debt.json` に負債として載せ、減る方向にだけ動かす。純粋関数は `mise run test-tools-file -- check/src/event-contract.test.ts` で確かめる。
- [x] T009 [Verify] 仕様生成物を再生成し、契約とセキュリティ制御の検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-event-contract`
- `mise run check-security-controls`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

応答を書いた後に発行する実装にすると、保存に失敗した操作の記録が残る、あるいは成立した操作の記録が落ちる。

受け入れテストは、成功経路で記録が読めることと、拒否経路で記録が増えないことの両方を確認する。

イベントの発行が保存トランザクションの外にあるため、発行の失敗そのものは操作を巻き戻さない。

この設計上の限界は Audit の既存方針と同じであり、本項目では変えない。

代わりに、発行経路が存在しない状態を検査で落とすことによって、同じ欠落が黙って再発しないようにする。

新しいイベント名は公開されたイベント語彙へ入り、追記のみで 7 年間保持される監査レコードとして残るため、`reversibility` は irreversible とする。

## 完了

- **完了日**: 2026-10-09
- **要約**:
  `mise run spec-diff` は、REQ-TENANCY-011、REQ-TENANCY-012、REQ-TENANCY-037 の変更、`TenantEndpointStyleChanged` の追加、`TenantQuotaUpdated` の変更を示した。
  System 管理者によるクォータ更新は、操作者、テナント、要求に含まれたリソースの名前（`changedFields`）を載せた `TenantQuotaUpdated` を記録する。
  正規ロケーションの切替は、操作者、テナント、切り替える前と後の `endpoint_style` を載せた新しい `TenantEndpointStyleChanged` を記録し、前と同じ値への切替でも記録する。
  拒否された要求と、上書きを保存できなかったクォータ更新は記録しない。
  監査の `category=tenant` は二つの型も返す。
  `check-event-contract` は、本番コードのどこも値を作らないイベント型を落とすようになり、既存の 4 件は `tools/check/event-emission-debt.json` に負債として載る。
- **主要ユースケースの証拠**:
  - id: quota-update-audited
    red: 実装前、TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog が `audit events = 0, want 1` で失敗した。
    fault_injection: 要求に含まれたかどうかの条件を反転すると要求していないリソースの一覧を、操作者を空にすると 0 件を報告して、TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog が失敗した。
  - id: endpoint-style-switch-audited
    red: 実装前、TestTenantEndpointStyleSwitchIsReadableFromTheSystemAuditLog が `audit events = 0, want 1` で失敗した。
    fault_injection: 発行を外すと 0 件で、切替前の値を切替後のテナントから取ると `previousEndpointStyle = "subdomain", want "path"` で、TestTenantEndpointStyleSwitchIsReadableFromTheSystemAuditLog が失敗した。
  - id: refused-change-not-audited
    red: 発行を use case の結果の確認より前へ移すと、TestRefusedEndpointStyleSwitchEmitsNoEvent が不正な値と `tenant_base_domain` のない `subdomain` の 2 件で `want none` により失敗した。存在しない realm は注入箇所より前で返るため、この注入では失敗しない。
    fault_injection: 同上の注入で検出した。変異テストは発行の位置を動かせないため手で注入した。
  - id: refused-quota-update-not-audited
    red: 保存の失敗の分岐で発行すると、TestRefusedOrUnsavedQuotaUpdateEmitsNoEvent の `save failed` が `want none` で失敗した。
    fault_injection: 同上の注入で検出した。
  - id: every-control-plane-change-audited
    red: 作成時の棚卸しの比較が echo の内部経路（`echo_route_not_found`）を状態変更の経路として数えて失敗したので、比較を POST、PUT、PATCH、DELETE に限った。
    fault_injection: 正規ロケーション切替の発行を外すと、TestEveryControlPlaneStateChangeEmitsAnAuditEvent が `events = 0, want 1` で失敗した。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/tenancy/handlers_http` は 161 の変異のうち 132 を検出し、18 が生き残った。
  変更した箇所で生き残ったのは `admin_tenant_handler.go:375`（`changedFields` の条件の反転）だけで、このパッケージのテストは `changedFields` を表明しない。同じ変異を手で入れると、server_http の TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog が検出した。
  残りの 17 は、この項目が触れていない branding、settings、属性スキーマ、連携エンドポイントのハンドラーにある既存の生存である。
  配線の故障（発行の除去、発行位置の前倒し、切替前の値の取り違え、操作者の欠落）は上の主要ユースケースの証拠のとおり手で注入し、すべて検出した。
  `check-event-contract` は、クォータ更新の発行を外した木で `fail  backend: event TenantQuotaUpdated is declared but no production code emits it` を出して失敗した。
- **検証結果**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功
  - `mise run check-event-contract` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 2 回とも 39 件中 1 件が失敗した。失敗したテストは 1 回目が同意の取り消し、2 回目がロゴのアップロードで、どちらも oven-sh/bun#43412 の WebKit の evaluate の詰まり（`an evaluate() is already pending`）であり、この項目が触れない画面である。
