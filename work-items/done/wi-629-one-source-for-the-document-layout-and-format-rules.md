---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-19
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "文書体系の執筆規約とリポジトリ検査だけを変更し、利用者が観測するプロダクトの変更はない。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/development/specification-first-workflow.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/design/application/design-guidelines.md
  typespec: []
  source:
    - mise.toml
    - tools/check/README.md
    - tools/workspace/src/document-layout.ts
    - tools/check/src/registry.ts
    - tools/check/src/runner.ts
  tests: [tools/check/src/repository-checks.acceptance.test.ts, tools/check/src/registry.test.ts]
  stop_before_reading: [backend, frontend, spec, infra, docs/domain]
spec_impact:
  kind: none
  reason: "規約文書の重複を落として正本を一つにするだけで、規範要素の意味も TypeSpec の契約も変えない。"
---

# 文書配置と文書フォーマットの正本を一つにし、実態との乖離を消す

## Motivation

`DOCUMENTATION_GUIDE.md`（935 行）、`SPECIFICATION_FORMAT.md`（409 行）、`tools/workspace/src/document-layout.ts` の三つが、同じ `docs/` の配置を別々に記述している。
`DOCUMENTATION_GUIDE.md` 自身が冒頭で「同じ定義、同じ一覧を二か所に置かない」と定めているのに、この規則が規約文書自身には適用されていない。

三つのうち実態と一致するのは `document-layout.ts` だけである。
人が読む二つは、次の点で実在しない配置を描いている。

| 記述 | `DOCUMENTATION_GUIDE.md` | `SPECIFICATION_FORMAT.md` | 実態 |
| --- | --- | --- | --- |
| Bounded Context の置き場所 | `docs/contexts/<context>/` | `contexts/<context>/` | `docs/domain/<context>/` |
| プロダクト概要 | `docs/product-overview.md` | `docs/product-overview.md` | `docs/design/product-overview.md` |
| 共通語彙、全体標準、構造、横断シナリオ | `docs/` 直下 | `docs/` 直下 | `docs/domain/` 直下 |
| ルート直下の変更履歴 | `CHANGELOG.md` を配置図に記載 | 記載しない | 存在しない |

`SPECIFICATION_FORMAT.md` は §1 の本文で「prose in `docs/domain/oauth2/`」と正しく書いた 20 行後の配置図で `contexts/<context>/` と描いており、一つの節の中で食い違っている。

フォーマットの規定も二重である。
`DOCUMENTATION_GUIDE.md` §3（122 行目から 343 行目）は、`README.md`、`glossary.md`、`standards.md`、`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md` の形式と、標準仕様表の `Adoption` と `Strength`、Gherkin の `REQ`/`EX` の形を定める。
同じ対象を `SPECIFICATION_FORMAT.md` §3 から §6 が定める。
`DOCUMENTATION_GUIDE.md` §5 と `SPECIFICATION_FORMAT.md` §2 も、どちらも TypeSpec の範囲を定める。

work item の形式は重複しているうえに古い。
`DOCUMENTATION_GUIDE.md` §6.2 のテンプレートには `evidence_policy`、`documentation_impact`、`primary_use_cases`、`maturity_evidence`、`reversibility` が無く、`affected_spec` を `{path, requirement}` の並びではなく文字列の配列として示す。
このテンプレートの通りに起票した記録は `mise run check-work-items` に落ちる。
規約文書が検査に落ちる書き方を教えている状態である。

この乖離はどの検査も見ていない。
`check-agent-guidance` が読むのは `.agents/skills/` の 3 ファイルだけで、ルート直下の規約文書は対象外である。

## Scope

- `DOCUMENTATION_GUIDE.md` から、`SPECIFICATION_FORMAT.md` と `WORK_ITEM_FORMAT.md` が定めるフォーマットの再掲を落とし、参照へ置き換える。対象は §3、§5、§6.2 である。
- `docs/` の配置図の正本を `SPECIFICATION_FORMAT.md` §1 に一本化し、`DOCUMENTATION_GUIDE.md` からは配置図を落として参照へ置き換える。
- 残した配置図を実態へ合わせる。上表の 4 点を直す。
- 配置図と `tools/workspace/src/document-layout.ts` の食い違いを検査で止める。`SYSTEM_DOCUMENT_DIRECTORIES` と `CONTEXT_DOCUMENTS` が定義する名前が、配置図の該当行として現れることを確かめる。
- リリース文書のファイル名の規約を一つに決める。`WORK_ITEM_FORMAT.md` は `docs/releases/changes/wi-<id>.md` と書くが、`<id>` が連番だけか、work item の識別子である stem 全体かを定めていない。`documentation-impact.ts` は接頭辞 `docs/releases/changes/wi-` と拡張子しか見ないため、実在する 21 件は `wi-257.md` の形が 12 件、`wi-453-userinfo-invalid-token-401.md` の形が 9 件に割れている。

## Out of Scope

- `DOCUMENTATION_GUIDE.md` を再利用可能なリポジトリへ切り出す作業。切り出し先と切り出し単位は別に決める。
- `docs/` 配下の文書の内容の書き換え。本項目が触るのはルート直下の規約文書と、配置を検査するコードだけである。
- 規約文書の英語散文を日本語へ書き換える作業。[[wi-631-write-the-human-readable-documents-in-japanese]] が扱う。本項目は節を落として参照へ置き換えるところまでを行い、残った文の言語には触れない。
- 既存 21 件のリリース文書のファイル名の改名。決めた規約は以降のリリース文書に適用し、既存の記録は書かれた時点の名前のまま残す。
- ルート直下の生成物（`CONFIGURATION.md`、`ROUTE_PRIORITY.md`）の移動。`DOCUMENTATION_GUIDE.md` は「生成物は追跡しない場所へ置く」と定めるがこの 2 件は追跡している。判断は [[wi-630-inventory-the-repository-checks-and-their-tests]] が扱う。
- `docs/` 配下の各 `README.md` の統廃合。索引としての責務は `SPECIFICATION_FORMAT.md` §1 が定めており、重複ではない。

## Design

配置の正本を `SPECIFICATION_FORMAT.md` へ置く。
`DOCUMENTATION_GUIDE.md` は文書体系の目的と分け方の理由を述べる文書、`SPECIFICATION_FORMAT.md` と `WORK_ITEM_FORMAT.md` は満たすべき形を定める文書、と役割を分ける。
読み手は「なぜこの分け方か」を前者で、「どう書くか」を後者で読む。
逆に `DOCUMENTATION_GUIDE.md` へ寄せる案は採らない。
機械検査が読む形の規定と、体系の理由の説明が一つのファイルに混ざると、935 行のどこを直せば検査が変わるのかが読めなくなる。

配置図を二つとも残して整合を検査する案も採らない。
検査が通っても、二つの図を同時に直す作業は残る。
同じ一覧を二か所に置かない、という規則をここで曲げる理由が無い。

`document-layout.ts` と配置図の整合は検査で止める。
人が読む図と機械が読む定義は、表現が違う以上どちらかに寄せられない。
片方だけが動く経路（新しい設計文書を `document-layout.ts` へ足して図を直し忘れる）が現に開いているので、ここは検査を足す側の判断をする。
検査は `tools/check/src` の既存の形に従い、規則を返す純粋関数と、ファイルを読む `check-*.ts` に分ける。
純粋関数 `verifyDocumentLayout(source: string): DocumentLayoutFinding[]` は、配置図を正規化したパス集合と、`SYSTEM_DOCUMENT_DIRECTORIES` および `CONTEXT_DOCUMENTS` から作る必須パス集合を比較する。
`DocumentLayoutFinding` は欠けたパスと診断文を持つ。
ファイル読み取りは `checkDocumentLayout(snapshot: WorkspaceSnapshot): Promise<CheckOutcome>` だけが行い、純粋関数へ本文を渡す。
時刻、乱数、永続化、通知は使わず、入力となる文書本文以外の作用はこの読み取り境界に置く。

## Plan

1. `SPECIFICATION_FORMAT.md` §1 の配置図を実態へ合わせ、`docs/domain/` 配下の構成と `docs/design/product-overview.md` を正しく描く。
2. `DOCUMENTATION_GUIDE.md` の §2 配置図、§3、§5、§6.2 を参照へ置き換える。落とす前に、参照先に無い内容（体系の理由、上位と下位の分かれ目、文書の見せ方）を特定し、それは残す。
3. リリース文書のファイル名の規約を決め、`WORK_ITEM_FORMAT.md` へ書く。work item の識別子は stem 全体である、という `tools/check/README.md` の規則に揃える案を出発点にする。
4. 配置図と `document-layout.ts` の整合検査を足し、先に現状の図で落ちることを確かめてから図を直す。
5. `docs/README.md` から三つの規約文書への案内が、新しい役割分担を表しているか確かめる。

## Tasks

- [x] T001 [Acceptance] `mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts` の「登録した文書配置検査が配置図の欠落を拒否する」で Acceptance RED を確認する。規範要件は `N/A: プロダクト仕様ではなくリポジトリ検査の変更`。
- [x] T002 [Unit] `mise run test-tools-file -- check/src/document-layout-format.test.ts` の「配置図にない定義済み文書のパスを報告する」で Unit RED を確認し、純粋関数を GREEN にする。規範要件は `N/A: プロダクト仕様ではなくリポジトリ検査の変更`。
- [x] T003 [Docs] `SPECIFICATION_FORMAT.md` §1 の配置図を実態へ合わせ、`mise run check-spec` を GREEN にする。
- [x] T004 [Docs] `DOCUMENTATION_GUIDE.md` の §2 配置図、§3、§5、§6.2 を参照へ置き換える。
- [x] T005 [Docs] リリース文書のファイル名の規約を `WORK_ITEM_FORMAT.md` へ書く。
- [x] T006 [Verify] `mise run test-tools-file -- check/src/document-layout-format.test.ts`、`mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts`、`mise run check-spec`、`mise run check-links`、`mise run check-work-items`、`mise run verify` で変更を検証する。

## Verification

- `mise run check-work-items`
- `mise run check-links`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

`DOCUMENTATION_GUIDE.md` から節を落とすときに、参照先に無い規定を一緒に落とす恐れがある。
落とす前に節ごとの内容を参照先と突き合わせ、参照先に無いものは残す。
突き合わせの結果を完了時に記録する。

リリース文書のファイル名の規約を決めると、既存 21 件のうち片方の形が規約から外れる。
既存の記録は書かれた時点の名前のまま残すので `documentation-impact.ts` の検査は通り続けるが、完了済みの work item が参照する経路が規約と食い違って見える。
規約を書く場所に、既存の記録は改名しないことと、その理由を併記する。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は `main` に対する規範仕様の変更がないことを報告した。
  文書配置と形式の一次情報を `SPECIFICATION_FORMAT.md` と `WORK_ITEM_FORMAT.md` へ集約し、`DOCUMENTATION_GUIDE.md` は体系の理由と参照だけを持つ形にした。
  `SPECIFICATION_FORMAT.md` の配置図と `tools/workspace/src/document-layout.ts` の不一致は、新しい登録済み検査が拒否する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts --test-name-pattern '登録した文書配置検査が配置図の欠落を拒否する'` と `mise run check-spec`
  - **Requirement**: N/A: プロダクト仕様ではなく、規約文書とリポジトリ検査の変更である。
  - **Observed Failure**: runner は未登録の `document-layout` selector を拒否し、接続後の `check-spec` は現行図にない `docs/domain/<context>/standards.md`、`docs/design/product-overview.md` など 56 パスを報告した。
  - **Detection Reason**: runner 境界のテストは検査の未登録を、実リポジトリの `check-spec` は人向け配置図と機械定義の食い違いを観測するため、検査または配置図のどちらかが欠けた実装を通さない。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/document-layout-format.test.ts`
  - **Requirement**: N/A: プロダクト仕様ではなく、規約文書とリポジトリ検査の変更である。
  - **Observed Failure**: `document-layout-format.ts` が存在せず、Bun が `Cannot find module './document-layout-format.ts'` を報告した。
  - **Detection Reason**: 純粋関数が無い状態では、配置図に欠けた system 文書と Context 文書をそれぞれ診断する表明を実行できない。
- **Change-Resistance Results**:
  `N/A: low-risk の文書・tooling 変更であり、risk-based-v3 が求める Acceptance RED と Unit RED を実施した。Go の変更はないため mutation testing は適用しない。`
- **Content Comparison Results**:
  `DOCUMENTATION_GUIDE.md` から落とした形式のうち、参照先に無かった glossary の所有範囲、Context 内の分割基準、一次情報文書へ置かない内容、TypeSpec 外の所有者表は `SPECIFICATION_FORMAT.md` へ移した。
  文書体系の理由、上位と下位の分かれ目、一次情報源の割り当て、文書の見せ方は `DOCUMENTATION_GUIDE.md` に残した。
- **Verification Results**:
  - `mise run test-tools-file -- check/src/document-layout-format.test.ts` - passed
  - `mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts` - passed (20 tests)
  - `mise run test-tools` - passed (572 tests)
  - `mise run check-spec` - passed
  - `mise run check-links` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed (tools, Go race, UI unit, lint, typecheck, build, and repository checks)
  - `mise run test-ui-e2e` - N/A: ブラウザーへ到達するプロダクト変更ではない。
