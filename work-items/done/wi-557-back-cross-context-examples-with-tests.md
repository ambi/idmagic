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
  reason: 宣言済みの具体例に既存の製品入口を通すテストを対応付ける保守作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/scenarios.feature.md
  typespec: []
  source:
    - backend/shared/http/testing_stack/stack.go
    - backend/shared/http/server_http/routes.go
    - backend/sharedsignals/usecases/revocation.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/provisioning/ports/capture.go
    - backend/idgovernance/usecases/user_mutation_committer.go
    - backend/application/usecases/assignments.go
  tests:
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/sharedsignals/handlers_http/scenario_examples_test.go
    - backend/shared/http/server_http/provisioning_api_token_scope_test.go
    - backend/provisioning/e2e_capture_delivery_test.go
    - backend/provisioning/usecases/capture_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - backend/provisioning/db_postgres
    - infra
---

# 横断シナリオが宣言する具体例 4 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決めたが、`docs/domain/scenarios.feature.md` が宣言する 4 件はどの Context にも属さない。**複数の Context が協調して初めて成り立つ振る舞いなので、どの Context の子 work item に入れても片側の話にしかならない。** 本項目がこの 4 件を引き取る。

対象は `EX-PLATFORM-001-01`、`EX-PLATFORM-002-01`、`EX-PLATFORM-003-01`、`EX-PLATFORM-003-02` である。`REQ-PLATFORM-004` の 2 件は既に名指しを持つ。

**`EX-PLATFORM-001-01` は近いテストがあるが、具体例を検証していない。** `backend/shared/http/server_http/routes_e2e_test.go` の `TestDisabledUserLoginAndExistingSessionAreRejected` は、利用者の状態を repository へ直接書いて無効化しており、具体例の `When`（管理者が無効化する）を通っていない。そのため 5 つの `Then` のうち、Agent の `AgentRevocationEpoch` の前進と `AgentAccessRevoked` の発行という 2 つを観測できる位置にない。

## Scope

- 4 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-PLATFORM-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。
- 各具体例が名指す参加 Context のすべてを、製品の正式な入口から 1 つの経路として通す。片方の Context だけを観測するテストは、この 4 件の根拠にならない。
- `EX-PLATFORM-001-01` は管理者の無効化操作を入口とし、5 つの `Then` をすべて観測する。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。
- `REQ-PLATFORM-004` の 2 件。既に `backend/shared/http/support_http/csrf_test.go` などが名指している。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 照合の結果

| 具体例 | 判定 | 解決 |
| --- | --- | --- |
| `EX-PLATFORM-001-01` | 実装は具体例どおりに振る舞う。既存の `TestDisabledUserLoginAndExistingSessionAreRejected` は保存先へ直接書いて無効化しており、Agent の 2 つの `Then` を観測できない | 管理 API を入口とする新しいテストを書く |
| `EX-PLATFORM-002-01` | 実装は具体例どおりに振る舞う。既存の `admin_user_handler_test.go` はログインの可否を観測していない | 管理 API を入口とする新しいテストを書く |
| `EX-PLATFORM-003-01` | 食い違う。User と割り当ての変更は、コミットの後で best-effort に capture を呼ぶ | 台帳に残し、`blocked_by` と `finding` を書く |
| `EX-PLATFORM-003-02` | 食い違う。変更と capture が別のコミットなので、片方だけが確定し得る | 同上 |

`EX-PLATFORM-003-*` の修正は、IdManagement、Application、Provisioning、IdGovernance のコミットを 1 つのトランザクションへ合成する設計判断を伴い、memory と Postgres の両方に及ぶ。
難しい修正に当たるため、[[wi-22350-capture-provisioning-deliveries-in-the-mutation-transaction]] へ切り出した。

### テストの置き場所

2 件のテストは `backend/shared/http/server_http/cross_context_examples_test.go` に置き、`testing_stack` の上に建てる。
管理者の操作は、`users:write` の API アクセストークンで管理 API を呼ぶ形にする。
`testing_stack` へ次を足した。

- `WithAgentRevocation()`：Agent の保存先と失効エポックの保存先を 1 つの option で配線する。片方だけでは、無効化から Agent の失効へ届く反応器が何もしない。
- `Browser.Get`：既存セッションの cookie を載せて認証必須 API を呼ぶ。
- `EventLog.All`：イベントを「どの Agent について発行されたか」まで読む。

### 選んだ実行レシピ

- RED、GREEN、故障注入：`mise run test-go-test -- ./backend/shared/http/server_http <test>`
- パッケージ：`mise run test-go-package -- ./backend/shared/http/server_http`、`./backend/shared/http/testing_stack`
- 変異：`mise run test-go-mutation -- backend/shared/http/testing_stack`

## Tasks

- [x] T001 [Readiness] 4 件を実装と照合し、`EX-PLATFORM-003-*` の食い違いを測定する。
- [x] T002 [Split] `EX-PLATFORM-003-*` の修正を wi-22350 へ切り出し、台帳の当該行へ `blocked_by` と `finding` を書く。
- [x] T003 [Harness] `testing_stack` に `WithAgentRevocation`、`Browser.Get`、`EventLog.All` を足す。
- [x] T004 [Acceptance] `EX-PLATFORM-001-01` と `EX-PLATFORM-002-01` のテストを書き、台帳に残したままの `check-spec` が 2 件を名指しで落とすことを観測してから台帳から外す。
- [x] T005 [FaultInjection] 反応器の配線、既存セッションの状態判定、ログインの状態判定、エポックの共有を手で壊し、テストが検出することを確かめる。
- [x] T006 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/scenarios.feature.md` の 4 件を `tools/check/example-coverage-debt.json` から外すか、`blocked_by` と `finding` を持つ行として残した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **片方の Context だけを観測して済ませる。** 4 件はいずれも複数の Context が協調して初めて成り立つ振る舞いであり、引き金を持つ Context と結果を観測する Context が違う。片側だけのテストは、この具体例の根拠にならない。`TestDisabledUserLoginAndExistingSessionAreRejected` が repository へ直接書いて無効化しているのが、その失敗の実例である。
- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `EX-PLATFORM-001-01` は 5 つの `Then` を持つ。`Then` の数だけ観測が要る。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change against main` を返した。
  横断シナリオの 4 件を実装と照合した。
  `EX-PLATFORM-001-01` と `EX-PLATFORM-002-01` は実装が具体例どおりに振る舞っていたが、具体例の `When` である管理者の操作を入口とするテストが無かった。管理 API を入口に、参加するすべての Context の結果を観測するテストを書き、台帳から外した。
  `EX-PLATFORM-001-01` のテストは、1 回の無効化で、ステータス、既存セッション、新規ログイン、所有 Agent 2 つの失効エポックと `/introspect`、Agent ごとの `AgentAccessRevoked` の 5 つを観測する。
  `EX-PLATFORM-002-01` のテストは、削除の予約でログインが閉じ、猶予期間内の復元で開き直すことを、同じ利用者の往復で観測する。
  `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` は実装が食い違っていた。User と割り当ての変更は、コミットの後で best-effort に capture を呼んでおり、配信行は変更と同じトランザクションで作られない。修正は複数の Context のコミットを 1 つのトランザクションへ合成する設計判断を伴うため、wi-22350 へ切り出し、台帳の 2 行へ `blocked_by` と `finding` を書いて残した。
  新しいテストのために `testing_stack` へ `WithAgentRevocation`、`Browser.Get`、`EventLog.All` を足した。製品コードは変えていない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`、`TestAdminDisablingAUserClosesEveryPathToItAtOnce`、`TestScheduledDeletionClosesLoginAndRestoreReopensIt`（`backend/shared/http/server_http`）
  - **Requirement**: REQ-PLATFORM-001
  - **Observed Failure**: 2 件のテストを置き、台帳を変えない状態の `check-spec` は、`EX-PLATFORM-001-01` と `EX-PLATFORM-002-01` を「now has a test that names it」で名指しして落ちた。
  - **Detection Reason**: 台帳は 2 件の id を含み、`//spec:covers` で名指すテストが現れた時点で検査が当の行を名指しで落とす。テストは製品と同じ `Register` の組み立てへ管理 API の要求を送り、ログイン、認証必須 API、`/introspect` の応答に加えて、失効エポック、イベント、セッションを保存先から読み直す。
- **Unit RED Evidence**:
  - **Test**: `TestAdminDisablingAUserClosesEveryPathToItAtOnce`、`TestScheduledDeletionClosesLoginAndRestoreReopensIt`
  - **Requirement**: REQ-PLATFORM-001
  - **Observed Failure**: N/A: 製品の振る舞いを変えない保守作業であり、単体の境界に新しい振る舞いが無い。代わりに故障を手で注入した。反応器の組み立てから失効エポックの保存先を外すと「A1 の失効エポックが進んでいない」で、既存セッションの解決から状態判定を外すと `/api/auth/account` の 200 で、ログインの状態判定を外すと 2 件のテストがログインの 200 で、エポックを Agent ごとの時刻にすると「A2 のエポック … と同一」で落ちた。
  - **Detection Reason**: 4 つの故障は、無効化から Agent の失効へ届く配線、既存セッションの経路、新規ログインの経路、同一エポックの要件をそれぞれ 1 つだけ壊しており、どれも片方の Context だけを観測するテストでは検出できない。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/shared/http/testing_stack` は、到達した 50 件の変異のうち 47 件を検出した。生存した 3 件は、応答本文の復号前の `len(raw) > 0` を `>= 0` にする等価な変異である。到達しなかった 7 件のうち 5 件は `Browser.Get` にあり、このパッケージ自身のテストは `Browser.Get` を呼ばない。`Browser.Get` は `server_http` の 2 件のテストが通しており、上の既存セッションの故障注入がその経路を検出した。
  変異器が表現できない配線の故障は、Unit RED Evidence に記した 4 件を手で注入し、すべて検出された。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run test-go-package -- ./backend/shared/http/server_http` - 成功
  - `mise run test-go-package -- ./backend/shared/http/testing_stack` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 未実行。ブラウザーへ届く変更が無い。
