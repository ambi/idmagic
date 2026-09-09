---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p1
change_kind: tooling
spec_impact:
  kind: none
  reason: "work item を実装する作業場所の置き場所と、途中経過を残す規律だけを変える。製品の観測可能な振る舞いも公開契約も変えない。"
---

# work item 実装用の worktree を消えない場所へ移し、タスクごとに checkpoint を残す

## Motivation

wi-257 の Timing Analysis は、能動作業時間 94 分のうち **35 分 (37%) を「一時 worktree 消失からの復旧」に費やした**と記録している。
分析はこれを「純粋な不要コスト」と呼び、OS が `/tmp` の worktree を除去して未コミット変更を再実装したと書いている。
これは単一の項目として最大であり、実装 38 分に匹敵する。

原因は偶発ではなく、指示と実行環境の食い違いである。

`parallel-work-items` skill は worktree の置き場所を `../<repo>-<work-item-id>` と指示する。
一方、エージェントの sandbox がファイルを書ける場所はプロジェクトディレクトリと `$TMPDIR` に限られる。
**指示された場所には書けない**ので、`/tmp` が選ばれ、OS の掃除で消えた。
同じ条件が続くかぎり、次も同じことが起きる。

失われたのが 35 分で済んだのは、消えた時点の未コミット変更が 35 分ぶんだったからである。
`implement-work-item` は完了時に 1 コミットを作るので、それまでの全作業が 1 回の消失で失われる形になっている。

## Scope

- work item 実装用の worktree の置き場所を決め、sandbox が書ける場所へ移す。
- 決めた場所を `.gitignore` と、リポジトリ全体を走査する検査の除外へ加える。
- `parallel-work-items` skill の worktree 作成手順を、決めた場所に合わせる。
- `implement-work-item` skill に、タスクごとの checkpoint commit を入れる。
- 完了時に 1 work item 1 コミットへ畳む手順を、checkpoint と両立する形で書く。

## Out of Scope

- 並列作業そのものの運用。`parallel-work-items` skill が別に持つ。
- コミットメッセージの形式。`commit` skill が持つ。
- 消失した作業の自動復旧。checkpoint があれば失われる量が 1 タスクに収まるので、追加の仕掛けを作る理由がない。

## Design

### 置き場所は `.worktrees/<work-item-id>` にする

リポジトリ配下なので sandbox が書ける。
`$TMPDIR` やシステムの一時領域と違い、OS の掃除の対象にならない。

副作用を 2 つ調べた。

**`go list ./...` は降りていかない。**
worktree は自身の `go.mod` を持つ nested module なので、親モジュールのパターンから除外される。
`build-go`、`test-go-race`、`lint-go` の対象は変わらない。

**`check-links` は降りていく。**
`tools/check/src/check-links.ts` はリポジトリ root から再帰的に走査し、除外しているのは `.git`、`node_modules`、`spec/generated` だけである。
`.worktrees/` を足さないと、worktree 内の Markdown が二重に検査される。
ほかの検査 (`check-ids`、`check-boundaries`、`check-security-controls`、`check-contract-drift`) は `work-items`、`backend`、`spec/contexts` のような名前付きの root から走査するので影響を受けない。
`rg` と `fd` は `.gitignore` に従う。

したがって必要な変更は、`.gitignore` への 1 行と `check-links` の除外への 1 行である。

**退けた案: sandbox の書き込み許可に `../<repo>-wi-*` を足し、skill の指示のまま repo の隣に置く。**
リポジトリ側の変更が要らないので一見すると軽い。
しかし許可を書く `.claude/settings.json` は sandbox の書き込み禁止対象なので、エージェントは自分で設定できず、機械ごと、利用者ごとに人手の設定が要る。
設定されていない機械では同じ事故が再発し、しかも再発するまで気付けない。
リポジトリの中で完結する案を選ぶ。

### checkpoint はタスクごとに残す

`implement-work-item` の Tasks は T001、T002 のように区切られている。
1 タスクが GREEN になった時点で checkpoint commit を作れば、消失で失われる量はそのタスクぶんに収まる。
wi-257 に当てはめれば、最大でも T003/T004 の 12 分である。

完了時は checkpoint を 1 コミットへ畳む。
「1 work item 1 コミット」は wi-497 の M8 が決めた規律であり、checkpoint はその規律を崩さない範囲の途中経過として扱う。

## Plan

1. `.gitignore` と `check-links` の除外を先に入れる。worktree を作る前に置き場所を安全にする。
2. `.worktrees/` に worktree を実際に作り、`mise run verify` が worktree の有無で結果を変えないことを確かめる。
3. 2 つの skill を直す。
4. 畳む手順を書き、checkpoint が残った状態から 1 コミットを作れることを確かめる。

実装内容を変える未決事項はない。置き場所は Design で決めた。

## Tasks

- [ ] T001 [Tooling] `.gitignore` に `.worktrees/` を加え、`check-links` の除外へ同じ名前を加える。実行: `mise run test-tools`。
- [ ] T002 [Verify] `.worktrees/` に worktree を作った状態で `mise run verify` を通し、worktree が無い状態と同じ結果になることを確かめる。`go list ./...` が worktree のパッケージを含まないことも確かめる。
- [ ] T003 [Docs] `parallel-work-items` skill の worktree 作成手順を `.worktrees/<work-item-id>` に直す。
- [ ] T004 [Docs] `implement-work-item` skill に、タスクごとの checkpoint commit と、完了時に 1 コミットへ畳む手順を入れる。
- [ ] T005 [Verify] checkpoint が複数残った状態から 1 コミットを作り、`commit` skill の形式を満たすことを確かめる。`mise run verify` を通す。

## Verification

- Acceptance RED: `.worktrees/wi-999` に worktree を作った状態で `mise run check-links` を実行し、worktree 内の Markdown が検査対象に入ることを観測する。
- `mise run test-tools`
- `mise run verify` (worktree がある状態と無い状態の 2 回)
- `mise run check-command-map`

## Risk Notes

- `check-links` 以外にリポジトリ root から走査する道具が残っていれば、同じ二重検査が起きる。T002 が `verify` 全体を worktree のある状態で走らせるので、残っていればそこで現れる。
- checkpoint commit は履歴に途中経過を残す。畳み忘れると 1 work item 1 コミットの規律が崩れるので、T005 で畳む手順を確かめる。畳む前に push してしまうと畳めなくなるが、`implement-work-item` は明示的な指示があるまで push しないと定めている。
- worktree をリポジトリ配下に置くと、worktree の中で `git status` を実行したときの表示や、エディターのファイル検索の対象が変わる。`.gitignore` に入れるので追跡はされない。
- 置き場所を変えても、消失そのものを禁止できるわけではない。checkpoint は消失が起きたときの損失を区切るためにあり、置き場所の変更とは独立に効く。
