---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/seeding/seed-run/README.md, requirement: REQ-SEEDING-009, impact: conforms }
  - { path: docs/modules/seeding/seed-run/README.md, requirement: REQ-SEEDING-010, impact: conforms }
---

# seed の適用を記録の正を持つ Context のユースケースへ通し、プロセスをまたいで排他する

## 動機

Seeding の判断は「SQL フィクスチャではなく記録の正を持つ Context の公開する操作を通す」ことで、seed で作るデータにも通常の作成の経路と同じ不変条件が成り立つことを根拠にしている。
しかし `backend/cmd/internal/bootstrap/seeding.go` の `Contributor` は、ユースケースではなく Repository の保存を直接呼ぶ。
ユースケースが行う検証、リソース上限の確認、ドメインイベントと監査の記録を、seed の書き込みは通らない。

あわせて次の二つも見つかった。

- 適用の排他は同じプロセスの中のミューテックスだけで、旧内部設計が書いていた PostgreSQL のアドバイザリーロックはない。二つのプロセスが同じ要求を同時に適用すると、保存が競合しうる。
- `development` のプロファイルでは、デモのデータが存在するかだけで完了を判定するので、手動の変更が `conflict` にならない（REQ-SEEDING-009 の手動の変更の検出が、デモのデータに効かない）。

wi-26063 で Seeding の内部設計をコードと照合して見つけ、`docs/modules/seeding/design/risks.md` に載せた。

## 対象範囲

- `Contributor` の書き込みを、記録の正を持つ Context のユースケースまたは公開する操作へ通す。
- 適用をプロセスをまたいで排他する。
- デモのデータのドリフトを、項目の比較で判定する。
- `docs/modules/seeding/design/risks.md` の該当の行を消す。

## 対象外

- マニフェストの文法と環境ポリシーの変更。

## 設計

ユースケースへ通すと、ドメインイベントと監査の記録が seed の書き込みでも発行される。
seed を監査に残すことは望ましいが、合成データの大量の投入で監査の記録が増えるので、`performance` のプロファイルの扱いを着手時に決める。
プロセスをまたぐ排他は、要求の鍵（環境、プロファイル、テナント）に対する PostgreSQL のアドバイザリーロックを候補とする。

## 計画

1. Context ごとに、seed が使う書き込みに対応するユースケースを洗い出す。
2. 一つずつユースケースへ移し、seed の再適用の冪等性（REQ-SEEDING-006）が保たれることを確かめる。
3. 排他とドリフトの判定を入れる。

## タスク

- [ ] T001 [App] seed の書き込みをユースケースへ通す。
- [ ] T002 [App] プロセスをまたぐ排他を入れる。
- [ ] T003 [App] デモのデータのドリフトを項目で判定する。
- [ ] T004 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

ユースケースのリソース上限の確認により、`performance` のプロファイルの大量の投入が拒否されうる。
上限の扱いを着手時に決める。
