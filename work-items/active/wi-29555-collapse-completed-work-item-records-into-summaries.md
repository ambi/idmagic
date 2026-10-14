---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-14
priority: p2
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "main に取り込んだ完了記録の本文を要約へ縮め、全文は git 履歴から読む形にする。製品の HTTP 応答、永続状態、イベント、TypeSpec と規範要件は維持する。"
---

# main に取り込んだ完了記録を要約だけにし、全文は git 履歴に任せる

## 動機

2026-10-14 の計測では、`work-items/done/` は 614 件で 9.0 MB あり、`backend/`（11.5 MB）に次いで大きい。
完了記録は当時の作業の記録であり、[作業項目の形式](../../docs/formats/work-item-format.md#着手時の文脈と設計)は完了後に検証し直さない。
それでも全文が作業ツリーにあるため、検索、`brief --include history`、リンクをたどる読み取りのたびにコンテキストを消費する。
全文は完了したコミットに残っており、作業ツリーに置き続ける理由は番号の衝突回避、`depends_on` の解決、リンク先の存在だけである。

## 対象範囲

- main に取り込み済みの完了記録と中止記録を、縮約形へ書き換えるツールと `mise` タスク。
- 縮約形の定義と、`check-work-items` と `check-links` が縮約形を受け付けること。
- 既存の 614 件の縮約。
- 縮約した記録への、アンカー付きリンクの書き換え。

## 対象外

- 完了節の書き方そのものの変更。[最終的な検証証拠の一回の記載](wi-82135-record-final-verification-evidence-once.md)が扱う。
- 履歴の revision で参照を解決する仕組み。縮約後に [wi-29959](wi-29959-resolve-completed-work-item-evidence-at-its-historical-revision.md) の残りの必要性を見直す。
- 完了時に自動で縮約すること。完了の変更は全文で検証する必要があるため、縮約は main に取り込んだ後に別のコミットで行う。

## 設計

縮約形は、ファイル名を保ったまま次だけを残す。

```markdown
---
status: completed
archived_from: <全文を含む最後のコミット>
---

# <題名>

<完了節の要約>
```

ファイル名を保つので、`work-item-number` の番号の衝突回避、`depends_on`、ファイルへのリンクは変わらない。
`archived_from` は、縮約の直前にそのファイルを最後に変えたコミットとし、`git show <commit>:<path>` で全文を読める。
1 件あたり 500 バイト前後を見込むと、9.0 MB は 0.3 MB 程度になる。

縮約の対象は、main とのマージベースで既に `work-items/done/` にあり、作業ツリーで変わっていない記録とする。
完了させる変更の中の記録は全文のまま検証を受け、取り込まれた後の縮約で初めて短くなる。
縮約したファイルは書き換えにあたるので、`check-work-items` は `archived_from` を持つ記録に完了節の証拠の検査を適用せず、形の検査だけを行う。

採らない案:

- ファイルを消して索引の 1 ファイルにまとめる案。番号の衝突回避と `depends_on` の解決を索引の読み取りへ移す必要があり、リンクもすべて書き換わる。
- 別ブランチやリポジトリ外へ移す案。全文を読む経路が二つになり、`git log` だけでたどれなくなる。
- frontmatter を残す案。`affected_spec` と `primary_use_cases` が残ると、それを解決し続ける費用が残る。

## 計画

1. 縮約形の検査を `check-work-items` に加え、`archived_from` の存在と、そのコミットに全文があることを確かめる。
2. 縮約ツールを `tools/work-item` に加え、`mise run archive-work-items` として公開する。
3. アンカー付きリンクで縮約した記録を指す箇所を、アンカーを外したファイルへのリンクと `archived_from` の案内に書き換える。
4. 既存の記録をまとめて縮約し、件数とサイズの前後を記録する。

`brief --include history` の出力は要約だけになる。
全文が必要な場合の読み方は `archived_from` を表示して示し、`brief` が全文を自動で取得することはしない。

## タスク

- [ ] T001 [Tooling] `check-work-items` が縮約形を受け付け、`archived_from` を検証する。RED を先に確かめる。
- [ ] T002 [Tooling] `archive-work-items` を追加する。対象の選択と書き換えをテストする。
- [ ] T003 [Tooling] 縮約した記録へのアンカー付きリンクを書き換える。
- [ ] T004 [Docs] 作業項目の形式の完了記録の節へ、縮約形と全文の読み方を追記し、置き換えた記述を削る。
- [ ] T005 [Cleanup] 既存の記録を縮約する。
- [ ] T006 [Verify] サイズの前後と検査の結果を記録する。

## 検証

- `mise run test-tools-file` で、縮約の対象の選択、縮約形の検査、リンクの書き換えを確かめる。
- `mise run check-work-items`、`mise run check-links`、`mise run verify` を縮約の前後で通す。
- 縮約した任意の 3 件で、`git show <archived_from>:<path>` が全文を返すことを確かめる。

## リスク

縮約の後に、過去の判断の理由を探す読み手が要約だけを見て結論を誤る。
縮約形の本文に `archived_from` の読み方を 1 行で示し、`brief --include history` も同じ案内を出す。
`git` の浅いクローンでは全文を読めないので、検査は `archived_from` の存在だけを確かめ、コミットを取得できない環境では未検証として示す。
