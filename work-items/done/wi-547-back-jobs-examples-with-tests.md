---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオは変えない。本項目で直した欠陥は、実装を宣言済みの具体例と decisions.md へ合わせるものであり、規範を変えない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と既存の検証を対応付け、実装を既存の規範へ合わせる保守作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-001
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-002
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-003
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-004
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-005
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-006
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-007
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-008
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-009
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-010
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-011
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-012
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-013
    - docs/domain/jobs/scenarios.feature.md#REQ-JOBS-014
    - docs/domain/jobs/decisions.md
  typespec: []
  source:
    - dev.sh
    - backend/cmd/idmagic-dev-infra/main.go
    - backend/cmd/idmagic-dev-infra/internal/devinfra/devinfra.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/jobs/usecases/runner.go
    - backend/jobs/usecases/handler_registry.go
    - backend/jobs/domain/job.go
    - backend/jobs/db_memory/repository.go
    - backend/jobs/db_postgres/jobs.sql
    - backend/jobs/handlers_http/admin_job_handler.go
    - backend/oauth2/logout/usecases/logout.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
    - backend/datakeys/usecases/reencrypt.go
    - backend/tenancy/context.go
  tests:
    - backend/cmd/idmagic-dev-infra/internal/devinfra/devinfra_test.go
    - backend/jobs/usecases/runner_test.go
    - backend/jobs/usecases/admin_test.go
    - backend/jobs/usecases/enqueue_test.go
    - backend/jobs/usecases/handler_registry_test.go
    - backend/jobs/domain/job_test.go
    - backend/jobs/db_postgres/postgres_test.go
    - backend/jobs/handlers_http/admin_job_handler_test.go
    - backend/oauth2/logout/usecases/logout_test.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher_test.go
    - backend/datakeys/usecases/reencrypt_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - infra
    - backend/jobs/db_postgres/jobs.sql.go
---

# Jobs が宣言する具体例 21 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/jobs/scenarios.feature.md` が宣言する具体例のうち、台帳に残る 21 件を引き取る。起票時は 24 件と見積もったが、readiness pass の時点で台帳に残っていたのは 21 件だった。残りの 8 件（`EX-JOBS-012-02`、`-012-04`、`-013-05`、`-015-01` から `-04`）は既にテストが名指している。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 21 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-JOBS-NNN-MM: <この具体例の何を固定しているか>` を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 照合で見つかった実装の欠陥は、利用者の指示により本項目で直す（「実装の欠陥の修正」節）。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。
- シナリオと具体例の追加、削除、書き換え。
- ハートビートが一時的な障害で失敗したときに、ハンドラーを走らせたままハートビートを止める現在の振る舞い。リース喪失（`ErrJobLeaseLost`）だけを中断の契機とし、一時的な障害の扱いは変えない。
- ハンドラーが個別に `tenancy.WithTenant` を重ねている箇所の整理。Runner が実行コンテキストを固定した後も害は無いので残す。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類を根拠にした台帳からの削除。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が用意した道具を使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。

## Design

### 証拠の境界

Acceptance RED は、台帳から対象 21 件を外した時点の `mise run check-spec` とする。
検査が具体例 id を名指しで落とすため、対応する `//spec:covers` がなければ完了できない。
修正する欠陥にはそれぞれ、修正前に落ちる観測を先に書く。

| 欠陥 | 具体例 | 修正前に落ちる観測 |
| --- | --- | --- |
| devinfra のテストが常に skip する | `EX-JOBS-001-01` | skip ではなく失敗として現れる、スキーマの所在の確認 |
| `dev.sh` が基盤の準備より先に UI を起動する | `EX-JOBS-001-02` | 基盤が失敗した `dev.sh` の起動記録に UI が残る |
| Runner がハンドラーの実行コンテキストを Job のテナントへ固定しない | `EX-JOBS-006-01` | ハンドラーが見る `tenancy.TenantID(ctx)` が既定テナントになる |
| 再暗号化ハンドラーが Job のテナントではなく params のテナントで操作する | `EX-JOBS-006-02` | 他テナントを指す params で、他テナントの行が再暗号化される |
| 取り消された `running` の Job のハンドラーが中断されない | `EX-JOBS-013-02` | 取り消し後もハンドラーのコンテキストが終わらない |

### 実装の欠陥の修正

#### devinfra のテストが常に skip する

`devinfra_test.go` はスキーマを `../../infra/schema/postgres.sql` で指す。
パッケージが `backend/cmd/idmagic-dev-infra/internal/devinfra` へ移った後もこの相対パスが残り、`Start` はスキーマを読めずに失敗する。
テストは `Start` の失敗をすべて「組込み PostgreSQL が使えない」として skip するので、埋め込み PostgreSQL が使える環境でも 2 件とも常に skip していた。
修正はスキーマの所在をテストファイル自身の位置から解決し（`testing_postgres` の `schemaPath` と同じ形）、ファイルが無ければ skip せず落とす。
skip が外れると、キューの共有を見るテストはさらに 2 か所で落ちた。
保存先の `Enqueue` をレーン無しで直接呼んで `jobs_lane_check` に反し、`RunnerConfig` にレーンが無く `NewRunner` が panic する形だった。
投入は API と同じ `jobsusecases.Enqueue` を通す形に、Runner には既定レーンを渡す形に直した。

#### `dev.sh` が基盤の準備より先に UI を起動する

`EX-JOBS-001-02` は、基盤の準備に失敗した標準開発環境が API と UI を起動せずに止まると定める。
`dev.sh` は起動時間の短縮のため UI を基盤より先に起動しており、基盤が失敗したときには UI を起動してから落としていた。
修正は UI の起動を基盤の準備完了の後へ移す。基盤を使わない `memory` の構成では、UI の起動位置は従来どおり API より前である。

| 採った案 | 退けた案 | 退けた理由 |
| --- | --- | --- |
| UI を基盤の準備完了の後に起動する | 具体例を「UI は起動してもよい」へ書き換える | 規範の変更であり本項目の範囲を越える。持続化したクラスターでは基盤の準備は数秒で済み、重ねる利得は小さい |

#### Runner がハンドラーの実行コンテキストを Job のテナントへ固定しない

`docs/domain/jobs/decisions.md` は、ハンドラーの実行コンテキストをその Job の `tenant_id` へ固定すると定める。
`Runner.execute` は `context.Background()` をそのままハンドラーへ渡しており、固定はハンドラーごとの実装に任されていた。
テナントを入れないハンドラーの中で `tenancy.TenantID(ctx)` を読むと、既定テナントの識別子が返る。
修正は `Runner.execute` がハンドラーを呼ぶ前に `tenancy.WithTenant` で Job のテナントを実行コンテキストへ入れる。

#### 再暗号化ハンドラーが params のテナントで操作する

`ReencryptionHandler` は操作対象のテナントを `ReencryptParams.TenantID` から取り、Job の `tenant_id` と照合しない。
params が他テナントを指せば、そのテナントの行を再暗号化する。
修正は params のテナントが Job のテナントと食い違うとき、操作せずにエラーを返す。

#### 取り消された `running` の Job のハンドラーが中断されない

取り消しは Job のリースを解放するので、実行中の `worker` の次のハートビートは `ErrJobLeaseLost` になる。
`heartbeatLoop` はこれを記録してハートビートを止めるだけで、ハンドラーの実行コンテキストは生きたまま残った。
修正は `heartbeatLoop` がリースの喪失を受け取ったとき、`ErrJobLeaseLost` を原因としてハンドラーの実行コンテキストを取り消す。
`execute` はその原因を見て、完了も失敗も報告せずに終える。
報告すれば、PostgreSQL の構成では取り消し済みのコンテキストで `FailJob` を呼んで警告を残し、実行時間の指標に `failed` を 1 件足す。

| 採った案 | 退けた案 | 退けた理由 |
| --- | --- | --- |
| 実行コンテキストの取り消しに原因を持たせ、`execute` が原因で分岐する | `heartbeatLoop` と `execute` がリース喪失のフラグを共有する | 共有の可変値が増える。原因はコンテキストが既に運ぶ |

### テストの置き場所

| 具体例 | 置き場所 |
| --- | --- |
| `EX-JOBS-001-01`、`-02` | `dev.sh` の起動順と環境は `backend/cmd/idmagic-dev-infra` の新しいテストが、`go` と `bun` を差し替えた PATH で `dev.sh` を実行して観測する。組込み PostgreSQL、スキーマ、キューの共有は `devinfra` の既存テストが観測する |
| `EX-JOBS-002-01`、`-005-01`、`-006-01`、`-009-01`、`-013-01`、`-013-02` | `backend/jobs/usecases` の Runner のテスト |
| `EX-JOBS-003-01` | `backend/oauth2/logout/usecases`。外部へ通知を送る実在のハンドラーで、再配送が通知を重ねないことを観測する |
| `EX-JOBS-004-01`、`-011-01` | `backend/jobs/db_postgres`。リースの失効と `lane` 列の既定値は SQL が決める |
| `EX-JOBS-006-02` | `backend/datakeys/usecases`。修正した再暗号化ハンドラー |
| `EX-JOBS-007-01`、`-008-01` | `backend/idgovernance/usecases`。ディスパッチャーと実行ハンドラーを jobs の Runner で動かす |
| `EX-JOBS-010-01`、`-02` | 既存の起動時検査のテスト |
| `EX-JOBS-012-01`、`-03`、`-013-03`、`-04`、`-014-01` | `backend/jobs/handlers_http` の管理 API のテスト |

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 埋め込み PostgreSQL を使うテスト | 同じタスクをサンドボックスの外で実行する（共有メモリが拒否され skip するため） |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 21 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [Fix] devinfra のテストのスキーマの所在を直し、skip ではなく実行されることを確認する。
- [x] T003 [Fix] `dev.sh` の起動テストを書き、基盤の失敗時に UI が起動しないよう直す（`EX-JOBS-001-01`、`-02`）。
- [x] T004 [Runner] `EX-JOBS-002-01`、`-005-01`、`-009-01`、`-013-01` の観測を書く。
- [x] T005 [Fix] Runner が実行コンテキストへ Job のテナントを固定する（`EX-JOBS-006-01`）。
- [x] T006 [Fix] 再暗号化ハンドラーがテナントの食い違いを拒否する（`EX-JOBS-006-02`）。
- [x] T007 [Fix] リースを失ったハンドラーの実行コンテキストを取り消す（`EX-JOBS-013-02`）。
- [x] T008 [Handler] `EX-JOBS-003-01`、`-007-01`、`-008-01` を利用側のハンドラーとディスパッチャーのテストへ対応付ける。
- [x] T009 [Storage] `EX-JOBS-004-01`、`-011-01` を PostgreSQL のテストへ対応付ける。
- [x] T010 [Startup] `EX-JOBS-010-01`、`-02` を起動時検査のテストへ対応付ける。
- [x] T011 [AdminAPI] `EX-JOBS-012-01`、`-03`、`-013-03`、`-04`、`-014-01` を管理 API のテストへ対応付ける。
- [x] T012 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、Jobs の 21 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `Then` の数だけ観測が要る。
- **埋め込み PostgreSQL のテストが skip したまま緑に見える。** 本項目で見つけた devinfra の欠陥がこれである。PostgreSQL を使うテストの結果は `-v` の出力で skip でないことを確かめる。

## Completion

- **Completed At**: 2026-09-22
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。
  Jobs が台帳に残していた 21 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  既存テストへディレクティブを足すだけで済んだのは `EX-JOBS-004-01`、`-010-01`、`-010-02` の 3 件である。
  他の 18 件は、状態遷移の途中の観測（running の保有者、滞留した bulk レーンの件数）、効果の不在（呼ばれないハンドラー、変わらない更新時刻、作られない Job）、利用側のハンドラーとディスパッチャーを jobs の Runner で動かす観測のいずれかを足した。
  照合で 5 件の欠陥が分かり、利用者の指示により本項目で直した。
  devinfra のテストはスキーマの所在を誤って常に skip しており、skip を外すとさらに 2 か所の腐敗が現れた。
  `dev.sh` は基盤の準備より先に UI を起動していた。
  Runner はハンドラーの実行コンテキストを Job のテナントへ固定しておらず、テナントを入れないハンドラーは既定テナントの範囲で動いた。
  再暗号化ハンドラーは Job のテナントではなく params のテナントで操作していた。
  取り消された running の Job のハンドラーは、リースを失っても中断されなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`、`TestDevScriptFailsFastWhenInfrastructureFails`
  - **Requirement**: REQ-JOBS-001
  - **Observed Failure**: 台帳から 21 件を外すと、`check-spec` は `EX-JOBS-001-01` から `EX-JOBS-014-01` までの 21 件を「declared, but no test names it」で名指しして落ちた。修正前の `dev.sh` のテストは、基盤が失敗した起動の記録が `["ui" "infra"]` になって落ちた。
  - **Detection Reason**: 台帳は REQ-JOBS-001 から REQ-JOBS-014 の具体例 21 件を含む。台帳の行を消すだけでは検査を通らず、各 id に観測を伴うテストが要る。`dev.sh` のテストは `go` と `bun` を代役へ差し替えて起動の順序と環境を記録するので、失敗で終わっても UI を先に起動する実装と区別できる。
- **Unit RED Evidence**:
  - **Test**: `TestRunner_BindsTheHandlerContextToTheJobsTenant`、`TestCancelJobForAdmin_ARunningJobsHandlerStopsAtItsNextReport`（`backend/jobs/usecases`）、`TestReencryptionHandler_RefusesParamsNamingAnotherTenant`（`backend/datakeys/usecases`）、`TestEmbeddedInfrastructureSharesJobQueueWithRunner`（`devinfra`）
  - **Requirement**: REQ-JOBS-006
  - **Observed Failure**: 修正前は、ハンドラーが見るテナントが `00000000-0000-4000-8000-000000000000`（既定テナント）で落ちた。取り消し後もハンドラーのコンテキストが 1 秒以内に終わらずに落ちた。他テナントを指す params を受け付けて落ちた。スキーマの所在の確認が `stat ../../infra/schema/postgres.sql: no such file or directory` で落ち、所在を直すと `jobs_lane_check` 違反で落ちた。
  - **Detection Reason**: 4 件はそれぞれ REQ-JOBS-006、REQ-JOBS-013、REQ-JOBS-001 に属する。いずれも Runner または実在のハンドラーを通して効果を読むので、応答や戻り値だけを見るテストでは区別できない配線の欠落を検出する。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/jobs/usecases` は 62 件の変異のうち 48 件を検出した。生き残った 14 件は既存の境界条件（ポーリング間隔などの既定値、ハートビート間隔、管理 API の件数上限、投入の既定値）にあり、今回変えた行（テナントの固定、リース喪失時の中断と報告の省略）には生存変異がない。
  `mise run test-go-mutation -- backend/datakeys/usecases` は 45 件のうち 42 件を検出した。今回足したテナントの照合は検出され、生き残りは継続 Job の投入失敗を記録するだけの既存の分岐にある。
  変異器が表現できない故障として次を手で注入し、すべて検出された。テナントの固定を外す形と `abandon` の呼び出しを外す形は、修正前の状態として Unit RED で観測済みである。`execute` のリース喪失時の早期 return を外す形は、`TestCancelJobForAdmin_ARunningJobsHandlerStopsAtItsNextReport` が実行時間の指標に `failed` が残ることで検出した。ログアウト通知の送信済み判定を外す形は、`TestBackChannelLogoutRedeliveredJobDoesNotResendTheNotification` が再配送の失敗で検出した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run test-go-package -- ./backend/jobs/usecases` - 成功
  - `mise run test-go-package -- ./backend/jobs/handlers_http` - 成功
  - `mise run test-go-package -- ./backend/jobs/db_postgres` - 成功（サンドボックスの外で実行し、skip 0 件）
  - `mise run test-go-package -- ./backend/cmd/idmagic-dev-infra/...` - 成功（サンドボックスの外で実行し、skip 0 件）
  - `mise run test-go-package -- ./backend/datakeys/usecases` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
