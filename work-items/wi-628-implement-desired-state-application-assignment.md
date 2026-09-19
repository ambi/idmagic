---
depends_on: [wi-625-application-assignment-accepts-any-subject]
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-19
priority: p1
change_kind: feature
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-014 }
---

# あるべき状態を指定する Application 割り当て操作を実装する

## Motivation

`EX-APPLICATION-014-01` と `EX-APPLICATION-014-02` は、IdManagement の LifecycleWorkflow が `AssignApplicationDesiredState` と `UnassignApplicationDesiredState` を呼び、グループ経由の割り当てを変えずに個人への直接割り当てだけを作成または削除すると定める。

`docs/domain/application/decisions.md` も、この 2 つを HTTP に公開せず同じテナント内の識別子しか受け取らない内部インターフェースとして宣言している。

しかし `AssignApplicationDesiredState` も `UnassignApplicationDesiredState` も、リポジトリのどこにも実装が存在しない（[[wi-543-back-application-examples-with-tests]] の測定）。

既存の `AssignApplication` と `UnassignApplication` は、指定した `subject_type` の行をそのまま保存または削除するだけで、あるべき状態との差分も `changed` の応答も持たない。

LifecycleWorkflow が割り当てを繰り返し適用する経路では、差分が無いことを応答で区別できなければ、同じ操作のたびに `ApplicationAssigned` が発行される。

## Scope

- `AssignApplicationDesiredState` と `UnassignApplicationDesiredState` のシグネチャと戻り値を決め、Application Context の内部インターフェースとして実装する。
- 個人への直接割り当て（`subject_type=user`）だけを作成または削除し、グループ割り当て（`subject_type=group`）の行を変えないことを確かめる。
- 指定どおりの `visibility` で直接割り当てが既に存在する場合、保存もイベント発行もせず `changed=false` を返すことを確かめる。
- 直接割り当てを削除した後もグループ割り当てによるフェデレーションが許可されることを確かめる。
- IdManagement の LifecycleWorkflow からこの操作へ到達する配線を決める。

## Out of Scope

- この 2 つの操作を HTTP へ公開すること。決定が内部インターフェースと定めている。
- 既存の `AssignApplication` と `UnassignApplication` の署名変更。
- 主体の実在検査。[[wi-625-application-assignment-accepts-any-subject]] が先に決着させる。
- グループ割り当てそのものの評価規則と、動的グループの所属判定。

## Design

主たるドメイン型は `domain.ApplicationAssignment` であり、`SubjectType`、`SubjectID`、`Visibility` が同一性を決める。

あるべき状態の操作は、`AssignmentRepo` から現在の直接割り当てを読み、指定した `visibility` と突き合わせ、差分があるときだけ保存する。

時刻は入力の `Now` として受け取り、イベント発行は既存の `Emit` port を通す。

判断点は戻り値の形である。

`changed bool` だけを返す案は呼び出し側が単純になるが、何が変わったかを記録できない。

割り当てと `changed` の組を返す案は監査に足りるが、差分が無いときに返す割り当ての意味を決める必要がある。

もう一つの判断点は、LifecycleWorkflow がこの操作をどの port で呼ぶかである。

IdGovernance が Application の usecases を直接呼ぶと Context 間の依存が増えるため、port を宣言して注入する。

## Plan

1. 2 つの操作のシグネチャ、戻り値、イベント発行の条件を決める。
2. グループ割り当てを持つ主体へ直接割り当てを作る経路を RED にする。
3. 差分が無い呼び出しが `changed=false` を返し、保存もイベントも起こさないことを確かめる。
4. 直接割り当ての削除後もグループ割り当てが残り、フェデレーションが許可されることを確かめる。
5. LifecycleWorkflow からの配線を決めて通す。

## Tasks

- [ ] T001 [Decision] 2 つの操作のシグネチャ、戻り値、イベント発行の条件を決める。
- [ ] T002 [Domain] あるべき状態との差分判定を、保存先を読み直せる形で実装する。REQ-APPLICATION-014。
- [ ] T003 [Unit] グループ割り当ての行が変わらないことと、`changed=false` の経路で保存もイベントも起きないことを確かめる。
- [ ] T004 [App] 直接割り当ての削除後もフェデレーションが許可されることを確かめる。
- [ ] T005 [Adapters] LifecycleWorkflow からこの操作へ到達する port を配線する。
- [ ] T006 [Verify] 仕様と被覆の検査を通し、2 件を台帳から外す。

## Verification

- `mise run test-go-package -- ./backend/application/usecases`
- `mise run test-go-package -- ./backend/idgovernance/usecases`
- `mise run test-go-mutation -- backend/application/usecases`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

差分判定を誤ると、グループ割り当ての行を直接割り当てとして上書きするか、削除してしまう。

操作の前後でグループ割り当ての行を読み直し、`subject_type` と `visibility` が変わらないことを観測する。

`changed=false` を応答だけで確かめると、保存もイベント発行も進む誤実装を通す。

保存ポートの呼び出し回数とイベントの発行数の双方を観測する。

冪等な呼び出しが毎回イベントを発行すると、監査ログが実際の変化を示さなくなる。

同じ入力を 2 回適用したときのイベント数を固定する。
