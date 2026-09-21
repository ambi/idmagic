# 作業項目フォーマット

作業項目は、一つの意味上の変更を説明、設計、実装、検証する作業単位である。
未完了の項目は `work-items/`、完了または中止した項目は `work-items/done/` に置く。
ファイル名には `wi-<識別番号>-<ケバブケースの題名>.md` を使う。
識別番号は、10000 から 99999 のうち、`work-items/` と `work-items/done/` のどちらにも現れない値を無作為に選ぶ。
`mise run work-item-number` がこの値を一つ出力する。
最大値に 1 を足す手順は使わない。
その手順は番号を配る主体が一つであることを前提にするが、各自がローカルで起票し、一人が並列のワークツリーで起票する運用では、まだ push されていない起票を見られないためである。
既存の記録は 3 桁以下に収まっているので、新規を 5 桁に限れば番号空間が重ならない。

作業項目は、タスクリスト、変更固有の設計文書、実装履歴も兼ねる。
完了時点でも有効な結論は、TypeSpec またはその種類の内容を扱う一次情報文書へ反映しなければならない。

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
evidence_policy: risk-based-v3 # 着手後は必須
documentation_impact: # 着手後は必須
  level: release_note
  reason: 新たにサポートする機能をリリースの読者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-48213-start-task.md }
initial_context: # 起票時ではなく着手時に記入する
  specification: [docs/domain/system/scenarios.feature.md#REQ-SYSTEM-001]
  typespec: [Product.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/system/main.tsp, symbol: Product.System.Operations.StartTask }
primary_use_cases: # feature、bugfix、standards.md の変更では着手後に必須
  - id: start-task
    requirement: REQ-SYSTEM-001
    observable_result: 呼び出し元がタスクの実行開始を観測できる。
    unit_test: { path: backend/system/usecases/start_task_test.go, name: TestStartTask_REQ_SYSTEM_001, task: test-go-race }
    e2e_test: { path: backend/system/e2e_test.go, name: TestE2E_StartTask_REQ_SYSTEM_001, task: test-go-race }
    unit_fault_model: ユースケースが開始コマンドを発行しない。
    e2e_fault_model: 構成済みの経路がハンドラーとユースケースを接続しない。
maturity_evidence: # 成熟度の昇格を検出した場合は完了時に必須
  - feature: start-task-v1
    from: preview
    to: supported
    security: セキュリティレビューで、対象ユースケースに未解決の統制不足がないことを確認した。
    compatibility: 既存の preview 設定は移行せずに引き続き受理される。
    documentation: docs/releases/changes/wi-48213-start-task.md
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
- `mise run verify`

## リスク
リスクと緩和策を書く。
```

節見出しは、上のテンプレートが示す日本語と、既存の記録が使う英語（`Motivation`、`Scope`、`Out of Scope`、`Plan`、`Tasks`、`Verification`、`Risk Notes`、`Completion`）のどちらでも機械検査が同じ項目として読む。
新しい記録はテンプレートの表記で書き、既存の記録を書き換えるためだけの変更はしない。

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

項目を `in_progress` にするときは `evidence_policy: risk-based-v3` を追加する。
リスクは完了までに必要な証拠を決めるが、push、merge、本番操作、外部システムの変更を許可するものではない。
作業の権限は項目の起票によって与えられるため、別の承認記録は設けない。
プロダクトの振る舞い、公開契約、採用する設計境界、タスク分割を変え得る問いは実装前に解決する。
実装中に規範の変更が必要だと分かった場合は仕様作業へ戻り、実装を通すためにシナリオを弱めてはならない。
リスクと証拠の対応は[仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md#4-証拠の要件)が定める。

`feature`、`bugfix`、`operations` の作業項目には `affected_spec` が必要である。
`affected_spec` からは、規範シナリオまたは標準仕様の ID、もしくは TypeSpec のシンボルを直接参照する。
仕様に影響しない変更（`refactor`、`docs`、`tooling`、`maintenance`）では、次の形を使用できる。

```yaml
spec_impact: { kind: none, reason: "具体的な理由。" }
```

`initial_context` は、一人のエージェントが最初に読む対象の一覧である。
起票時ではなく、作業項目を `in_progress` にするときに書く。
バックログにある間に書いた一覧は着手前に古くなり、移動または削除されたファイルを指す一覧は、一覧がない状態より作業を誤らせる。
`pending` の作業項目は、Motivation、Scope、Out of Scope だけでも役に立つ。

作業項目を `in_progress` にすると、`mise run check-work-items` が一覧を解決する。
すべてのパスが存在し、`docs/domain/<context>/scenarios.feature.md#REQ-<CONTEXT>-NNN` の項目が、その文書で宣言されたシナリオを指さなければならない。

`affected_spec` は、完了済みを含むすべての記録で解決する。
これは当時読んだものではなく、変更が触れた規範要素の索引だからである。
規範要素を別のファイルへ移したら参照先を更新し、廃止した場合は廃止済みの見出しを残して参照を解決できるようにする。

中リスク以上の変更では、`Design` と `Plan` を具体的に書く。
中核ロジックを変更する場合は、主要なドメインデータ型と操作のシグネチャを挙げる。
時刻、乱数、識別子生成、設定、永続化、通知などの作用は、計算へ入る境界または計算から出る境界を明示する。
Domain、Use Cases、Adapters の各タスクには、自己証明となる対応テストと規範シナリオ ID を残す。

`in_progress` になる作業項目は、`documentation_impact` を一つ宣言し、完了まで同じ構造化フィールドを保つ。
`level` には `none`、`release_note`、`upgrade_note`、`deprecation_notice`、`removal_notice` のいずれかを指定する。
`none` には具体的な理由が必要であり、リリース文書への参照を含めない。
それ以外の水準では、実装前にリリース文書の予定パスを宣言する。
完了時には、そのパスが存在し、作業項目名を記載し、`affected_spec` の要件または TypeSpec のシンボルへリンクしていなければならない。
リリース文書のファイル名には、識別番号とケバブケースの題名を含む作業項目の完全なファイル名から、拡張子を除いた部分を使う。
たとえば、`work-items/wi-48213-start-task.md` のリリースノートには `docs/releases/changes/wi-48213-start-task.md`、アップグレードノートには `docs/releases/upgrades/wi-48213-start-task.md` を使う。
既存のリリース文書は、対応する作業項目が完了したときの名前を保つ。
この規則へ合わせるためだけに過去の記録を改名しない。
`upgrade_note`、`deprecation_notice`、`removal_notice` では、注目すべき差分と、必要な操作または互換性情報を読者へ示すため、両方の種類の文書が必要になる。
検査は、変更種別、規範仕様の差分、TypeSpec の非推奨指定、機能レジストリの成熟度の差分から最低水準を導く。
作成者は最低水準より強い水準を選べるが、弱い水準は選べない。

機能レジストリの差分で `experimental` から `preview`、または `preview` から `supported` へ昇格する場合は、完了時に昇格した機能ごとの `maturity_evidence` も記録する。
各項目には、正確な遷移、セキュリティ検査の結果、互換性情報または移行情報、新しい成熟度を示すリリース文書のパスを書く。
該当する作業項目には引き続き `primary_use_cases` が必要である。
成熟度の証拠は、Unit RED、E2E RED、フォールト注入の結果を置き換えない。
この契約より前に書かれた完了記録は履歴であり、再解釈しない。

`feature`、`bugfix`、および `affected_spec` から `standards.md` の要件を参照する作業項目では、実装前に `primary_use_cases` を追加する。
各項目では、中心となる一つの正常経路について、安定したケバブケースの `id`、正確な `REQ-*` または標準要件、最終的な `observable_result`、Unit テストと E2E テストへの参照、各テストが検出すべき互いに異なる現実的な障害を宣言する。
テストへの参照には、リポジトリ相対の `path`、安定した `name`、必要な `mise` または CI の `task` を含める。
作業項目が `in_progress` の間は、予定したテストがまだ存在しなくてもよい。
完了時には、ファイルと識別子が存在し、テストソースが要件を参照し、宣言した標準タスクから到達できることを検査する。
本番がより外側の入口と構成経路を使う場合は、入力の受理、列挙値の検証、行カバレッジ、直接構築した下位コンポーネントを E2E の結果にしない。

`risk-based-v3` は、新しく着手する作業にこの契約を適用する。
完了済みの `risk-based-v1` と `risk-based-v2` の記録は、有効な履歴として残す。
導入時点ですでに `in_progress` だった該当項目は v3 へ移行して計画を追加するが、完了済みの記録は書き換えない。

作業が完了したら `status` を `completed` にし、次の節を追加して、ファイルを `work-items/done/` へ移す。

```markdown
## 完了
- **Completed At**: 2026-01-01
- **Summary**:
  記憶ではなく `mise run spec-diff` の結果から読み取った、この作業による意味上の差分。
- **Acceptance RED Evidence**:
  - **Test**: 実装前に失敗を観測したテストまたは検査。
  - **Requirement**: `REQ-CONTEXT-NNN`。規範となる製品要件がない作業では `N/A: <理由>`。
  - **Observed Failure**: 実際に観測した、想定どおりの失敗。
  - **Detection Reason**: その表明によって、現実的な誤実装と要求された振る舞いを区別できる理由。受け入れ境界が該当しない場合は、実際に失敗した代替検査を示す。
- **Unit RED Evidence**:
  - **Test**: 実装前に失敗を観測したテストまたは検査。
  - **Requirement**: `REQ-CONTEXT-NNN`。規範となる製品要件がない作業では `N/A: <理由>`。
  - **Observed Failure**: 実際に観測した、想定どおりの失敗。
  - **Detection Reason**: その表明によって、現実的な誤実装と要求された内部の振る舞いを区別できる理由。単体境界が該当しない場合は、実際に失敗した代替検査を示す。
- **Change-Resistance Results**:
  中リスク以上では、代表的な誤実装、差分による変異、または明示的なフォールト注入と、それをテストが検出したかを記録する。
  高リスクまたは重大リスクの純粋ロジック変更では、代表例が一つだけでは足りない。
  変更したロジックへ体系的に変異を加えるか、全体に明示的な障害を注入し、等価な変異と手法の限界を隠さず記録する。
  変異ツールは、既存のトークンを書き換える体系的な部分を担う。
  振る舞いを追加、削除、転送する変異は手作業で行う。
  この分担と、生き残った変異を点数化せず読む方法は、[仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md)で定める。
- **Verification Results**:
  - `mise run verify` - 成功
```

上の Acceptance RED と Unit RED のフィールドは、主要ユースケースの要件がない作業で使う完了形式である。
該当する `risk-based-v3` の作業項目では、代わりに次のフィールドを使う。
主要でない振る舞いの Acceptance 証拠または Unit 証拠を追加で残してもよいが、次の証拠の代わりにはならない。

```markdown
- **Primary Use Case Evidence**:
  - id: start-task
    unit_red: 開始コマンドを発行しないため、TestStartTask_REQ_SYSTEM_001 が失敗した。
    e2e_red: 構成済みの経路から実行中のタスクが生成されないため、TestE2E_StartTask_REQ_SYSTEM_001 が失敗した。
    unit_fault_injection: コマンドの発行を削除すると、TestStartTask_REQ_SYSTEM_001 が失敗した。
    e2e_fault_injection: 経路の接続を外すと、TestE2E_StartTask_REQ_SYSTEM_001 が失敗した。
```

`id` は、`primary_use_cases` の計画項目と一致させる。
計画した各項目には、完了項目がちょうど一つ必要である。
RED とフォールト注入の各結果には、将来の指示ではなく、実際に観測した内容を書く。
