---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/README.md]
  source: [DOCUMENTATION_GUIDE.md, SPECIFICATION_FORMAT.md, AGENTS.md, README.md]
  tests: []
  stop_before_reading: [backend, frontend, spec]
risk: low
created_at: 2026-08-23
priority: p3
change_kind: docs
spec_impact: { kind: none, reason: "運用文書を 1 階層浅くするだけの変更である。正規文書の集合にも、規範的な要素（REQ-* のシナリオ、規範 ID、TypeSpec symbol）にも触れない。docs/operations/ は閉じた集合の外側にあり、検査の対象でもない。" }
---

# 子が 1 つしかない `docs/operations/runbooks/` を 1 階層畳む

## Motivation

[[wi-405-spec-and-docs-boundary-is-not-legible]] が `docs/operations/runbooks/` を作ったが、**`docs/operations/` の子は `runbooks/` 1 つだけである。** 中身は 3 ファイルしかない。

階層が 1 つ増えると、参照する側は毎回 1 段深く降り、`docs/operations/` を開いた人は `runbooks/` へもう一度入ることになる。その代償に見合う分岐がここには無い。DOCUMENTATION_GUIDE §3 の「必要が生じていない文書を作らない」は、必要が生じていないディレクトリにも同じだけ当てはまる。

**将来の予定を確認したうえで畳む。** backlog を突き合わせると、`docs/operations/` の下に**サブディレクトリ**を足す計画は 1 件も無い。

| work item | `docs/operations/` へ足すもの | 種類 |
|---|---|---|
| [[wi-290-alert-runbook-catalog-and-on-call-operations]] | アラートごとの runbook と、その索引 | ファイル |
| [[wi-400-service-objectives-need-stable-ids]] | `page` 級アラートの runbook | ファイル |

足されるのはすべてファイルであって、ディレクトリではない。

## Scope

- `docs/operations/runbooks/*.md` を `docs/operations/` 直下へ移す。
- DOCUMENTATION_GUIDE §4 の構成図と §11.4 の見出しを、`runbooks/` を持たない形に改める。
- SPECIFICATION_FORMAT.md §1 の構成図を追随させる。
- `AGENTS.md` の言語表、`README.md`、`infra/backup/` のスクリプトとREADME、進行中の wi-290 と wi-400 の参照を張り替える。

## Out of Scope

- `docs/operations/` の中身を書くこと。runbook の追加は wi-290 と wi-400 が持つ。
- 完了済み work item の散文に残るパス。当時の記録であり、書き換えれば履歴の改竄になる。
- `docs/development/` の新設。まだ該当する内容が無い（wi-405 の Left Undone）。

## Design

**`docs/runbooks/` ではなく `docs/operations/` に畳む。**

畳み先は 2 つありえた。

| 案 | 得るもの | 失うもの |
|---|---|---|
| `docs/runbooks/` | 今ある 3 ファイルに対して名前がいちばん正確 | `reliability.md` が来たときに置き場所が無い。`docs/` 直下は正規文書の閉じた集合なので入れられず、`operations/` を作り直して runbooks を再度動かすことになる |
| `docs/operations/`（採用） | 運用文書の家が 1 つで、ファイルが増えても移動が要らない | ディレクトリ名が中身の種類（runbook）を言わない |

**後者を採る理由は、DOCUMENTATION_GUIDE §11 との一致である。** ガイドは運用文書を「SLO、リリースと後退、バックアップ、runbook」というカテゴリとして定義しており、リポジトリだけが `runbooks/` と名乗ると、[[wi-405-spec-and-docs-boundary-is-not-legible]] が直したばかりの「ガイドと実態の食い違い」が再発する。ディレクトリ名は成果物の種類ではなく読み手の関心（運用）を表すべきで、それは §11 が「読み手で分ける」と決めたことと同じ向きでもある。

**`reliability.md` が来るかもしれないことは理由にしない。** wi-400 の Design 1 はそれを未決の論点として並べているだけで、同じ節が「増やさず `capacity.md` に ID を足す側に寄せるのが素直」と書いている。**予定されていないものを根拠に階層を残すのは、この work item が畳もうとしている無駄と同じ性質である。**

runbook が増えて索引が要るようになったら、wi-290 が計画している索引は `docs/operations/README.md` になる。**これはガイドの「`README.md` はディレクトリを開いたときに表示されるため、索引の置き場所として使う」に素直に乗る。** `runbooks/README.md` より 1 段浅いところで同じ役目を果たす。

runbook が十分に増えて他の運用文書を押しのけるようになったら、そのときサブディレクトリを与える。**その判断は件数を見てから行う。いまは 3 件である。**

## Tasks

- [x] T001 [Docs] `docs/operations/runbooks/*.md` を `docs/operations/` へ移す。
- [x] T002 [Docs] DOCUMENTATION_GUIDE §4 と §11.4、SPECIFICATION_FORMAT.md §1 を追随させる。
- [x] T003 [Docs] `AGENTS.md`、`README.md`、`infra/backup/`、wi-290、wi-400 の参照を張り替える。
- [x] T004 [Verify] `mise run verify` を通し、相対リンクが全件解決することを確認する。

## Verification

- `mise run verify`
- 手動: 全 Markdown の相対リンクが解決することを確認する。
- 手動: `infra/backup/` のスクリプトが表示するパスが実在することを確認する。**スクリプトの中の文字列はリンク検査に掛からないので、ここだけは目で見る。**

## Risk Notes

リスクは low。3 ファイルの移動と参照の張り替えであり、検査の対象外の領域である。

**この変更が間違いになる条件は 1 つだけである。** `docs/operations/` に runbook 以外のファイルが増えず、かつ runbook が数十件に育った場合、`operations/` という名前は中身を言わないまま肥大する。そのときは runbooks/ を作り直すことになり、いま畳んだ手間が無駄になる。**それでも畳むのは、無駄になるのが将来の 1 回の移動であるのに対し、畳まないことの代償は今日から毎回の 1 階層だからである。**

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  `docs/operations/runbooks/` の 3 ファイルを `docs/operations/` 直下へ移し、子が 1 つしかない階層を畳んだ。DOCUMENTATION_GUIDE §4 の構成図と §11.4、SPECIFICATION_FORMAT.md §1 の構成図を追随させ、§11.4 には「runbook のためだけのサブディレクトリを最初から作らない。件数が増えてから与える。索引が要るなら `operations/README.md`」という判断基準を書いた。`AGENTS.md`、`README.md`、`docs/README.md`、`infra/backup/` のスクリプトと README、進行中の wi-290 と wi-400 の参照を張り替えた。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - 手動: Markdown の相対リンク検査 - 561 ファイルを検査し、残る 17 件はすべて完了済み work item 内の歴史的な参照（`file://` の絶対リンク 11 件、削除済みの `decisions/ADR-*` と `ARCHITECTURE.md` へのパス 6 件）であることを確認した。
  - 手動: `infra/backup/*.sh` が表示するパスが実在することを確認 - passed

## Left Undone

- **リンク検査が仕組みになっていない。** 本 work item で判明したとおり、私が wi-404 と wi-405 で「全件確認した」と記録した検査は `fd` の呼び出しを誤っており、**0 ファイルしか見ていなかった**。両方の Completion にその訂正を追記した。作り直した検査はコードフェンス内の例を除外し、561 ファイルを実際に読む。**ただしこれは使い捨てのスクリプトであり、CI にも `mise` タスクにも入っていない。** 同じ誤りは次も起きうる。`check-links` として仕組みに入れるかどうかは別の work item が判断する。
- **完了済み work item に残る 17 件の壊れたリンク。** `file://` の絶対リンクと、削除済みの `decisions/ADR-*` や `ARCHITECTURE.md` を指すパスである。いずれも本 work item より前から壊れており、当時の記録なので直さない（wi-405 の Out of Scope と同じ理由）。
