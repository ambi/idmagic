---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-19
priority: p1
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 既に宣言済みの主体実在性とテナント境界の拒否を実装へ一致させる修正であり、公開契約や利用手順は変わらない。
  references: []
initial_context:
  specification: [docs/domain/application/scenarios.feature.md#REQ-APPLICATION-007]
  typespec: []
  source:
    - backend/application/usecases/assignments.go
    - backend/application/ports/repository.go
    - backend/application/module.go
    - backend/application/handlers_http/admin_application_handler.go
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/application/usecases/applications_admin_test.go
    - backend/application/handlers_http/catalog_examples_test.go
    - backend/shared/http/server_http/application_api_token_scope_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading: [frontend, backend/application/db_postgres, spec]
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-007 }
primary_use_cases:
  - id: assign-existing-tenant-subject
    requirement: REQ-APPLICATION-007
    observable_result: 管理者が同じテナントに実在する User または Group を Application に割り当てると保存され、存在しない主体または別テナントの主体では割当も ApplicationAssigned も残らない。
    unit_test: { path: backend/application/usecases/assignment_subjects_test.go, name: TestAssignApplicationAcceptsOnlyExistingTenantSubjects, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/application_assignment_e2e_test.go, name: TestApplicationAssignmentAcceptsOnlyExistingTenantSubjects, task: test-go-race }
    unit_fault_model: 主体照合の false を無視して AssignmentRepository.Save と ApplicationAssigned の発行を続ける。
    e2e_fault_model: 製品の module 組み立てが IdManagement の User/Group 保存先を Application の主体照合 port へ接続しない。
---

# Application の割り当てが存在しない主体と別テナントの主体を受け入れる

## Motivation

`EX-APPLICATION-007-02` は、別テナントの主体または存在しない主体を指定した割り当てを `InvalidRequestError` で拒否すると定める。

`AssignApplication` は対象 Application の実在とテナントを確かめたあと、`SubjectType` の妥当性と `SubjectID` が空でないことだけを検査し、そのまま保存する。

主体が呼び出し元のテナントに実在するかどうかは、ユースケースにも `handleAssignApplication` にも検査が無い。

結果として、他テナントの利用者 id やタイプミスした id が割り当てとして保存され、`ApplicationAssigned` も発行される。

割り当てはプロトコル経由のフェデレーションの関門であるため（[[wi-543-back-application-examples-with-tests]] の測定）、実在しない主体の行が残ることは関門の意味を弱める。

## Scope

- 割り当ての主体が呼び出し元テナントに実在することを、保存より前に確かめる。
- `subject_type=user` と `subject_type=group` の双方について、実在の確認先を決める。
- 拒否した要求が割り当てを作らず、`ApplicationAssigned` も発行しないことを観測する。
- 同じ欠落が `UnassignApplication` と、あるべき状態を指定する経路にもあるかを棚卸しする。

## Out of Scope

- 越境した Application id を指定した参照と更新の拒否契約。[[wi-626-align-cross-tenant-application-read-refusals]] が扱う。
- あるべき状態を指定する割り当て操作の実装。[[wi-628-implement-desired-state-application-assignment]] が扱う。
- 割り当ての一覧、ページング、`visibility` の意味の変更。
- 既に保存済みの、主体が実在しない割り当て行の掃除。

## Design

主体の実在は `AssignApplication` のユースケース境界で確かめる。
HTTP ハンドラーだけで確かめる案は、ユースケースを直接呼ぶ経路が検査を迂回するため採用しない。
Application Context が IdManagement の保存型へ直接依存する案も採用せず、Application が所有する読み取り port を module の組み立てで IdManagement の User/Group repository へ接続する。

主たるドメイン型は `domain.ApplicationAssignment` と `domain.AssignmentSubjectType` である。
読み取り port は `SubjectDirectory.SubjectExists(context.Context, tenantID string, subjectType domain.AssignmentSubjectType, subjectID string) (bool, error)` とし、`AssignmentDeps` が受け取る。
User は `FindBySub` の結果の `TenantID` を照合し、Group は tenant-scoped な `FindByID` を使う。
存在しない主体、別テナントの主体、または照合 port が未構成の経路は `ErrSubjectNotFound` で fail closed にする。

永続化は `AssignmentRepository`、イベント発行は `Emit`、プロビジョニング通知は `ProvisioningNotifier` の既存 port に残す。
主体照合をこれらの作用より前へ置き、拒否時は保存、イベント、通知のいずれも実行しない。

`UnassignApplication` は、主体削除後にも残った既存行を冪等に解除できる必要があり、現在は主体の実在を成功条件にしていない。
あるべき状態を指定する操作は [[wi-628-implement-desired-state-application-assignment]] が扱うため、どちらにも本項目の照合を広げない。

## Plan

1. 製品と同じ module 組み立てを使う HTTP 入口で、存在しない主体と別テナントの主体の割り当てが 201 で成立する E2E RED を確認する。
2. `AssignApplication` の単体境界で同じ誤保存とイベント発行を観測し、Unit RED を確認する。
3. `SubjectDirectory` port と IdManagement adapter を配線し、User/Group の許可と、存在しない主体/別テナント主体の拒否を保存前に実装する。
4. 単体と E2E の故障注入、変更パッケージ、mutation、仕様被覆、標準検証の順で確認する。

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 複数パッケージの配線 | `mise run test-go-changed` |
| 変更した純粋ロジック | `mise run test-go-mutation -- backend/application/usecases`、`mise run test-go-mutation -- backend/application` |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] `TestApplicationAssignmentAcceptsOnlyExistingTenantSubjects` で、User/Group の許可と、存在しない主体/別テナント主体の拒否後に割当と `ApplicationAssigned` が残らない E2E RED を確認する。REQ-APPLICATION-007、EX-APPLICATION-007-02。
- [x] T002 [Decision] `SubjectDirectory.SubjectExists` を Application 所有 port とし、module の adapter で IdManagement の User/Group repository へ接続する。
- [x] T003 [App] `AssignApplication` で主体を保存とイベント発行より前に照合し、User/Group の双方を fail closed にする。`TestAssignApplicationAcceptsOnlyExistingTenantSubjects`、REQ-APPLICATION-007。
- [x] T004 [Unit] `TestAssignApplicationAcceptsOnlyExistingTenantSubjects` の Unit RED を確認し、拒否時に保存された割当と `ApplicationAssigned` が無いことを読み直す。
- [x] T005 [Verify] 仕様と被覆の検査を通し、`EX-APPLICATION-007-02` を台帳から外す。

## Verification

- `mise run test-go-package -- ./backend/application/usecases`
- `mise run test-go-package -- ./backend/application/handlers_http`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-package -- ./backend/shared/http/testing_stack`
- `mise run test-go-changed`
- `mise run test-go-mutation -- backend/application/usecases`
- `mise run test-go-mutation -- backend/application`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

拒否応答だけを確認すると、応答を書いた後に保存が進む誤実装を検出できない。

保存ポートの呼び出し回数と、拒否後に読み直した割り当て一覧の双方を観測する。

主体の実在確認を足すと、正規の割り当て経路が壊れる余地がある。

利用者とグループの双方について、許可側のテストを同じ変更に含める。

User ID は repository 全体では一意だが tenant-scoped な参照 port ではないため、adapter が取得後の `TenantID` を必ず比較する。
Group は tenant-scoped な `FindByID` を使い、別テナントと不在を同じ `false` に畳む。

互換性上、これまで 201 になっていた不正な割り当ては 400 `invalid_request` になるが、既存の規範契約へ戻す修正である。
保存済みの孤児割り当ての移行や掃除は行わず、ロールバックはコードを戻すだけでデータ変換を要しない。

## Completion

- **Completed At**: 2026-09-20
- **Summary**:
  `mise run spec-diff` は、`main` と比較して規範仕様の変更がないと判定した。
  `AssignApplication` は、保存前に Application が所有する `SubjectDirectory` ポートを通じて User と Group の主体を解決し、存在しない主体と別テナントの主体を `invalid_request` で拒否する。拒否後には割り当ても `ApplicationAssigned` イベントも残さない。
  モジュールアダプターは IdManagement のリポジトリ型をユースケースの外側に保ち、本番構成を通る E2E テストによって `EX-APPLICATION-007-02` を被覆不足台帳から削除した。
- **Primary Use Case Evidence**:
  - id: assign-existing-tenant-subject
    unit_red: '実装前の `TestAssignApplicationAcceptsOnlyExistingTenantSubjects` は、`AssignmentDeps.SubjectDirectory` と `ErrSubjectNotFound` が存在しないためコンパイルに失敗した。'
    e2e_red: '実装前の `TestApplicationAssignmentAcceptsOnlyExistingTenantSubjects` は、存在しない User、存在しない Group、別テナントの User、別テナントの Group に対して 400 ではなく 201 を返し、割り当てを保存した。'
    unit_fault_injection: '`SubjectDirectory` が返す false を無視すると、存在しない User と存在しない Group のケースが `error=<nil>, want assignment subject does not exist in tenant` で失敗した。'
    e2e_fault_injection: 'モジュールからハンドラーへの `AssignmentSubjectDirectory` の配線を外すと、存在する User と存在する Group のケースが 201 ではなく 400 を返した。'
- **Change-Resistance Results**:
  - `mise run check-go-mutation-tool` - 成功。最初のサンドボックス内での試行はホストの Go ビルドキャッシュを読み取れなかったが、承認後に環境外で再実行し、コールド時とキャッシュ時の両方でカナリア判定に成功した。
  - `mise run test-go-mutation -- backend/application` - モジュールアダプターの変異 7 件中 7 件を検出した。生存、未被覆、実行不能の変異はなかった。
  - `mise run test-go-mutation -- backend/application/usecases` - 171 件の変異を検出し、154 件を検出、14 件が生存、3 件が未被覆だった。今回変更した `AssignApplication` の検証経路に対するトークン変異はすべて検出した。生存および未被覆の変異は、既存の通知、割り当て解除、または今回と無関係な Application ユースケースの分岐に属する。
  - 変異ツールでは新しい `if !subjectExists` ガードの削除やモジュール配線の切断を表現できないため、その限界は上記二つの明示的な故障注入で補った。
- **Verification Results**:
  - `mise run spec-diff` - 成功（`main` と比較して規範仕様の変更なし）
  - `mise run check-work-items` - 成功
  - `mise run check-spec` - 成功（標準 157 件、ルール 316 件、例 771 件、名前付き ID 725 件）
  - `mise run lint-go` - 成功（問題 0 件）
  - `mise run test-go-package -- ./backend/application` - 成功
  - `mise run test-go-package -- ./backend/application/usecases` - 成功
  - `mise run test-go-package -- ./backend/application/handlers_http` - 成功
  - `mise run test-go-package -- ./backend/shared/http/server_http` - 成功
  - `mise run test-go-package -- ./backend/shared/http/testing_stack` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（38 テスト）
