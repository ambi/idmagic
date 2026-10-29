---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p2
depends_on: [wi-80420-enforce-agent-access-at-external-resource-servers]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-042 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-043 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-050 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9396-MONOTONIC-NARROWING }
---

# 人間が承認した操作を実行時の対象と引数に結び付ける

## 動機

CIBA の `auth_req_id` は一回だけ token へ交換できるが、発行した token 自体は有効期間内に繰り返し使える。
人が一件の送金や削除を承認したつもりでも、資源側が承認対象を確かめなければ別の引数への利用や同じ操作の再実行を防げない。
RAR の一回ごとの上限、累積上限、操作の一回性は異なる保証である。
本項目は既存 CIBA の不具合を断定せず、承認から業務作用までを結ぶ機能の不足を扱う。

## 対象範囲

- 承認対象の resource、action、業務対象、セキュリティ上意味のある引数、期限を安定した形で束縛する。
- 参照 PEP が実際の要求と承認内容を照合し、不一致、期限切れ、取消、停止時は実行しない。
- 一回性を要求する操作について、並行要求と再送で許可を二重消費しない契約を資源側と定める。
- 承認済み内容と実行結果を秘密値なしに相関できるようにする。
- `supervised` の CIBA 強制について、現行 REQ-OAUTH2-050 と矛盾する実現方式と設計判断の旧記述を整理する。

## 対象外

- [[wi-59-agent-governance-guardrails-audit-inventory]] の Agent 全体の回数、予算、レート。
- [[wi-55-token-vault-federated-connections]] の資格情報保管と HTTP 仲介そのもの。
- 汎用決済エンジン、外部資源との分散トランザクション、ネットワーク越しの exactly-once の宣言。
- 未確定の Transaction Token draft の独自拡張、外部へ提示する独自委譲受領書。

## 設計

要件差分の候補は、承認対象の変更を拒否し、一回性を要求した承認の再利用で二つの業務作用を許可しないことである。
既存 `ApprovalRequest` と登録済み RAR type を使い、同意とは別の短命な判断として保つ。
正規化した入力の digest は照合の材料であり、人が理解できる承認画面の代わりにしない。

境界の入力は OAuth2 の承認ライフサイクル、外部 PEP の実行直前の評価、業務資源の idempotency と結果の確定である。
D1/D2/D7/D9 に従い、承認状態は OAuth2、業務作用の冪等性は外部資源に置き、公開契約だけで結ぶ案を第一候補とする。
「token を一度発行すれば操作も一回」という案は二つの確定点を混同するため採らない。
資源側が既存 idempotency を提供する案と、実行前の短い予約と確定を使う案を、タイムアウト後の再試行で比較する。
資源が冪等性を提供できない場合に一回性を保証できると宣言しない。

主要型の候補は `ApprovedOperation`、`OperationBinding`、`ApprovalUse` である。
`MatchApprovedOperation(approved, actual, now)` は純粋に照合し、消費の比較交換、実行、結果記録を作用として分ける。
認可の消費と資源側の実行がずれたときの状態、digest の正規化、機密引数の除外、期限と取消の順序は実装前に決める。
構築内容を変える問いが残る場合は、仕様の段階で決定し、単なる拡張点として残さない。

移行では一回性を要求する操作だけに明示的に適用し、一般の OAuth token の再利用可能性を変えない。
ロールバックは新しい保護操作を停止し、未完了予約と結果を保持して再送による重複を避ける。

## 計画

1. 一つの参照業務操作で承認対象、引数、冪等性、タイムアウト時の観測結果を決める。
2. CIBA と PEP の公開契約を仕様化し、対象変更と並行再送の拒否を先に確かめる。
3. 既存承認と最小の資源側実装を接続する。

## タスク

- [ ] T001 [Design] 正規化、消費、業務資源の冪等性、失敗後の状態を比較して確定する。
- [ ] T002 [Spec] 承認と実行の束縛を規範化し、supervised 強制に関する旧い記述を整合させる。
- [ ] T003 [Acceptance] 承認と同じ操作の成功、引数差分の拒否、並行再送で業務作用が一つである RED を作る。
- [ ] T004 [App] 既存 ApprovalRequest と RAR を拡張し、照合と消費の操作を実装する。
- [ ] T005 [PEP] 実際の要求の照合と業務資源の冪等性を接続し、失敗と不明な結果を区別する。
- [ ] T006 [Verify] 取消、期限、Agent 停止、改ざん、二重送信、クラッシュと再試行、機密値の非露出を検証する。

## 検証

- `mise run check-spec`、`mise run check-boundaries`、`mise run check-published-api-compat`：責務と通信契約を確認する。
- `mise run test-go-package -- ./oauth2/...`：承認の状態、同一対象と引数、期限、並行消費を確認する。
- `mise run test-go-package -- ./oauth2/internal/handlers_http`：外部資源の記録する実際の作用件数を検査し、token 発行回数だけを証拠にしない。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP と DB を使う検証はサンドボックス外で実行する。
- `mise run test-ui-e2e-file -- <承認の対象を表示する spec>`：UI を変える場合、実際に承認する対象と引数が表示されることを確認する。具体的な既存ファイルを着手時に選ぶ。

## リスク

「人が承認した」という説明だけでは、実行する操作との一致も二重実行の防止も保証できない。
資源側の冪等性と責任分界を明示し、結果が不明な要求を無条件にやり直さない。
引数と digest の監査が機密情報の複製にならないよう、表示用情報、照合用情報、監査用要約を分ける。
