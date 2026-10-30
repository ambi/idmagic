---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p1
depends_on: [wi-59-agent-governance-guardrails-audit-inventory]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-046 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-050 }
---

# Agent のガードレールを版付きのポリシーとしてトークンの発行と交換で評価する

## 動機

[[wi-59-agent-governance-guardrails-audit-inventory]] は起票時にガードレールと棚卸しの両方を含んでいたが、2026-10-30 に棚卸しへ絞り、ガードレールを本項目へ分けた。
二つは独立して受け入れられる結果であり、棚卸しはポリシーの設計の未決事項に依存しないためである。

IdMagic は Agent ごとの区分（REQ-OAUTH2-050）、状態と所有者（REQ-OAUTH2-046）、委譲深さ（REQ-OAUTH2-048）を発行の境界で評価するが、Agent ごとのレート、行動回数、有効期限、許可する資源と `authorization_details` の範囲を管理者が版付きで宣言する手段を持たない。
上限を超えたときに拒否するか、人間の承認へ降格するかも宣言できない。

## 対象範囲

- Agent、所有者、アプリケーション、資源の単位で、許可する資源と `authorization_details` の範囲、レート、行動回数、有効期限、委譲深さ、人間の承認要否を版付きのポリシーとして保持する。
- 評価点はトークンの発行（client_credentials、CIBA）とトークン交換に限る。2026-10-30 の判断で、IdMagic が自ら通す行動はこの二つであり、外部資源での行動の仲介は [[wi-80420-enforce-agent-access-at-external-resource-servers]] の外部 PEP が持つ。
- 超過時に拒否するか、[[wi-52-ciba-async-human-approval]] の `ApprovalRequest` へ降格するかをガードレールの種別ごとに宣言する。降格の経路を使えない場合は許可へ倒さず拒否する。
- 評価の結果に版、理由、相関 ID を残し、[[wi-377-agent-and-delegation-chain-audit-axes]] の監査軸から辿れるようにする。緊急の拒否とキルは [[wi-58-continuous-access-evaluation-agent-revocation]] の失効エポックへ接続する。
- 設定の管理 API と、[[wi-59-agent-governance-guardrails-audit-inventory]] のエージェント詳細への実効ガードレールの表示。

## 対象外

- 予算の上限。実行環境が消費量を報告する API と、その認証方式の設計が要る。2026-10-30 の判断で今回の評価点に含めず、必要になれば別の項目で扱う。
- 外部資源での行動の仲介と評価（[[wi-80420-enforce-agent-access-at-external-resource-servers]]）。
- 異常検知、外部への監査ログ配信、Web Bot Auth（起票時の wi-59 の対象外と同じ理由）。

## 設計

未定。着手時に次を確定する。

- ガードレールの種別ごとの超過時の既定の挙動。
- ポリシーの版と、評価中にポリシーが差し替わった場合の扱い。
- レートと回数の計数の保存先。既存の `shared/ratelimit` を Agent 単位の鍵で使えるか。
- ポリシーの所有モジュールと、OAuth2 の発行経路からの読み取りの向き（境界を選ぶ判断手順を適用する）。

## 計画

1. 種別、既定の挙動、版、所有者を決め、仕様と TypeSpec を更新する。
2. 発行と交換の評価器を実装し、拒否と降格を観測する受け入れテストを先に書く。
3. 管理 API と UI を加える。

## タスク

- [ ] T001 [Design] 種別、既定の挙動、版、計数の保存先、所有モジュールを確定する。
- [ ] T002 [Spec] ポリシー、評価、違反イベント、管理 API を仕様と TypeSpec に加える。
- [ ] T003 [Acceptance] 上限内の許可、超過の拒否、承認への降格、降格できないときの拒否の RED を作る。
- [ ] T004 [App] 評価器を発行と交換へ接続する。
- [ ] T005 [UI] 設定とエージェント詳細の表示を加える。
- [ ] T006 [Verify] 版の競合、計数の並行、テナント境界、PII の秘匿を検証する。

## 検証

- `mise run check-spec`、`mise run check-boundaries`。
- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases` と、正式な `/token` を通す受け入れテスト。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

評価を発行の経路へ挟むため、判定の重さとすり抜けが課題になる。迷ったら拒否へ倒し、判断の記録は既存の監査基盤へ載せて二重化しない。
