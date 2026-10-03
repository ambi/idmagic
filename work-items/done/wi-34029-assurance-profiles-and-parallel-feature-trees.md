---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  references: []
  reason: "開発用の証拠契約と探索方法だけを変更し、製品の利用方法とアップグレード手順は変えない。"
initial_context:
  specification: [SPECIFICATION_FORMAT.md, WORK_ITEM_FORMAT.md, DOCUMENTATION_GUIDE.md, docs/development/specification-first-workflow.md, docs/development/testing.md]
  typespec: []
  source: [tools/check/src/primary-use-case-evidence.ts, tools/render-docs/src/render.ts, backend/idmanagement/group/usecases/admin_groups.go]
  tests: [backend/idmanagement/group/usecases/group_rules_test.go, backend/idmanagement/group/testing_contract/contract.go]
  stop_before_reading: [frontend, backend/oauth2]
affected_spec:
  - { path: docs/domain/identity-management/groups/group/README.md, requirement: REQ-IDMANAGEMENT-060, impact: modifies }
---

# 設計情報とテストの検出能力を増やしながら、重複記述と読み取り範囲を減らす

## 動機

設計文書の情報も、テストのカバレッジも、現状では不足している。
内容を削ったり、未検証の規則を検査から外したりして小さくすることは、この作業の目的ではない。
不足を補っても、手で維持する記述量と変更のたびに読む範囲が比例して増えない構成を作る。

Context 内では仕様、TypeSpec、実装、テストの粒度も揃っていない。
同じ feature の変更でも複数の大きな文書とアダプターの集積場所を横断するため、必要な範囲を特定しにくい。

## 対象範囲

- 一機能で、入力と期待結果を一度だけ記述し、テスト実行と文書の例の表示で共有する試行を行う。
- 同じポートの複数実装には一つのテスト契約を使い、同じ業務規則を各層で再記述しない。
- 機能の公開契約から必要な内部設計へたどれる入口を作り、変更に必要な読み取り範囲を測る。
- 維持する規範 ID、観測結果、拒否後の無作用と、独立した故障の検出能力を変更前後で比較する。
- 発見の還元、選択した故障モデルに対する最小境界、並行した機能配置と生成索引を開発手順へ統合する。

## 対象外

- 自然言語の規則またはシナリオから製品コードや製品テストを生成すること。
- 既存 Context の文書、TypeSpec、実装を一斉に移動すること。
- 製品の API、認証、認可、永続データ、実行時の振る舞いを変えること。
- 未検証の規則の免除、期待結果を本番実装から算出するテスト、設計理由の削除。
- あらゆるシナリオを表現する新しい DSL または汎用ステップ実行基盤の導入。

## 設計

### 一つの事実の維持箇所を一つにする

人が書く設計は、責務、不変条件、判断理由、依存方向、故障時の振る舞いを扱う。
契約の項目一覧、具体的な例の値、対応するテストの一覧は、それぞれの一次情報から表示する。
短い文書へ言い換えるだけでは不足を解消しないので、必要な説明は残し、別の一次情報と同じ内容だけを派生表示にする。

具体例の入力と期待結果は、人が仕様に基づいて決める一つのデータとして保持する。
テストはそのデータを使って実物を呼び、独立した期待結果と比較する。
文書の具体例は同じデータから表示し、Gherkin と Go へ値を二重に転記しない。
一般則とその理由は文章に残す。
非同期処理や複雑な操作列を無理にデータへ押し込まず、コードで表現した方が短いものは既存のテストに残す。

### 検証本体を再利用する

ポートの共有契約を複数の実装に実行する方式は、既存の `backend/idmanagement/group/testing_contract/contract.go` にある。
この方式を再利用し、業務規則の表、アダプター固有の変換、正式な入口の配線をそれぞれの所有箇所で検査する。
同じ入力を異なる境界から実行する必要がある場合も、期待結果と共通の表明を再利用する。
外側でしか観測できないエラー変換、認可、テナント隔離、永続化、最終作用の検査は残す。
共通化して行数が減ることと、検出できる故障が維持されることは別々に確認する。

### 読み取り範囲を機能の契約から絞る

並行ツリーは所有する機能を探す補助であり、全ファイルの深さをそろえること自体は目的にしない。
既存の機能グループ、名前照合、共有契約と共有設計を維持する。
機能の入口から、公開契約と不変条件、必要な場合だけ内部の判断理由へ進めるようにする。
内部実装を変更しない作業まで、詳細設計と全テストを読むことを要求しない。

### 試行と未決事項

候補は Group の入力検証と正規化である。
`REQ-IDMANAGEMENT-060` のメールアドレスの具体値は、本文、例、Go テストへ重複して記述されている。
Group の属性の拒否は、ユースケースと HTTP の両方に同種の例がある。
一次情報は実装に隣接する `testdata/*.examples.json` とする。
Go と TypeScript が既存の JSON デコーダーで同じ値を読むため、新しい言語のパーサーを導入しない。
既存の Go テスト表を直接文書化する案は、Go AST を別のランタイムから抽出する固定費が今回の小さな表より大きいため採用しない。
各例は `id`、`title`、`given`、`input`、`expected` だけを持ち、実行する操作は Go テスト自身に置く。
汎用の操作名、ステップ、式評価をデータへ持ち込まない。
`renderExampleCases(cases)` は Gherkin の具体例を決定的に表示する。
`updateExampleBlocks(document, read)` は文書内の一次情報への参照を解決し、派生区間を置き換える。
標準検査は再生成との差を拒否し、通常の EX 被覆検査と spec-diff は派生文書を従来どおり読む。
Go の `TestGroupNormalizationExamples` は同じ表を作成と更新の両方へ実行し、保存値、拒否後の状態、イベントと通知を観測する。
複雑な既存テストと永続化の共有契約は維持する。

従来案の保証区分と全ツリーの配置強制は採用しない。
証拠契約は新規着手向けの `risk-based-v4` として、検出する故障と選択した境界を一組にし、固定 Unit/E2E 対を廃止する。
旧バージョンの記録は再解釈しない。
別の保証分類や台帳は作らず、既存の `primary_use_cases` と完了証拠の項目数を減らす。
`verifyPrimaryUseCaseEvidence(record, environment)` を拡張し、計画と結果の id、テスト、標準タスク、実際の RED と故障注入を照合する。
E2E の選択には下位境界では検出できない理由を要する。
機能索引は既存の feature map と contextAliases を再利用し、手書きの第二の対応台帳を作らない。
Regenerative Architecture の目的は、独立した期待結果を失わずに、復元できる情報を手で重ねて維持しないこととする。

## 計画

1. 一機能の重複記述、検査する振る舞い、変更に必要な読み取り範囲を変更前に測る。
2. 一次情報の置き場所と最小の実行方法を決め、元の保証区分の草案を取り込み対象から外す。
3. 具体例の共有と派生表示を試し、独立した期待結果と既存 ID を維持する。
4. 不足するケースを足し、同じ規則の複数実装と必要な外側の検証へ再利用する。
5. 手書きの維持量、読み取り量、実行時間、被覆と故障検出を比較し、効果のある変更だけを一般化する。

## タスク

- [x] T001 [Design] 設計情報と検出能力を減らさない条件で、試行の一次情報と実行方法を決める。
- [x] T002 [Baseline] 重複記述、読む範囲、未被覆の振る舞いを測り、維持する観測結果を列挙する。
  変更前の正規化は、手書きの文書 3 例と Go の専用テスト 25 行で具体値を二重に維持していた。
  新規テストを除くパッケージ被覆は `mise run test-go-cover -- ./backend/idmanagement/group/usecases TestGroupNormalizationExamples` で 64.8%。
  保存値、名前衝突、メールアドレスの正規化、空白だけの説明を維持し、作成と更新、拒否後の状態と通知へ観測を広げる。
- [x] T003 [Acceptance] 例の表示と実行の不一致と、選択した証拠の欠落を検出する最小の RED を確認する。
  Acceptance RED：表示関数を空実装にすると executable-examples の 4 テストが失敗。機能地図の実装候補がないことでも失敗。
  Unit RED：v4 の検査を適用しない旧実装では、計画が空でも通るため `primary-use-case-evidence.test.ts` の新規テストが失敗。
- [x] T004 [App] 具体例と検証本体を共有し、不足する検証を追加する。
  JSON の 9 例を作成と更新の 18 通りへ実行する一つのランナーに統合し、旧テストの重複した値を除去した。
  v4 は別の台帳を作らず、既存の証拠フィールドで最小境界を選べる。v3 の記録とリリース文書の検査は維持した。
  機能地図は配置から候補を生成し、具体例の付録は値の表として表示する。
- [x] T005 [Verify] 被覆と故障検出を保ち、具体値の維持量と読み取り範囲の削減を固定費と分けて測る。
  具体値の維持先は文書と Go テストの 2 か所から JSON の 1 か所になった。
  同じ最初の 3 例の値を確認する範囲は、旧文書 18 行と旧専用テスト 25 行の計 2,044 bytes に対して、JSON の該当行と配列の囲み 5 行で 835 bytes になった。
  これは例の値の追加と修正に限った比較であり、トークン数、初回の理解時間、全リポジトリのコンテキスト削減を測ったものではない。
  全 9 例のデータは 11 行で、操作と表明を持つランナーは 128 行の固定費として増えた。
  生成器も新設したため、今回の総コード量は減っていない。
  パッケージのカバレッジは 64.8% から 64.9%、`normalizeGroupEmail` は 90% から 100%、`CreateGroup` は 80% から 82.9% になり、`UpdateGroup` は 87.7% を維持した。
  パッケージ実行時間は比較前 0.663 秒、比較後 0.692 秒で、キャッシュなどの条件を統制していないため速度改善とは判断しない。
  通知の配線を外すと成功する 5 例の作成と更新、計 10 通りが失敗し、元に戻すと通過した。
  生成文書の一致判定と v4 の証拠検査をそれぞれ無効化しても、対応する回帰テストが失敗した。
  最終レビューで、表の値だけの変更を spec-diff が見逃すことを発見した。
  値の読み取りを省くと、値だけの変更、入力と期待結果の対応、決定表の 3 テストが失敗した。
  公式解析器が返す表、DocString、Outline の行を例ごとに比較し、操作順を保持する修正後は関連 36 テストが通過した。

## 検証

- `mise run test-tools`
- `mise run typecheck-tools`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run check-agent-guidance`
- `mise run verify`

## リスク

- 共通化が故障の観測を消す可能性がある。
  境界固有の観測と故障注入を維持し、行数だけでは成功としない。
- データ形式と生成器が別の維持負担になる可能性がある。
  一機能の具体例だけで試し、汎用のシナリオ言語へ広げない。
- 設計を短くする過程で判断理由を失う可能性がある。
  削除するのは一次情報から復元できる内容に限り、理由と見直し条件を保持する。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff -- main` が示す規範差分は `REQ-IDMANAGEMENT-060` のみであり、TypeSpec と標準要件の差分はない。
  正規化の具体例を 3 例から 9 例へ増やし、独立した入力と期待結果を一つの JSON から仕様の表示と作成と更新の 18 通りへ渡した。
  保存値、拒否後の状態、イベントと通知を検査し、製品コードの振る舞いは変えていない。
  一般則と判断理由は文書に残し、実装とテストで得た発見を、規則、具体例、記述的な参照、未決事項へ還元する手順を統合した。
  新しい証拠契約は故障を検出できる最小境界を選び、Unit/E2E の固定対と証拠欄の重複を減らした。
  着手済みの v3、境界固有の検査、既存の共有テスト契約とリリース文書の検査は維持した。
  機能名と既存の配置から実装、契約、テスト、具体例の候補を生成し、手書きの対応台帳と一斉移動を避けた。
  表の値だけの変更と入力に対応する期待結果の変更も、親の規則の仕様差分へ返すようにした。
  削減を確認できたのは例の値の維持先と読み取り範囲であり、生成器とランナーの固定費を含む総コード量の削減は主張しない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/executable-examples.test.ts render-docs/src/render.test.ts`
  - **Requirement**: N/A: 開発用の派生表示と探索の仕組みには製品の受け入れ境界がないため、表示と探索用の検査を代替とした。
  - **Observed Failure**: 表示関数が空の状態で具体例の 4 テストが失敗し、機能地図に実装候補がない状態でも候補を要求するテストが失敗した。
  - **Detection Reason**: 読める具体値と、所属する実装、共有契約、テストへの候補を表明するため、見出しだけの表示と探索先の欠落を区別できる。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/primary-use-case-evidence.test.ts`
  - **Requirement**: N/A: 証拠契約の検査は製品の内部判断ではないため、契約を適用する開発ツールの局所テストを代替とした。
  - **Observed Failure**: v4 を適用しない旧実装では、空の計画が受理され、新しい検査を要求するテストが失敗した。
  - **Detection Reason**: 宣言したバージョンを無視する検査と、故障に応じた境界、テスト、RED と故障注入結果を照合する検査を区別できる。
- **Change-Resistance Results**:
  通知の配線を外すと `TestGroupNormalizationExamples` の 10 通りが失敗し、復元後は 18 通りすべて通過した。
  生成文書の一致判定を無効化するとドリフト検査のテストが失敗し、v4 の証拠検査を迂回すると計画の欠落を検出するテストが失敗した。
  spec-diff から表の値の比較を外すと、値だけの変更と入力との対応を固定するテストが失敗した。
  各故障は復元済みであり、製品の純粋ロジックは変更していないため Go の構文変異は行っていない。
  生成には、区切り、改行、引用、マーカーに似た値を含む 256 個の再現可能な入力と境界値で再生成の一致を検査した。
- **Verification Results**:
  - `mise run verify`：修正後のツリーで成功。開発ツール 823 テスト、Go の競合検査、UI の単体テスト、型検査、静的検査、ビルドを含む。
  - `mise run test-tools-file -- check/src/executable-examples.test.ts check/src/spec-diff.test.ts check/src/gherkin-scenarios.test.ts`：36 テスト成功。
  - `mise run test-go-test -- ./backend/idmanagement/group/usecases TestGroupNormalizationExamples`：作成と更新の 18 通りが成功。
  - `mise run test-go-cover -- ./backend/idmanagement/group/usecases`：成功。パッケージのカバレッジは 64.9%。
  - `mise run spec-render`：1,216 ページを生成し、303 文書、342 操作、19 API tag、908 TypeSpec symbol の検査に成功。
  - `mise run check-spec`、`mise run check-api-compat`、`mise run check-boundaries`、`mise run check-agent-guidance`：成功。
  - `mise run spec-diff -- main`：変更した規則は `REQ-IDMANAGEMENT-060` のみ。
  - `mise run check-work-items`：完了記録を含む依存関係と証拠の検査に成功。
  - ブラウザー E2E は製品の UI と実行経路を変更していないため対象外とし、文書の生成と表示は renderer のテストと生成サイトの検査で確認した。
