---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-25
priority: p2
group: development-process
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "文書の書き方の規則を案内するだけで、製品の規範仕様、TypeSpec、検査器の判定は変えない。" }
---

# 表記の検査が拒否する語を、書く前に文書から分かるようにする

## 動機

`mise run check-terminology` は、`docs/` 以下の Markdown と根の `AGENTS.md`、`CONTRIBUTING.md`、`README.md`、`SECURITY.md` から採らない表記を拒否する。
採らない表記と採用する表記の組は、`tools/check/src/terminology.ts` の `TERMINOLOGY_RULES` にしかない（2026-11-25 時点で 28 語。「配備」「既定」「ランブック」「観測可能性」など）。
文書の書き方を定める `docs/formats/documentation-guide.md` の用語の箇条も、文章の言語を定める `docs/development/writing-language.md` も、この一覧を載せず、検査へのリンクもない。
書き手は、検査で落ちて初めてその語が採らない表記だと知る。
完了時の `move-work-item` も同じ検査を適用するので（`docs/formats/work-item-format.md` の「完了」）、作業の最後に書き直しが生じる。

[[wi-65845-document-primary-use-case-setup-and-task]] の調査で、`docs/formats/` の文書と検査器を突き合わせたときに見つけた。

## 対象範囲

- `documentation-guide.md` の用語の箇条に、表記の検査があることと、採らない表記と採用する表記の組の一次情報が `TERMINOLOGY_RULES` であることを書き、リンクする。
- 検査の対象（`docs/` 以下の Markdown と根の四文書）と、照合が文脈を読まない部分文字列の一致であることを一文で書く。

## 対象外

- 規則の追加、削除、言い回しの変更。
- 検査の対象の範囲の変更。
- 一覧を文書へ生成する仕組み。必要になれば別に起票する。

## 設計

一覧は `TERMINOLOGY_RULES` を一次情報のままにし、文書はそこへリンクする。
文書へ一覧を写すと、規則を足したときに片方だけが古くなるためである。
各規則は採用する表記をコードの中で日本語で示しているので、リンク先を読めば書き手は選べる。

採らない案：一覧を `documentation-guide.md` の表として持ち、検査器がその表を読む。
`document-layout-format.ts` が配置図を文書から読むのと同じ形で、ずれは生じない。
ただし規則ごとの `allow`（採用する表記が採らない表記を含む場合の例外）を表で表すには列が増え、28 語の表は用語の箇条の読みやすさを損なう。
リンクで足りなくなったときの改善として残す。

## 計画

1. `documentation-guide.md` の用語の箇条に、表記の検査の案内とリンクを足す。
2. リンクが解決し、追記した文が表記の検査を通ることを確かめる。

## タスク

- [ ] T001 [Docs] `documentation-guide.md` に表記の検査の案内と `TERMINOLOGY_RULES` へのリンクを足す。
- [ ] T002 [Verify] 変更を検証する。

## 検証

- `mise run check-links`：`terminology.ts` へのリンクが解決することを確かめるため。
- `mise run check-terminology`：追記した文が採らない表記を含まないことを確かめるため。

## リスク

リンク先がコードなので、規則のファイルを移すとリンクが切れる。
`check-links` が切れたリンクを拒否するので、移した変更の中で気付ける。
