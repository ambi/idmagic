---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-78794-rename-spec-contexts-to-modules]
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回公開前であり、変えるのはリポジトリ内の検査ツールと開発文書だけである。製品の利用者と運用者が観測する振る舞い、設定、API、既存データは変わらない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/check-work-items.ts
    - tools/check/src/work-item-references.ts
    - tools/check/src/documentation-impact.ts
    - tools/check/src/primary-use-case-evidence.ts
    - tools/check/src/check-links.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/specification-doc.ts
    - tools/check/src/feature-specification.ts
    - tools/check/src/specification-rules.ts
    - tools/check/src/agent-guidance.ts
    - tools/check/src/canonical-document-set.ts
    - tools/check/schemas/work-item.schema.json
    - tools/workspace/src/workspace.ts
    - tools/workspace/src/document-layout.ts
    - tools/render-docs/src/main.ts
    - docs/formats/work-item-format.md
    - docs/formats/specification-format.md
  tests: []
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "検査ツールから、旧い配置と旧い形式を読むための処理と、完了済みの作業項目を後から検証し直す処理を外すだけで、製品の振る舞い、TypeSpec の契約、REQ-* と標準 id は変えない。"
---

# 完了済みの作業項目を完了の時点でだけ検証し、旧い配置と旧い形式のための移行用の処理を検査ツールから外す

## 動機

検査ツールには、文書と仕様の配置や形式を移した後も、移す前の状態を読むための処理が残っている。

| 処理 | 置き場所 | 必要になっている理由 |
| --- | --- | --- |
| 旧パスから新パスへの対応表 | `tools/check/relocated-spec-paths.json` と、それを引く `check-work-items` の処理 | `check-work-items` が、完了済みの記録（約 280 件）の `affected_spec` を、毎回いまのファイルに対して解決し直す |
| 旧パスの許可 | `work-item.schema.json` の `docs/domain/` と `SPECIFICATION.md` | 同上 |
| 履歴の読み替え | `spec-diff` の旧パス（`docs/contexts`、`docs/domain`、`spec/contexts`、`docs/` 直下）、旧形式（`SPECIFICATION.md`、`scenarios.md`、`states.md`、`examples.feature.md`） | 比較の基準が、移動より前のリビジョンになる場合に備える |
| 旧形式の名前の分類 | `specification-doc.ts` の、モジュールの直下の旧い名前（`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md`） | 同上 |
| 旧形式への逆戻りの拒否 | `Rule:` の見出し、`SPECIFICATION.md` への言及、旧い欄名（「理由」「上位の規則」）、旧い文書名（`ARCHITECTURE.md` など）の検出 | 移行の直後に、旧い書き方が戻るのを防ぐ |
| 旧い出力先の掃除 | 描画器の `spec/generated/docs` の削除 | 出力先を `site/` へ移す前の生成物を消す |
| 旧い証拠の契約 | `risk-based-v3` の検証と、作業項目の番号 452 未満を除外する分岐 | 旧い契約で完了した記録を、旧い契約のまま検証し直す |

これらは移行のためだけにあり、移すたびに増える。
移すたびに対応表を足し、完了済みの記録のリンク先を張り替える作業も生じる。
いずれも、完了済みの記録を後から検証し直していることと、比較の基準が旧い配置でありうることが原因である。

## 対象範囲

- 完了済みの作業項目を、完了の時点でだけ検証する。完了の時点は、作業ツリーでその記録が基準のリビジョンから変わっていることで判定する（`documentation-impact` がすでに使っている判定）。
- リンク検査の対象から `work-items/done/` を外す。リリース文書は利用者が読むので、検査を続ける。
- 上の表の処理、対応表のファイル、スキーマの旧パスの許可を削除する。
- 作業項目フォーマットと仕様フォーマットの、旧パスの対応表と旧形式の扱いの記述を、新しい方針に書き換える。

## 対象外

- `*-debt.json`（`boundary-debt` など）。いまある負債を減らす方向にだけ動かすための記録であり、移行用ではない。
- 境界検査の `legacy` 公開方式。`internal/` へ移していないモジュールの現在の状態を表す。
- 完了済みの記録とリリース文書の本文。

## 設計

### 完了済みの記録の検証

完了済みの記録は、当時の作業の記録であり、後のファイルの移動や形式の変更に追従させる必要がない。
完了の時点の検証（`affected_spec` の解決、主要ユースケースの証拠、文書影響、リンク）は、記録が完了する変更の中で一度だけ行えば足りる。
後から検証し直すと、移動のたびに対応表と読み替えが必要になり、記録が現在のファイルへ縛られる。

判定には、作業ツリーで基準のリビジョンから変わった記録の集合を使う。
`in_progress` の記録と、この集合に含まれる `completed` の記録だけを検証する。
完了の変更では記録が `work-items/done/` へ移るので、必ずこの集合に含まれる。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| 対応表を残し、移動のたびに足し続ける | 移動のたびに移行用の処理が増え、完了済みの記録のリンクの張り替えも続く。今回の問題そのものである |
| 完了済みの記録の旧パスを、移動のたびに書き換える | 記録が当時の状態を示さなくなる。作業項目フォーマットが、完了済みの記録を書き換えないと定めている |

### 履歴の読み替えと旧形式の拒否

比較の基準は main とのマージベースであり、作業項目の開始の時点（`pending` でなくなったコミットの親）は、その作業項目のブランチの中にある。
wi-78794 の統合後は、どの基準も現在の配置と形式になるので、履歴の読み替えは不要になる。
旧形式への逆戻りを拒否する検査は、文書の形式の検査（節の構成、置ける名前）が同じ文書を拒否する場合に限って外す。
着手時に検査ごとに確かめ、形式の検査では拒否できない旧形式が残るものは、移行用ではなく現在の規則の一部として残す。

着手時に確かめた結果は次のとおりである。
`in_progress` の記録はこの作業項目のほかになく、`pending` の記録は `evidence_policy` を持たない。
比較の基準（main）は wi-78794 の統合（`e121c10f7`）より後である。

| 検査 | 形式の検査が同じ文書を拒否するか | 判断 |
| --- | --- | --- |
| 機能仕様の `Rule:` 付きの見出し（`feature-specification.ts`） | 拒否しない。宣言の正規表現に合わず、規則が黙って宣言から落ちる | 残す |
| 要件の旧い欄名「理由」「上位の規則」（`specification-rules.ts`） | 拒否しない。ただの箇条書きとして読まれる | 残す |
| エージェント向け手順の `SPECIFICATION.md` と旧い証拠の契約への言及（`agent-guidance.ts`） | 拒否しない。手順の文書は形式の検査の対象外である | 残す |
| リポジトリ全体の `ARCHITECTURE.md`、`requirements.md`、`SPECIFICATION.md`（`workspace.ts`） | `docs/` の下は文書の集合の検査が拒否するが、リポジトリの直下と `docs/` の外は対象外である | 残す |
| `work-items/` 直下の記録 | 拒否しない | 残す |
| 任意の名前の段で使えない旧い名前（`document-layout.ts` の `RESERVED_FREE_NAMES`） | この検査自体が名前の検査である | 残す |

逆戻りの拒否はどれも外さない。
外すのは、旧い配置と旧い形式を読む処理（対応表、`spec-diff` と `documentKind` の読み替え）と、完了済みの記録を再検証するための処理である。

完了済みの記録を変わったときだけ検証すると、スキーマの形式の検査も同じ扱いになる。
そうしないと、スキーマから旧パスの許可を外した時点で、旧パスを指す完了済みの記録が形式の検査で落ちる。
番号と依存の検査（`depends_on` の解決、番号の重複）は、記録の間の現在の関係を見るので、すべての記録へ続けて適用する。

旧い証拠の契約は、`risk-based-v3` だけでなく `risk-based-v1` と `risk-based-v2`、作業項目の番号で旧い記録を除外するスキーマの分岐（410、412、452）も外す。
いずれも、旧い契約で完了した記録を後から検証し直すためだけにあり、変わった完了済みの記録は現在の契約で検証する。
主要ユースケースの証拠の、`backend/<module>/` を `backend/<module>/internal/` として読み替える処理も、完了済みの記録の再検証のためだけにあるので外す。

状態遷移の表は、機能仕様の `## 状態遷移` の節として現在も使うので、`spec-diff` はその節を読み続ける。
`states.md` の種別と、`SPECIFICATION.md` の `State Transitions` の節の読み取りは外す。

## 計画

1. 着手時に、`in_progress` の記録が旧い契約（`risk-based-v3`）でなく、比較の基準が wi-78794 の統合より後であることを確かめる。旧形式への逆戻りを拒否する検査ごとに、外しても形式の検査が同じ文書を拒否するかを確かめる。
2. 完了済みの記録を変わったときだけ検証するよう、`check-work-items` と関連する検査を変える。テストを先に変え、旧いパスを指す完了済みの記録が、変わっていなければ通り、変わっていれば失敗することを確かめる。
3. リンク検査から `work-items/done/` を外す。
4. 移行用の処理、対応表、スキーマの旧パスの許可、旧形式の拒否、旧い証拠の契約を削除する。
5. 作業項目フォーマットと仕様フォーマットの記述を書き換える。

未解決の問いはない。

## タスク

- [x] T001 [Tooling] 完了済みの作業項目を、作業ツリーで変わったときだけ検証する。
  検査は `mise run test-tools-file -- check/src/check-work-items.test.ts`。
  RED は「基準から変わっていない記録は、参照先が消えていても通す」が `affected_spec path does not exist` で失敗したこと。
- [x] T002 [Tooling] リンク検査から、変わっていない `work-items/done/` の記録を外す。
  完了の時点のリンク検査を保つため、T001 と同じ変わった記録の集合を使う。
  検査は `mise run test-tools-file -- check/src/check-links.test.ts`。
  RED は「基準から変わっていない完了済みの記録のリンクは検査しない」が壊れたリンクを報告して失敗したこと。
- [x] T003 [Tooling] 移行用の処理、対応表、スキーマの旧パスの許可、`risk-based-v1` から `v3` の検証を削除する。
  旧形式への逆戻りの拒否は、設計の表のとおり現在の規則として残す。
  検査は `mise run test-tools`。
  RED は、新しいスキーマと `documentKind` のテストを変更前の本番コードで実行し、5 件が失敗したこと。
- [x] T004 [Docs] 作業項目フォーマット、仕様フォーマット、仕様先行の開発ワークフロー、テスト方針の記述を新しい方針に書き換える。
  N/A: 文書の変更であり、代わりに `mise run check-spec` と `mise run check-links` を通した。
- [x] T005 [Verify] 検証する。

## 検証

- `mise run test-tools`
- `mise run check-work-items`
- `mise run check-links`
- `mise run check-spec`
- `mise run verify`
- `rg` で、旧パス（`docs/domain`、`docs/contexts`、`spec/contexts`）と旧形式の名前が、完了済みの記録、リリース文書、`docs/formats/` の履歴の説明の外の検査ツールに残っていないことを確かめる。

## リスク

| リスク | 緩和策 |
| --- | --- |
| 完了の変更の中で検証すべき記録を、変わっていないと判定して検証を飛ばす | 完了の変更では記録が `work-items/done/` へ移ることをテストで固定する |
| 長く続く作業項目の比較の基準が、移動より前のリビジョンになる | 着手時に `in_progress` の記録を確かめる。将来に配置を移す作業項目は、その作業項目の中だけで必要な読み替えを足し、統合後に外す |
| 旧形式の文書が紛れ込む | 文書の形式の検査が、節の構成と置ける名前で拒否する |

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
  変わったのは検査ツール、作業項目のスキーマ、開発文書である。
  `check-work-items` と `check-links` は、`work-items/done/` の記録を、main とのマージベースから、または作業ツリーで変わったときだけ検証する。
  変わった記録の判定は `tools/check/src/work-item-changes.ts` にまとめ、完了の変更で新しく作った `work-items/done/` の中の記録も拾えるよう、未追跡のファイルを一つずつ列挙する。
  依存と識別番号の検査は、記録の間の現在の関係なので、すべての記録へ続けて適用する。
  旧パスの対応表（`tools/check/relocated-spec-paths.json`）とその解決、主要ユースケースの証拠の `internal/` への読み替え、`spec-diff` と `documentKind` の旧い配置と旧形式（`SPECIFICATION.md`、`scenarios.md`、`states.md`、`examples.feature.md`）の読み取り、描画器の旧い出力先の削除を外した。
  スキーマは `risk-based-v4` だけを受け付け、`risk-based-v1` から `v3` の分岐、作業項目の番号による除外、旧い SCL 参照、`docs/domain/` のパスを外した。
  完了した記録には `evidence_policy` と `documentation_impact` を求める。
  旧形式への逆戻りの拒否は、形式の検査が同じ文書を拒否しないことを確かめたので、現在の規則として残した。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/check-work-items.test.ts` の「基準から変わっていない記録は、参照先が消えていても通す」と、`tools/check/src/check-links.test.ts` の「基準から変わっていない完了済みの記録のリンクは検査しない」。
  - **要件**: N/A: 製品の振る舞いを変えない検査ツールの変更であり、規範となる製品要件はない。
  - **観測した失敗**: 変更前の検査は、Git の workspace で main にコミット済みの完了記録を検証し直し、`affected_spec path does not exist: docs/modules/demo/moved/README.md` と、壊れたリンクの所見を返して失敗した。
  - **検出できる理由**: fixture は main にコミットした完了記録を変えずに検査するので、完了済みの記録を毎回検証し直す実装と、変わったときだけ検証する実装を、所見の有無で区別できる。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/lib.test.ts` の「v4 だけを受理し、より古い契約を拒否する」「rejects an affected_spec path below the retired docs/domain tree」「requires the evidence policy and the documentation impact when an item completes」、`tools/check/src/specification-doc.test.ts` の `documentKind` の拒否。
  - **要件**: N/A: 製品要件のない検査の変更である。
  - **観測した失敗**: 変更前のスキーマと `documentKind` で実行すると、`lib.test.ts` の 3 件と `specification-doc.test.ts` の 1 件が失敗した。旧スキーマは `risk-based-v1` から `v3` と `docs/domain/` のパスを受理し、`documentKind` は `docs/domain/` を `docs/modules/` へ読み替えていた。
  - **検出できる理由**: 旧い契約、旧パス、旧い名前を受理する実装は、それぞれの表明を満たせない。
- **変更耐性の結果**:
  TypeScript は Go の変異器の対象外なので、配線と分岐の故障を手で注入し、各テストファイルを実行した。
  すべての故障をテストが検出した。

  | 注入した故障 | 検出したテスト |
  | --- | --- |
  | 変わっていない完了記録も常に検証する | `check-work-items.test.ts` の 1 件が失敗 |
  | main とのマージベースからの差分を読まない | `check-work-items.test.ts` の「main から分けたブランチで、コミット済みの変更を検証する」が失敗 |
  | `git status` の `--untracked-files=all` を外す | `check-work-items.test.ts` の「新しく作った work-items/done へ直接書いた記録を検証する」が失敗 |
  | Git の外の workspace で完了記録を検証しない | `check-work-items.test.ts` の「旧配置と status に反するディレクトリを拒否する」が失敗 |
  | リンク検査で完了記録を絞り込まない | `check-links.test.ts` の 1 件が失敗 |

  最初の注入で `--untracked-files=all` を外す故障が生き残った。
  完了の移動では、`work-items/active/` の削除が同じ識別子を示すからである。
  移動の前身を持たずに `work-items/done/` へ直接書く記録のテストを足し、検出を確かめた。
- **検証結果**:
  - `mise run test-tools` - 成功（948 件）
  - `mise run typecheck-tools` - 成功
  - `mise run lint-tools` - 成功
  - `mise run check-work-items` - 成功
  - `mise run check-links` - 成功（705 文書）
  - `mise run check-spec` - 成功
  - `mise run verify` - 成功
  - `rg` による旧パスと旧形式の名前の検索 - 検査ツールに残るのは、拒否を確かめるテストと、設計の表で残すとした逆戻りの拒否だけである
