---
depends_on: [wi-526-durable-work-item-worktrees-and-checkpoints, wi-527-move-layer-checks-next-to-green]
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p3
change_kind: tooling
spec_impact:
  kind: none
  reason: "作業開始時に一度だけ払う準備と、狭い段のテストを並べて走らせる書き方を手順に足すだけである。検査の内容も製品の振る舞いも変えない。"
---

# 作業開始時の setup を 1 回にまとめ、狭い段の並列実行を手順に書く

## Motivation

wi-257 の Timing Analysis には、準備の欠落が最終ゲートの失敗として現れた項目が 2 つある。

- 仕様生成の初回 3.61 秒が Mermaid 未導入で失敗した。分析は「作業開始時に `setup-tools` を一度完了すれば、初回 3.61 秒は不要だった」と書いている。
- UI 依存関係の導入 1.72 秒が必要になった。永続 worktree に `node_modules` が無かったためで、分析は「worktree 作成直後の共通 setup に含めれば最終ゲートの失敗は避けられた」と書いている。

秒数は小さいが、現れ方が悪い。
どちらも最終集約ゲートの初回失敗 37.90 秒の一部として、実装が終わったつもりの時点で出ている。

同じ Timing Analysis に、逆向きの観測もある。
SSRF 対策後の対象検証で **4 テストを並列実行し、直列なら約 26 秒のところを約 14 秒に短縮した**。
この書き方は手順のどこにも書かれていないので、毎回その場で選び直されている。

## Scope

- 作業開始時に一度だけ払う準備を、`implement-work-item` skill の着手手順に入れる。
- 準備の対象を、Mermaid を含む `setup-tools`、UI 依存関係、Go と lint のキャッシュの暖機に定める。
- 狭い段の複数テストを並べて走らせる `mise` の書き方を、skill と `docs/development/local-development.md` に書く。

## Out of Scope

- 準備を自動で行う新しい `mise` タスクの追加。Design のとおり、既存タスクの組み合わせで足りる。
- 検査そのものの高速化と、層ごとの検査の前倒し。wi-527 が持つ。
- worktree の置き場所。wi-526 が持つ。
- `test-ui-e2e` の spec 並列実行。wi-497 が Out of Scope として据えた判断を変えない。

## Design

### 準備は既存タスクの列挙で足りる

作業開始時に払うのは次である。

| 対象 | 手段 | 実測 |
| --- | --- | ---: |
| Mermaid を含むツール | `mise run setup-tools` | 1.24秒 |
| UI 依存関係 | `mise run install-ui` | 1.72秒 |
| lint のキャッシュ | `mise run lint-go` (冷状態) | 18.3秒 |

`lint-go` の冷状態 18.3 秒は 2026-09-10 の実測である。
実装中の各回が 5.3 秒で済むのはこのキャッシュが効いているからなので、暖機は wi-527 が前倒しした検査の前提でもある。

新しいタスクは作らない。
1 行で呼べる `setup-workspace` のようなタスクを足すと、`mise run setup` との関係を説明する行が要り、`check-command-map` の対象も増える。
skill に 3 つ並べるほうが、何を払っているかが読んで分かる。

### 並列実行の書き方を実測で確かめた

2026-09-10 に確かめた。

```
mise run -c test-go-package ./backend/apitoken/domain ::: test-go-package ./backend/oauth2/domain
```

この形は動く。
一方、`--` を挟む形 (`test-go-package -- <package> ::: ...`) では `:::` 以降のタスクが実行されない。
`mise run <task> -- <args>` は単独実行のときの書き方なので、並べるときは `--` を外す。

この違いは実行しないと分からず、間違えても後続が黙って走らないだけなので気付きにくい。
手順に書く価値があるのはこの点である。

## Plan

1. wi-526 が worktree の置き場所を、wi-527 が第 8 段の検査の対応を決めた後に着手する。3 件とも `implement-work-item` skill の同じ範囲を編集するので、直列にする。
2. 準備の 3 行を着手手順に入れる。
3. 並列実行の書き方を、`--` を外すという注意とともに書く。
4. `mise run check-agent-guidance` と `mise run verify` を通す。

実装内容を変える未決事項はない。

## Tasks

- [ ] T001 [Docs] `implement-work-item` skill の着手手順に、作業開始時の準備 3 行を入れる。
- [ ] T002 [Docs] 狭い段の並列実行の書き方を、`--` を外す注意とともに skill と `docs/development/local-development.md` に書く。
- [ ] T003 [Verify] 書いた形をそのまま実行し、2 つのタスクが両方走ることを確かめる。`--` を挟んだ形では後続が走らないことも確かめる。
- [ ] T004 [Verify] `mise run check-agent-guidance` と `mise run verify` を通す。

## Verification

- Acceptance RED: 変更前の `implement-work-item` skill と `docs/development/local-development.md` に、作業開始時の準備と `mise run -c` の書き方が無いことを観測する。
- `mise run check-agent-guidance`
- `mise run check-links`
- `mise run verify`

## Risk Notes

- `depends_on` に 2 件を置いたのは、同じ skill ファイルの近い範囲を 3 件が編集するためである。内容上の依存ではないので、先行 2 件の結論が変わってもこの work item の内容は変わらない。
- 準備を手順に書いても、払い忘れは防げない。払い忘れたときに何が起きるかは wi-257 が記録しているとおり最終ゲートの失敗なので、失敗の形からたどれる。機械的な強制は入れない。
- `mise` の `:::` の扱いは実測で確かめた版に依存する。`mise` の版が上がったときに書き方が変わりうるので、手順には実測した形をそのまま書き、根拠としてこの work item を参照する。
- 冷状態の 18.3 秒を作業開始時に払うと、着手の体感は 18 秒遅くなる。実装中の各回が 5.3 秒で済むことと引き換えであり、1 回の work item で lint を 4 回以上走らせるなら得になる。wi-257 の lint 修正は 3 分かかっており、回数はこれを超える。
