---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と検証を対応付け、配送の最終失敗で発行するイベントを既存の具体例と状態遷移表へ合わせる保守作業であり、運用手順と公開 API の形を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/sharedsignals/scenarios.feature.md
    - docs/domain/sharedsignals/states.md
  typespec:
    - spec/contexts/sharedsignals/models.tsp
  source:
    - backend/sharedsignals/usecases/receive.go
    - backend/sharedsignals/usecases/revocation.go
    - backend/sharedsignals/usecases/admin_streams.go
    - backend/sharedsignals/usecases/deliver.go
    - backend/sharedsignals/usecases/project.go
    - backend/sharedsignals/handlers_http/routes.go
    - backend/sharedsignals/verify_jose/verifier.go
    - backend/shared/http/server_http/routes.go
    - backend/shared/http/testing_stack/stack.go
    - backend/oauth2/token/usecases/introspect_token.go
    - backend/shared/security/tokens_jose/jwks_resolver.go
  tests:
    - backend/sharedsignals/usecases/receive_test.go
    - backend/sharedsignals/usecases/receive_subject_identifier_test.go
    - backend/sharedsignals/usecases/deliver_test.go
    - backend/sharedsignals/usecases/project_test.go
    - backend/sharedsignals/usecases/admin_streams_quota_test.go
    - backend/sharedsignals/handlers_http/routes_test.go
    - backend/sharedsignals/handlers_http/e2e_stream_delivery_test.go
    - backend/idmanagement/handlers_http/extra_identity_test.go
    - backend/oauth2/token/usecases/introspect_token_agent_revocation_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - infra
    - backend/sharedsignals/db_postgres
affected_spec:
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SecurityEventDeliveryFailed }
---

# SharedSignals が宣言する具体例 20 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/sharedsignals/scenarios.feature.md` が宣言する 20 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 20 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-SHAREDSIGNALS-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 食い違いの判定

具体例と実装の食い違いは、具体例とは別の規範とも照合して判定した。

| 食い違い | 具体例 | 具体例以外の根拠 | 判定 |
| --- | --- | --- | --- |
| 配送の最終失敗で `SecurityEventDeliveryDeadLettered` だけを発行し、`SecurityEventDeliveryFailed` を発行しない | `EX-SHAREDSIGNALS-006-01` | `states.md` は `pending →(SecurityEventDeliveryFailed)→ failed →(SecurityEventDeliveryDeadLettered)→ dead_letter` と定め、`failed` を「再試行を予定するか、上限に達すれば `dead_letter` へ進む」状態とする。TypeSpec の `SecurityEventDeliveryFailed` の説明だけが「a retry was scheduled」と書いていた | 実装の欠陥。利用者の判断により、実装と TypeSpec の説明を具体例と `states.md` へ合わせる |

### 修正の設計

`deliverOne`（`backend/sharedsignals/usecases/deliver.go`）は、配送が失敗した試行で常に `SecurityEventDeliveryFailed`（`AttemptCount` は失敗した試行の回数）を発行する。
上限に達した試行では、続けて `SecurityEventDeliveryDeadLettered` を発行する。
保存する状態は従来どおりであり、上限に達した試行は `dead_letter`、それ以外は `failed` と次の試行時刻である。
TypeSpec の `SecurityEventDeliveryFailed` の説明を「配送の試行が失敗するたびに発行し、再試行を予定するか、上限に達していれば `SecurityEventDeliveryDeadLettered` が続く」へ改める。

### テストの置き場所

| 具体例 | 置き場所 | 観測 |
| --- | --- | --- |
| `EX-SHAREDSIGNALS-001-01`、`-02`、`-003-01`、`-004-01`、`-005-01`、`-007-01`、`-008-01`、`-009-01`、`-02`、`-03`、`-011-01`、`-02` | `backend/sharedsignals/handlers_http/scenario_examples_test.go`（新設） | 製品と同じ `server_http.Register` の組み立てに、管理 API、`/token`、`/introspect`、受信エンドポイントを通す。SET は外部の送信者の鍵で実際に署名し、受信ストリームへインライン JWKS として登録する。効果は保存先（失効エポック、ストリーム、配送、クォータの利用量）とイベントの記録から読み直す |
| `EX-SHAREDSIGNALS-006-01`、`-02` | `backend/sharedsignals/usecases/deliver_test.go` | 試行ごとの状態、イベントの順序、試行回数 |
| `EX-SHAREDSIGNALS-010-01` から `-06` | `backend/sharedsignals/usecases/receive_subject_identifier_test.go`（既存） | 既存の表が 6 件の入力と、失効エポック、`SecurityEventReceived`、`rejected_subject_unresolved` をすでに観測している。ディレクティブを足す |

`testing_stack` は管理 API の認証を API トークンで行うが、API トークンのスコープに `shared-signals:*` が無いため使わない。同じパッケージの既存テストと同じく、`DemoHeaderResolver` の管理者セッションで管理 API を呼ぶ。

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 台帳と仕様 | `mise run check-spec`、`mise run check-api-compat` |
| frontmatter | `mise run check-work-items` |

## Tasks

- [x] T001 [Spec] `SecurityEventDeliveryFailed` の TypeSpec の説明を改め、`check-spec` と `check-api-compat` を通す。
- [x] T002 [Acceptance] 20 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T003 [Fix] 最終失敗でも `SecurityEventDeliveryFailed` を発行する（`EX-SHAREDSIGNALS-006-01`）。Unit RED を先に観測する。
- [x] T004 [Delivery] `EX-SHAREDSIGNALS-006-02` の観測を書く。
- [x] T005 [HTTP] `EX-SHAREDSIGNALS-001-01`、`-02`、`-003-01`、`-004-01`、`-005-01`、`-007-01`、`-008-01` の観測を書く。
- [x] T006 [AdminAPI] `EX-SHAREDSIGNALS-009-01`、`-02`、`-03`、`-011-01`、`-02` の観測を書く。
- [x] T007 [Directive] `EX-SHAREDSIGNALS-010-01` から `-06` を既存テストへ結び付ける。
- [x] T008 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/sharedsignals/scenarios.feature.md` の 20 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 予定する Acceptance RED は、台帳から 20 件を外した `mise run check-spec` である。
- 予定する Unit RED は、3 回連続で失敗した配送のイベント列が `[Failed Retried Failed Retried Failed DeadLettered]` にならないことの `backend/sharedsignals/usecases` の観測である。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。TypeSpec の変更はイベント 1 件の説明文だけであり、契約の形は変えていない。
  SharedSignals が台帳に残していた 20 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  既存テストへディレクティブを足すだけで済んだのは、RFC 9493 の Subject Identifier を扱う 6 件（`EX-SHAREDSIGNALS-010-01` から `-06`）だけだった。この既存テストは 6 件の入力、失効エポック、`SecurityEventReceived`、`rejected_subject_unresolved` をすでに観測している。同じ箇所で、過去の一括移行が途中で切っていた `RFC9493-SUBID-ISS-SUB` のディレクティブの文も元の意味へ復した。
  配送の 2 件（`EX-SHAREDSIGNALS-006-01`、`-02`）は `backend/sharedsignals/usecases/deliver_test.go` で消化した。006-01 は既存テストへイベント列全体、失敗ごとの試行回数、再試行までの間隔が伸びること、dead_letter に次の試行時刻が残らないことを足し、006-02 は 3 回目で配送が成功する事例を新しく書いた。
  残る 12 件は `backend/sharedsignals/handlers_http/scenario_examples_test.go` を新設し、製品と同じ `server_http.Register` の組み立てで観測した。Agent の強制終了は IdManagement の管理 API、トークンは `/token` と `/introspect`、SET は外部の送信者の鍵で実際に署名して受信エンドポイントへ送る。拒否の効果は失効エポック、ストリーム、付随する設定の保存回数、配送、クォータの利用量、発行イベントから読み直している。
  照合で分かった食い違いは 1 件だった。配送の最終失敗で `SecurityEventDeliveryFailed` を発行していなかった。`states.md` の遷移表（`pending →(Failed)→ failed →(DeadLettered)→ dead_letter`）とも食い違っており、実装の欠陥と判定して利用者の指示で本項目で直した。TypeSpec の同イベントの説明だけが「a retry was scheduled」と限定していたので、具体例と `states.md` に合わせて改めた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-SHAREDSIGNALS-001
  - **Observed Failure**: 台帳から 20 件を外すと、`check-spec` は `EX-SHAREDSIGNALS-001-01` から `EX-SHAREDSIGNALS-011-02` までのちょうど 20 件を「declared, but no test names it」で名指しして落ちた。
  - **Detection Reason**: 台帳の行を消すだけでは検査を通らず、各 id を名指しするディレクティブと、その id を実際に動かすテストの両方が要る。HTTP のテストは製品と同じ `Register` の組み立てで要求を送り、応答に加えて保存先とイベントを読み直すので、拒否を書いてから効果を残す実装とも区別できる。
- **Unit RED Evidence**:
  - **Test**: `TestProcessDueDeliveries_ExhaustingMaxAttemptsDeadLetters`（`backend/sharedsignals/usecases`）
  - **Requirement**: REQ-SHAREDSIGNALS-006
  - **Observed Failure**: 修正前のイベント列は `[Failed Retried Failed Retried DeadLettered]` であり、期待する `[Failed Retried Failed Retried Failed DeadLettered]` に対して最終失敗の `SecurityEventDeliveryFailed` が欠けていた。
  - **Detection Reason**: 最後のイベントだけを見る従来の表明は、最終失敗の `Failed` が無くても通っていた。イベント列全体と、`Failed` が運ぶ試行回数 `[1 2 3]` を比べることで、発行の漏れと順序の入れ替わりの双方を検出する。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/sharedsignals/usecases` は 126 件の変異のうち 112 件を検出した（最初の実行では 109 件）。最初の実行で、配送の再試行の間隔を決める `nextAttemptDelay` に生存変異があったので、失敗のたびに予定の間隔が伸びることを `TestProcessDueDeliveries_ExhaustingMaxAttemptsDeadLetters` へ足して 3 件を検出させた。今回変えた行（失敗の分岐と 2 つのイベントの発行）に生存変異は残っていない。
  残る 13 件の生存変異はすべて既存の行にある。`nextAttemptDelay` の上限 30 分に関する境界の変異（この上限に達する試行回数を誰も観測していない）、`rejectEvent` が監査の記録を書く条件、`initiatingEntityForReason` の分岐、クォータ解放のログ分岐、`max_delivery_attempts` の `> 0` である。被覆されない 1 件は受信側設定の削除の分岐にある。
  変異器が表現できない故障として次の 4 つを手で注入し、すべて検出された。`server_http.Register` の失効イベントから SET の射影を外す形は、`TestKillAdvancesTheEpochWhileTheReceiverIsUnreachable` と `TestDisabledStreamNeitherReceivesNorTransmits/transmit` が配送 0 件で検出した。受信側で無効なストリームの判定を外す形は、`TestDisabledStreamNeitherReceivesNorTransmits/receive` が 202 で検出した。SharedSignals の管理 API の組み立てから `QuotaRepo` を外す形は、クォータの 3 件が利用量 0 で検出した。管理者の判定の結果を返さず処理を続ける形（拒否を書いてから保存する実装）は、`TestNonAdminCannotRegisterStreams` がストリームの増加で検出した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功
  - `mise run test-go-package -- ./backend/sharedsignals/usecases` - 成功
  - `mise run test-go-package -- ./backend/sharedsignals/handlers_http` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 実行していない。変更は backend の配送ロジックとテストだけであり、ブラウザーへ到達しない。
