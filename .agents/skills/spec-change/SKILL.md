---
name: spec-change
description: 機能と振る舞いの変更で、実装の前にモデル、API、認証の TypeSpec と docs/ 以下の担当する一次情報の文書を更新する。要件、TypeSpec、機能仕様、状態機械を追加、変更、廃止するときに使う。
---

# 仕様を先に変える

作業項目では `mise run brief -- <work-item> --target <対象パス>` から読む節を選ぶ。
公開 HTTP 契約を変えるときは `--concern contract`、作用の保証を変えるときは `--concern effects` を付ける。
起点と追加取得は `docs/development/task-context.md` に従い、変更を所有する最小の仕様と、その形式の該当節を読む。

実装の前に、変更を所有する最小の仕様を更新する。
`docs/formats/specification-format.md` が現在の文書の種類と文法を定める。
該当する節を読み、その根拠を説明する文書は読まない。

1. 要件の差分を、まず作業項目の設計に書く。
   追加、変更、廃止する要件を、それぞれの ID と要件文とともに書く。
   一次情報の文書へは、その後で適用する。
   一つの変更のために機能全体を書き直さない。
2. モデル、API の操作、HTTP のバインディング、要求と応答の形、ステータスコード、エラーの union、非推奨のメタデータ、認証方式は `spec/modules/<module>/{models,main}.tsp` に置く。
3. モジュールの境界は `docs/modules/<module>/README.md` に、複数の機能が共有する値オブジェクト（正規化、比較、一意性の範囲）はモジュールの `data.md` に置く。
   語彙は `glossary.md`、採用したプロトコルの規則は `standards.md` に置く。
   一つの機能のモデル、状態機械、要件、セキュリティの考慮は、機能仕様 `docs/modules/<module>/features/<feature>.md`（またはその横の章 `<feature>.<chapter>.md`）に置く。
   各操作は `## 操作` の下に専用の H3 を持たせ、要件だけを、正常系、状態による分岐、拒否の順に書く。
   章のページ `<feature>.<chapter>.md` では、操作を H2、要件を H3 にする。
   操作ごとの要約の表や `## エラー` の節は書かない。
   各状態機械は、状態表、遷移表、すべてのセルを埋めた状態遷移マトリクスを持つ。
   状態機械は機能仕様ではなく、その機械の `testdata/<Machine>.statemachine.json` で編集し、`mise run generate-spec-from-testdata` を実行する。
   モジュールに割り当てた品質要件は `docs/modules/<module>/quality.md` に、機能の取り分は `## 品質` に置く。
   機能のノードは、ファイル名でコードのスライス `backend/<module>/<feature>/` と対応する（名前を比べるときはハイフンを除く）。
   複数の機能が共有する仕組みは、それを実装する機能の `## 実現方式` か、`observability.md` のようなモジュールの設計領域の文書に置く。
   代替案を比べた判断は `decisions.md` に置く。
   コードから読めないこと（仕組みがどう結果を保証するか、どう修復するか）だけを、機能仕様の任意の最後の節 `## 実現方式` に置く。
   システム全体の事実には、`docs/requirements/` または `docs/design/` の該当するファイルを使う。
4. 要件を書く前に、触れる操作ごとに `docs/development/specification-first-workflow.md` の観点の表（仕様の漏れを探す観点）のすべての問いに答える。
   答えのない問いは漏れである。
   各漏れを、同じ文書の「仕様にない振る舞いの分類」の定義に従って、(a) 要件にする、(b) 未規定のままにする、(c) 実装を変える、に分類し、分類を作業項目に記録する。
   分類が利用者の観測する結果を変えるときは利用者に確認する。
5. 外部から観測できる新しい振る舞いには、未使用の `REQ-<MODULE>-NNN` を与え、それが支配する操作の下に `#### REQ-<MODULE>-NNN <タイトル>` の見出し（章のページでは `###`）で宣言する。
   `docs/formats/specification-format.md` の日本語の EARS 形式（前提、応答者としてのモジュール名、応答）で、箇条ごとに一文を書く。
   条件が組み合わさるときは表を使い、理由が自明でないときだけ `**判断**` の欄を書く。
   `**担保手段**` と `**要判断**` は書かない。
   テストは `//spec:covers` で要件を辿り、未解決の問いは作業項目にする。
   参照されている振る舞いを廃止するときは、ID を消したり再利用したりせず、見出しに `(superseded by REQ-<MODULE>-NNN)` を付ける。
   製品が保つ値（上限、期間、計算式）は、設計だけでなく要件に書く。
   その値と、標準が任意とする選択は、`docs/development/specification-first-workflow.md` の「同種の製品の参照」に従って主要な製品の扱いを調べてから決める。
   複数の操作が共有する要件は、最も直接に支配する操作の下に一度だけ書き、ほかの操作は `**上位の要件**` からリンクして差分だけを書く。
6. `<feature>.feature.md` に例を足すのは、境界や意外な振る舞いが要件文から読み取りにくいときだけにする。
   例は `testdata/*.examples.json` から生成するほうを選ぶ。
   要件を言い直すだけの例は足さない。
7. 複数のモジュールを要する保証や、複数のモジュールに同じく当てはまる保証は `docs/modules/scenarios.feature.md` に置き、参加するモジュールを名指し、モジュール内に閉じた断片を入れない。
8. プロジェクトがポリシー言語を採るまで、細かな認可の振る舞いはコードとテストに置く。
   TypeSpec は認証と強制する操作のスコープを記録する。
   共有する主体、スコープ、テナント境界、fail-closed の規則は `docs/design/security/authorization.md` が所有する。
9. 作業項目の `affected_spec` を、要件または標準の ID、または TypeSpec のシンボルと同期させる。
10. `mise run check-spec` を通し、派生成果物を再生成して検査する。
    - `mise run build-site` を実行する。
      これは先に TypeSpec をコンパイルするので、サイトが古い OpenAPI を描くことはない。
      OpenAPI がモジュールごとのタグを持つことと、`site/index.html` が作られることを確かめる。
    - 変更が TypeSpec の操作を追加、削除、変更するときは、`mise run generate-contract` も実行する。
      これは追跡されている `backend/shared/spec/operations_gen.go` を書き換え、`check-generated-contract` がそれを TypeSpec と比べる。
      このファイルは変更と一緒にコミットする。
    - `mise run check-published-api-compat` で公開状態に応じた互換性を検査する。
      初回公開前の差分を任意に調べる場合は `mise run check-api-compat` を使う。
      公開状態と初回公開の切り替え、告知と既存バージョン互換性の適用条件は `docs/development/release.md` に従う。
    - `spec/generated/` と `site/` は追跡しない派生ビューである。
      コミットしない。
    - ベースラインはリリースの間だけ、明示したリリースの手順として `mise run update-api-baseline` で更新する。
      通常の機能変更で互換性の検査を避けるためにベースラインを更新しない。
      検査を飛ばすと本当の後退が出荷され、リリースなしに固定したベースラインは誰も受け取っていないものを表すことになる。

モジュール、全体構造、技術、ランタイムの構成、主要な設計規則が変わるときだけ、`update-design` も使う。
