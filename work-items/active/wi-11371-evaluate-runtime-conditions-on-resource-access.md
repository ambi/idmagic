---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-064 }
  - { path: spec/modules/oauth2/models.tsp, symbol: ResourceAccessRule }
---

# リソースアクセスの評価で属性の条件を判定する

## 動機

現在の評価は「100ドルまでは自動、超えたら人の承認」を表現できない。
要求に業務属性の入口がなく、`operation_parameters` は承認の `fields` との比較にしか使わないからである。
[Ping の連携例](https://developer.pingidentity.com/identity-for-ai/integrations/idai-authorizing-gemini-on-behalf-of-users-with-ping.html)はツール名と金額で拒否し、時刻を使う例もある。

## 対象範囲

- `ResourceAccessRule` に版付きの条件を追加し、REQ-OAUTH2-062 の論理積へ一段加える。
- 条件の入力を、検証済みトークンのクレーム、IdMagic の時刻、リソースサーバーが渡す属性に分ける。
- 結果を許可、拒否、承認要求の三つにし、承認要求は `approval_required` の経路へ接続する。
- 評価できない条件を不許可にする。監査に条件の版と理由を残す。

## 対象外

- ReBAC への交差と差集合の追加。規則の追加が既存の許可を取り消し得るとして、AccessControl は和だけを採用している。
- 発行と交換の評価（[[wi-30707-agent-guardrail-policies-at-issuance-and-exchange]]）。
- 回数の計数（[[wi-25649-limit-agent-tool-calls-at-runtime]]）。

## 設計

判定は `EvaluateCondition(condition, attributes) -> Allow | Deny | RequireApproval | Indeterminate` の計算にし、時刻は入力で受ける。
T001 で式の表現を次から選び、記録する。

| 候補 | 利点 | 懸念 |
| --- | --- | --- |
| CEL（第一候補） | 副作用がなく、評価の費用に上限を付けやすい | 依存が増える |
| 独自の比較式 | 依存がなく検査しやすい | 表現の追加要求ごとに言語を設計することになる |
| OPA への委譲 | 既存の方針資産を使える | 別の可用性の境界と、判断の説明の分断が増える |

リソースサーバーが渡す属性は申告であり、IdMagic は業務の事実を保証しない。
条件の所有者は、規則を持つ OAuth2 を第一候補とし、境界の判断手順で確かめる。

## 計画

1. 式の表現、所有者、属性の上限を決める。
2. 仕様と TypeSpec を更新する。
3. 金額、時刻、ツール名の例を受け入れテストにする。

## タスク

- [ ] T001 [Design] 式の表現、所有者、属性の検証、版の扱いを確定する。
- [ ] T002 [Spec] 規則、評価の要求と応答、管理 API を更新する。
- [ ] T003 [Acceptance] 許可、拒否、承認要求、評価不能の RED を正式な入口で観測する。
- [ ] T004 [Domain] 条件の判定を実装する。
- [ ] T005 [App] 評価と管理 API へ接続する。
- [ ] T006 [Verify] 属性の型の揺れ、上限、版の差し替え、テナント境界を検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases`：判定の表を確かめる。
- `mise run test-go-package -- ./backend/shared/http/server_http`：正式な入口を確かめる。
- `mise run check-spec` と `mise run check-boundaries`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

属性の申告を業務の事実と誤解すると保証を偽る。
仕様とイベントで申告の出どころを区別する。
式の評価が重いと評価の遅延が増えるので、費用の上限を設ける。
