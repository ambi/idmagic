---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "seed の判断と結線の配置を分け、既存の計画結果、ドリフト判定、保存の順序、永続状態とイベントを維持する。" }
---

# seed のリソース生成と差分判定を組み立て地点から移す

## 動機

[[wi-26339-audit-module-boundaries-against-independent-deployment-and-contract-only-coupling]] の棚卸しで、Seeding が隠すはずのリソース生成と差分判定を、System の組み立て地点が実装していることを確認した。
`backend/cmd/internal/bootstrap/seeding.go` の `seedContributor` は、計画と適用だけでなく、クライアントの比較、デモの完了判定、性能用 User の生成と保存を行う。
結線を変更せずに seed の規則を変更できる配置へ戻す。

## 対象範囲

- [既知のリスク](../../docs/design/architecture/risks.md)の Seeding と組み立て地点の責務の混在を解消する。
- リソースの生成と差分判定を Seeding に、状態更新の保証を各所有モジュールに置き、bootstrap を構築と結線へ限定する。
- 公開する入力と操作を、現在の Contributor の計画、適用、再計画から選ぶ。

## 対象外

- Repository 経由の書き込みの是正、プロセス間の排他、デモのドリフト検出の強化。これらは [[wi-24943-route-seed-writes-through-context-use-cases]] が扱う。
- マニフェストの文法、環境ポリシー、イベントと監査の変更。

## 設計

入力は [Seeding](../../docs/modules/seeding/README.md)、[seed の計画と適用](../../docs/modules/seeding/features/seed-run.md)、`backend/cmd/internal/bootstrap/seeding.go` と `seed.go` とする。
[境界の判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D1、D2、D4、D6 と D8 を適用する。
Seed はサーバーと同じ内部状態の契約を使う管理コマンドであり、外部配布用の HTTP クライアントに作り替えない。

| 案 | 制約への適合と変更の波及 | 判断 |
| --- | --- | --- |
| bootstrap に判断を残す | 組み立て地点が業務判断を含み、seed の変更が起動の結線へ波及する | 不採用 |
| Seeding が生成と計画を担当し、要求側の操作を所有者へ結ぶ | 状態の不変条件を所有者に残し、変更理由を分けられる | 採用候補 |
| seed 専用の共有ライブラリに業務型を移す | D5 の技術的な規則だけという条件を満たさない | 不採用 |

着手時に既存の計画と保存の操作列を固定し、必要なポートを確定する。
wi-24943 が先に進んだ場合は更新済みの公開操作へ接続し、同項目の振る舞いの変更をこの構造変更へ混ぜない。

## 計画

1. Contributor と seed 補助の生成、比較、保存、通知を分類する。
2. 同じ入力と作用の順序を保って、判断を Seeding と状態所有者へ移す。
3. bootstrap の結線、現在状態の文書と既存の検査を揃える。

## タスク

- [ ] T001 [Design] 型と操作、作用の所有者と既存項目との分担を確定する。
- [ ] T002 [Refactor] 生成と差分判定を組み立て地点から移す。
- [ ] T003 [Verify] 計画、適用と再適用の既存テスト、境界と race 検査を通す。

## 検証

- `mise run test-go-package -- ./backend/seeding/...` と bootstrap の対象テスト：計画結果と適用順を維持する。
- `mise run check-boundaries`：循環と非公開 import の追加を拒否する。
- `mise run check-spec` と `mise run check-links`：文書の形式と参照を検査する。

## リスク

生成と状態更新を同じポートへ押し込むと、現在の保存型の依存を別の場所へ写すだけになる。
入力と結果の知識を比較し、所有者の更新規則を Seeding へ複製しない。
