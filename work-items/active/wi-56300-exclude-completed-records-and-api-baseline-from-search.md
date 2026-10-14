---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-14
priority: p1
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "検索ツールが既定で読む範囲だけを変える。製品の HTTP 応答、永続状態、イベント、TypeSpec と規範要件、機械検査の判定は維持する。"
---

# 完了した作業記録と OpenAPI のベースラインを既定の検索対象から外す

## 動機

2026-10-14 の計測では、追跡されているテキストのうち `work-items/done/`（614 件）が 9.0 MB、`spec/idmagic.openapi.baseline.json` が 1 ファイルで 2.0 MB ある。
どちらも日常の変更では読む必要がないが、`rg`、`fd`、`ast-grep` とエージェントの検索ツールは既定でこれらを走査する。
その結果、要件 ID や型名の検索に過去の記録とベースラインの一致が大量に混ざり、検索結果がコンテキストを消費する。

## 対象範囲

- リポジトリ直下に `.ignore` を置き、`work-items/done/` と `spec/*.openapi.baseline.json` を除外する。
- `mise run spec-where` のうち `work-items` を探す行が、除外後に完了記録を返さなくなる挙動の確認と、必要なら明示的な指定。
- 除外した範囲を検索する方法（`rg --no-ignore-dot` など）を、[リポジトリの配置](../../docs/development/repository-layout.md)に 1 行だけ追記する。

## 対象外

- 完了記録の本文の削減。[完了記録の縮約](wi-29555-collapse-completed-work-item-records-into-summaries.md)が扱う。
- 未公開の間のベースラインの要否と削除。`check-api-compat` の判断を伴うので別に扱う。
- `.gitignore` の変更。両方とも追跡は続ける。

## 設計

`.gitignore` ではなく `.ignore` を使う。
`rg`、`fd`、`ast-grep` は `.ignore` を読み、git の追跡と `git grep` には影響しない。
リポジトリ内のツールは完了記録を `readdir` と `git grep` で読むため、`check-work-items`、`brief`、`work-item-number`、`move-work-item` の判定は変わらない。

採らない案:

- エージェントの設定ごとに除外を書く案。実行環境ごとに設定が分かれ、`rg` を直接使う経路を覆えない。
- 完了記録を別のディレクトリ階層へ移す案。相対リンクの深さが変わり、[配置の改善](../done/wi-35829-keep-active-and-completed-work-items-at-the-same-depth.md)の前提を崩す。

## 計画

1. `.ignore` を追加し、除外の前後で `rg -c` による代表的な検索の一致件数を比べる。
2. `mise.toml` で `rg` と `fd` を使うタスクを確かめ、完了記録を探す意図のある行には `--no-ignore-dot` を付ける。
3. 配置の文書へ 1 行を追記する。

## タスク

- [ ] T001 [Tooling] `.ignore` を追加する。
- [ ] T002 [Tooling] `spec-where` などの `rg` を使うタスクの意図を確かめ、必要な行だけ明示する。
- [ ] T003 [Docs] 除外した範囲を検索する方法を配置の文書へ追記する。
- [ ] T004 [Verify] 検索の一致件数の前後と、作業記録の検査の結果を記録する。

## 検証

- `rg -c REQ-AUTHN` などの代表的な検索で、`work-items/done/` とベースラインが一致に現れないことを確かめる。
- `mise run check-work-items`、`mise run check-links`、`mise run brief -- <完了済みの記録>` が除外の前と同じ結果になることを確かめる。

## リスク

完了記録を探したい場面で、見つからないことを「存在しない」と誤読する。
除外を配置の文書に明記し、`--include history` を持つ `brief` は `git grep` を使うので影響を受けないことを確かめる。
