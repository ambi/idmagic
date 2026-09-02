---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-011 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-012 }
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-001 }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantQuotaUpdated }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
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

## Plan

1. クォータ更新と正規ロケーション切替を成功させた後、`ListAdminAuditEvents` に記録が現れないことを HTTP 境界で観測し、RED を確認する。
2. REQ-TENANCY-011 と REQ-TENANCY-012 へ監査記録の規範を加え、`TenantEndpointStyleChanged` と `TenantQuotaUpdated` の変更キーを TypeSpec へ定義する。
3. 生成物を再生成し、公開イベント語彙の差分を確認する。
4. 両ハンドラーで主体を保持し、保存が成立した経路だけで発行する。
5. 拒否、`tenant_base_domain` 未設定、保存失敗の各経路で発行が起きないことを確認する。
6. Tenancy の制御面変更ハンドラーを棚卸しし、結果をテスト名として残す。
7. 発行経路のないイベント型を検出する検査を追加し、今回の欠落と同じ形が再発したときに落ちることを確認する。

## Tasks

- [ ] T001 [Acceptance] クォータ更新と正規ロケーション切替の後に監査イベントが読めないことを観測し、RED を確認する。
- [ ] T002 [Spec] REQ-TENANCY-011 と REQ-TENANCY-012 へ監査記録を規範化し、TypeSpec のイベント定義を更新する。
- [ ] T003 [App] `UpdateTenantQuota` の保存成功時に `TenantQuotaUpdated` を変更キー付きで発行する。
- [ ] T004 [App] `SetTenantEndpointStyle` の切替成功時に `TenantEndpointStyleChanged` を切替前後の値付きで発行する。
- [ ] T005 [Unit] 拒否および保存失敗の経路で発行が起きないことを確認する。
- [ ] T006 [Acceptance] 発行されたイベントが制御面の監査一覧から主体と対象テナントで検索できることを確認する。
- [ ] T007 [Inventory] Tenancy の制御面変更ハンドラーの監査イベント発行を棚卸しし、回帰テストで固定する。
- [ ] T008 [Tooling] 発行経路を持たないイベント型を `check-event-contract` で検出する。
- [ ] T009 [Verify] 仕様生成物を再生成し、契約とセキュリティ制御の検査を通す。

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
