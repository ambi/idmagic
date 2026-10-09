---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-92970-adopt-information-hiding-modular-design, wi-26986-rename-remaining-context-terms-to-module]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発者向けの文書体系と生成サイトを整理する変更であり、製品の API、設定、運用手順は変わらず、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - docs/formats/documentation-guide.md
    - docs/formats/specification-format.md
    - docs/formats/work-item-format.md
    - docs/README.md
    - docs/domain/README.md
    - tools/render-docs/src/render.ts
    - tools/render-docs/src/main.ts
    - tools/workspace/src/document-layout.ts
    - tools/check/src/terminology.ts
  tests:
    - tools/render-docs/src/render.test.ts
    - tools/check/src/terminology.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading: [backend, frontend, spec]
spec_impact: { kind: none, reason: "文書体系、入口、生成サイトの区分名と境界判断の案内を更新する。HTTP の応答、認証方式、永続状態、監査イベント、外向きの通知と同期、TypeSpec と規範要件は変更しない。" }
---

# 文書体系と生成サイトを情報隠蔽に基づくモジュール設計へ合わせる

## 動機

wi-92970 と wi-26986 は、実装の単位をモジュールへ改名し、境界の判断を情報隠蔽と検査可能な制約へ置き換えた。
しかし `docs/README.md` とサイトのサイドバーは「ドメイン設計文書」を案内し、文書体系はモデルと用語体系の範囲としてのコンテキストを別に定義している。
読者は現在のモジュール設計と旧来の分類を読み分けなければならない。

さらに `docs/domain/README.md` は廃止済みの 7 文書を要求し、仕様フォーマットは語彙の違いから境界を引き直す表を残している。
入口、記述規約、生成器をそろえて、現行の設計と文書配置へ導く。

## 対象範囲

- 文書の区分名を「モジュール設計」へ統一する。
- 利用者が選んだ表示名「コード構成」へ、`structure.md` の題名と参照ラベルを統一し、旧い配置図を更新する。
- 仕様の階層をシステム、モジュール、機能群、機能とし、方法論文書のコンテキストという階層名を改める。
- モジュールの仕様と実装の対応を責務表で宣言し、境界は設計ガイドラインの判断手順で選ぶと定める。
- `docs/domain/README.md` を現在の機能仕様、任意の例の付録、話題ごとの設計へ更新する。
- 生成サイトのナビゲーションと用語検査を更新し、サイトを再生成する。
- 3 文書を `docs/formats/documentation-guide.md`、`specification-format.md`、`work-item-format.md` へ移し、`README.md` を索引にする。
- 3 文書の役割、重複する規則、見出しと参照を整理し、文書内容もリファクタリングする。
- 文書配置の定義、検査、生成器、エージェント指示と作業スキル、未完了項目の参照を新しいパスへ更新する。
- エージェント指示は、専用ツールが公開されている場合に優先し、ない場合は通常の読み取りコマンドと `apply_patch` を使えるようにする。

## 対象外

- モジュールの追加、分割、統合、公開範囲、依存、テーブル所有者、組み立て地点の変更。
- Aggregate、値オブジェクト、状態、不変条件、イベント、作用分離など、wi-92970 が維持した規則の撤去。
- `docs/domain/`、`spec/contexts/`、`REQ-*`、TypeSpec の名前空間、コード中の識別子の改名。
- Go や認証プロトコル、実行の文脈を指すコンテキストの改名と完了済み記録の内容の書き換え。移動で切れる Markdown リンクの宛先だけは更新する。

## 設計

「モジュール設計」は責務、公開契約、要件、内部機構を収める区分の名前とする。
モジュールの境界は、隠す設計判断、要求と不変条件、公開範囲、書き込みの所有、変更の波及を比較して選ぶ。
仕様の粒度は観測可能な機能で決め、モデルの語彙だけから実装の分割を決めない。

| 案 | 採否と理由 |
| --- | --- |
| サイトの区分名だけを変更する | 採用しない。入口が廃止済み文書を案内し、規約の境界判断が残る |
| 仕様のコンテキストと実装のモジュールを呼び分け続ける | 採用しない。現在の各モジュールの仕様文書と用語が一致しない。対応は責務表で明示する |
| 区分名、文書体系、境界判断の案内を合わせる | 採用する。公開済み識別子を維持し、現行の設計へ直接たどれる |
| DDD 由来の語をすべて削除する | 採用しない。wi-92970 が維持した状態と整合性の規則まで変わる |

境界そのものは変更しないため、D1〜D8 による候補比較の記録は該当しない。
用語検査では旧区分名を拒否し、設計の階層としてのコンテキストに限って検出する。
実行や認証の文脈を指す用法は維持する。

### 形式文書の配置と責務

| 文書 | 定めること | 他の文書へ委ねること |
| --- | --- | --- |
| `documentation-guide.md` | 内容の担当、文書の目的、一次情報と派生表示の区別 | 仕様の正確な構文、作業項目の証拠の形式、開発サイクル |
| `specification-format.md` | 検査される配置と記述規則、仕様と設計の形式 | 配置を選んだ理由、境界の判断手順、変更の履歴 |
| `work-item-format.md` | 一つの変更の計画、仕様影響、文書影響、着手と完了の証拠 | 証拠の水準を決める方針、実装と検証の実行手順 |

`docs/formats/` は固定の集合として文書配置の定義へ追加する。
生成器は既存の「フォーマット」区分と `format/*.html` の URL を維持し、新しい `README.md` をその入口として描画する。
旧い root 直下の特別扱いを外し、通常の一次情報文書と同じ収集経路へ置く。
完了済み項目の本文、仕様参照と着手時の記録は維持する。
移動で切れる Markdown リンクの宛先だけを機械的に更新し、未完了項目の読み取り先は新しいパスへ合わせる。

### 移動で見つかった検査の不具合

形式文書を通常の文書収集へ移すと、検査がコードブロック内のテンプレートを本文として数え、複数の H1 と規範 ID の宣言を報告した。
テンプレートは記述例であり、規範の宣言ではない。
Markdown のパーサーが返すコードブロックの範囲を検査対象から除き、原稿の行数と位置は維持する。
入れ子のバッククォートとチルダのフェンスを含む例を受理し、本文の規範宣言と H1 の欠落は引き続き拒否するテストで固定する。

エージェント指示には `Read` と `Edit` の提供を前提とする禁止があった。
利用可能な専用ツールを優先するという目的を明記し、専用の読み取りツールがない場合の `cat` と `sed`、編集の `apply_patch` を許可する。
利用者とは、画面に表示される読み取り操作の「Read」と専用ツールの名前を取り違えて会話したため、指示を特定のツール名に依存させない。

## 計画

1. 生成サイトの既存テストで新しい区分名の RED を確認する。
2. 一次情報文書と記述規約を更新する。
3. 生成器と用語検査を更新する。
4. 規範差分がないこと、リンク、文書サイト、ツールの検証結果を確認する。

## タスク

- [x] T001 [Docs] 入口と記述規約を現在の文書構成と境界の判断へ合わせる。N/A: 製品要件の変更はなく、一次情報のレビューと `mise run check` で配置、用語、リンク、規範差分を検証した。
- [x] T002 [Tooling] 3 文書の移動、文書配置の定義、生成サイトの区分名と用語検査を合わせる。検査：`mise run test-tools-file -- render-docs/src/render.test.ts`、`mise run test-tools-file -- check/src/terminology.test.ts`、`mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts`。
  N/A: 製品の受け入れ境界を変えない。代替の RED は生成サイトの既存テスト 5 件の失敗と、形式文書のコードブロックを本文と誤認する仕様検査の失敗で確認した。
- [x] T003 [Verify] 文書サイトを再生成し、規範差分と検証結果を記録する。

## 検証

- `mise run check-work-items`
- `mise run check-terminology`
- `mise run check-spec`
- `mise run check-links`
- `mise run check-rendered-docs`
- `mise run render-docs`
- `mise run check-api-compat`
- `mise run spec-diff -- main`
- `mise run verify`

## リスク

コンテキストは実行の文脈やプロトコルの値にも使われる。
一括改名は方法論文書の階層名に限り、その他の用法は検索結果と差分を個別に確認する。
文書の置き場所と規範 ID を維持し、既存リンクの切断を避ける。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff -- main` は規範仕様の差分がないことを示した。
  文書の区分を「モジュール設計」、コード配置と依存規則の文書を利用者が選んだ「コード構成」へそろえた。
  廃止済みの必須文書一覧と、モデルの語彙だけで境界を分ける説明を、現在の機能仕様、任意の受け入れ例、話題ごとの設計、責務表と境界を選ぶ判断手順へ改めた。
  3 文書は `docs/formats/` の小文字ファイル名へ移し、内容の担当、仕様の形式、作業の記録という責務を分け、重複する規則と古い参照を整理した。
  生成器は移動先を通常の文書収集で読み、既存の `format/*.html` の URL を保ち、索引には `docs/formats/README.md` を使う。
  文書検査はコードブロック内のテンプレートを本文の H1 と規範宣言に数えず、本文への診断の行番号は維持する。
  エージェント指示は特定の専用ツールの提供を仮定せず、専用の読み取りツールがない場合は `cat` と `sed`、編集には `apply_patch` を使えるようにした。
  仕様、設計、指示、スキル、未完了項目の参照を更新し、完了済み項目は移動で切れる Markdown リンクの宛先だけを修正した。
  製品コードの差分は仕様文書への参照コメント一件であり、製品の振る舞いは変更していない。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/render-docs/src/render.test.ts` の生成サイトの既存テスト。
  - **要件**: N/A: 製品の規範要件を変更しない、文書配置と生成サイトの変更である。
  - **観測した失敗**: 新しい区分名と移動先を期待するテストは、生成器の変更前に 5 件失敗した。
  - **検出できる理由**: 生成ファイルの集合、サイドバーの区分名、リンク先、コード構成のラベルを出力の HTML で比較するため、旧名の残存と移動先の誤分類を区別できる。移動先の索引に固有の本文とリンクを置き、生成した索引に置き換わらないことも確認した。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/specification-doc.test.ts` の「形式文書のコードブロックを見出しや規範宣言として数えない」。
  - **要件**: N/A: 形式文書の記述例を規範宣言と誤認する文書検査の不具合である。
  - **観測した失敗**: コードブロックの除外前は、テンプレート内の H1 と `REQ-DEMO-*` に対する診断 3 件が返り、診断なしを期待したテストが失敗した。
  - **検出できる理由**: 入れ子のバッククォートとチルダのフェンス内の例を受理し、本文の規範宣言は行 14 で拒否し、本文の H1 の欠落も拒否する。単に見出しの検査を止める誤実装は通らない。
- **変更耐性の結果**:
  低リスクの文書と検査ツールの変更であり、追加の変異試験は選択していない。
  生成結果の具体的なリンクと本文、本文の不正な規範宣言の拒否、正しい診断行を確かめるテストを変更後も通した。
  Go の振る舞いは変更しておらず、参照コメントの更新に変異試験は該当しない。
- **検証結果**:
  - `mise run verify` - 成功。ツールのテスト 920 件、Go の静的検査と race テスト、フロントエンドの整形、静的検査、型検査、単体テスト、ビルドを含む。
  - `mise run check` - 最終の文言調整後も成功。文書の配置、仕様文法、リンク、用語、エージェント指示、生成サイト、API 互換性を含む。
  - `mise run spec-diff -- main` - 規範仕様の差分なし。
  - `mise run test-tools-file -- render-docs/src/render.test.ts` - 31 件成功。
  - `mise run test-tools-file -- check/src/specification-doc.test.ts` - 48 件成功。
  - `mise run test-tools-file -- check/src/terminology.test.ts` - 16 件成功。
  - `mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts` - 26 件成功。
  - `mise run test-tools-file -- render-docs/src/documentation-quality.test.ts` - 5 件成功。
  - `mise run lint-tools`、`git diff --check` - 成功。
  - `mise run render-docs` - 443 文書から 1353 ページを再生成した。
