---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-17
priority: p3
depends_on: [wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy]
change_kind: tooling
spec_impact: { kind: none, reason: "負債の比較検査が基準 revision の責務表を読む方法だけを変える。製品のコード、HTTP の応答、永続状態、ドメインイベント、作業ツリーに対する境界検査の判定は変えない。" }
---

# 負債の比較検査から、移行前の責務表を読む互換を外す

## 動機

[残りのモジュールの internal/ への移行](wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で、作業ツリーの責務表は `internal` だけを宣言できるようになった。
負債の比較検査は基準 revision の責務表も読むため、基準 revision が `legacy` の行や旧形式の表を持つ間は、それを読む互換と、`legacy` から `internal` へ切り替えたモジュールの利用元のパスの読み替えを残してある。
wi-75383 が `main` に入れば、比較の基準になる revision の責務表はすべて `internal` になり、この互換は使われなくなる。

## 対象範囲

- `parseLogicalArchitecture` から基準 revision だけが読む形（公開方式 `legacy`、旧形式の表）を外し、`ArchitectureRevision` の区別をなくす。
- `domain` と `ports` の区画名による公開の判定と、組み立て地点から `legacy` のモジュールへの直接の import を許す分岐を外す。
- 負債の比較検査から、`legacy` から `internal` へ切り替えたモジュールの利用元の読み替えを外す。
- 検査のテストと fixture から、移行前の形を基準にした場合を外す。
- バックエンド設計の公開範囲の節から、移行前の責務表を読む規則を外す。

## 対象外

- 完了した作業項目が記録したテストのパスを、`internal/` を挿入して読む規則。完了した記録は書き換えないので残す。

## 設計

着手の条件は、`mise run check-boundary-debt-ratchet` が使う基準 revision（既定では `main`）の責務表に `legacy` の行も旧形式の表もないことである。
着手時に、比較の基準として使われうる revision を確かめる。

## タスク

- [ ] T001 [Inventory] 基準 revision の責務表が新しい形だけであることを確かめる。
- [ ] T002 [Tooling] 互換の読み取りと読み替えを検査から外し、テストと fixture を更新する。
- [ ] T003 [Docs] バックエンド設計から移行前の責務表の規則を外す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-tools-file -- check/src/boundary-fitness.test.ts`
- `mise run test-tools-file -- check/src/boundary-debt-ratchet.test.ts`
- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`

## リスク

- 基準 revision に移行前の責務表が残っているうちに外すと、負債の比較検査が基準を解析できずに失敗する。
  着手の条件を T001 で確かめる。
