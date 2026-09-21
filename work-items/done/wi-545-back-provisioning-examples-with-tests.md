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
  reason: 宣言済みの具体例と既存の検証を対応付ける保守作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-001
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-002
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-003
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-004
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-005
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-006
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-007
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-008
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-009
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-010
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-011
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-012
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-013
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-014
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-015
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-016
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-017
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-018
  typespec: []
  source:
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/deliver.go
    - backend/provisioning/usecases/dispatcher.go
  tests:
    - backend/provisioning/handlers_http/admin_connection_handler_test.go
    - backend/provisioning/usecases/admin_test.go
    - backend/provisioning/usecases/capture_test.go
    - backend/provisioning/usecases/deliver_test.go
    - backend/provisioning/usecases/dispatcher_test.go
    - backend/provisioning/usecases/job_handler_test.go
    - backend/provisioning/e2e_capture_delivery_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - infra
    - backend/provisioning/db_postgres
---

# Provisioning が宣言する具体例 27 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/provisioning/scenarios.feature.md` が宣言する 27 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 27 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-PROVISIONING-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
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
検査が具体例 id を名指しで落とすため、対応する `//spec:covers` が無ければ完了できない。

Unit RED は N/A とする。
新しい単体ロジックを作らず、既存テストの観測を具体例と突き合わせる作業だからである。
代替の RED は同じ `mise run check-spec` とし、追加または補強した各テストは所属パッケージの絞り込みタスクで GREEN を確認する。

### テストの対応単位

一つの具体例が捕捉、ジョブ投入、下流への送出など複数の境界を持つ場合は、同じ id を複数のテストから参照し、それぞれのディレクティブには固定する観測だけを書く。
拒否の既存テストに効果の不在の観測が無い場合は、id を対応付けたうえで [[wi-392-refusal-tests-assert-the-absent-effect]] の対象として残す。

### readiness pass の結果

27 件のうち 14 件は既存テストへの具体例ディレクティブ追加、または On-Demand Provision と Provisioning API のスコープ境界を観測する小さなテスト追加で対応できた。

残る 13 件は、猶予期間・Full Resync 完了・ライフサイクルイベント・テナント境界の公開 HTTP 観測のいずれかが欠けるため、実装と証拠の両方を別 work item へ切り出した。
台帳の各エントリーに先行 work item と実測した `finding` を残す。

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 27 件を台帳から一時的に外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [Connection] `EX-PROVISIONING-001`、`-002`、`-011`、`-012`、`-014`、`-015` を管理接続と管理操作のテストへ対応付ける。
- [x] T003 [Capture] `EX-PROVISIONING-003` から `-006`、`-016` を捕捉と永続化のテストへ対応付ける。
- [x] T004 [Delivery] `EX-PROVISIONING-007` から `-010`、`-018` を配信とジョブ処理のテストへ対応付ける。
- [x] T005 [Dispatch] `EX-PROVISIONING-013`、`-017` を再同期と定期ディスパッチャーのテストへ対応付ける。
- [x] T006 [Triage] 実装と食い違う具体例を個別の work item へ切り出し、台帳へ `blocked_by` と `finding` を記録する。
- [x] T007 [Verify] 対応済み id を台帳から外し、対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/provisioning/scenarios.feature.md` の 27 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。
  Provisioning が台帳に残していた 27 件の具体例を実装と照合し、14 件を既存テストまたは追加した Provisioning API スコープと On-Demand Provision のテストへ結び付けて台帳から外した。
  残る 13 件は猶予期間、Full Resync 完了、ライフサイクルイベント、テナント境界の公開 HTTP 観測が欠けるため、[[wi-96960-defer-and-cancel-user-deprovisioning-after-grace-period]]、[[wi-22987-track-full-resync-completion]]、[[wi-85060-publish-provisioning-lifecycle-events]]、[[wi-37560-return-access-denied-for-cross-tenant-provisioning-api-tokens]]、[[wi-93622-acceptance-test-provisioning-tenant-isolation]] へ切り出し、台帳へ `blocked_by` と `finding` を残した。
  総合検証を妨げていた [[wi-595-declare-uniqueness-conflict-responses]] の完了記録の相対リンクも、`work-items/done/` から解決するパスへ修正した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-PROVISIONING-001
  - **Observed Failure**: `EX-PROVISIONING-001-01` を台帳へ一時的に戻すと、検査は `backend/shared/http/server_http/provisioning_api_token_scope_test.go:30` が id を名指しているため当該行を台帳から削除するよう拒否した。
  - **Detection Reason**: 台帳に残した id は、テストが id を名指す状態では検査を通らない。対応済みの具体例を台帳から外し、未対応または実装不一致の具体例だけを根拠とともに残す境界を固定する。
- **Unit RED Evidence**:
  - **Test**: `mise run check-spec`（代替 RED）
  - **Requirement**: N/A: 製品ロジックと規範を変更せず、既存の観測と具体例を対応付ける保守作業である。
  - **Observed Failure**: Acceptance RED と同じ台帳・テスト対応の不整合を検出した。
  - **Detection Reason**: 新しい単体ロジックは作らず、追加したテストは既存ユースケースと HTTP 配線を観測する。規範 id とテストの対応を検査する最小の受け入れ境界を RED にした。
- **Change-Resistance Results**:
  製品 Go を変更していないため、`mise run test-go-mutation` の対象はない。
  追加した HTTP テストは `provisioning:read` による接続登録を 403 と保存先の不変で検出し、別 tenant 発行トークンが 401 `invalid_token` を返す実装不一致は [[wi-37560-return-access-denied-for-cross-tenant-provisioning-api-tokens]] へ分離した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run test-go-package -- ./backend/provisioning/usecases` - 成功
  - `mise run test-go-package -- ./backend/shared/http/server_http` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
