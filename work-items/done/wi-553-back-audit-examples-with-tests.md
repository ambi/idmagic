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
documentation_impact: { level: none, reason: "宣言済みの具体例へテストを対応付けるだけで、製品の振る舞いも公開契約も変えない。読者へ知らせる差分がない。", references: [] }
initial_context:
  specification: [docs/domain/audit/scenarios.feature.md]
  typespec: []
  source:
    - backend/audit/handlers_http/admin_audit_event_handler.go
    - backend/audit/usecases/audit_search_extractor.go
    - backend/audit/db_memory/audit_event_store.go
    - backend/shared/http/support_http/pagination_request.go
    - backend/shared/http/support_http/pagination_cursor.go
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/idgovernance/domain/events.go
  tests:
    - backend/audit/handlers_http/admin_audit_event_handler_test.go
    - backend/audit/handlers_http/admin_audit_event_list_pagination_test.go
    - backend/audit/usecases/audit_search_test.go
    - backend/audit/db_postgres/audit_events_test.go
    - backend/audit/db_memory/audit_event_store_test.go
    - backend/audit/ports/audit_search_attribute_test.go
    - backend/cmd/idmagic-worker/worker_audit_test.go
    - backend/cmd/internal/bootstrap/audit_event_record_test.go
  stop_before_reading: [frontend, docs/domain/claim-mapping, docs/domain/workloadidentity]
---

# Audit が宣言する具体例 14 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/audit/scenarios.feature.md` が宣言する 14 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 14 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-AUDIT-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
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

## Verification

- `mise run check-spec` が、`docs/domain/audit/scenarios.feature.md` の 14 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
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
  `mise run spec-diff` は `no normative specification change against main` を返す。`docs/domain/audit/scenarios.feature.md` の宣言は 1 行も変えていない。
  変わったのは、REQ-AUDIT-001〜006 が宣言する 14 件 (`EX-AUDIT-001-01`、`002-01`、`003-01`、`004-01`〜`06`、`005-01`〜`04`、`006-01`) すべてが `//spec:covers` で id を名指すテストを持つようになったことと、その 14 件を `tools/check/example-coverage-debt.json` から外した (Audit 分 14 件、全体は 71→57) ことである。製品コードの変更は無い。
  内訳: 8 件は既存テストに観測が無く新しく書いた（`TestAdminAuditEventsFiltersRecentWindowAndExportsSameTenantScope`、`TestAdminAuditEventListFollowsPrevLastAndFirstLinks`、`TestAdminAuditEventListReturnsEmptyPageWhenFilterMatchesNothing`、`TestAdminAuditEventListFailsClosedWhenCountFails`、`TestAdminAuditEventListRejectsCursorWhenFilterChanges`、`TestAdminAuditEventListRejectsTamperedCursor`、`TestAdminAuditEventListRejectsExpiredLegacyCursor`、`TestNewAuditEventRecordLifecycleWorkflowSearchExcludesUnrelatedRunsAndSensitiveFields`）。6 件は具体例を実際に検証している既存テストへ「何を固定しているか」を書いた注記を足しただけで済んだ（`TestAdminAuditEventListSetsLinkHeaderWhenMorePagesExist`、`TestAdminAuditEventListNextPageContinuesWithoutOverlap`、`TestAdminAuditEventListRejectsCursorFromAnotherTenant`、`TestAdminAuditEventsRequiresAdminRole`、`TestAuditEventRepositoryDelegationAxes`（`db_postgres`、005-01/005-03/006-01 の 3 件をまとめて観測）、`TestExtractSearchAttributesAgentAsTargetKeepsTheHumanActor`）。`TestUserImportApplyRecordsUserCreatedAuditEvent`（worker、EX-AUDIT-002-01）は wi-205 の既存回帰テストで、CSV インポート適用という worker 専用の経路が共有の `AuditEventRepository` へ書き込み、`type=UserCreated` の絞り込みで見えることを既に検証していたので注記だけを足した。`TestAdminAuditEventsRejectsUnknownFilterField`（EX-AUDIT-005-04）は注記に加え、拒否時に既存イベントが応答へ漏れないこと（wi-392 が定める効果の不在の観測）を足した。`TestAuditEventRepositoryDelegationAxes` には EX-AUDIT-005-03（委譲の軸を持たない過去のイベントがどの軸にも一致しない）を独立に観測させるため、`legacy` イベントと専用のケースを 1 件足した。
  **`backend/shared/http/testing_stack` は使わなかった。** 作業の進め方は wi-565 の道具に乗ることを求めるが、`testing_stack` は Audit の配線 option (`WithAudit` 相当) を持たない。この audit パッケージの既存テストは `httpadapter.Register` を直接呼ぶ `newAuditAdminServer` / `newAuditAdminPaginationServer` という同種の最小 helper を既に持っており（`Deps` を手でフィールドごとに埋めるのではなく `Register` と同じ配線を 1 か所へ集約する点は `testing_stack` と同じ目的）、新しい option を `testing_stack` へ増設するのは本項目の範囲（宣言済み具体例へのテスト対応付け）を超える。触る必要が出た範囲だけ移すという wi-565 の原則どおり、既存の audit 専用 helper を使い、拡張はしなかった。
  EX-AUDIT-003-01（`workflow_run.id` での検索）は、LifecycleWorkflowRunPartiallyFailed と LifecycleWorkflowStepFailed が 1 ステップの失敗で揃って発行されること自体は idgovernance 側の `EX-IDGOVERNANCE-008-01`（`backend/idgovernance/usecases/scenario_examples_test.go`）が既に固定しているため、Audit 側の新しいテストはその 2 イベントを構成して監査記録へ変換し、`workflow_run.id` 絞り込みが無関係な run を含めないこと、記録された payload に属性値やメール本文が含まれないこと（イベントがそもそもそれらのフィールドを宣言していないことの回帰ピン）を固定した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 宣言済みの具体例へテストを対応付けるだけの作業で、新しい `REQ-*` も導入しない（`spec_impact: none`）。
  - **Observed Failure**: 14 本のテストへ注記を書き足した直後、対象 id が `tools/check/example-coverage-debt.json` に残ったまま実行すると、14 件それぞれについて `EX-AUDIT-NNN-MM now has a test that names it, at <path>:<line>. Remove it from the list; the list only shrinks.` という指摘で検査全体が失敗した（`checkNormativeCoverage` が finding を積み `ok: false` を返す）。台帳から 14 件を外すと exit 0 へ変わった。
  - **Detection Reason**: この指摘は「テストが id を名指ししているのに台帳がまだ残っている」という食い違いだけを検出する。id が本当に未検証のままなら別の文言（`is declared, but no test names it`）になり、消化済みで台帳も正しく外れていれば指摘そのものが出ないので、3 つの状態を取り違えない。
- **Unit RED Evidence**:
  - **Test**: 新しく書いた 8 本のテスト（代表: `TestAdminAuditEventListFollowsPrevLastAndFirstLinks`、`TestAdminAuditEventListRejectsTamperedCursor`）。
  - **Requirement**: N/A: 製品コードの変更を伴わない。既存実装が宣言済みの具体例を既に満たしていることを確認する作業のため、実装前に落ちる Unit RED は無い。
  - **Observed Failure**: 実際に落ちた代替検査として、`TestAdminAuditEventListRejectsTamperedCursor` は最初の実装で `status=200`（期待は 400）を返して落ちた。カーソル末尾 1 文字を隣接文字へ置き換える改ざん方法が、base64 raw-url エンコードの末尾グループが持つ zero-padding のせいで同じデコード結果になる組み合わせ（`A`→`B`）を引いており、署名検証をすり抜けていたことが分かった。改ざん位置を `v3.` 直後の payload 先頭 1 文字へ変えたところ GREEN になり、5 回連続実行しても安定した。
  - **Detection Reason**: 署名検証を経由しない改ざんでは応答が変わらないため、「意図した障害（実際に改ざんされたカーソル）を作れているか」がテストの pass/fail に直結する。今回は自分のテストコード側の障害注入が不完全だったことをテスト自身が検出した。
- **Change-Resistance Results**:
  N/A: `risk: low` であり、本項目は製品コードを一切変更していない（差分はテストと `tools/check/example-coverage-debt.json`、work item だけ）。変異を注入する対象のロジック差分が無いため、体系的な変異注入は行っていない。上記 Unit RED Evidence の改ざんカーソルの一件が、この作業で実際に観測できた唯一の「検出漏れ→修正→検出」の事例である。
- **Verification Results**:
  - `mise run check-spec` - passed（Audit の 14 件を台帳から除去した状態で通過）
  - `mise run test-go-package -- ./backend/audit/handlers_http` - passed
  - `mise run test-go-package -- ./backend/audit/usecases` - passed
  - `mise run test-go-package -- ./backend/audit/db_postgres` - passed
  - `mise run test-go-package -- ./backend/audit/db_memory` - passed
  - `mise run test-go-package -- ./backend/audit/ports` - passed
  - `mise run test-go-package -- ./backend/cmd/idmagic-worker` - passed
  - `mise run test-go-package -- ./backend/cmd/internal/bootstrap` - passed
  - `mise run check-work-items` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run verify` - passed
