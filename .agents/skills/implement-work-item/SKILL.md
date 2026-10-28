---
name: implement-work-item
description: 選択した作業項目を専用のブランチで実装し、仕様、故障に合う検証、要約への縮約、コミットまで仕上げる。
---

# 作業項目の実装

このスキルは一つの作業項目を完了する実行手順である。
規則の形式と判断は、各段階の参照先に従う。
参照先は通読せず、今回の条件に当たる節を読む。

## 1. 作業場所

一つの項目には `work-item/<work-item-id>` ブランチを使う。
作業場所は、既定のチェックアウト（リポジトリの根）が空いていればそこにし、使用中のときだけ worktree を作る。
新しいディレクトリでは Go のビルドと lint のキャッシュが初回だけ効かず、十数秒余計にかかるためである。

- 現在のチェックアウトがそのブランチを開いていれば、そのまま使う。
- 既定のチェックアウトが base（利用者が指定したもの、未指定なら `main`）を開き、`git status --short` が空なら、そこでブランチを切る。

  ```sh
  git switch -c work-item/wi-42
  ```

- そうでなければ、既定のチェックアウトの変更には触れず、`.worktrees/` に worktree を作り、依存を入れる。ブランチが既にあり、どこでも開かれていなければ `-b` を付けずに接続する。

  ```sh
  git worktree add -b work-item/wi-42 .worktrees/wi-42 <base-branch>
  mise -C .worktrees/wi-42 run setup
  ```

`git status --short --branch` でブランチとクリーンな作業場所を確認し、その作業場所の中で続ける。
worktree は `.worktrees/` に置き、`/tmp` や `$TMPDIR` へ置かない。

## 2. 着手前の確認

`mise run brief -- <work-item> --target <changed-path>` を実行する。
複数の対象には `--target` を繰り返し、境界は `--concern boundary`、作用は `--concern effects` で指定する。
参照元と理由から、読む節と検証タスクを選ぶ。
起点と追加取得は[作業文脈の導出](../../../docs/development/task-context.md#起点と追加取得)に従う。

frontmatter を変える前に、項目、直接の規範と標準の宣言、参照する TypeSpec、所有文書、最小のコードとテストを読む。
読んだ内容から必要になった参照だけを追加する。
外部製品や標準への対応と、その前提機能では、[外部制約と利用側の成立](../../../docs/development/specification-first-workflow.md#外部制約と利用側の成立)を実装前に確かめる。
標準被覆の項目では、対象の全 ID について宣言、負債、名指されたテスト、最小の本番入口、欠落を担当する active 項目を解決する。
完了に対象外の実装が必要なら、前提または欠陥を記録し、実装と集約検査の前に止める。

共有コード、構成入口を含め、製品の HTTP 契約へ影響する変更では、[着手時の影響と契約証拠](../../../docs/development/specification-first-workflow.md#着手時の影響と契約証拠)で予定するファイルと参照不足を照合する。

選んだ検査の実行に必要な依存だけを準備する。
導入タスクと先行 lint の条件は[検証の進め方](../../../docs/development/specification-first-workflow.md#検証の進め方)に従う。

## 3. 仕様と計画

仕様を変える場合は `spec-change` で先に更新し、`mise run check-spec` を通す。
確認後も `spec_impact: none` が正しければ、仕様の編集と仕様だけの基準検査は追加しない。

製品結果、公開契約、設計境界、タスク分割を変え得る問いを解決し、先送りする選択は対象外へ記す。
方式の候補を比べるときも、[外部制約と利用側の成立](../../../docs/development/specification-first-workflow.md#外部制約と利用側の成立)に従い、選択の根拠が候補の採用を前提にしていないか確かめる。
複数の改善案を持つ項目では、すべての案の実装済み、今回の採用、保留、対象外と理由を記録し、今回実装すべき案を残していないことを確認する。

一つの記録は独立して受け入れられる一つの結果を扱う。
量だけでは分割しない。
モジュール間の依存や共有コードを追加する場合は、既存の負債台帳の範囲内でも[境界を選ぶ判断手順](../../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
先に取り込める構造変更だけは別の記録と Pull Request にし、それ以外のコミット境界は[コーディングスタイル](../../../docs/development/coding-style.md#構造変更と振る舞いの変更を分ける)に従う。
反復する単位ごとの作業が高コストなら、原因を測り、fixture、索引、判断を失うツールなどの欠陥として起票する。

## 4. 着手の記録

[証拠の要件](../../../docs/development/specification-first-workflow.md#証拠の要件)から、行う証拠の手順を選ぶ。
主要ユースケースを要求する作業では、[テスト方針](../../../docs/development/testing.md#主要ユースケース)から観測結果、`fault_model`、最小の境界を選び、[記入形式](../../../docs/formats/work-item-format.md#主要ユースケースの証拠)に従って `primary_use_cases` に書く。
機能の成熟度を上げる作業では、[成熟度の昇格](../../../docs/formats/work-item-format.md#成熟度の昇格)も記入する。
`status: in_progress` にして `mise run check-work-items` を通す。
中核ロジックは型と操作のシグネチャ、境界変更は候補の比較を、コードの前に設計へ記す。
依存の構築条件を変える、または同じ操作の呼び出し元を追加する場合は、[依存の向きと差し替え点](../../../docs/design/application/design-guidelines.md#依存の向きと差し替え点)で公開の経路と必須の作用を比べる。
必須依存の配線、HTTP の経路差、派生表現の参照を変える場合は、[組み立てと経路差](../../../docs/development/testing.md#組み立てと経路差)から観測結果と故障を選ぶ。
対象の選択や上限の判定には[判定の両側と境界値](../../../docs/development/testing.md#判定の両側と境界値)、外部互換性と定型設定には[外部契約と設定の利用](../../../docs/development/testing.md#外部契約と設定の利用)を適用する。
状態の更新入口や共通の検査を変える場合は、[型の所有と不変条件](../../../docs/design/application/design-guidelines.md#型の所有と不変条件)から、既存の入口が所有者の検査へ到達するかを確かめる。
イベントの payload を追加または変更する場合は、[イベントの送信表現](../../../docs/development/testing.md#イベントの送信表現)から直列化後の観測を選ぶ。
実装中に規範変更が必要になれば手順 3 へ戻る。

## 5. 実装と局所検証

最初のコードまたはテストの編集前に、[コーディングスタイル](../../../docs/development/coding-style.md)を読む。
触れる既存 Go の振る舞いを `//spec:covers` テストが固定していない場合は、[特性化テスト](../../../docs/development/specification-first-workflow.md#特性化テスト)の条件に従う。
公開状態が `published` なら特性化テストを先に行い、本番コードを変えない別コミットへ記録する。
`unpublished` なら、触れる範囲の保証を表す要件のテストが欠けていないかだけを確かめ、欠けていれば先に足す。
既存の `TestCharacterize*` の失敗は期待値を書き換えて通さず、変更を戻すか、利用者に分類を確認する。

[検証の進め方](../../../docs/development/specification-first-workflow.md#検証の進め方)に従い、選んだ最小の検査で一つの振る舞いを RED、GREEN にし、GREEN を保ってリファクタリングする。
必要な範囲で Domain、Use Cases、Adapters、Infrastructure、UI へ進み、各層の局所検査をその場で実行する。
選んだ recipe はタスクに記し、同じ選択を繰り返さない。
タスクには、実際の失敗、テスト名、規範 ID、該当しない境界の理由と代替検査を残す。
振る舞いを保つ変更には成功した検査を記す。

仕様にない分岐、エラー、イベント、作用を書くことになったら、[仕様にない振る舞いの分類](../../../docs/development/specification-first-workflow.md#仕様にない振る舞いの分類)で判断して設計へ記録する。
利用者が観測する結果を変える分類は、続ける前に利用者へ確認する。
[実装とテストで見つけた振る舞いの反映](../../../docs/development/specification-first-workflow.md#実装とテストで見つけた振る舞いの反映)に従い、要件、例、生成できる参照を一次情報へ移す。
例をデータにすると短くなる場合は[実行可能な具体例](../../../docs/formats/specification-format.md#実行可能な具体例と派生表示)を使い、複雑な操作列はコードとして残す。

テストの引用は[規範 ID の参照](../../../docs/development/specification-first-workflow.md#規範-id-をテストから参照する)に従う。
`//verify:operation` を追加または変更したら `mise run check-operation-references` で形式と配置を局所検査する。
手書きの解析、デコード、分割、正規化、比較では、[プロパティテストとファジング](../../../docs/development/testing.md#プロパティテストとファジング)から費用対効果のある検査を選ぶ。
テスト補助を含むモジュール間の依存を追加したら、集約検査へ進む前に `mise run check-boundaries` を局所検査に選ぶ。
各タスクが GREEN になったら、[チェックポイント](../../../docs/development/specification-first-workflow.md#チェックポイントと最終コミット)を作る。

## 6. 設計と追加検査

[現在状態の文書](../../../docs/development/specification-first-workflow.md#現在状態の文書)の条件に当たれば `update-design` を使う。
境界を変える場合は、手順 4 で記録した比較に基づいて更新する。

故障注入、変異テスト、fuzz の探索の実施の要否と範囲は、[追加検査の費用対効果](../../../docs/development/testing.md#追加検査の費用対効果)で選ぶ。
実施する場合は同文書の該当手法に従い、仕様にない振る舞いが見つかれば手順 5 の分類へ戻る。

## 7. 完了とコミット

対象の振る舞いを実装し、証拠の手順を終えてから、七つの視点で差分をレビューする。
[最終検査の選択](../../../docs/development/specification-first-workflow.md#最終検査の選択)に従い、実際の差分と影響範囲に必要な最小の検査を選ぶ。
影響が不明なら検査の範囲を広げる。
Go のテストの race の扱いは[通常テストと race 検査](../../../docs/development/testing.md#go-の通常テストと-race-検査)に従う。
必要な検査が最終差分を検証していることを確認し、未検査または入力が変わった検査を実行する。
前提が未解決の間は集約ゲートを状態確認に使わない。

`mise run spec-diff` を読み、[完了](../../../docs/formats/work-item-format.md#完了)に従って `## 要約` を書く。
正式コミットの前に `mise run move-work-item -- <id>` を実行し、縮約前の仕様影響の照合と完了時の検査を通す。
移動後に `mise run check-work-items` と `mise run check-links` を通す。
[チェックポイントと最終コミット](../../../docs/development/specification-first-workflow.md#チェックポイントと最終コミット)に従って折り畳み、完了の変更もすべてステージして、`commit` スキルで実装と要約だけの完了記録を一つの正式コミットにする。

最終報告には、ブランチ、コミット、対象外、残した未実施事項を示す。
明示された場合だけ push する。
時間分析を求められた場合は、判断の区間とコマンドの実行時間を分け、コールドとキャッシュ済みの実行、環境の再試行、承認待ちをそれぞれ記録する。
