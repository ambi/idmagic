---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-17
priority: p2
group: agent-governance
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-DELEGATION-DEPTH }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-048 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-060 }
  - { path: docs/modules/oauth2/features/guardrail.md, requirement: REQ-OAUTH2-073 }
  - { path: docs/modules/oauth2/features/guardrail.md, requirement: REQ-OAUTH2-078 }
---

# Agent が呼び出せる Agent を規則で定め、連鎖を監査する

## 動機

IdMagic は Token Exchange の `act` の連鎖で Agent から Agent への委譲を表し、深さの上限（REQ-OAUTH2-048）とガードレール（REQ-OAUTH2-073）で制限する。
しかし「どの Agent がどの Agent を呼べるか」を管理者が宣言する規則はない。
交換を許可されたクライアントなら、委譲の深さの範囲で任意の Agent を連鎖に加えられる。

Okta の [Agent-to-Agent Connections](https://www.okta.com/newsroom/press-releases/ai-innovations-oktane-2026/) は一般提供で、呼び出してよい組み合わせを定め、引き継ぎを監査可能な連鎖として記録する。
A2A 1.0 も公開され、Agent 間の委譲は実利用に入った。
競合が一般提供した統制の基本で、許可の過剰を防ぐ規則なので、群の基準の p3 より上げて p2 とする。

## 対象範囲

- 呼び出す Agent と呼び出される Agent（または Agent の区分）の組を許可する規則と管理 API。
- トークン交換で、連鎖に新しい Agent を加えるときの規則の評価と拒否。
- 規則の評価結果、連鎖全体、理由の監査。
- 既存のテナントでの規則の既定（移行）。

## 対象外

- A2A プロトコル固有の対応（Agent Card の検証など）。[[wi-65990-decide-a2a-agent-support]] が扱う。
- ラベルによる規則の対象選択。[[wi-60406-govern-agents-by-label]] の結果を使えるようにだけ設計する。

## 設計

規則はガードレールと同じく発行と交換の時点で評価し、和だけで許可を広げる AccessControl の判断とは独立させる。
`AllowAgentHop(chain []AgentRef, next AgentRef, rules RuleSet) Decision` に相当する作用のない計算に閉じ、規則の取得と監査の確定はユースケースが編成する。

既存のテナントの既定は二案ある。
規則がなければ従来どおり許可する案は互換性を保つが、統制の効果が導入まで出ない。
規則がなければ拒否する案は安全だが、既存の連鎖を止める。
着手時に、テナントの設定で既定を選べるようにし、新規テナントは拒否を既定にする案を第一候補として比較する。

## 計画

1. 規則のモデル、既定、ガードレールとの関係を確定し、仕様と TypeSpec を書く。
2. 受け入れ試験の RED を確認してから実装する。

## タスク

- [ ] T001 [Spec] 規則、評価の時点、既定、監査の規範と TypeSpec を書く。
- [ ] T002 [Acceptance] 未許可の組、連鎖の途中の未許可、規則の削除後の交換の RED を作る。
- [ ] T003 [Domain] 判定の計算を単体テストで固める。
- [ ] T004 [App] 交換のユースケース、管理 API、保存を接続する。
- [ ] T005 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases`。
- `mise run test-go-package -- ./backend/shared/http/server_http`：正式な配線での交換。
- `mise run check-spec`、`mise run check-schema`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。

## リスク

規則の評価を交換の一部の経路にだけ入れると、別のグラントで連鎖へ加えられる。
REQ-OAUTH2-073 と同じく、Agent が関わるすべての発行で評価する。
