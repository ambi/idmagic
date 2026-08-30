---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-30
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
reversibility: irreversible
initial_context:
  source:
    - docs/development/testing.md
    - docs/development/specification-first-workflow.md
    - WORK_ITEM_FORMAT.md
    - tools/check/schemas/work-item.schema.json
    - tools/check/src/main.ts
    - tools/check/src/work-item-references.ts
    - tools/check/src/spec-diff.ts
    - tools/workspace/src/check-workspace.ts
    - mise.toml
    - .agents/skills/implement-work-item/SKILL.md
    - work-items/done/wi-439-outbound-scim-resource-body-omits-schemas.md
    - work-items/done/wi-440-outbound-scim-oauth2-client-credentials-unimplemented.md
    - work-items/done/wi-441-push-groups-produces-no-delivery.md
  tests:
    - tools/check/src/lib.test.ts
    - tools/check/src/work-item-markdown.test.ts
    - tools/check/src/work-item-references.test.ts
    - tools/check/src/spec-diff.test.ts
  stop_before_reading: [backend, frontend, spec]
spec_impact: { kind: none, reason: "機能と標準対応に要求するテスト証拠を強化するだけで、製品契約の意味は変えない。検査が見つけた製品欠陥は個別に直す。" }
---

# 機能と標準対応の主要ユースケースに単体テストと E2E テストを要求する

## Motivation

`mise run verify` を全部通しながら、仕様にある機能が実際の利用経路では動かない欠陥が残っていた。`wi-440` では `oauth2_client_credentials` に触れるテストが列挙値の妥当性しか検査せず、機能を実装した後に追加した受け入れテストも認証方式から供給元を選ぶ分岐を迂回していた。`wi-441` では `push_groups=true` を受理しても Group の配信を生む配線が無く、`wi-439` では属性対応付けを設定できても必須の `schemas` を出力する手段が無かった。

この問題は、設定値と列挙型に限らない。機能名や規範 ID がテストに現れること、対象行が実行されること、下位の部品を直接組み立てたテストが成功することのいずれも、利用者が使う入口から要求された効果まで到達することを示さない。現在の証拠契約も Acceptance RED と Unit RED を要求するが、製品機能の主要ユースケースを E2E で固定することや、その E2E が実配線を迂回していないことまでは要求していない。

機能または標準への対応を実装するときは、その変更が成立したと判断する中心的な正常系を主要ユースケースとして明示し、内部の判断を単体テストで、正式な入口から最終的な観測結果までの接続を E2E テストで検証する必要がある。二つのテストは同じ成功を重ねて数えるのではなく、単体テストは業務規則やユースケースの分岐、E2E テストは設定、構成、アダプター、経路制御を含む実際の組み立てを担当する。

テストの存在だけを検査しても、`Valid()` を呼ぶだけのテストや常に成功する E2E を増やせる。各主要ユースケースについて、内部判断を壊した実装を単体テストが検出し、配線を外すか効果を無作用にした実装を E2E テストが検出することを実測して初めて、「機能を実装したことを確認できるテスト」と判断する。

## Scope

- `docs/development/testing.md` と `docs/development/specification-first-workflow.md` に、主要ユースケースの選び方と、単体テストと E2E テストが担う別々の責任を定める。
- `change_kind: feature` と `change_kind: bugfix` の work item、および `affected_spec` が `standards.md` の規範 ID を参照する work item を対象にする。標準対応は `change_kind` にかかわらず対象とする。
- 対象の work item は実装前に、各主要ユースケースが対応する `REQ-*` または規範 ID、観測する結果、予定する単体テストと E2E テスト、両テストがそれぞれ検出する誤実装を構造化して宣言する。
- 単体テストは、変更するドメイン規則またはユースケースの分岐を公開されたモジュール境界から実行し、戻り値だけでなく状態遷移または外へ出す効果を表明する。
- E2E テストは、製品が対応すると宣言する入口から開始し、対象の機能を選ぶ設定、構成、経路制御を迂回せず、利用者、永続状態、または外部境界から観測できる最終結果を表明する。外部サービスをテストダブルにするときも、送信先、方式、本文など、この製品が外へ出した効果を検査する。
- 完了時には主要ユースケースごとに Unit RED と E2E RED の結果を記録し、内部判断を壊した故障と配線または最終効果を壊した故障を注入して、対応するテストが失敗した結果を残す。
- `WORK_ITEM_FORMAT.md`、work item スキーマ、Markdown の解析と検査を更新し、対象項目に主要ユースケースの計画または完了証拠が無ければ `mise run check-work-items` を失敗させる。
- 完了証拠が参照するテストが実在し、対応する `REQ-*` または規範 ID をテスト名かテストコードに持ち、必須の `mise` タスクまたは CI 検査から実行されることを機械検査する。実装前の計画には予定するテスト識別子を要求するが、まだ作成していないテストの実在は要求しない。
- 適用開始時に進行中の対象 work item を新しい証拠方針へ移し、その過程で判明したテスト不足または製品欠陥は許可リストへ入れず、個別の work item として切り出す。
- `wi-439`、`wi-440`、`wi-441` の修正前の状態を回帰用の検査材料にし、設定値に固有の規則ではなく、主要ユースケースの証拠が不足している例として検出できることを確認する。

## Out of Scope

- 全分岐、全代替経路、全拒否を E2E テストにすること。本 work item が必須にするのは、機能または標準対応が成立したと判断する主要ユースケースである。追加のシナリオはリスクと仕様に従って各水準へ置く。
- 行カバレッジと分岐カバレッジの閾値。量としての被覆は `wi-131` が扱い、本 work item は主要ユースケースに対するテストの責任と検出能力を扱う。
- `REQ-*` と規範 ID を名指しするテストの全件被覆。参照の網羅性は `wi-418` が扱い、本 work item は機能または標準対応を実装する変更の主要ユースケースに限定する。
- ドキュメント、ツール、純粋なリファクタリングに一律で E2E テストを要求すること。ただし、`standards.md` の規範に対応する変更は `change_kind` にかかわらず対象にする。
- 汎用の変異試験基盤を導入すること。代表的な誤実装を一時的な差分または明示的な故障注入で検出できればよく、高リスク以上に既存の証拠契約が要求する体系的な変異試験は維持する。
- 検査の実装または移行で見つけた個別の製品欠陥を本 work item で修正すること。
- 完了済みの全機能と全標準対応へ新しい証拠形式を遡及適用すること。完了済み記録は当時の証拠方針の履歴として保持し、具体的な不足を確認した場合だけ後続の work item を作る。

## Design

主要ユースケースは「その機能または標準対応を実装済みと判断するために成功しなければならない、中心的な利用経路」と定義する。単に入力が受理されること、列挙値が妥当であること、ハンドラーが成功ステータスを返すことは、それ自体が製品の提供する効果でない限り主要ユースケースの結果にしない。主要ユースケースは `REQ-*` または規範 ID に結び付け、work item 内だけの自由記述を仕様の代わりにしない。

単体テストと E2E テストは異なる故障を受け持つ。単体テストは、主要ユースケースを決定するドメイン規則またはユースケースの分岐を最小のモジュール内で実行する。E2E テストは、正式な外部入口から本番の構成経路を通り、最終効果まで到達することを検査する。たとえば設定で認証方式を選ぶ機能では、供給元を直接構築したテストは単体テストにはなり得ても E2E の証拠にはならない。E2E は設定の読み込みと方式の選択を通り、外部へ送った認証要求まで観測する必要がある。

E2E を必須にする理由は、単体テストと同じ表明を重複させるためではない。どの製品機能にも、内部判断が正しくても入口、構成、アダプター、通知先のいずれかが接続されず、利用者へ効果が届かない故障がある。`docs/development/testing.md` にある「下位の水準で同じ失敗を検出できない理由」は、この接続の故障を E2E が担当することによって満たす。

実装前の構造化宣言には、安定した要求参照、主要ユースケースの結果、単体テストの識別子、E2E テストの識別子、単体側の故障モデル、E2E 側の故障モデルを持たせる。完了証拠には、それぞれの RED で観測した失敗と、故障注入時の失敗結果を持たせる。既存の `Acceptance RED Evidence` と `Unit RED Evidence` をどう移行するかは、完了済み記録を再解釈しない版付きの証拠方針として設計する。

機械検査が判断するのは、適用対象、要求参照の解決、テスト参照の存在、必須タスクからの到達、RED と故障注入の記録である。アサーションの意味をソースの字面から推測しない。テストの検出能力は、宣言した誤実装を実際に与えたときに失敗した結果で判定する。これにより、テスト名や規範 ID の記載だけで門を通る方法を避ける。

標準対応の適用漏れは二方向から検出する。着手時は `affected_spec` に `standards.md` の規範 ID があれば証拠計画を要求し、完了時は `mise run spec-diff` が追加または変更した規範行と `affected_spec` を突き合わせる。既存の規範に対する実装修正では仕様本文を変更しなくても、その規範 ID を `affected_spec` に残す。

既存項目を一度に形式だけ満たす移行は行わない。新しい証拠方針を適用する変更から fail-closed にし、監査で見つけた既存の不足は主要ユースケースと観測結果を特定した個別の work item にする。完了済み記録は旧方針の履歴として読み続けられるようにする。

新しい版は `risk-based-v3` とする。`risk-based-v1` と `risk-based-v2` の完了済み記録はそのまま受理し、新たに着手する項目は v3 を使う。対象項目は frontmatter の `primary_use_cases` に計画を置き、完了時に `Completion` の `Primary Use Case Evidence` へ結果を置く。主要型は `TestReference = { path: string; name: string; task: string }`、`PrimaryUseCasePlan = { id; requirement; observable_result; unit_test; e2e_test; unit_fault_model; e2e_fault_model }`、`PrimaryUseCaseEvidence = { id; unit_red; e2e_red; unit_fault_injection; e2e_fault_injection }` とする。検査の主操作は `verifyPrimaryUseCaseEvidence(record, environment): string[]` とし、ファイル読み取りと必須タスク集合は `environment` から受け取る。したがって検査判断は純粋で、ファイルシステム、`mise.toml`、CI 定義の読み取りは作業空間アダプターに閉じる。

対象条件は `change_kind` が `feature` または `bugfix`、あるいは `affected_spec[].path` が `standards.md` で終わることである。`in_progress` では計画の構造と要求参照だけを検査し、予定テストは未作成でもよい。`completed` ではテストファイルと識別子、テストコード内の要求 ID、標準検証または CI から到達する `mise` タスク、計画と一対一に対応する RED と故障注入結果を検査する。アサーションの意味はソースから推測しない。

本 work item 自身は製品機能でも標準対応でもないため、主要ユースケースは `N/A` である。Acceptance RED は主要ユースケース計画の無い対象 fixture を `mise run check-work-items` が現在は受理すること、Unit RED は新しい純粋検査の期待を `mise run test-tools` が満たさないこととする。Markdown の構造化結果は既存の `Bun.YAML` へ委ね、手書きの入力分割規則を増やさないため、新しい fuzz 対象は不要である。

## Plan

1. 主要ユースケース、単体テスト、E2E テスト、故障モデルを `docs/development/testing.md` と証拠契約へ定義し、既存の Acceptance RED と Unit RED からの移行規則を決める。
2. work item の実装前計画と完了証拠を表す構造をスキーマと Markdown 形式へ追加し、適用条件を `feature`、`bugfix`、`standards.md` 参照として実装する。
3. 検査を RED で追加し、主要ユースケース未宣言、単体テスト欠落、E2E テスト欠落、未解決の要求参照、必須タスクから到達しないテスト、故障注入結果の欠落をそれぞれ落とす。
4. `wi-439`、`wi-440`、`wi-441` の修正前の形を検査材料にし、値やフラグの種類を列挙しなくても証拠不足として検出できることを確認する。
5. 適用開始時に進行中の対象 work item を新しい方針へ移し、完了済み記録を変更せず検査できることを確認する。移行中に見つけたテスト不足と実装欠陥は分けて個別の work item にする。

## Tasks

- [x] T001 [Docs] 主要ユースケースに対する単体テストと E2E テストの責任、表明対象、故障モデルをテスト方針と証拠契約へ定める。
- [x] T002 [Design] `risk-based-v3`、`PrimaryUseCasePlan`、`PrimaryUseCaseEvidence`、既存記録の移行規則、効果境界を確定する。
- [x] T003 [Acceptance] `check-workspace --work-items > rejects an applicable in-progress item without a primary-use-case plan` を追加し、`wi-439` 相当の feature fixture に `primary_use_cases` が無くても CLI が終了コード 0 を返すため、`expect(result.code).not.toBe(0)` が失敗する Acceptance RED を確認した。製品の規範要求は `N/A` で、観測境界は work item CLI である。Unit RED は `verifyPrimaryUseCaseEvidence` の単体検査を追加し、`Cannot find module './primary-use-case-evidence.ts'` で失敗することを確認した。
- [x] T004 [Tooling] `verifyPrimaryUseCaseEvidence` と作業空間アダプターで、適用条件、`affected_spec` との要求照合、単体テスト、E2E テスト、標準 verify / CI の必須タスクからの到達を検査する。
- [x] T005 [Tooling] 主要ユースケースごとの Unit RED、E2E RED、内部判断と実配線の故障注入結果を完了条件として検査し、Markdown の構造化 YAML を既存パーサーへ接続した。
- [x] T006 [Regression] `wi-439`、`wi-440`、`wi-441` の修正前の形を、設定値の種類を判定しない三つの fixture として主要ユースケース未宣言で検出することを確認した。
- [x] T007 [Migration] 適用時点で進行中の対象 work item は存在しなかった。完了済み `risk-based-v1` / `risk-based-v2` 記録を再解釈せず、新規着手だけを `risk-based-v3` に固定した。
- [x] T008 [Verify] 適用条件、計画、要求参照、単体テストと E2E テストの分離、テスト実在性、識別子、必須タスクからの到達、四つの完了証拠を fixture から一つずつ欠落させて失敗を確認した。タスク到達性の判定を一時的に無効化すると完了 fixture の検査が失敗することも実測し、復元後に `mise run verify` を通した。

## Verification

- `mise run check-work-items`
- `mise run check-spec`
- `mise run verify`
- `feature`、`bugfix`、`standards.md` 参照の各 fixture から主要ユースケースの計画を外すと失敗する。
- 主要ユースケースから単体テストまたは E2E テストの参照を片方ずつ外すと失敗する。
- 存在しないテスト、要求参照を持たないテスト、必須の `mise` タスクから到達しないテストを指定すると失敗する。
- 完了 fixture から Unit RED、E2E RED、内部判断の故障注入、配線または最終効果の故障注入を一つずつ外すと失敗する。
- `wi-439`、`wi-440`、`wi-441` の修正前の状態について、設定値の種類を特別扱いせず主要ユースケースの証拠不足を指摘する。

## Risk Notes

リスクは medium。最大の失敗は、主要ユースケースという名前だけを追加し、既存の薄いテストを参照して門を通すことである。テスト参照や要求 ID の存在だけではこの失敗を防げないため、主要ユースケースごとに内部判断と実配線を別々に壊し、対応するテストが実際に失敗した結果を完了条件にする。

主要ユースケースを広く取りすぎると、すべての分岐を E2E に重複実装する運用になる。主要ユースケースは機能または標準対応の成立を示す中心的な正常系に限定し、代替経路と拒否は仕様上のリスクに応じて単体、アダプター統合、受け入れへ置く。

反対に狭く取りすぎると、入力の受理や成功ステータスだけを主要結果として宣言できてしまう。要求参照と最終的な観測結果を対にし、配線または最終効果を無作用にした故障を E2E が検出することを要求して、機能の効果まで検証対象に含める。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。製品の規範仕様は変更せず、機能、欠陥修正、`standards.md` の規範対応を行う新規作業項目に `risk-based-v3` を適用した。主要ユースケースごとに要求参照、観測結果、単体テストと E2E テスト、異なる故障モデルを実装前に宣言し、完了時に両方の RED と故障注入結果を残す契約を追加した。作業項目検査は、完了時のテスト実在性、識別子、要求 ID、標準検証または CI から到達可能な `mise` タスクまで検査する。完了済みの `risk-based-v1` / `risk-based-v2` 記録は履歴として再解釈しない。
- **Acceptance RED Evidence**:
  - **Test**: `check-workspace --work-items > rejects an applicable in-progress item without a primary-use-case plan` (`tools/workspace/src/check-workspace.test.ts`)
  - **Requirement**: N/A: 製品の規範要求を変更しないリポジトリ検査の変更であるため。
  - **Observed Failure**: 主要ユースケース計画を持たない `change_kind: feature` の fixture を CLI が終了コード 0 で受理し、`expect(result.code).not.toBe(0)` が失敗した（5 pass、1 fail）。
  - **Detection Reason**: 観測境界を検査関数ではなく実際の work item CLI に置いたため、スキーマ、Markdown 解析、作業空間アダプターのいずれかが新規規則へ接続されていなければ、対象項目が受理される失敗として検出できる。
- **Unit RED Evidence**:
  - **Test**: `verifyPrimaryUseCaseEvidence > requires a plan for feature, bugfix, and standards work` (`tools/check/src/primary-use-case-evidence.test.ts`)
  - **Requirement**: N/A: 製品の規範要求を変更しないリポジトリ検査の変更であるため。
  - **Observed Failure**: 純粋検査を実装する前は `Cannot find module './primary-use-case-evidence.ts'` で失敗した（0 pass、1 error）。
  - **Detection Reason**: `feature`、`bugfix`、`standards.md` 参照を同じ公開操作へ与え、設定値や規範 ID の個別列挙ではなく適用条件そのものを固定した。後続の検査で計画、要求照合、テスト分離、完了証拠の各欠落も個別に固定した。
- **Change-Resistance Results**:
  `verifyPrimaryUseCaseEvidence` の必須タスク到達性判定を一時的に常時無効化し、CI または標準検証から到達しない `test-go` を参照する完了 fixture が誤って受理される変異を与えた。`check-workspace --work-items > validates completed primary-use-case evidence and required task reachability` の `expect(unreachable.code).not.toBe(0)` が失敗し（6 pass、1 fail）、この判定を検出できることを確認した。変異を直ちに復元し、同じ対象検査が 7 pass、0 fail になることを再確認した。適用条件、計画、要求参照、単体テストと E2E テストの重複、テスト実在性、識別子、要求 ID、Unit RED、E2E RED、二種類の故障注入結果は、それぞれ一つだけ欠落させる単体 fixture で失敗する。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-tools` - 354 pass、0 fail
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-spec` - passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run spec-diff` - `no normative specification change against main`
