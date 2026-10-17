# 作業項目フォーマット

作業項目は、一つの意味上の変更を説明、設計、実装、検証する作業単位である。
未完了（`pending`、`in_progress`）の項目は `work-items/active/`、完了または中止した項目（`completed`、`cancelled`）は `work-items/done/` に置く。
両者は同じ深さにあるため、共通文書やコードへの相対リンクは完了時に変更しない。
`work-items/` 直下には項目を置かず、状態と配置の不一致は `mise run check-work-items` が拒否する。
この文書は記録の形式を定める。
証拠の水準と開発サイクルは[仕様先行の開発ワークフロー](../development/specification-first-workflow.md)、現在有効な仕様と設計の書式は[仕様フォーマット](specification-format.md)が定める。

## 識別子と配置

ファイル名には `wi-<識別番号>-<ケバブケースの題名>.md` を使う。
識別番号は、10000 から 99999 のうち、`work-items/active/` と `work-items/done/` のどちらにも現れない値を無作為に選ぶ。
`mise run work-item-number` がこの値を一つ出力する。
最大値に 1 を足す手順は使わない。
その手順は番号を配る主体が一つであることを前提にするが、各自がローカルで起票し、一人が並列のワークツリーで起票する運用では、まだ push されていない起票を見られないためである。
既存の記録は 3 桁以下に収まっているので、新規を 5 桁に限れば番号空間が重ならない。

作業項目は、タスクリスト、変更固有の設計文書、実装履歴も兼ねる。
完了時点でも有効な結論は、TypeSpec またはその種類の内容を扱う一次情報文書へ反映しなければならない。

## 基本形式

```markdown
---
status: pending
authors: [name]
risk: low
reversibility: reversible # 任意。このテンプレートを写さず、変更ごとに判断する
created_at: 2026-01-01
priority: p1
depends_on: []
change_kind: feature
documentation_impact: # 公開後に着手する項目では必須
  level: release_note
  reason: 新たにサポートする機能をリリースの読者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-<number>-start-task.md }
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/modules/system/main.tsp, symbol: Product.System.Operations.StartTask }
primary_use_cases: # feature、bugfix、standards.md の変更では着手後に必須
  - id: start-task
    requirement: REQ-SYSTEM-001
    observable_result: 呼び出し元がタスクの実行開始を観測できる。
    boundary: acceptance
    test: { path: backend/system/usecases/start_task_test.go, name: TestStartTask_REQ_SYSTEM_001, task: test-go-race }
    fault_model: ユースケースが開始コマンドを発行しない。
maturity_evidence: # 成熟度を昇格する場合は完了前に必須。告知パスは公開後に必須
  - feature: start-task-v1
    from: preview
    to: supported
    security: セキュリティレビューで、対象ユースケースに未解決の統制不足がないことを確認した。
    compatibility: 既存の preview 設定は移行せずに引き続き受理される。
    documentation: docs/releases/changes/wi-<number>-start-task.md
---

# 意味上の変更を表す一文

## 動機
変更が必要な理由を書く。

## 対象範囲
- 変更に含める仕様と実装を書く。

## 対象外
- 変更から明示的に除外する作業を書く。

## 設計
採用する設計、考慮事項、採用しない代替案を書く。

## 計画
実装順序、移行、未解決の問いを書く。
作るものを変え得る問いは、実装前にすべて解決する。

## タスク
- [ ] T001 [Spec] 仕様を更新する。
- [ ] T002 [Acceptance] 観測可能な境界で受け入れ RED を確認する。
- [ ] T003 [App] 単体 RED を確認し、GREEN にしてからリファクタリングする。
- [ ] T004 [Verify] 変更を検証する。

## 検証
- 変更と影響範囲に応じて選んだ検査タスク、対象、選択理由を書く。

## リスク
リスクと緩和策を書く。
```

節見出しは、上のテンプレートが示す日本語と、既存の記録が使う英語（`Motivation`、`Scope`、`Out of Scope`、`Plan`、`Tasks`、`Verification`、`Risk Notes`）のどちらでも機械検査が同じ項目として読む。
新しい記録はテンプレートの表記で書き、既存の記録を書き換えるためだけの変更はしない。
検証タスクと対象は[最終検査の選択](../development/specification-first-workflow.md#最終検査の選択)に従って決める。

## 優先順位とリスク

`priority`（`p0`〜`p3`）と `depends_on` は別の問いに答える。
`depends_on` は先に完了すべき項目を示し、機械検査によって作業順を制約する。
`priority` は、依存関係に妨げられていない項目のうち何を先に扱うべきかを示す参考値である。
未設定なら順位を付けていないことを表す。

`risk` と `reversibility` も別の問いに答える。
`risk` は変更を誤った場合の被害を、`reversibility` は判断を後から取り消せるかを表す。
両者は独立している。
レプリカ構成、キャッシュ方針、画面配置は影響が大きくても元に戻せる場合がある。
一方、通信フォーマット、識別子の意味、公開済みスキーマ、破棄した鍵、割り当てた `REQ` 番号は、小さな変更に見えても取り消せない。
判断を戻すためにリポジトリ外の利用者が保存済み、送信済み、または信頼済みのものを変える必要がある場合は `irreversible` とする。

`reversibility` 自体は必要な証拠を選ばない。
後から読む人が、まだ選び直せる判断と、今後も維持すべき判断を区別できるように記録する。
`reversible` としても `risk` が求める証拠は緩和しない。
このフィールドは導入前の記録を有効に保つため任意であり、未記入は可逆を意味せず、評価していないことを意味する。

リスクと証拠の対応は[仕様先行の開発ワークフロー](../development/specification-first-workflow.md#4-証拠の要件)が定める。

## 仕様影響

`feature`、`bugfix`、`operations` の作業項目には `affected_spec` が必要である。
`affected_spec` からは、規範シナリオまたは標準仕様の ID、もしくは TypeSpec のシンボルを直接参照する。
仕様に影響しない変更（`refactor`、`docs`、`tooling`、`maintenance`）では、次の形を使用できる。

```yaml
spec_impact: { kind: none, reason: "具体的な理由。" }
```

`affected_spec` の各項目には、規範要素との関係を `impact` で示す。

| `impact` | 意味 | 機械検査が求めるもの |
| --- | --- | --- |
| `modifies`（省略時の値） | 規範要素を追加、変更、廃止する | 作業範囲の仕様差分に、その規範要素が現れる |
| `conforms` | 規範要素を変えず、実装をそれに合わせる | 作業範囲の仕様差分にその規範要素が現れず、それを引くテストを同じ作業範囲で追加または変更している |

```yaml
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-001 }
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-002, impact: conforms }
```

`conforms` の項目を引くテストとは、規則またはその子の例を `//spec:covers` で引くテストである。
TypeSpec のシンボルには引く ID がないため、宣言名を含むテストファイルの追加または変更を求める。
作業範囲は、その作業項目の `status` が `pending` でなくなったコミットの親から作業ツリーまでである。
`impact` は、導入後に着手または完了する記録に書く。
完了済みの記録へ遡って書き加えない。

`spec_impact.reason` には、何が変わらないかを名指して書く。
「リファクタリングのみ」「振る舞いは変えない」のように、維持する結果、永続状態、イベント、外向きの呼び出しを一つも名指さない理由は、機械検査が拒否する。
`spec_impact: none` を宣言した作業範囲に規範仕様の差分があれば、同じく拒否する。
これらの検査は `mise run check` が行い、仕様影響の判断とコミットのトレーラーは[仕様先行の開発ワークフロー](../development/specification-first-workflow.md#仕様影響の宣言)が定める。

## 着手時の文脈と設計

最初に読む範囲は、着手時に[作業文脈の導出](../development/task-context.md)の `brief` で求め、記録には書かない。
`pending` の作業項目は、動機、対象範囲、対象外だけでも役に立つ。

`affected_spec` は未完了の記録で解決し、完了の直前に `move-work-item` がもう一度解決する。
規範要素を別のファイルへ移したら、未完了の記録は参照先を更新する。
廃止した場合は廃止済みの見出しを残して参照を解決できるようにする。
規範要素を取り除くまたは改名する記録は、旧名も `affected_spec` に挙げる。
`main` との差分が取り除いた要素として示す間は、その参照を解決できなくても受け入れる。
`work-items/done/` の縮めた記録は、変わったときに形だけを検査し、後のファイルの移動に追従させない。

中リスク以上の変更では、設計と計画の節を具体的に書く。
中核ロジックを変更する場合は、主要なドメインデータ型と操作のシグネチャを挙げる。
時刻、乱数、識別子生成、設定、永続化、通知などの作用は、明示的な入力、出力、またはポートとして計算との境界を記す。
分離によって規則をテストしやすく、または読みやすくできる場合は、決定論的な判断をデータ上の計算に保ち、ユースケースで作用を編成する。
これは変更するロジックの設計記録であり、リポジトリ全体へ純粋性やラッパー型の割り当て目標を課すものではない。
Domain、Use Cases、Adapters の各タスクには、自己証明となる対応テストと規範シナリオ ID を残す。

モジュールの境界を変える作業項目（モジュールの追加、分割と統合、公開範囲の拡大、モジュール間の依存の追加、テーブルの所有者または組み立て地点の変更）では、設計の節に境界の判断の記録を書く。
記録は、入力の参照、適用した判断の制約、候補ごとの制約への適合、変更シナリオの波及、採否と理由、残る仮定を含む。
手順と制約は[設計ガイドライン](../design/application/design-guidelines.md#境界を選ぶ判断手順)が定める。
境界を変えない局所的な変更には、この記録を求めない。

## 文書影響と成熟度

[公開状態](../development/release.md#公開状態と日常開発)が `published` の間に着手する作業項目は、`documentation_impact` を一つ宣言し、完了まで同じ構造化フィールドを保つ。初回公開前は宣言しない。
`level` には `none`、`release_note`、`upgrade_note`、`deprecation_notice`、`removal_notice` のいずれかを指定する。
`none` には具体的な理由が必要であり、リリース文書への参照を含めない。
それ以外の水準では、実装前にリリース文書の予定パスを宣言する。
完了時には、そのパスが存在し、作業項目名を記載し、`affected_spec` の要件または TypeSpec のシンボルへリンクしていなければならない。
リリース文書のファイル名には、識別番号とケバブケースの題名を含む作業項目の完全なファイル名から、拡張子を除いた部分を使う。
たとえば、`work-items/active/wi-<number>-start-task.md` のリリースノートには `docs/releases/changes/wi-<number>-start-task.md`、アップグレードノートには `docs/releases/upgrades/wi-<number>-start-task.md` を使う。
既存のリリース文書は、対応する作業項目が完了したときの名前を保つ。
この規則へ合わせるためだけに過去の記録を改名しない。
`upgrade_note`、`deprecation_notice`、`removal_notice` では、注目すべき差分と、必要な操作または互換性情報を読者へ示すため、両方の種類の文書が必要になる。
検査は、変更種別、規範仕様の差分、TypeSpec の非推奨指定、機能レジストリの成熟度の差分から最低水準を導く。
規範要素の追加と除去による水準は、その要素を `affected_spec` に挙げる記録だけに求める。
変更された記録のどれもその要素を挙げていなければ、差分を持つすべての記録に求める。
作成者は最低水準より強い水準を選べるが、弱い水準は選べない。

明示的に宣言した告知文書への参照は公開状態によらず検査する。

機能レジストリの差分で `experimental` から `preview`、または `preview` から `supported` へ昇格する場合は、完了の前に、昇格した機能ごとの `maturity_evidence` も記録する。
各項目には、正確な遷移、セキュリティ検査の結果、互換性情報または移行情報を書く。
新しい成熟度を示すリリース文書の `documentation` パスは公開後に必須とし、初回公開前は省略できる。
省略しても一次情報文書の成熟度表示と、昇格の検証は維持する。
該当する作業項目には引き続き `primary_use_cases` が必要である。

## 主要ユースケースの証拠

[主要ユースケースの証拠契約](../development/specification-first-workflow.md#主要ユースケースの証拠)に該当する作業では、実装前に `primary_use_cases` を追加する。
各項目には次のフィールドを記す。
中心となる利用経路の定義と、故障からの境界選択は[テスト方針](../development/testing.md#主要ユースケース)に従う。
同じ利用経路の内部判断と外部配線が別の故障なら、同じ要件を引く別の項目にする。

| フィールド | 記入する内容 |
| --- | --- |
| `id` | 安定したケバブケースの識別子 |
| `requirement` | 正確な `REQ-*` または標準要件 |
| `observable_result` | 利用者、永続状態、外部境界から見える最終結果 |
| `boundary` | `unit`、`adapter`、`acceptance`、`contract`、`property`、`e2e` |
| `test` | 一つのテストへの参照 |
| `fault_model` | そのテストで検出する現実的な故障 |
| `reason` | `e2e` の場合、下位境界で同じ故障を検出できない理由 |

テストへの参照には、リポジトリ相対の `path`、安定した `name`、必要な `mise` または CI の `task` を含める。
作業項目が `in_progress` の間は、予定したテストがまだ存在しなくてもよい。
完了の直前に `move-work-item` が、ファイルと識別子が存在し、テストソースが要件を参照し、宣言した標準タスクから到達できることを検査する。

## 完了

作業が完了したら、記録の末尾に `## 要約` の節を書き、`mise run move-work-item -- <id>` を実行する。
要約には、記憶ではなく `mise run spec-diff` の結果から読み取った意味上の差分と、後から読む人が知るべき判断を数行で書く。
中止するときも、中止の判断を `## 要約` に書き、`mise run move-work-item -- <id> --cancel` を実行する。

コマンドは、完了では次の順に処理し、どこかで失敗すれば何も書き込まない。

1. 全文に完了時の検査を適用する。`affected_spec` の解決、`primary_use_cases` のテストの実在、宣言した告知文書の実在と内容を確かめる。
2. 記録を、状態、題名、要約だけに縮める。リンクは文字だけを残す。
3. 縮めた記録を `work-items/done/` へ移し、ほかの Markdown から記録へのリンクを新しいパスへ直す。

RED の観測や検証の実行は、手順として行い、記録には書かない。
記録は自己申告であり、検査は書いたかどうかしか確かめられないためである。
実行で確かめられる事実は、上の完了時の検査と `mise run check-spec-impact` が確かめる。

縮めた記録の全文は、`git log -1 -- work-items/active/<ファイル名>` が示すコミット（記録を消したコミットならその親）で読む。
初期に縮めた記録は、全文のコミットを `archived_from` に書いてある。
縮めた記録へはアンカーなしでリンクする。
`depends_on` は配置を含まない識別子なので、移動で変わらない。
移動後に `mise run check-work-items` と `mise run check-links` を実行する。
