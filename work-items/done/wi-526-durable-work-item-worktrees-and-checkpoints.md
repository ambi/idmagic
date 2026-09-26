---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: エージェントの作業場所と途中経過の残し方を変えるだけで、製品の振る舞いも公開契約も変わらない。リリースの読み手に見えるものが無い。
  references: []
spec_impact:
  kind: none
  reason: "work item を実装する作業場所の置き場所と、途中経過を残す規律だけを変える。製品の観測可能な振る舞いも公開契約も変えない。"
initial_context:
  specification: []
  typespec: []
  source:
    - .gitignore
    - tools/check/src/check-links.ts
    - tools/check/src/check-boundaries.ts
    - tools/workspace/src/workspace.ts
    - .agents/skills/parallel-work-items/SKILL.md
    - .agents/skills/implement-work-item/SKILL.md
  tests:
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading:
    - backend
    - frontend
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

**`check-links` と `check-boundaries` は降りていく。**
どちらも `WorkspaceSnapshot.files('', <除外名>)` でリポジトリ root から再帰的に走査する。
`check-links` が除外するのは `.git`、`node_modules`、`spec/generated` だけなので、`.worktrees/` を足さないと worktree 内の Markdown が二重に検査される。
`check-boundaries` は `backend/` と `frontend/src/` で始まるパスだけを検査するので worktree の Go を誤検出しないが、`*/architecture.yaml` はどの深さでも拒否し、worktree の全ファイルを毎回たどる。
ほかの検査 (`check-ids`、`check-security-controls`、`check-contract-drift`) は `work-items`、`backend`、`spec/contexts` のような名前付きの root から走査するので影響を受けない。
`discoverWorkspaceConfig` の旧仕様文書の走査は `.` で始まるディレクトリを降りない。
`rg` と `fd` は `.gitignore` に従う。

`lint-repo` の `betterleaks dir .` も root から走査する。
`.betterleaks.toml` の許可は `^dev\.sh$` のように root 起点で書かれているので、worktree 内の同じファイルが許可から外れ、既知の開発用資格情報を秘密として報告する。
この走査は readiness では見落とし、T002 の `mise run verify` で 6 件の検出として現れた。

したがって必要な変更は、`.gitignore` への 1 行、`check-links` と `check-boundaries` の除外名への 1 語ずつ、`.betterleaks.toml` への `^\.worktrees/` の許可である。

**退けた案: sandbox の書き込み許可に `../<repo>-wi-*` を足し、skill の指示のまま repo の隣に置く。**
リポジトリ側の変更が要らないので一見すると軽い。
しかし許可を書く `.claude/settings.json` は sandbox の書き込み禁止対象なので、エージェントは自分で設定できず、機械ごと、利用者ごとに人手の設定が要る。
設定されていない機械では同じ事故が再発し、しかも再発するまで気付けない。
リポジトリの中で完結する案を選ぶ。

### checkpoint はタスクごとに残す

`implement-work-item` の Tasks は T001、T002 のように区切られている。
1 タスクが GREEN になった時点で checkpoint commit を作れば、消失で失われる量はそのタスクぶんに収まる。
wi-257 に当てはめれば、最大でも T003/T004 の 12 分である。

checkpoint は件名を `checkpoint(<work-item-id>): <task-id> <内容>` とし、途中経過であることを件名だけで見分けられるようにする。
`implement-work-item` は、1 つの work item を 1 コミットにし、構造の変更と振る舞いの変更を分けるときだけ複数コミットにすると定めている。
checkpoint はこの規律の外にある途中経過なので、正式なコミットを作る時点で畳む。

畳む操作は `git reset --soft <直前の正式なコミット>` の後に正式なコミットを作る形にする。
対話的な rebase はこの環境で使えないので採らない。
構造の変更を先にコミットする場合は、振る舞いの作業へ移る前に、それまでの checkpoint を構造のコミットへ畳む。
畳む前に `git log --oneline <直前の正式なコミット>..HEAD` で、対象が checkpoint だけであることを確かめる。

## Plan

1. `.gitignore` と `check-links` の除外を先に入れる。worktree を作る前に置き場所を安全にする。
2. `.worktrees/` に worktree を実際に作り、`mise run verify` が worktree の有無で結果を変えないことを確かめる。
3. 2 つの skill を直す。
4. 畳む手順を書き、checkpoint が残った状態から 1 コミットを作れることを確かめる。

実装内容を変える未決事項はない。置き場所は Design で決めた。

## Tasks

- [x] T001 [Tooling] `.worktrees/` 配下の Markdown と `architecture.yaml` を検査が拾うことを `check-links`、`check-boundaries` の単体テストで RED にし、`.gitignore` と両検査の除外名へ `.worktrees` を加えて GREEN にする。実行: `mise run test-tools-file -- <file>`、`mise run test-tools`。
- [x] T002 [Verify] `.worktrees/` に worktree を作った状態で `mise run verify` を通し、worktree が無い状態と同じ結果になることを確かめる。`go list ./...` が worktree のパッケージを含まないことも確かめる。
- [x] T003 [Docs] `parallel-work-items` skill の worktree 作成手順を `.worktrees/<work-item-id>` に直す。
- [x] T004 [Docs] `implement-work-item` skill に、タスクごとの checkpoint commit と、完了時に 1 コミットへ畳む手順を入れる。
- [x] T005 [Verify] checkpoint が複数残った状態から 1 コミットを作り、`commit` skill の形式を満たすことを確かめる。`mise run verify` を通す。

## Verification

- Acceptance RED: `.worktrees/wi-999` に worktree を作った状態で `mise run check-links` を実行し、検査した文書数が倍になることを観測する。
- Unit RED: `tools/check/src/worktree-exclusion.test.ts` の 2 件が、`.worktrees/` 配下の壊れたリンクと `architecture.yaml` を指摘として拾うことを観測する。
- `mise run test-tools`
- `mise run verify` (worktree がある状態と無い状態の 2 回)
- `mise run check-command-map`

## Risk Notes

- `check-links` 以外にリポジトリ root から走査する道具が残っていれば、同じ二重検査が起きる。T002 が `verify` 全体を worktree のある状態で走らせるので、残っていればそこで現れる。
- checkpoint commit は履歴に途中経過を残す。畳み忘れると 1 work item 1 コミットの規律が崩れるので、T005 で畳む手順を確かめる。畳む前に push してしまうと畳めなくなるが、`implement-work-item` は明示的な指示があるまで push しないと定めている。
- worktree をリポジトリ配下に置くと、worktree の中で `git status` を実行したときの表示や、エディターのファイル検索の対象が変わる。`.gitignore` に入れるので追跡はされない。
- 置き場所を変えても、消失そのものを禁止できるわけではない。checkpoint は消失が起きたときの損失を区切るためにあり、置き場所の変更とは独立に効く。

## Completion

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` は規範仕様の差分を報告しない。変更はエージェントの作業場所と、リポジトリ root から走査する検査の除外に閉じる。
  work item 実装用の worktree は `.worktrees/<work-item-id>` に置く。`.gitignore`、`check-links`、`check-boundaries`、`.betterleaks.toml` がこの場所を除外する。
  `parallel-work-items` skill の作成、統合、片付けの手順はすべて `.worktrees/` を使い、`implement-work-item` skill は別の worktree を `/tmp` や `$TMPDIR` に作らないと定める。
  `implement-work-item` skill は、タスクが GREEN になるたびに確認なしで checkpoint commit を作り、正式なコミットの前に `git reset --soft <直前の正式なコミット>` で畳むと定める。checkpoint を確認なしで作ることは利用者が選んだ。
  readiness で見落とした `betterleaks` の root 走査は、worktree のある状態で走らせた `mise run verify` で見つけて直した。
- **Acceptance RED Evidence**:
  - **Test**: HEAD から `.worktrees/wi-999` に worktree を作り、`mise run check-links` と `git status --short` を実行した。
  - **Requirement**: N/A: 製品の振る舞いを変えないツールと手順の変更である。
  - **Observed Failure**: `check-links` の検査文書数が 927 から 1853 に倍増し、`git status` に `?? .worktrees/` が現れた。worktree のある状態の `mise run verify` は `lint-repo` の `betterleaks` が `.worktrees/wi-999/dev.sh` など 6 件を報告して失敗した。
  - **Detection Reason**: 文書数と検出件数そのものを観測するので、worktree 内のファイルが走査に入ったかを直接区別できる。修正後は文書数 927、`git status` に現れず、`verify` は成功した。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/worktree-exclusion.test.ts` の `check-links does not inspect Markdown inside work-item worktrees`、`check-boundaries does not inspect files inside work-item worktrees`
  - **Requirement**: N/A: 製品の振る舞いを変えないツールの変更である。
  - **Observed Failure**: 修正前は `check-links` が `.worktrees/wi-999/docs/README.md:3: relative link target does not exist` を報告し、`check-boundaries` が `ok: false` を返して、2 件とも失敗した。
  - **Detection Reason**: worktree 内には、走査に入れば必ず指摘される壊れたリンクと `architecture.yaml` を置いている。どちらかの検査の除外を外すと、その検査のテストだけが失敗する。
- **Change-Resistance Results**:
  - Go の変更は無いので `test-go-mutation` は対象外である。除外名を外す故障は上の Unit RED そのものであり、2 件の検査それぞれで独立に検出した。
  - 畳む手順は本 work item 自身で確かめた。T001 と T003/T004 の checkpoint commit を 2 件作り、`git log --oneline 686dac2a..HEAD` が checkpoint だけであることを確かめてから `git reset --soft 686dac2a` で畳み、全変更が staged のまま残ることを確認した。
- **Verification Results**:
  - `mise run test-tools` - 成功（594 件）
  - `mise run check-links` - 成功（927 文書、worktree の有無で同じ）
  - `mise run check-command-map` - 成功
  - `go list ./...` - worktree のある状態で 308 パッケージ、`.worktrees` を含むパッケージは 0 件
  - `mise run verify`（worktree のある状態、サンドボックス外） - 成功
  - `mise run verify`（worktree の無い状態、サンドボックス外） - 成功
