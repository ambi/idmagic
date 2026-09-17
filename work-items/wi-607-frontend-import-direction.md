---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-586-frontend-architecture-and-user-interface-design]
change_kind: refactor
spec_impact:
  kind: none
  reason: "フロントエンドの import の向きだけを変え、画面の振る舞いは変えない。"
---

# フロントエンドの import を設計どおりの向きにそろえる

## Motivation

[フロントエンド設計](../docs/design/application/frontend.md#機能スライスの境界)は、機能スライスどうしの参照を禁じ、`lib/` から `components/` への参照も許していない。
現行のコードには、この規則に反する参照が次の 4 件ある。

| 参照元 | 参照先 |
| --- | --- |
| `frontend/src/features/admin-audit-events/AuditEventsBrowser.tsx` とそのテスト | `features/admin-dashboard/AdminDashboardPage.i18n.ts` の `friendlyEventName` |
| `frontend/src/features/admin-provisioning/AdminProvisioningOverviewPage.tsx` | `features/admin-applications/AdminApplicationProvisioning.i18n.ts` の `provisioningDictionary` |
| `frontend/src/features/admin-provisioning/AdminProvisioningOverviewPage.tsx` | `features/admin-applications/AdminApplicationsShared.tsx` の `provisioningURL` |
| `frontend/src/lib/adminNav.ts`、`frontend/src/lib/systemNav.ts` | `components/shell.i18n.ts` の `shellDictionary` |

この向きを検査する仕組みもないため、同じ種類の参照が今後も増えうる。

## Scope

- 4 件それぞれについて、意図的な参照かどうかを確認する。確認の材料は、参照が入ったコミットと、参照先を `components/` や `lib/` へ移したときの影響である。
- 意図的でないものは、参照先を `components/` または `lib/` へ移すか、参照元へ複製して解消する。
- 意図的であり、規則のほうを改めるべきだと判断したものは、[フロントエンド設計](../docs/design/application/frontend.md)の規則を改める。
- import の向きを機械的に検査する仕組みを入れ、`mise run verify` から実行する。

## Out of Scope

- スライスの分割や統合。
- 画面の振る舞いと文言の変更。

## Design

判断の基準は次のとおりとする。

| 状況 | 対応 |
| --- | --- |
| 参照先が二つ以上の機能から使われる | `components/` または `lib/` へ移す |
| 参照先が一方の機能の内部事情に依存する（`provisioningURL` が経路の形を知っている、など） | 参照元へ必要な部分だけ複製するか、経路の生成を `lib/` に置く |
| ナビゲーションの定義が画面の外枠の辞書を使う | 辞書を `lib/i18n/` へ移すか、ナビゲーションの定義を `components/` へ移す。規則を改める場合は設計文書を更新する |

検査の方式は着手時の readiness pass で決める。
候補は、Biome の import 制限、TypeScript の path 単位の検査を行う小さな自前スクリプト（`tools/check/`）、依存関係の検査ツールの導入である。
新しい依存を増やさずに済むなら、自前スクリプトを優先する。

## Plan

1. 4 件の参照が入ったコミットを読み、意図の有無を記録する。
2. 検査を先に書き、現行の 4 件で失敗することを確認する（Acceptance RED）。
3. 参照を 1 件ずつ解消し、検査を GREEN にする。
4. 規則を改めた場合は設計文書を更新する。

## Tasks

- [ ] T001 [Design] 4 件の参照の意図を確認し、対応を決める。
- [ ] T002 [Acceptance] import の向きの検査を追加し、現行コードで失敗することを確認する。
- [ ] T003 [Refactor] 参照を解消するか、規則を改める。
- [ ] T004 [Verify] `mise run verify-ui` と `mise run verify` を通す。

## Verification

- `mise run verify-ui`
- `mise run verify`

## Risk Notes

辞書を移すと、辞書を参照するテストの import も変わる。テストの期待値は辞書の値を参照しているため、移動後も同じ値を指していることを型検査で確かめる。
