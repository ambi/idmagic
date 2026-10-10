---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 仕様ファイルの置き場所、ID の文法の表記、ツールの識別子、コメントの参照先を改める変更であり、製品の利用者と運用者が観測する振る舞い、設定、API は変わらない。
  references: []
initial_context:
  specification: []
  typespec: [spec/main.tsp]
  source: [tools/workspace/src/document-layout.ts, tools/render-docs/src/render.ts, tools/check/relocated-spec-paths.json]
  tests: []
  stop_before_reading: [frontend/src]
spec_impact:
  kind: none
  reason: "TypeSpec のファイルの置き場所、文書の ID の文法の表記、検査ツールの識別子、コードのコメントの参照先を改めるだけで、TypeSpec の名前空間とタグ、生成する OpenAPI、REQ-* と EX-* と標準 id の値、外部から観測できる振る舞いは変えない。"
---

# 設計と仕様の単位の名前を「context」から「module」へそろえ、spec/contexts を spec/modules へ移す

## 動機

設計と仕様の単位は「モジュール」に改めた（文書は `docs/modules/`、用語集の Module、`check-terminology` は単独の「コンテキスト」を拒否する）。
しかし、TypeSpec の置き場所、ID の文法の表記、検査ツールの識別子には、まだ「context」が残っている。

| 箇所 | 現状 | 問題 |
| --- | --- | --- |
| TypeSpec の配置 | `spec/contexts/<context>/{models.tsp,main.tsp}`。`spec/main.tsp` が `./contexts/...` を import する | 文書の `docs/modules/<module>/` と、同じ単位を別の名前で呼ぶ |
| ID の文法の表記 | `REQ-<CONTEXT>-NNN`、`EX-<CONTEXT>-<REQ-NNN>-<sequence>`、配置の `<context>` | 仕様フォーマット、仕様先行の開発ワークフロー、作業項目フォーマット、skills が、廃した語で ID と配置を説明する（`docs/` と skills の 10 文書、37 か所） |
| 検査ツールの識別子 | `CONTEXT_DOCUMENTS`、`contextAliases`、`contextTags`、`contextDocument` など（`tools/` の約 270 か所。`feature-slice-debt.json` のキー `contextAliases` を含む） | 文書の区分と検査の語彙が一致せず、読み手がツールと文書の対応を名前からたどれない |
| コードのコメントの参照先 | `backend/` と `frontend/` の 72 ファイル、140 か所が `spec/contexts/<module>.yaml` を指す | その YAML の仕様ファイルは TypeSpec と Markdown の仕様へ移した後に削除されており、参照先が存在しない |
| TypeSpec のファイルへの参照 | `backend/`、`tools/`、`docs/` の約 20 ファイルが `spec/contexts/<module>/` のパスを書く | 移動に合わせて張り替える必要がある |

## 対象範囲

- `spec/contexts/` を `spec/modules/` へ移し、`spec/main.tsp` の import と、パスを読む検査と描画器を合わせる。`spec/tspconfig.yaml` と `mise.toml` はこのパスを参照していない。
- ID の文法と配置の表記を、`REQ-<MODULE>-NNN`、`EX-<MODULE>-<REQ-NNN>-<sequence>`、`<module>` に改める。
- 検査ツールの「context」を名前に含む識別子、関数名、JSON のキーを「module」へ改める。
- コードのコメントにある `spec/contexts/<module>.yaml` への参照を、現存する仕様の文書へ張り替える。
- 未完了の作業項目のパスを新しいパスへ直す。完了済みの作業項目の `affected_spec` は、`tools/check/relocated-spec-paths.json` の接頭辞の対応（`spec/contexts/` → `spec/modules/`）で解決する。

## 対象外

- REQ-*、EX-*、標準 id の値。ID は取り消せないので、`REQ-IDMANAGEMENT-001` のような既存の ID の接頭辞は変えない。変えるのは文法を説明する表記だけである。
- TypeSpec の名前空間（`IdMagic.Tenancy` など）と OpenAPI のタグ。どちらも「context」を含まず、変えると公開契約が変わる。
- 文書のモジュール名（`identity-management`）とコードのモジュール名（`idmanagement`）の違い。`feature-slice-debt.json` の別名の対応で扱っており、名前をそろえるかは別の判断である。
- Go の `context.Context` と、認証コンテキストのように設計の単位ではない「context」。
- 完了済みの作業項目とリリース文書の本文。リンク検査を通すためのリンク先の張り替えだけを行う。

## 設計

### 新しい名前

`spec/modules/` とする。
文書の `docs/modules/` と同じ名前にすると、責務表の一行、`docs/modules/<module>/`、`spec/modules/<module>/`、`backend/<module>/` が同じ語で対応する。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| `spec/` の直下にモジュールのディレクトリを並べる（`spec/tenancy/`） | `spec/generated/` と `spec/main.tsp` と同じ段にモジュールが混ざり、モジュールでないディレクトリとの区別をパスから読めなくなる |
| `spec/contexts/` のまま、文書の説明だけを直す | 同じ単位を二つの名前で呼ぶ状態が残る。この作業項目が解こうとしている問題そのものである |

### 存在しない YAML への参照の張り替え

コメントの `spec/contexts/<module>.yaml` は、`docs/modules/<module>/` へ張り替える。
コメントは要件のタイトルや節の名前を一緒に引用していることが多く、モジュールの文書へ向ければ、その名前で現在の要件を探せる。
一件ずつ現在の `REQ-*` を特定して書き換える案は採らない。140 か所の判断が要り、コメントの参照先の精度のために払う費用が、得られる精度に見合わない。
引用している要件が現在の仕様にない場合は、そのコメントの記述が古い可能性があるが、この作業項目では直さず、リスクとして記録する。

### 検査ツールの識別子

識別子だけを改め、振る舞いは変えない。
`feature-slice-debt.json` のキー `contextAliases` は `moduleAliases` に改め、読む側と同じ変更で直す。
ツールのテストは識別子の改名の前後で同じ結果になることを確かめる。

### 互換性

TypeSpec の名前空間、タグ、モデル名は変わらないので、生成する OpenAPI は変わらない。
`mise run check-api-compat` と `mise run check-generated-contract` が、移動の前後で差分がないことを確かめる。

## 計画

1. `spec/contexts/` を `git mv` で `spec/modules/` へ移し、`spec/main.tsp` の import、パスを読む検査と描画器、TypeSpec のパスを書く文書とコードを同じコミットで直す。
2. 完了済みの作業項目のために、`relocated-spec-paths.json` へ接頭辞の対応を足す。`spec-diff` が履歴の `spec/contexts/` を新しいパスとして読むように、履歴のパスの読み替えも足す。
3. ID の文法と配置の表記を改める。
4. 検査ツールの識別子を改める。
5. コードのコメントの YAML への参照を張り替える。

未解決の問いはない。

## タスク

- [x] T001 [Tooling] `spec/contexts/` を `spec/modules/` へ移し、import、検査、描画器、パスの参照を直す。生成する OpenAPI に差分がないことを確かめる。
- [x] T002 [Tooling] 完了記録の解決と `spec-diff` の履歴の読み替えに `spec/contexts/` → `spec/modules/` を足す。
- [x] T003 [Docs] ID の文法と配置の表記を `<MODULE>` と `<module>` に改める。
- [x] T004 [Tooling] 検査ツールの識別子と `feature-slice-debt.json` のキーを改める。
- [x] T005 [Docs] コードのコメントの存在しない YAML への参照を、モジュールの文書へ張り替える。
- [x] T006 [Verify] 検証する。

## 検証

- `mise run test-tools`
- `mise run check-spec`
- `mise run check-api-compat`
- `mise run check-generated-contract`
- `mise run check-links`
- `mise run check-work-items`
- `mise run verify`
- `rg` で `spec/contexts`、`<CONTEXT>`、`<context>` が、完了済みの作業項目とリリース文書の本文の外に残っていないことを確かめる。

## リスク

| リスク | 緩和策 |
| --- | --- |
| TypeSpec の import の張り替え漏れで、コンパイルや生成が壊れる | `mise run check-spec` がコンパイルし、`check-api-compat` が OpenAPI の差分を検出する |
| 完了済みの作業項目の `affected_spec` が解決できなくなる | 接頭辞の対応を足し、`mise run check-work-items` で全記録の解決を確かめる |
| 識別子の一括置換が、Go の `context` や設計の単位でない「context」まで変える | 置換の対象を `tools/` の TypeScript の識別子と、決めた語の組み合わせに限り、差分を読む |
| 張り替えたコメントが引用する要件が、現在の仕様にない | 張り替えは参照先の存在だけを保証する。引用の内容の正しさは、コメントのある機能を次に変える作業項目で確かめる |

## 完了
- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff -- main` は「no normative specification change against main」を報告した。TypeSpec の名前空間、タグ、モデル名は変えていない。`spec/generated/openapi/idmagic.openapi.json` は、移動の前後で同じハッシュ（`9332aa4a…`）になった。
  `spec/contexts/` を `spec/modules/` へ移し、`spec/main.tsp` の import、検査、描画器、文書とコードのパスの参照を合わせた。ID の文法と配置の表記を `REQ-<MODULE>-NNN`、`EX-<MODULE>-…`、`<module>` に改めた。検査ツールでは、「context」を名前に含む複合語の識別子と `feature-slice-debt.json` のキー（`moduleAliases`）を改めた。単独の `context` は、モジュールの意味で使っているファイルに限って `module` に改めた。Go の `context.Context`、Gherkin の Context 手順、作業項目の `initial_context` を指すものは残した。
  コードのコメント 140 か所（72 ファイル）にあった、存在しない `spec/contexts/<module>.yaml` への参照は、`docs/modules/<module>/` へ張り替えた。spec-change の skill にあった、存在しない `tools/check/legacy-spec-layout.json` を指す一文は削除した。
  完了済みの記録とリリース文書は、本文を書き換えず、移したファイルへのリンク先だけを張り替えた。完了記録の `affected_spec` は、対応表の接頭辞の対応（`spec/contexts/` → `spec/modules/`）で解決する。`spec-diff` は、履歴の `spec/contexts/` の TypeSpec の宣言を、現在のパスで同定する。これらの移行用の対応は、この作業項目の比較の基準（main）が旧い配置であるあいだだけ必要であり、統合後に別の作業項目で削除する。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-links`。
  - **要件**: N/A: 仕様ファイルの配置と名前の変更であり、製品の要件を変えない。
  - **観測した失敗**: `spec/contexts/` を移した時点で、リリース文書 9 件のリンクが存在しない `spec/contexts/<module>/*.tsp` を指して失敗した。
  - **検出できる理由**: リンク検査は、移したファイルを指すリンクをすべて失敗として報告する。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/spec-diff.test.ts` の「reports nothing when TypeSpec moves from spec/contexts to spec/modules」。
  - **要件**: N/A: 検査ツールの変更であり、製品の要件を変えない。
  - **観測した失敗**: 履歴のパスの読み替えを足す前は、同じ宣言を削除と追加の両方として報告して失敗した。
  - **検出できる理由**: 宣言をパスごとに同定するので、読み替えがなければ移動を宣言の変更と区別できない。
- **変更耐性の結果**:
  エスケープしたパスの一括置換が、`spec-diff` の履歴の読み替え（`spec\/contexts`）まで `spec\/modules` に書き換えた状態では、上のテストが失敗した。読み替えを戻すと通った。識別子の改名では、返す項目名の変更を `specification-rules.test.ts` が検出した。
- **検証結果**:
  - `mise run test-tools`、`mise run lint-tools` - 成功
  - `mise run check-spec`、`mise run check-api-compat`、`mise run check-links`、`mise run check-work-items` - 成功
  - `rg` による走査 - `spec/contexts`、`<CONTEXT>`、`<context>` は、完了済みの記録、リリース文書の本文、この作業項目、履歴を読むための対応表と `spec-diff` のテストの外に残っていない
  - `mise run verify` - 成功（138 秒）
