---
name: parallel-work-items
description: Set up and coordinate parallel git worktrees and branches for multiple Regenerative Architecture work items. Use when the user wants to implement work items concurrently, create per-work-item workspaces, prepare branch/worktree commands, assign agents to work items, or integrate completed work-item branches back through an integration branch.
---

# 並列ワークアイテム実装

## Overview

複数の work item を別々の git worktree + branch で進める。各 worktree では
`implement-work-item` Skill を使い、統合地点では `scl-render` Skill を使って派生物を一度だけ
再生成する。

## 0. 前提を確認

1. リポジトリルートで `git status --short` を確認する。未コミット差分がある場合は、
   それが今回のセットアップ対象か、ユーザの既存作業かを分ける。
2. 対象 work item のファイルを確認し、id と状態を読む。未作成なら `new-work-item` Skill で
   先に作る。
3. base branch を決める。指定がなければ現在の branch を base とする。
4. worktree 置き場を決める。指定がなければリポジトリ親ディレクトリ直下に
   `<repo>-<short-id>` 形式で作る。

## 1. ブランチと worktree を作る

work item ごとに branch を作る。branch は実装・レビュー・統合の単位なので、worktree だけで
済ませない。

- branch: `work-item/<work-item-id>`
- worktree: `../<repo>-<work-item-id>` または短縮して `../<repo>-wi-<nn>`

新規 branch の例:

```sh
git fetch --all --prune
git worktree add -b work-item/wi-42-example ../idmagic-wi-42 <base-branch>
```

既存 branch を別 worktree に出す例:

```sh
git worktree add ../idmagic-wi-42 work-item/wi-42-example
```

複数 work item のセットアップでは、各 worktree 作成後に次を確認する。

```sh
git worktree list
git -C ../idmagic-wi-42 status --short --branch
```

## 2. 各 worktree で実装する

各 worktree は独立した agent / terminal に割り当てる。agent へ渡す指示は短くし、対象
work item と worktree パスを明示する。

例:

```text
Use the implement-work-item Skill in /path/to/idmagic-wi-42.
Implement work-items/wi-42-example.md end to end, verify it, update completion,
move it to done, and commit the branch. Do not push.
```

各 branch では `implement-work-item` Skill の順序に従う。

- SCL 変更がある場合は `scl-change` Skill を先に使う。
- work-item branch では HTML / JSON Schema / OpenAPI などの SCL 派生物を原則 commit しない。
- 検査用に `just scl-render` を実行してよいが、生成差分は統合 branch で作り直す。
- 完了時は work item に `completion` を追記し、`work-items/done/` へ移して commit する。
- push はユーザの明示指示があるまでしない。

## 3. 並列作業中の調整

衝突を早く見つけるため、各 branch の作業範囲を work item の `scope` に寄せる。

- 同じ `spec/scl.yaml` の同じ section を複数 branch が触る場合は、先に順序を決める。
- 同じ id / ファイル名を作った場合は `just check-ids` の結果に従って片方を採番し直す。
- 生成物の衝突は手で解かず、統合済み SCL から再生成する。

## 4. 統合する

統合用 branch / worktree を用意してから completed branch を取り込む。指定がなければ
`integration/work-items` branch 名を使う。

```sh
git worktree add -b integration/work-items ../idmagic-integration <base-branch>
cd ../idmagic-integration
git merge --no-ff work-item/wi-42-example
git merge --no-ff work-item/wi-43-example
```

統合後は SCL 派生物を同期する。

```sh
just yaml-check
just scl-render
just verify
```

`just scl-render` で出た派生物差分は、統合 branch の commit に含める。work-item branch 側へ
逆流させない。

## 5. 片付け

merge 済みで不要になった worktree はユーザ確認後に削除する。削除は作業ファイルを消す操作
なので、未コミット差分がないことを確認してから行う。

```sh
git -C ../idmagic-wi-42 status --short
git worktree remove ../idmagic-wi-42
```

branch 削除や remote push / delete は、ユーザの明示指示がある場合だけ実行する。
