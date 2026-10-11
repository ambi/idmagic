---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p1
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  references: []
  reason: "未公開期間の文書形式と開発用検査の変更であり、利用者の操作と契約を変えない。"
initial_context:
  specification:
    - docs/formats/design-format.md
    - docs/formats/documentation-guide.md
    - docs/formats/work-item-format.md
    - docs/development/task-context.md
    - docs/development/specification-first-workflow.md
    - docs/development/coding-style.md
    - docs/development/writing-language.md
    - docs/development/release.md
    - docs/requirements/quality.md
    - docs/design/README.md
    - docs/design/architecture/logical.md
    - docs/design/verification/README.md
    - docs/design/verification/system-acceptance.md
    - docs/modules/authentication/design/README.md
    - docs/modules/oauth2/design/README.md
    - docs/modules/claim-mapping/design/README.md
  typespec: []
  source:
    - tools/README.md
    - tools/check/src/specification-doc.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/markdown-links.ts
    - tools/workspace/src/document-layout.ts
  tests:
    - tools/check/src/feature-layout.acceptance.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading:
    - backend
    - frontend/src
    - spec/modules
spec_impact:
  kind: none
  reason: "設計の担当、委譲先、品質要件の割り当てと規約の参照方法を明確にする。HTTP 応答、認証と認可、永続状態、イベント、外向きの通知と、品質要件の目標値と測定境界は維持する。"
---

# 共通設計を既定にして設計索引を縮小し、固有設計と品質要件の担当を明示する

## 動機

設計索引の「該当なし」は、固有設計が不要な場合と、システムや機能の設計へ委譲する場合の両方を表す。
また、システムの[品質要件](../../docs/requirements/quality.md)にはログインと OAuth2 の SLO がある一方、該当モジュールの設計索引には割り当てた品質要件がないと記載される。
領域を列挙する形式検査は通っても、保証を実現する担当と参照先を AI が推測する余地がある。

全モジュールに同じ九領域の表を要求すると、小さなモジュールでも共通設計への委譲や対象外を繰り返す。
ClaimMapping は九領域のうち八領域が「該当なし」であり、固有の判断を読むために定型の行を維持する費用がある。
担当の明示と、毎回同じ欄を埋めることは分け、共通設計を既定にした索引へ縮小する。

現在は未リリースであり、当面も未リリースを継続する。
公開後の互換性やリリース文書の準備より、現在の設計を誤読せず変更できることを優先して p1 とする。
未リリースであることを理由に、既存の品質目標や仕様を弱めない。

## 対象範囲

- 設計索引で、固有設計あり、共通設計へ委譲、対象外を区別する形式と検査。
- 全モジュールで九領域を列挙する義務を外し、固有設計、既定からの例外、担当する品質要件だけを記述する形式。
- 委譲先の所有文書と該当節への参照、対象外の理由の明示。
- 宣言済みの品質要件を、担当する機能、設計、検証へ割り当てる参照。
- 目標設計、実装状況、検証済みの事実を混同させる記述の整理。

## 対象外

- 品質目標、SLO の母集団、測定境界、製品の規範要件の変更。
- 内容のない領域別文書の追加、全機能への独立した品質文書の強制。
- 別の責務台帳、品質目標の数値の複製。
- 全文書の一括改名と、読みやすさに関係しない表記だけの変更。
- 開発規約の重複と背景説明の分離は[規約の整理](wi-12973-consolidate-daily-development-rules-and-separate-explanations.md)が扱う。

## 設計

[設計フォーマット](../../docs/formats/design-format.md)と[文書ガイド](../../docs/formats/documentation-guide.md)を先に更新し、索引の構造と検査を同じ契約にする。
委譲は担当が存在すること、対象外はその要素では扱わないことを表し、どちらも空白の代用にしない。
品質目標はシステムの一次情報から ID で参照し、モジュールは自分の担当と例外だけを書く。
参照の存在は検証済みを意味しないので、実装状況や実測の証拠は別に表示する。
共通設計はシステム側で適用範囲を定め、各モジュールに同じ委譲と理由の写しを要求しない。
全領域を見渡す必要がある場合は、共通設計と固有の記述から導く表示にする。
未記入を対象外と推定せず、宣言済みの品質要件の担当と例外の欠落は引き続き検出する。

索引は `設計領域 | 区分 | 参照先 | 内容` の四列にする。
区分は「固有設計」「委譲」「対象外」の三つとし、固有設計と委譲は担当文書の節へのリンク、対象外は参照先 `—` と理由を必要とする。
システム索引は九領域を維持し、モジュールは共通設計からの差分だけを同じ順序で書く。
共通設計だけで完結するモジュールは表を省略できる。
対象外はそのモジュールで実現対象がない場合に限り、システム全体の保証を外す意味にはしない。

品質要件の割り当ては論理アーキテクチャに一度だけ置く。
`品質要件 | 担当機能 | 設計 | 検証` の表で宣言済みの `SLO-*` と `CAP-*` をすべて参照し、複数の機能が担当する母集団は同じ行で示す。
数値を持たない品質特性には既存の共通設計と検証の参照を適用し、新しい ID や独立文書を作らない。
JWKS の HTTP 契約は SigningKeys の境界宣言どおり OAuth2 が担当する。
可用性の監視未実装と実測未実施は既存の監視設計と検証設計へ参照し、割り当ての存在を達成の証拠にはしない。

検査は既存の `validateDocument(path, source): SpecificationValidation` で索引の各行を判定し、`checkDocuments(snapshot, options): Promise<CheckOutcome>` で文書間のリンクと品質要件の網羅性を確かめる。
索引の移行で、Markdown の有効なアンカーと生成サイトの ID の句読点処理が異なることを発見した。
製品の振る舞いを追加する問題ではなく、一次情報への参照を派生表示でも解決する実装上の不整合（分類 c）として、この項目で修正する。
`markdownHeadingTargets(source): ReadonlyMap<string, string>` で、既存のアンカー規則から元の見出しへの対応を導き、生成サイトが同じ節へ解決する。
依存と作用の境界は変えず、既存の Markdown 検査モジュールから生成側への参照を使う。
ファイルの読み取りは既存の snapshot 境界に保ち、判定は文字列と文書 Map を入力とする。
`verifyDesignDocumentCoverage(documents)` は、実在する固有の領域文書を索引から参照しているかを照合する。
共通設計への継承を認めることと、既存の固有設計の書き漏らしを認めることは分ける。
製品の受け入れ境界と単体境界は該当しないため、代替 RED は `mise run test-tools-file -- check/src/feature-layout.acceptance.test.ts` と純粋な形式検査のテストで観測する。
製品要件 ID は N/A とし、形式文書の規則を根拠にする。
共通設計へのフォールバック、委譲先と節の欠落、区分の矛盾、対象外の理由の欠落、品質要件の担当欠落を故障として選ぶ。

| 案 | 判断 |
| --- | --- |
| 共通設計を既定とし、固有設計と例外、担当する品質要件だけを記述する | 採用候補。定型行の保守を減らし、必要な担当を追跡できる |
| 全モジュールの九行へ区分の欄を追加する | 担当の区別は改善するが、定型行の維持費用が残るため採らない |
| すべての領域へ独立した文書を作る | 共通設計の複製と空の文書を増やすため採らない |
| 「該当なし」の散文だけを書き直す | 状態と担当を機械的に区別できないため採らない |

## 計画

1. 設計索引の委譲と対象外、宣言済み品質要件の担当を棚卸しする。
2. 形式、所有先の参照、割り当ての検査を決める。
3. 形式文書、検査、代表モジュールを更新し、残る索引へ展開する。
4. 定型行の削減と担当への到達を比較し、既存の目標値と測定境界が変わっていないことを確かめる。

## タスク

- [x] T001 [Inventory] 委譲、対象外、品質要件の担当の不明点を列挙する。Authentication と OAuth2 の「品質要件はない」は既存 SLO と矛盾し、委譲を対象外と記した行が全モジュールにある。`mise run setup` と `mise run lint-go` は成功。
- [x] T002 [Design] 索引の区分、委譲先、品質要件の割り当てを決める。四列の索引と一か所の品質要件割り当てを採用し、製品要件 ID と製品テストは N/A。
- [x] T003 [Docs] 形式文書と Authentication、OAuth2 の索引を更新する。四列の区分と論理アーキテクチャの割り当て表を採用し、独立した品質文書と数値の複製は追加しない。`mise run check-spec` と `mise run check-links` が成功。
- [x] T004 [Tooling] 委譲先の欠落と対象外の理由の欠落を検査する。`mise run test-tools-file -- check/src/specification-doc.test.ts` で初期 10 失敗（旧ヘッダーしか受け入れない）から GREEN にした。`feature-layout.acceptance.test.ts` の「委譲先の存在しない節を文書検査で拒否する」「宣言済みの品質要件から担当への参照がない場合を拒否する」は初期に終了コード 0 を返して失敗し、配線後に成功した。製品要件 ID は N/A。生成表示の句読点アンカーも `render-docs/src/render.test.ts` で RED と GREEN を確認した。
- [x] T005 [Docs] 21 モジュールの索引を縮小し、共通設計の適用範囲と例外を同期する。領域行は 189 行から 103 行、ClaimMapping は 9 行から 4 行。旧形式を要求した `update-design` スキルを一次情報への参照へ更新した。
- T004 の追加検証：`feature-layout.acceptance.test.ts` の「既存の固有設計を索引から消すと共通設計への委譲として隠せない」は、索引から固有設計の行を削除しても終了コード 0 になる RED を確認し、実在する領域文書との照合後に GREEN にした。製品要件 ID は N/A。
- [x] T006 [Verify] 参照の解決、19 件の品質要件の担当の網羅性、固有設計の索引、規範の維持を検証する。`mise run verify` と `mise run test-ui-e2e` が成功し、`mise run spec-diff` は規範変更なし。品質要件の照合をランナーから外すフォールト注入は担当欠落のテストが検出し、復元後に成功した。

## 検証

- mise run test-tools-file で索引の形式と参照の検査を実行する。
- 委譲先がない、参照先の節がない、区分が矛盾する索引を検査が拒否する。
- ログインと OAuth2 の宣言済み SLO から担当機能、設計、検証へ到達できる。
- ClaimMapping のような小さなモジュールで九行の再掲が不要になり、適用する共通設計と固有の例外へ到達できる。
- mise run spec-diff で品質目標、測定境界、製品の規範要件に意味上の変更がないことを確かめる。
- mise run check-spec、mise run check-links、mise run check-work-items、mise run verify を実行する。

## リスク

担当を明示するつもりで新しい品質要件を加えたり、既存の保証を対象外へ移したりしない。
割り当てに実質的な仕様判断が必要なら、着手時に本項目の仕様影響を更新し、規範を先に決める。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は `no normative specification change against main` を報告した。
  品質目標、母集団、測定境界、製品要件は維持した。
  共通設計をデフォルトにし、21 モジュールの索引の領域行を 189 行から 103 行へ減らした。
  固有設計、委譲、対象外を区別し、委譲先の節と実在する固有文書の参照を検査する。
  宣言済みの 16 SLO と 3 キャパシティ要件を、担当機能、設計、検証へ一か所で割り当てた。
  実装状況と達成の証拠は担当の宣言から分離し、未実測と監視の不足を維持した。
  移行で見つかった Markdown と生成サイトの句読点アンカーの不整合も修正した。
  `wi-98584` へ三つの観察、原因の分類、改善案と検証した効果を記録した。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/feature-layout.acceptance.test.ts` の「委譲先の存在しない節を文書検査で拒否する」「宣言済みの品質要件から担当への参照がない場合を拒否する」。追加した「既存の固有設計を索引から消すと共通設計への委譲として隠せない」も同じ境界で確認した。
  - **要件**: N/A: 製品の振る舞いを変更しない文書形式とツールの作業。根拠は設計フォーマットの索引と品質要件の割り当ての規則。
  - **観測した失敗**: 存在しない節、未割り当ての SLO、索引から消した固有設計のいずれも、実装前は文書検査が終了コード 0 を返し、拒否を期待した表明が失敗した。
  - **検出できる理由**: 製品の受け入れ境界は N/A。代替として実際のランナーを最小の文書集合へ実行し、参照と担当の欠落を成功と誤判定する配線を検出する。
- **単体 RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/specification-doc.test.ts` の「共通設計を既定とする索引」。生成表示は `mise run test-tools-file -- render-docs/src/render.test.ts --test-name-pattern 'Markdown の句読点'`。
  - **要件**: N/A: 製品のドメイン計算を変更せず、形式検査と生成表示を変更する。根拠は設計フォーマットと Markdown の参照先。
  - **観測した失敗**: 索引の 10 例は旧二列ヘッダーを要求されて失敗した。生成表示は `OAuth 2.0` の Markdown アンカー `#oauth-20` を存在しない生成 ID へ変換し、生成サイトの参照検査が例外を返した。
  - **検出できる理由**: 製品の単体境界は N/A。代替として純粋な形式検査の所見と生成 HTML の実際の参照先を比較し、省略、区分、句読点処理の誤りを判別する。
- **変更耐性の結果**:
  品質要件の照合を `checkDocuments` から外す配線の故障を注入した。
  「宣言済みの品質要件から担当への参照がない場合を拒否する」が、終了コード 0 を観測して失敗した。
  配線を復元すると同じテストが成功した。
  固有文書を残して索引の行だけを消す故障も、追加したランナーのテストが拒否する。
  変更した Go はないため、Go の変異検査は N/A。
  機械検査が保証するのは区分の形式、宣言済み ID の網羅性、文書と節の存在、固有文書の索引であり、担当する判断の意味は七つの視点と文書のレビューで確認した。
- **検証結果**:
  - `mise run setup`、`mise run lint-go`、`mise run format-tools`、`mise run typecheck-tools`、`mise run lint-tools` - 成功。
  - `mise run test-tools-file` による形式、ランナー、Markdown リンク、生成表示の局所検証 - 成功。
  - `mise run check-spec`、`mise run check-links`、`mise run check-work-items`、`mise run check-terminology` - 成功。
  - `mise run render-docs` - 1,353 ページを生成。OpenAPI と `site/index.html` を生成し、生成物は追跡しない。
  - `mise run check-published-api-compat` - 成功。未公開状態を維持した。
  - `mise run spec-diff` - 規範変更なし。`docs/requirements/quality.md`、TypeSpec、製品コードにも差分なし。
  - `mise run verify` - 成功。ツール 1,003 テスト、UI 単体 709 テスト、全 Go race テストと各静的検査を通過した。
  - `mise run test-ui-e2e` - 39 テスト成功。最初の実行ではサンドボックスが WebView の起動を拒否し、サンドボックス外で再実行した。
  - 総合検証の最初の失敗は文書の用語「既定」を「デフォルト」へ直した。再実行中の Go lint の読み込みエラーは、単独のサンドボックス外実行と、その後の総合検証で成功を確認した。
