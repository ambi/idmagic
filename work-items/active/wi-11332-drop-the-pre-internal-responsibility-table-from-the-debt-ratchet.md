---
status: in_progress
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

[残りのモジュールの internal/ への移行](../done/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で、作業ツリーの責務表は `internal` だけを宣言できるようになった。
負債の比較検査は基準 revision の責務表も読むため、基準 revision が `legacy` の行や旧形式の表を持つ間は、それを読む互換と、`legacy` から `internal` へ切り替えたモジュールの利用元のパスの読み替えを残してある。
wi-75383 が `main` に入れば、比較の基準になる revision の責務表はすべて `internal` になり、この互換は使われなくなる。

## 対象範囲

- `parseLogicalArchitecture` から基準 revision だけが読む形（公開方式 `legacy`、旧形式の表）を外し、`ArchitectureRevision` の区別をなくす。
- `domain` と `ports` の区画名による公開の判定と、組み立て地点から `legacy` のモジュールへの直接の import を許す分岐を外す。
- 負債の比較検査から、`legacy` から `internal` へ切り替えたモジュールの利用元の読み替えを外す。
- データベース設計のテーブル一覧から、基準 revision だけが読む改名前の見出し「所有 Context」の読み取りを外す。基準 revision の責務表と同じ理由で残した互換であり、比較の基準になる revision はすでに「所有モジュール」だけを持つ。
- 検査のテストと fixture から、移行前の形を基準にした場合を外す。
- バックエンド設計の公開範囲の節から、移行前の責務表を読む規則を外す。

## 対象外

- 完了した作業項目が記録したテストのパスを、`internal/` を挿入して読む規則。完了した記録は書き換えないので残す。

## 設計

着手の条件は、`mise run check-boundary-debt-ratchet` が使う基準 revision（既定では `main`）の責務表に `legacy` の行も旧形式の表もないことである。
着手時に、比較の基準として使われうる revision を確かめる。

比較の基準として使われうる revision は次の三つである。

| 基準 | 使われる場面 |
| --- | --- |
| `main` | upstream のないブランチでの既定 |
| `origin/main` | `main` の checkout での既定（upstream） |
| CI の `github.event.before` または Pull Request の base | CI の `mise run check` |

基準の責務表に `legacy` の行や旧形式の表が残っていれば、解析の失敗をそのまま報告する。
基準を当時の規則で読む経路は持たない。

## タスク

- [x] T001 [Inventory] 基準 revision の責務表が新しい形だけであることを確かめる。
  - `main` と `origin/main`（ともに 10a48fdcb）の `docs/design/architecture/logical.md` に `legacy` の行も旧形式の見出しもなく、`docs/design/data/database.md` に「所有 Context」の見出しもない。
  - 着手時点の `origin/main` は wi-75383 を含まず `legacy` の行を持っていたため、利用者が `main` を push してから続けた。CI の次の push の基準もこの revision になる。
- [x] T002 [Tooling] 互換の読み取りと読み替えを検査から外し、テストと fixture を更新する。
  - `ModuleDefinition` から公開方式の欄を外した。値が `internal` だけになり、判定に使わなくなったためである。
  - 基準 revision の解析は作業ツリーと同じ `analyzeModuleBoundaries` だけを通る。
  - 代替検査: `boundary-debt-ratchet.test.ts` の基準が `legacy` を宣言する場合、`boundary-fitness.test.ts` の旧形式の表と `legacy`、`schema-tables.test.ts` の「所有 Context」の見出し。
- [x] T003 [Docs] バックエンド設計から移行前の責務表の規則を外す。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-tools-file -- check/src/boundary-fitness.test.ts`
- `mise run test-tools-file -- check/src/boundary-debt-ratchet.test.ts`
- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`

## リスク

- 基準 revision に移行前の責務表が残っているうちに外すと、負債の比較検査が基準を解析できずに失敗する。
  着手の条件を T001 で確かめる。

## 要約

負債の比較検査は、基準 revision の責務表も作業ツリーと同じ規則で読むようになった。
公開方式 `legacy`、旧形式の責務表、データベース設計の「所有 Context」の見出しを読む互換と、`legacy` から `internal` へ切り替えたモジュールの利用元の読み替えを外した。
基準 revision がこれらの形を持てば、比較検査は解析の失敗を報告する。
バックエンド設計の公開範囲の節から、移行前の責務表を読む規則を外した。
`spec-diff` に規範の差分はない。
