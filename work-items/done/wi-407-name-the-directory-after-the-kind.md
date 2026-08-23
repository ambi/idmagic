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
spec_impact: { kind: none, reason: "運用文書のディレクトリ名と、ガイド §11 の章立てを直す変更である。正規文書の集合にも、規範的な要素（REQ-* のシナリオ、規範 ID、TypeSpec symbol）にも触れない。" }
---

# ディレクトリを読み手ではなく種類で名付ける — `operations/` をやめて `runbooks/` にする

## Motivation

[[wi-406-operations-holds-only-runbooks]] は `docs/operations/runbooks/` を `docs/operations/` へ畳んだ。**畳む判断は正しかったが、残す名前を間違えた。**

この文書体系は「**ファイル名が内容の種類を表す**」で貫かれている（DOCUMENTATION_GUIDE §4 冒頭）。`scenarios.md`、`decisions.md`、`states.md`、`standards.md`——すべて種類である。

`runbook` も種類である。決まった形（発火条件、最初に確認すること、緩和、確認、エスカレーション）と、固有の入口（**原因が分からないまま呼び出される**）を持つ。

**`operations` は種類ではなく読み手である。** ガイド §11 が `operations/` に置くとした三つ——SLO 定義、リリースと後退の手順、バックアップ方針——は互いに無関係な種類であり、「同じ役割の人が読む」という一点だけで束ねられている。この体系で読み手を軸に分けているのは、wi-405 が §11 へ書き足した一節だけである。**異物はそちらだった。**

現物を見ると、束ねる対象すら無い。

| ガイド §11 の文書 | このリポジトリでの実際 |
|---|---|
| `reliability.md`（SLO） | `docs/capacity.md` が既に持つ。[[wi-400-service-objectives-need-stable-ids]] の Design 1 は「増やさず `capacity.md` に ID を足す側に寄せるのが素直」と書く |
| `release-and-rollback.md` | wi-400 の Out of Scope。「まだリリースを行っていない」ため無期限に延期されている |
| `backup-and-recovery.md` | `backup-restore-dr.md` として既に実在し、**それ自身が runbook である**（冒頭が「この手順書では」） |

三つとも、別の運用文書としては来ない。`operations/` は**読み手で名付けたうえに、束ねる中身が無い**ディレクトリだった。

## Scope

- `docs/operations/` を `docs/runbooks/` に改名する。
- DOCUMENTATION_GUIDE §4 の構成図と §11 を改める。§11 は「運用文書」という**章としては残し、ディレクトリとしては解体する。** 三つの種類それぞれの行き先を書く。
- §11 から、wi-405 が書き足した「読み手で分ける規則が、対象で分ける規則に優先する」を撤回し、**読み手による分類は章立てで行い、ファイルシステムは種類で分ける**に置き換える。
- SPECIFICATION_FORMAT.md §1 の構成図を追随させる。
- `AGENTS.md`、`README.md`、`docs/README.md`、`infra/backup/`、進行中の wi-290 と wi-400 の参照を張り替える。

## Out of Scope

- `docs/reliability.md` を `ROOT_DOCUMENTS` へ足すこと。**SLO をどう置くかは wi-400 が決める未決の論点であり、来ると決まっていないものの席をいま作るのは、本 work item が正そうとしている誤りそのものである。** ガイドには「SLO は正規文書として `docs/` 直下に置く」という一般的な規則だけを書き、集合は触らない。
- `docs/development/` の新設。リリース手順が実在しないので、置くものが無い。
- 完了済み work item の散文に残るパス。当時の記録である。
- リンク検査を仕組みにすること（wi-406 の Left Undone）。

## Design

**読み手による分類は章立てで行い、ファイルシステムは種類で分ける。**

これがガイド §11 の解体後に残る規則である。当番担当者が読むものをひとまとめに示したいという要求は本物だが、それに応えるのはガイドの章であって、ディレクトリではない。章は複数の場所を指せるが、ディレクトリは 1 か所にしか置けない。**読み手でディレクトリを切ると、同じ種類の文書が読み手ごとに分かれてしまう。** SLO 定義は運用者も実装者も読むし、`capacity.md` の隣にあるべき現在状態の記述でもある。

三つの種類の行き先は、§5.9 の判定（それが変わったとき、外部から観測できる振る舞いか、守るべき境界が変わるか）で決まる。

| 内容 | 判定 | 行き先 |
|---|---|---|
| SLO の目標値、測定条件 | 変わる。守ると約束した水準である | 正規文書（`docs/` 直下） |
| バックアップの RPO / RTO、復旧の順序 | 変わる。失ってよい範囲の宣言である | 正規文書（`docs/` 直下） |
| リリースと後退の手順 | 変わらない。やり方の話である | 手順（`docs/development/`） |
| 障害時の手順 | 変わらない | runbook（`docs/runbooks/`） |

`backup-and-recovery` が 2 行に分かれるのは分割ではなく、**もともと別の種類が 1 つのファイル名に同居していた**ということである。目標値は宣言であり、訓練の手順は runbook である。

### 却下した案

**`docs/operations/` のまま維持する（wi-406 の結論）。** 名前が読み手を指す一方で中身は 1 種類しかなく、増える予定も無い。ガイドとの一致を理由にしたが、**ガイドのほうが体系の原則（名前が種類を表す）から外れていた。** 一致させる先が間違っていた。

**`docs/operations/runbooks/` へ戻す。** `operations/` に他の種類が来るなら正しい形だが、上表のとおり来ない。来ないものの容れ物を維持する費用は、参照するたびの 1 階層である。

## Tasks

- [x] T001 [Docs] `docs/operations/` を `docs/runbooks/` へ改名する。
- [x] T002 [Docs] DOCUMENTATION_GUIDE §4 と §11 を改める。§11 を章として残し、ディレクトリとしては解体する。
- [x] T003 [Docs] SPECIFICATION_FORMAT.md §1 を追随させる。
- [x] T004 [Docs] `AGENTS.md`、`README.md`、`docs/README.md`、`infra/backup/`、wi-290、wi-400 の参照を張り替える。
- [x] T005 [Verify] `mise run verify` とリンク検査を通す。

## Verification

- `mise run verify`
- 手動: Markdown の相対リンク検査（wi-406 で作り直したもの）を全 561 ファイルに対して実行する。
- 手動: ガイド内の `§N.M` 参照が実在する節を指すことを確認する。**§11 の小節を組み替えるので、ここが壊れやすい。**
- 手動: 直した §4 と §11 だけを読んで、SLO 定義・リリース手順・障害手順の 3 つをそれぞれどこへ置くか判断できるかを確かめる。

## Risk Notes

リスクは low。3 ファイルの改名と、ガイドの章立ての組み替えである。

**このディレクトリについて、私は 1 セッションで 3 回立場を変えた**（`operations/runbooks/` → `operations/` → `runbooks/`）。ファイルが 3 つの今なら改名は数分だが、runbook が 20 件になり、外部から `docs/runbooks/...` を参照する資材が増えてからでは同じ判断が高くつく。**いま確定させることに価値がある。**

確定の根拠を「件数が少ないから」に置かないこと。件数は変わる。根拠は「名前が種類を表す」という体系の原則であり、これは runbook が何件になっても変わらない。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  `docs/operations/` を `docs/runbooks/` に改名し、ガイド §11 を「読み手でまとめた案内の章」として残しつつ、ディレクトリとしては解体した。§11 冒頭に、読み手による分類は章立てで行いファイルシステムは種類で分けるという規則と、その理由（章は複数の場所を指せるがディレクトリは一か所にしか置けない／読み手で切ると同じ種類が読み手ごとに分かれる）を書いた。§5.9 の判定をそのまま適用した行き先の表を置き、11.1〜11.4 の各小節に**配置**を明示した——SLO は正規文書、リリースと後退は `docs/development/`、バックアップは目標値が正規文書で手順が runbook、障害時の手順は `docs/runbooks/<event>.md`。wi-405 が §11 へ書き足した「読み手で分ける規則が対象で分ける規則に優先する」は撤回した。`ROOT_DOCUMENTS` は触っていない。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run check-work-items` / `check-ids` - 406 件 OK
  - 手動: リンク検査 562 ファイル - 残る 17 件はすべて完了済み work item 内の歴史的な参照（`file://` 11 件、削除済みの `decisions/ADR-*` と `ARCHITECTURE.md` 6 件）
  - 手動: ガイド内の `§N.M` 参照が実在する節を指すことを全件確認 - passed

## Left Undone

- **`docs/development/` は依然として存在しない。** ガイド §11.2 がリリース手順の置き場所として名指ししたが、このリポジトリにリリース手順は無い。§3 に従い、置くものができるまで作らない。
- **SLO の置き場所は wi-400 の判断待ち。** ガイド §11.1 は「`capacity.md` が兼ねてよい。独立させるなら `reliability.md` として正規文書の集合へ加える」と両方を許す形にした。**`ROOT_DOCUMENTS` は意図的に触っていない。** 来ると決まっていない文書の席を先に作るのは、本 work item が正した誤りと同じ形だからである。
- **このディレクトリは 1 セッションで 3 回動いた**（`infra/runbooks/` → `docs/operations/runbooks/` → `docs/operations/` → `docs/runbooks/`）。確定の根拠は件数ではなく「名前が種類を表す」という体系の原則なので、runbook が増えても再検討は要らない。**逆に、この原則から外れる置き方を提案するときは、原則の側を先に議論すること。**
