---
depends_on: []
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-19
priority: p1
change_kind: bugfix
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-007 }
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

判断点は、主体の実在をどの境界で確かめるかである。

`AssignApplication` のユースケースへ利用者とグループの参照 port を足すと、Application Context が IdManagement の保存先へ直接依存する。

`handlers_http` で確かめると、ユースケースを直接呼ぶ経路が同じ検査を通らない。

どちらを採るにせよ、確認先は port として宣言し、テストが呼び出し回数と拒否後の状態を読み直せるようにする。

## Plan

1. 現在の挙動を HTTP 入口で観測し、存在しない主体の割り当てが 201 で成立することを RED として固定する。
2. 主体の実在を確かめる境界を決め、port の署名を定める。
3. 拒否を保存より前に置き、割り当てもイベントも作られないことを確かめる。
4. `subject_type=group` と越境した主体についても同じ拒否を確かめる。

## Tasks

- [ ] T001 [Acceptance] 存在しない主体と別テナントの主体の割り当てが現在成立することを観測し、RED を確認する。REQ-APPLICATION-007。
- [ ] T002 [Decision] 主体の実在を確かめる境界と port の署名を決める。
- [ ] T003 [App] 保存とイベント発行より前に拒否を置く。
- [ ] T004 [Unit] 拒否時に `AssignmentRepo.Save` が呼ばれず `ApplicationAssigned` も発行されないことを確かめる。
- [ ] T005 [Verify] 仕様と被覆の検査を通し、`EX-APPLICATION-007-02` を台帳から外す。

## Verification

- `mise run test-go-package -- ./backend/application/usecases`
- `mise run test-go-package -- ./backend/application/handlers_http`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

拒否応答だけを確認すると、応答を書いた後に保存が進む誤実装を検出できない。

保存ポートの呼び出し回数と、拒否後に読み直した割り当て一覧の双方を観測する。

主体の実在確認を足すと、正規の割り当て経路が壊れる余地がある。

利用者とグループの双方について、許可側のテストを同じ変更に含める。
