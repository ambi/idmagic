---
name: implement-work-item
description: 選択した作業項目を専用 worktree で実装し、仕様、故障に合う検証、要約への縮約、コミットまで仕上げる。
---

# 作業項目の実装

このスキルは一つの作業項目を完了する実行手順である。
規則の形式と判断は、各段階の参照先に従う。
参照先は通読せず、今回の条件に当たる節を読む。

## 1. 専用の作業場所

一つの項目には `work-item/<work-item-id>` ブランチと `.worktrees/<work-item-id>` worktree を使う。
現在の worktree がそのブランチ専用なら、そのまま使う。
そうでなければ `git status --short` で既存の変更を確認して保ち、利用者が指定した base、未指定なら現在のブランチから作る。

```sh
git worktree add -b work-item/wi-42 .worktrees/wi-42 <base-branch>
```

ブランチが既に存在し、別の worktree で使用されていなければ接続する。

```sh
git worktree add .worktrees/wi-42 work-item/wi-42
```

`git status --short --branch` でブランチとクリーンな作業場所を確認し、その worktree 内で続ける。
worktree は `.worktrees/` に置き、`/tmp` や `$TMPDIR` へ置かない。

## 2. 着手前の確認

`mise run brief -- <work-item> --target <changed-path>` を実行する。
複数の対象には `--target` を繰り返し、境界は `--concern boundary`、作用は `--concern effects` で指定する。
参照元と理由から、読む節と検証タスクを選ぶ。
起点と追加取得は[作業文脈の導出](../../../docs/development/task-context.md#起点と追加取得)に従う。

frontmatter を変える前に、項目、直接の規範と標準の宣言、参照する TypeSpec、所有文書、最小のコードとテストを読む。
読んだ内容から必要になった参照だけを追加する。
標準被覆の項目では、対象の全 ID について宣言、負債、名指されたテスト、最小の本番入口、欠落を担当する active 項目を解決する。
完了に対象外の実装が必要なら、前提または欠陥を記録し、実装と集約検査の前に止める。

共有コード、構成入口を含め、製品の HTTP 契約へ影響する変更では、[着手時の影響と契約証拠](../../../docs/development/specification-first-workflow.md#着手時の影響と契約証拠)で予定するファイルと参照不足を照合する。

選んだ検査の実行に必要な依存だけを準備する。
導入タスクと先行 lint の条件は[検証の段階](../../../docs/development/specification-first-workflow.md#5-検証の段階)に従う。

## 3. 仕様と計画

仕様を変える場合は `spec-change` で先に更新し、`mise run check-spec` を通す。
確認後も `spec_impact: none` が正しければ、仕様の編集と仕様だけの基準検査は追加しない。

製品結果、公開契約、設計境界、タスク分割を変え得る問いを解決し、先送りする選択は対象外へ記す。
複数の改善案を持つ項目では、すべての案の実装済み、今回の採用、保留、対象外と理由を記録し、今回実装すべき案を残していないことを確認する。

一つの記録は独立して受け入れられる一つの結果を扱う。
量だけでは分割しない。
モジュール間の依存や共有コードを追加する場合は、既存の負債台帳の範囲内でも[境界を選ぶ判断手順](../../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
先に取り込める構造変更だけは別の記録と Pull Request にし、それ以外のコミット境界は[コーディングスタイル](../../../docs/development/coding-style.md#構造変更と振る舞いの変更を分ける)に従う。
反復する単位ごとの作業が高コストなら、原因を測り、fixture、索引、判断を失うツールなどの欠陥として起票する。

## 4. 着手の記録

[証拠の要件](../../../docs/development/specification-first-workflow.md#4-証拠の要件)から、行う証拠の手順を選ぶ。
主要ユースケースを要求する作業では、[テスト方針](../../../docs/development/testing.md#主要ユースケース)から観測結果、`fault_model`、最小の境界を選び、[記入形式](../../../docs/formats/work-item-format.md#主要ユースケースの証拠)に従って `primary_use_cases` に書く。
公開後は[文書影響と成熟度](../../../docs/formats/work-item-format.md#文書影響と成熟度)も記入する。
`status: in_progress` にして `mise run check-work-items` を通す。
中核ロジックは型と操作のシグネチャ、境界変更は候補の比較を、コードの前に設計へ記す。
依存の構築条件を変える、または同じ操作の呼び出し元を追加する場合は、[依存の向きと差し替え点](../../../docs/design/application/design-guidelines.md#依存の向きと差し替え点)で公開の経路と必須の作用を比べる。
必須依存の配線、HTTP の経路差、派生表現の参照を変える場合は、[組み立てと経路差](../../../docs/development/testing.md#組み立てと経路差)から観測結果と故障を選ぶ。
実装中に規範変更が必要になれば手順 3 へ戻る。

## 5. 実装と局所検証

最初のコードまたはテストの編集前に、[コーディングスタイル](../../../docs/development/coding-style.md)を読む。
触れる既存 Go の振る舞いを `//spec:covers` テストが固定していない場合は、[特性化テスト](../../../docs/development/specification-first-workflow.md#特性化テスト)の条件に従う。
公開状態が `published` なら特性化テストを先に行い、検出能力を確認して本番コードを変えない別コミットへ記録する。
`unpublished` なら、触れる範囲の保証を表す要件のテストが欠けていないかだけを確かめ、欠けていれば先に足す。
既存の `TestCharacterize*` の失敗は期待値を書き換えて通さず、変更を戻すか、利用者に分類を確認する。

[検証の段階](../../../docs/development/specification-first-workflow.md#5-検証の段階)に従い、選んだ最小の検査で一つの振る舞いを RED、GREEN にし、GREEN を保ってリファクタリングする。
必要な範囲で Domain、Use Cases、Adapters、Infrastructure、UI へ進み、各層の局所検査をその場で実行する。
選んだ recipe はタスクに記し、同じ選択を繰り返さない。
タスクには、実際の失敗、テスト名、規範 ID、該当しない境界の理由と代替検査を残す。
振る舞いを保つ変更には成功した検査を記す。

仕様にない分岐、エラー、イベント、作用を書くことになったら、[仕様にない振る舞いの分類](../../../docs/development/specification-first-workflow.md#仕様にない振る舞いの分類)で判断して設計へ記録する。
利用者が観測する結果を変える分類は、続ける前に利用者へ確認する。
[実装とテストで見つけた振る舞いの反映](../../../docs/development/specification-first-workflow.md#実装とテストで見つけた振る舞いの反映)に従い、要件、例、生成できる参照を一次情報へ移す。
例をデータにすると短くなる場合は[実行可能な具体例](../../../docs/formats/specification-format.md#実行可能な具体例と派生表示)を使い、複雑な操作列はコードとして残す。

テストの引用は[規範 ID の参照](../../../docs/development/specification-first-workflow.md#規範idをテストから参照する)に従う。
`//verify:operation` を追加または変更したら `mise run check-operation-references` で形式と配置を局所検査する。
手書きの解析、デコード、分割、正規化、比較には[プロパティテストとファジング](../../../docs/development/specification-first-workflow.md#プロパティテストとファジング)を適用する。
各タスクが GREEN になったら、[チェックポイント](../../../docs/development/specification-first-workflow.md#チェックポイントと最終コミット)を作る。

## 6. 設計と変更耐性

[現在状態の文書](../../../docs/development/specification-first-workflow.md#6-現在状態の文書)の条件に当たれば `update-design` を使う。
境界を変える場合は、手順 4 で記録した比較に基づいて更新する。

選んだ故障について、[ミューテーションテスト](../../../docs/development/specification-first-workflow.md#ミューテーションテスト)で変更耐性を検査する。
変更した Go には `mise run test-go-mutation -- <package-directory>` を使い、ツールで表現できない配線の切断などだけを手で注入する。
比較基準、変更行と既存の全体調査の選択、生存変異の到達経路、並行時の時間超過の扱いは、同じ節から選ぶ。
生存した変異を読み、未使用の補完、等価な変異、観測の不足を分け、仕様にない振る舞いがあれば手順 5 の分類へ戻る。

## 7. 完了とコミット

対象の振る舞いを実装し、証拠の手順を終えてから、七つの視点で差分をレビューする。
[最終検査の選択](../../../docs/development/specification-first-workflow.md#最終検査の選択)に従い、実際の差分と影響範囲に必要な最小の検査を選ぶ。
影響が不明なら検査の範囲を広げる。
Go では[通常テストと race 検査](../../../docs/development/specification-first-workflow.md#go-の通常テストと-race-検査)も選ぶ。
必要な検査が最終差分を検証していることを確認し、未検査または入力が変わった検査を実行する。
前提が未解決の間は集約ゲートを状態確認に使わない。

`mise run spec-diff` を読み、[完了](../../../docs/formats/work-item-format.md#完了)に従って `## 要約` を書き、`mise run move-work-item -- <id>` を実行する。
移動後に `mise run check-work-items` と `mise run check-links` を通す。
[チェックポイントと最終コミット](../../../docs/development/specification-first-workflow.md#チェックポイントと最終コミット)に従って折り畳み、`commit` スキルで残る正式コミットを作る。

最終報告には、ブランチ、コミット、対象外、残した未実施事項を示す。
明示された場合だけ push する。
時間分析を求められた場合は、判断の区間とコマンドの実行時間を分け、コールドとキャッシュ済みの実行、環境の再試行、承認待ちをそれぞれ記録する。
