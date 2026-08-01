---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-01
depends_on: [wi-312-migrate-radix-to-base-ui]
---

# 未使用になった Radix UI / hugeicons パッケージを package.json から除去する

## Motivation

[[wi-312-migrate-radix-to-base-ui]] で `frontend/src` 配下のソースコードは Base UI
（`@base-ui/react`）へ完全移行済みで、`@radix-ui/*` を import しているファイルは
現在ゼロである（`grep -rl "@radix-ui" frontend/src` が空）。しかし
`frontend/package.json` からは実際にはパッケージを削除できておらず、以下が
未使用のまま残っている:

- `@radix-ui/react-dropdown-menu`
- `@radix-ui/react-label`
- `@radix-ui/react-slot`
- `@hugeicons/core-free-icons`（shadcn init 直後の初期 `iconLibrary` が `hugeicons`
  だった名残。すぐに `tabler` へ変更したため一度も使われていない）
- `@hugeicons/react`（同上）

wi-312 の Completion Summary には「`@radix-ui/*` を `@base-ui/react` に置き換えた」
と記載したが、これは誤りで実際には `@base-ui/react` を追加しただけで旧パッケージの
削除が漏れていた。レビューで指摘を受け、本 WI として切り出す。

## Scope

- `frontend/package.json`
- `frontend/bun.lock`

## Out of Scope

- ソースコードの変更（既に Base UI 移行済みで変更不要）。
- `shadcn` パッケージ自体の扱い（`dependencies` にあるが devDependency 相当。
  今後 `shadcn add` を使う運用が続く前提で、本 WI では動かさない）。

## Design

`bun remove` で未使用パッケージを外し、`bun install` で `bun.lock` を再生成する。
Radix / hugeicons のどちらも `frontend/src` から一切参照されていないことを事前に
`grep` で再確認してから実施する。

## Tasks

- [x] T001 [調査] `grep -rl "@radix-ui\|@hugeicons" frontend/src` が空であることを
      再確認する（着手時点で状態が変わっていないか念のため確認）。
      → 空(exit 1)を確認。
- [x] T002 [App] `cd frontend && bun remove @radix-ui/react-dropdown-menu @radix-ui/react-label @radix-ui/react-slot @hugeicons/core-free-icons @hugeicons/react`
      で `package.json` / `bun.lock` を更新する。
      → 5 パッケージ削除、`package.json`/`bun.lock` に `radix`/`hugeicons` の
      文字列が一切残っていないことを確認。
- [x] T003 [Verify] `just verify-ui`(format-check / lint / test-ui-unit / build)を
      通す。→ passed。
- [x] T004 [Verify] `just test-ui-e2e` を通す。→ passed(4 spec ファイル・22 テスト)。

## Verification

- `just verify-ui`
- `just test-ui-e2e`
- `grep -rl "@radix-ui\|@hugeicons" frontend/src` が空であること
- `frontend/package.json` に該当パッケージが残っていないこと

## Risk Notes

- 低リスク。対象パッケージはソースコードから一切参照されておらず、削除は
  ビルド成果物にもテストにも影響しない想定。`just verify-ui` / `just test-ui-e2e`
  で機械的に確認できる。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  `frontend/package.json` に残っていた未使用の `@radix-ui/react-dropdown-menu` /
  `@radix-ui/react-label` / `@radix-ui/react-slot`（[[wi-312-migrate-radix-to-base-ui]]
  で Base UI へ移行済みだが削除し忘れていた）と `@hugeicons/core-free-icons` /
  `@hugeicons/react`（shadcn init 直後の初期 `iconLibrary` の名残で一度も使われて
  いなかった）を `bun remove` で除去した。これにより frontend の Radix UI 依存は
  package.json レベルでも完全に無くなった。ソースコードの変更は無し。
- **Verification Results**:
  - `just verify-ui`(format-check / lint / test-ui-unit / build) - passed
  - `just test-ui-e2e` - passed(4 spec ファイル・22 テスト)
  - `grep -rl "@radix-ui\|@hugeicons" frontend/src` - 空(該当なし)
  - `frontend/package.json` / `frontend/bun.lock` に `radix`/`hugeicons` の
    文字列が残っていないことを確認
