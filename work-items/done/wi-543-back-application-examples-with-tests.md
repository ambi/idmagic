---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と既存の検証を対応付ける作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-001
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-002
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-003
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-004
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-005
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-006
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-007
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-008
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-009
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-010
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-011
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-012
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-013
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-014
    - docs/domain/application/decisions.md
  typespec:
    - IdMagic.Application.Operations.GetAdminApplication
    - IdMagic.Application.Operations.GetApplicationIcon
  source:
    - backend/application/handlers_http/admin_application_handler.go
    - backend/application/handlers_http/application_provisioning.go
    - backend/application/usecases/assignments.go
    - backend/application/usecases/sign_in_policy.go
    - backend/application/domain/applications.go
    - backend/application/ports/repository.go
    - backend/claimmapping/usecases/floor.go
    - backend/oauth2/handlers_http/authorize_completion.go
    - backend/shared/http/testing_stack/stack.go
    - frontend/src/features/admin-sign-in-policy/AdminSignInPolicyPage.tsx
  tests:
    - backend/application/handlers_http/application_handler_test.go
    - backend/application/handlers_http/extra_handlers_test.go
    - backend/application/handlers_http/claim_release_test.go
    - backend/application/usecases/applications_admin_test.go
    - backend/shared/http/server_http/application_api_token_tenant_test.go
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - frontend/src/features/admin-applications/AdminApplicationDetailPage.test.tsx
    - frontend/src/features/admin-applications/AdminApplicationEditPage.test.tsx
    - frontend/src/features/admin-applications/ClientSecretRotationPanel.test.tsx
    - frontend/src/features/admin-sign-in-policy/AdminSignInPolicyPage.test.tsx
  stop_before_reading:
    - docs/architecture
    - infra
    - backend/saml/handlers_http
---

# Application が宣言する具体例 32 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/application/scenarios.feature.md` が宣言する 32 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

`docs/domain/application/scenarios.feature.md` が宣言する 32 件のうち、`EX-APPLICATION-003-02` と `EX-APPLICATION-004-03` は既に `//spec:covers` を持つテストから名指されており、台帳にも載っていない。

本項目が引き取るのは、台帳に残る 30 件である。

- 30 件を 1 件ずつ確認し、実装と一致する 25 件を次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-APPLICATION-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 実装と一致しない 5 件は個別の work item へ切り出し、台帳へ `blocked_by` と `finding` を残す。
- ディレクティブの説明は「何を固定しているか」を書く。id だけの記述は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 証拠の境界

本項目は規範も製品ロジックも変えず、宣言済みの具体例とテストの対応を検証可能な形にする。

Acceptance RED は、台帳から対象行を外した時点の `mise run check-spec` とする。

検査が具体例 id を名指して落ちるため、対応する `//spec:covers` が無ければ完了できない。

Unit RED は N/A とする。

新しい単体ロジックを作らず、既存テストの観測を具体例と突き合わせる作業だからである。

代替の RED は同じ `mise run check-spec` であり、追加または補強した各テストは所属パッケージか UI テストファイルの絞り込みタスクで GREEN を確認する。

### readiness pass の結果

30 件のうち 25 件は、既存テストへの `//spec:covers` 追加か、既存テストの小さな観測補強か、`backend/shared/http/testing_stack` の上に書く新しいテストで対応できる。

残る 5 件は、実装が具体例のとおりに振る舞わないため台帳に残す。

| 具体例 | 実測結果 | 所有する記録 |
| --- | --- | --- |
| `EX-APPLICATION-007-02` | `AssignApplication` は主体の実在もテナント所属も確かめず、空でない `subject_id` をそのまま保存する。拒否そのものが無い | [[wi-625-application-assignment-accepts-any-subject]] |
| `EX-APPLICATION-007-03` | 越境した id の参照は拒否されるが、応答は `InvalidRequestError` ではなく `404 application_not_found` であり、`GetAdminApplication` の契約は 404 を宣言していない | [[wi-626-align-cross-tenant-application-read-refusals]] |
| `EX-APPLICATION-008-03` | 越境したアイコン取得は内容を返さないが、応答は `InvalidRequestError` ではなく `404 not_found` である。契約は 404 `ApplicationIconNotFoundError` を宣言しており、具体例だけが食い違う | [[wi-626-align-cross-tenant-application-read-refusals]] |
| `EX-APPLICATION-014-01` | `AssignApplicationDesiredState` は決定にだけ現れ、実装が存在しない | [[wi-628-implement-desired-state-application-assignment]] |
| `EX-APPLICATION-014-02` | `UnassignApplicationDesiredState` と `changed=false` の応答も同様に存在しない | [[wi-628-implement-desired-state-application-assignment]] |

この 5 件は製品の公開契約または規範の変更を要する。

本項目では直さず、台帳の `blocked_by` と `finding` に測定結果を残す。

### テストの対応単位

一つの具体例が HTTP、保存、UI の複数境界を持つ場合は、同じ id を複数のテストから参照し、それぞれのディレクティブに固定する観測を限定して書く。

一つのテストだけへ全体を検証したような説明を置かない。

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| UI の 1 ファイル | `mise run test-ui-unit-file -- <file>` |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 25 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [UI] `EX-APPLICATION-001-01` と `EX-APPLICATION-002-01` から `-03` までを、Application 詳細・編集画面とクライアントシークレット節のテストへ結び付ける。SAML の SSO URL と SLO URL の表示観測を足す。`mise run test-ui-unit-file` を使う。
- [x] T003 [Scope] `EX-APPLICATION-003-01`、`-003-03`、`EX-APPLICATION-004-01`、`-004-02` を、`testing_stack` の上に書く API トークンのスコープ判定テストへ結び付ける。`mise run test-go-package -- ./backend/shared/http/server_http`。
- [x] T004 [Protocol] `EX-APPLICATION-005-01`、`-005-02`、`EX-APPLICATION-006-01` から `-006-03` までを、SAML 設定更新とクレーム公開規則のテストへ結び付ける。`mise run test-go-package -- ./backend/application/handlers_http`。
- [x] T005 [Catalog] `EX-APPLICATION-007-01`、`EX-APPLICATION-008-01`、`-008-02`、`EX-APPLICATION-013-01` を、Application のライフサイクル、アイコン、非管理者の拒否のテストへ結び付ける。`mise run test-go-package -- ./backend/application/handlers_http`。
- [x] T006 [Policy] `EX-APPLICATION-009-01` から `-009-03` までと `EX-APPLICATION-010-01`、`-010-02` を、サインインポリシーの保存と OAuth2.Authorize での評価のテストへ結び付ける。`mise run test-go-package -- ./backend/application/usecases`、`./backend/oauth2/handlers_http`。
- [x] T007 [Federation] `EX-APPLICATION-011-01`、`-011-02`、`EX-APPLICATION-012-01` を、割り当て関門と `hidden` の扱いのテストへ結び付ける。`mise run test-go-package -- ./backend/oauth2/handlers_http`、`./backend/application/usecases`。
- [x] T008 [Triage] 5 件の食い違いを個別の work item へ切り出し、台帳へ `blocked_by` と `finding` を記録する。
- [x] T009 [Verify] `mise run verify` を通し、完了記録を作る。

## Verification

- `mise run check-spec` が、実装と一致する 25 件を `tools/check/example-coverage-debt.json` から外し、食い違う 5 件へ `blocked_by` と `finding` を残した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。


## Completion

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。
  Application が宣言する具体例のうち台帳に残っていた 30 件を実装と照合し、実装と一致する 25 件を `//spec:covers` を持つテストへ結び付けて被覆台帳から外した。
  既存テストへのディレクティブ追加だけで済んだのは 4 件で、残りは SAML 詳細の SSO URL と SLO URL、作成応答が運ぶ平文シークレット、上書きを持たない隣の Application に適用され続けるデフォルトポリシー、MFA 未登録人数の表示といった観測の補強か、`backend/shared/http/testing_stack` と Application の管理 API 上に書いた新しいテストを要した。
  実装と一致しない 5 件は [[wi-625-application-assignment-accepts-any-subject]]、[[wi-626-align-cross-tenant-application-read-refusals]]、[[wi-628-implement-desired-state-application-assignment]] に切り出し、台帳へ `blocked_by` と `finding` を残した。
  readiness pass の途中で `EX-APPLICATION-010-01` を実装なしと誤って測ったが、`unenrolled_user_count` が管理 API から `AdminSignInPolicyPage` まで通っていることを確認して消化側へ戻し、いったん起票した切り出し先を取り下げた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-APPLICATION-001
  - **Observed Failure**: 対象 24 件を台帳から外した時点で、検査が 24 件すべてを名指しして落ちた。先頭の失敗は `docs/domain/application/scenarios.feature.md:7: EX-APPLICATION-001-01 is declared, but no test names it.` である。`EX-APPLICATION-010-01` を消化側へ戻した後は 25 件が同じ形で落ちた。
  - **Detection Reason**: 台帳から外した具体例は、その id を名指す `//spec:covers` が無い限り検査を通らない。ディレクティブの説明と観測内容までは検査が保証しないため、各テストを具体例の `Then` と 1 つずつ照合した。
- **Unit RED Evidence**:
  - **Test**: `mise run check-spec`（代替 RED）
  - **Requirement**: N/A: 製品ロジックと規範を変更せず、既存の観測と具体例を対応付ける保守作業である。
  - **Observed Failure**: Acceptance RED と同じ未対応 id を検出した。
  - **Detection Reason**: 新しい単体ロジックが無いため、単体テストの失敗を作る対象が無い。代わりに、規範 id とテストの対応を検査する最小の受け入れ境界を RED にした。
- **Change-Resistance Results**:
  製品 Go を変更していないため、`mise run test-go-mutation` は対象を持たない。
  変更したのは `backend/shared/http/testing_stack` の配線とテストだけである。
  代わりに、新しく書いた観測が誤実装を実際に検出するかを、変異器が表現できない「検査を外す」「効果の後ろへ移す」形の故障注入 3 件で測った。
  - F1 `handleUpdateSamlConfig` から AuthnRequest 署名証明書の検査を削除する: 検出された (`TestUpdateSamlConfigRefusesAuthnRequestSigningWithoutACertificate` が `status=204, want 400`)。
  - F2 同じ検査を `SamlSPRepo.Save` の後ろへ移す: 検出された (`want_authn_requests_signed saved despite the refusal`)。応答は 400 のままなので、状態を読み直さない拒否テストでは取り逃がす形である。拒否テストへ効果の不在を入れた理由がこれである。
  - F3 `requireAdminApiTokenScope` を素通しにする: 検出された (`TestApplicationsReadScopeAloneCannotChangeAnApplication` が `status=201, want 403`)。
  - 手法の限界: 故障注入は新しい観測が名指す 3 つの判定に閉じており、既存テストへディレクティブを足しただけの 4 件については、そのテストの検出能力を測り直していない。
- **Verification Results**:
  - `mise run spec-diff` - passed（`main` に対する規範差分なし）
  - `mise run check-spec` - passed（157 standards、316 rules、771 examples、693 ids named）
  - `mise run check-work-items` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-go-package` - passed（`./backend/application/handlers_http`、`./backend/oauth2/handlers_http`、`./backend/shared/http/server_http`、`./backend/shared/http/testing_stack`）
  - `mise run test-ui-unit-file` - passed（変更した 5 個の UI テストファイル）
  - `mise run test-go-changed` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 実行していない。製品の UI コードを変更しておらず、変更はテストと `testing_stack` の配線に閉じているため、ブラウザーへ到達する経路が無い。
