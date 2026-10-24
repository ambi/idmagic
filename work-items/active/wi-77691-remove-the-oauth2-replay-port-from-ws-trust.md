---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "WS-Trust の再送防止の依存先を分け、同じ MessageID の拒否、記録の有効期間とキー、トークンの応答とイベントを維持する。" }
---

# WS-Trust の再送防止を OAuth2 のクライアント認証ポートから分ける

## 動機

[[wi-26339-audit-module-boundaries-against-independent-deployment-and-contract-only-coupling]] の棚卸しで、WsFederation から OAuth2 への本番 import は `ClientAssertionReplayStore` だけだと確認した。
`backend/wsfederation/internal/handlers_http/wstrust_handler.go` は、WS-Trust の MessageID に `wstrust:` を付け、再送を記録するために使っている。
OAuth のクライアント認証の意味は使っておらず、再送防止の変更が別のプロトコルの公開ポートへ波及している。

## 対象範囲

- [既知のリスク](../../docs/design/architecture/risks.md)の WsFederation の再送防止の依存を解消する。
- WsFederation が必要な最小の再送防止ポートを定め、組み立て地点で現在と同じストアを接続する。
- キーの名前空間、記録する時点、有効期間、エラーと未配線時の既存の結果を照合する。

## 対象外

- 再送防止のアルゴリズム、PostgreSQL のスキーマ、エラーの状態コードとイベントの変更。
- 未配線時に処理を続ける現状の是正。必要性を評価する場合は、仕様と別項目で扱う。
- ApplicationCatalog からの RP の管理。

## 設計

入力は [WS-Trust](../../docs/modules/ws-federation/features/active-sts.md)、`backend/wsfederation/module.go`、`backend/wsfederation/internal/handlers_http/routes.go`、`backend/oauth2/ports/replay_store.go` とする。
[境界の判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D3〜D6 と D8 を適用する。

| 案 | 制約への適合と変更の波及 | 判断 |
| --- | --- | --- |
| OAuth2 のポートを使い続ける | 技術的な再送の記録のために OAuth クライアントの語彙を学ぶ必要が残る | 不採用 |
| WsFederation の要求側ポートを、現在のストアが直接満たす | プロトコルの意味を WsFederation に閉じ、委譲だけの変換を増やさず接続できる | 採用候補 |
| 業務の意味を含まない再送ストアを共有する | D5 を満たすか既存の `backend/shared/security/tokens_jose` の契約と比べる。WS-Trust のキーと有効期間は共有側へ移さない | 比較候補 |
| WsFederation と OAuth2 を統合する | 共有する判断に比べて、両プロトコルの契約と変更範囲が広すぎる | 不採用 |

着手時にポートのシグネチャと、ストアが構造的に適合する経路を確定する。
同じ MessageID の再送を拒否する既存テストと、本番の組み立ての経路を確認してから import を外す。

## 計画

1. メモリーと PostgreSQL のストア、MessageID の記録と利用側を照合する。
2. 必要なポートを定義し、同じストアを組み立て地点で渡す。
3. OAuth2 への import を除き、公開範囲と依存図を更新する。

## タスク

- [ ] T001 [Design] 技術的な記録と WS-Trust の意味を分け、ポートを確定する。
- [ ] T002 [Refactor] 依存と結線を移し、OAuth2 への import を取り除く。
- [ ] T003 [Verify] 再送の拒否、既存のエラーと配線を確認する。

## 検証

- `mise run test-go-package -- ./backend/wsfederation/...`：既存の再送とトークンのテストを race 検査する。
- `mise run test-go-changed`：ストアと組み立てを変更する場合の利用側を確認する。
- `mise run check-boundaries` と `mise run generate-module-dependencies`：依存を外し、新しい違反を追加しない。
- `mise run check-spec` と `mise run check-links`：文書の担当と参照を検査する。

## リスク

ストアの共有を外すと、同じ MessageID の重複をプロセス間で見逃すおそれがある。
キーの名前空間と保存先を変えず、型の所有者と結線を分ける。
