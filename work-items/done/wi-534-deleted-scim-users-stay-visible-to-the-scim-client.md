---
depends_on: [wi-535-standards-row-additions-cannot-be-claimed-by-any-record]
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 標準の行 RFC7644-DELETE-SEMANTICS を新設し、削除済み User の id に対する SCIM の応答が変わるので、リリースの読み手に見える差分がある。未リリースなので移行の案内は要らず、リリースノートは何が変わったかだけを書く。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-534.md }
initial_context:
  specification:
    - docs/contexts/sourcing/scenarios.feature.md#REQ-SOURCING-002
    - docs/contexts/sourcing/standards.md
  typespec:
    - IdMagic.Sourcing.Operations.GetScimUser
    - IdMagic.Sourcing.Operations.UpdateScimUser
    - IdMagic.Sourcing.Operations.PatchScimUser
    - IdMagic.Sourcing.Operations.DeleteScimUser
  source:
    - backend/sourcing/scim/usecases/usecases.go
    - backend/sourcing/scim/usecases/users.go
    - backend/sourcing/scim/usecases/groups.go
    - backend/sourcing/scim/usecases/list.go
    - backend/sourcing/scim/handlers_http/handlers.go
    - backend/sourcing/scim/domain/mutation.go
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/user/db_memory/users.go
    - backend/idmanagement/user/db_postgres/users.sql
  tests:
    - backend/sourcing/scim/handlers_http/scim_test.go
    - backend/sourcing/scim/handlers_http/standards_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
    - backend/sourcing/scim/usecases/users_test.go
  stop_before_reading:
    - frontend
    - backend/provisioning/client_scim
    - backend/sourcing/scim/db_postgres
    - backend/sourcing/scim/domain/filter.go
affected_spec:
  - { path: docs/contexts/sourcing/scenarios.feature.md, requirement: REQ-SOURCING-002 }
  - { path: docs/contexts/sourcing/standards.md, requirement: RFC7644-DELETE-SEMANTICS }
  - { path: docs/contexts/sourcing/standards.md, requirement: RFC7644-RESOURCE-OPERATIONS }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.GetScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.UpdateScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.PatchScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.DeleteScimUser }
primary_use_cases:
  - id: scim-delete-removes-the-user
    requirement: RFC7644-DELETE-SEMANTICS
    observable_result: 削除した id への以後の SCIM 操作が 404 になり、その User が一覧、Group の members、他の User の manager から消える。
    unit_test: { path: backend/sourcing/scim/usecases/users_test.go, name: TestScimDeletedUserIsGoneFromEveryUsecasePath, task: test-go-race }
    e2e_test: { path: backend/sourcing/scim/handlers_http/standards_test.go, name: TestScimDeleteSemantics_DeletedUserIsGoneFromTheScimSurface, task: test-go-race }
    unit_fault_model: 削除済みの判定を参照系にだけ置き、変更系と member / manager の解決および射影を素通りさせる。
    e2e_fault_model: handler が usecases の ErrNotFound を 404 へ写さず、削除済みの id への GET が 200 と本文を返す。
---

# SCIM で削除した User を、以後の SCIM 操作と一覧から消す

## Motivation

`DeleteScimUser` は内部 User を soft delete し、`Lifecycle.Status` を `PendingDeletion` にする。
しかし SCIM の側では、この User が消えていない。
削除した id へ `GET /scim/v2/Users/{id}` を送ると 200 と本文が返り、`GET /scim/v2/Users` の一覧にも現れ、`PUT` と `PATCH` も通る。
Group の member として参照されていれば、`GET /scim/v2/Groups/{id}` の `members` にも残る。

RFC 7644 §3.6 は、サービス提供者が実際にレコードを消さない選択をしてよいと認めたうえで、削除した id に対する以後のすべての操作に 404 を返し、以後の照会結果からも除くことを MUST として要求している。
つまり本件は soft delete という実装選択の問題ではなく、**その選択を SCIM の表現に反映していない**ことの問題である。

この食い違いは [[wi-505-back-sourcing-standards-rows-with-tests]] が `RFC7644-RESOURCE-OPERATIONS` にテストを対応付ける過程で見つかった。
同項目は「4 操作を提供する」としか宣言していない行に対する欠陥ではないと判断し、削除後の到達性を規範として扱うことを本項目へ渡した。

外部 IdP から見ると、この状態は退職者の扱いを壊す。
削除を送った SCIM クライアントは、次の同期で同じ User を「まだ存在する現役の利用者」として読み戻す。
`active` が false になっているので無効化と区別できず、クライアント側の照合表に削除済みの id が残り続ける。

## Scope

- SCIM の入口から見て、削除済みの User を消す。
  - `GetScimUser`、`UpdateScimUser`、`PatchScimUser`、`DeleteScimUser` は、削除済みの id に対して 404 の `ScimProtocolError` を返す。
  - `ListScimUsers` は削除済みの User を `Resources` と `totalResults` の双方から除く。
  - Group の member 射影は削除済みの User を出さない。`CreateScimGroup` / `UpdateScimGroup` / `PatchScimGroup` が削除済みの id を member として受け取ったときは `invalidValue` で拒否し、Group を変更しない。
  - Enterprise 拡張の `manager` が削除済みの User の id を指すときは `invalidValue` で拒否する。読み取りの射影も同じで、`manager_sub` が削除済みの User を指す User の応答に `manager` を載せない。
- 削除済みと判定する状態を 1 箇所で決め、上記のすべてがその判定を通る形にする。
- `docs/contexts/sourcing/standards.md` に、削除後の意味論を宣言する行 `RFC7644-DELETE-SEMANTICS` を足し、その id を名指すテストを同時に書く。
- `docs/contexts/sourcing/scenarios.feature.md` の `REQ-SOURCING-002` に、削除後の到達性を観測する Example を足す。
- TypeSpec を宣言に合わせる。`UpdateScimUser` と `PatchScimUser` に 404 を足し、`DeleteScimUser` と `GetScimUser` の `@doc` を削除後の意味論まで含む記述にする。

## Out of Scope

- soft delete そのものをやめること。管理側の復元 (`PendingDeletion` からの復帰) は idmanagement が持つ機能であり、本項目はその手前で SCIM の表現だけを変える。
- `userName` の衝突判定から削除済み User を外すこと。判断と理由は Design に置く。
- Group の削除の意味論。Group は実削除なので既に 404 を返す。
- `PendingDeletion` を経ずに `Deleted` へ至る purge の経路と、その保持期間。
- 管理コンソールと管理 API における削除済み User の見え方。
- 削除済み User を member に持つ Group の membership 行そのものの掃除。射影から消すだけで、`group_members` の行は残す。
- `RFC7644-RESOURCE-OPERATIONS` の `Adoption` と `Strength` の変更。

## Design

### 削除済みの判定を 1 箇所に置く

判定の対象は `userdomain.User` の lifecycle であり、SCIM から見て消えているのは `PendingDeletion` と `Deleted` の 2 状態である。
`Disabled` は消えていない。無効化は `active: false` として表現する状態であり、削除と区別できなくなってはいけない。

判定を `backend/sourcing/scim/usecases` の述語 1 つに集める。

```go
// scimDeleted は SCIM の表現から消えている User を判定する (RFC7644-DELETE-SEMANTICS)。
func scimDeleted(user *userdomain.User) bool
```

置き場所を usecases にするのは、`sourcing/scim/domain` が idmanagement の型を知らないからである。
lifecycle は idmanagement が持つ概念であり、SCIM 側の domain へ持ち込むと、境界を越えた型の依存が 1 つ増える。
この述語は idmanagement の状態を SCIM の可視性へ翻訳する変換であり、その役割は adapter 側の usecases に属する。

呼ぶ場所は次の 7 箇所である。

| 場所 | 削除済みだったときの振る舞い |
|---|---|
| `Usecases.GetUser` | `ErrNotFound` |
| `Usecases.UpdateUser` | `ErrNotFound` |
| `Usecases.PatchUser` | `ErrNotFound` |
| `Usecases.DeleteUser` | `ErrNotFound`（二重の削除は 404） |
| `Usecases.ListUsers` | 射影せず読み飛ばす |
| `Usecases.resolveMemberUserIDs` と `resolveManagerSub` | `invalidValue` の `MutationError` |
| `Usecases.toScimUser` の `manager` 射影 | `manager` を載せない |

### 読み取りの `manager` 射影も落とす

`manager` の読み取り射影は、起票時の Scope では書き込み側だけを対象にしていた。
実装に入る前に読み取り側も含める形へ広げた。

書き込み側だけを拒否すると、読み取りが出した値を書き込みが拒む状態になるからである。
`toScimUser` は `manager_sub` を SCIM id へ解決して `manager` に載せ、`ParseUserWrite` は PUT の本文からその `manager` を読む。
SCIM クライアントが読んだ resource をそのまま PUT で送り返す経路（read-modify-write）は SCIM の同期実装では普通の形であり、そこで 400 `invalidValue` が返る。
この不一致は本項目が書き込み側だけを直したことで生まれるものなので、本項目が持つ。

`members` と同じく、削除済みの User を指す参照を応答から消せば、読み取りと書き込みが一致する。

`toScimGroup` の member 射影も同じ判定を通す。
member の一覧は `GroupRepo.ListMembersByGroup` が返す内部 User の id なので、射影の前に User を引いて判定する必要がある。
この追加の読み取りが member 数に比例することは受け入れる。1 回の Group 応答あたりの member 数は SCIM の同期単位に収まる規模であり、ここで N+1 を避けるための一括取得を先に入れると、判定の置き場所が 2 つに割れる。

`DeleteUser` は現在 `ref == nil` と `user == nil` で `errors.New("user not found")` を返しており、`ErrNotFound` を返していない。
handler 側が delete の失敗をすべて 404 に落としているので今は同じ結果になるが、判定を足すのに合わせて `ErrNotFound` へ揃える。

### 削除済み User は `userName` の衝突判定に残す

RFC 7644 §3.6 は、削除済みリソースを衝突判定に含めないことを SHOULD として求めている。
本項目はこれを採らない。

`users_preferred_username_active_idx` は `lifecycle->>'status' <> 'deleted'` を条件とする部分一意索引であり、`PendingDeletion` の行は `preferred_username` を占有し続ける。
衝突判定から外すには索引の条件を緩める必要があり、そうすると同じ `preferred_username` を持つ行が 2 つ存在しうる状態になる。
その状態は SCIM の外側、すなわち `FindByUsername` を使う認証の利用者名解決にまで届く。
本項目が持つのは SCIM の表現であって、利用者名の一意性が何を意味するかではない。

したがって、削除した `userName` で作り直そうとした SCIM クライアントは 409 の `uniqueness` を受け取る。
この振る舞いは `RFC7644-DELETE-SEMANTICS` の `Statement` に明記し、`Adoption` を `partial` にする。
採用していない範囲を行に書かずに残すと、後から読んだ者が §3.6 を全部満たしていると読む。

再作成の経路が塞がることの逃げ道は、管理側の復元と purge である。
この選択が SCIM クライアントにとって不便であると判明した場合は、索引と認証側の利用者名解決を含む別の work item が扱う。

### 規範の置き方

新しい行を足す。`RFC7644-RESOURCE-OPERATIONS` の `Statement` を広げる案は採らない。

同行は「User と Group リソースに作成、参照、置換、削除の操作を提供する」という、操作の存在についての宣言である。
削除後の到達性はそれとは別の性質であり、1 つの行に 2 つの性質を入れると、テストがどちらを固定しているのかが行から読めなくなる。
[[wi-495-burn-down-the-standards-coverage-debt]] が `Statement` に動詞が 2 つあれば観測も 2 つ要ると記録しているのは、この読みにくさのことである。

足す行は次の形にする。

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-DELETE-SEMANTICS | partial | MUST | 削除した User の id に対する以後の操作は 404 を返し、コレクションの照会結果、Group の `members`、Enterprise 拡張の `manager` のいずれにも現れない。内部の User レコードは soft delete として残す。削除済みの User は `userName` の一意性判定には残るため、同じ `userName` での再作成は 409 の `uniqueness` になる。 |

`Adoption` を `partial` にするのは、§3.6 の MUST を満たし SHOULD を満たさないからである。
テストは採用した側（404 と照会結果からの除外）と採用していない側（再作成の 409）の両方を観測する。

### 効果の境界

この変更は新しい効果を持ち込まない。
時刻、乱数、識別子生成、永続化のいずれも、既存の呼び出し位置から動かない。
追加されるのは、永続化から読んだ User を SCIM の表現へ射影するかどうかの判定だけである。

## Plan

1. `RFC7644-DELETE-SEMANTICS` の行と `REQ-SOURCING-002` の Example を足し、`mise run check-spec` を通す。
2. TypeSpec に `UpdateScimUser` と `PatchScimUser` の 404 を足し、`DeleteScimUser` と `GetScimUser` の `@doc` を直して、`mise run check-contract-drift` と `mise run check-api-compat` を通す。
3. HTTP 境界の受入検査を書き、削除後の `GET` が 200 を返すこと（現状）で RED を観測する。
4. `scimDeleted` を入れ、参照系（`GetUser`、`ListUsers`）を GREEN にする。
5. 変更系（`UpdateUser`、`PatchUser`、`DeleteUser`）を GREEN にする。
6. Group の member 射影、member / manager の解決、`manager` の読み取り射影を GREEN にする。
7. 再作成が 409 になることを、採用していない側の観測として固定する。

赤緑の 1 回ごとに使う recipe をここで選び、以後は選び直さない。

| 用途 | recipe |
|---|---|
| 単体の RED / GREEN / 故障注入 | `mise run test-go-test -- ./backend/sourcing/scim/usecases <Test>` |
| 受入の RED / GREEN / 故障注入 | `mise run test-go-test -- ./backend/sourcing/scim/handlers_http <Test>` |
| 1 つの振る舞いが GREEN になった後 | `mise run test-go-package -- ./backend/sourcing/scim/usecases` |
| パッケージ境界を越えた後 | `mise run test-go-changed` |
| Go が GREEN になった後 | `mise run lint-go` |

## Tasks

- [x] T001 [Spec] `RFC7644-DELETE-SEMANTICS` の行と `REQ-SOURCING-002` の Example (`EX-SOURCING-002-05`) を足す。
- [x] T002 [Spec] TypeSpec の 404 と `@doc` を宣言に合わせる。
- [x] T003 [Acceptance] 削除後の `GET` / `PUT` / `PATCH` / `DELETE` が 404 になり、一覧から消えることを HTTP 境界で観測する検査を書き、RED を確認する (`RFC7644-DELETE-SEMANTICS`、`EX-SOURCING-002-05`)。
- [x] T004 [Domain] `scimDeleted` を置き、参照系 (`GetUser`、`ListUsers`) を GREEN にする。
- [x] T005 [Use Cases] 変更系 (`UpdateUser`、`PatchUser`、`DeleteUser`) を GREEN にする。
- [x] T006 [Use Cases] Group の member 射影、member / manager の解決、`manager` の読み取り射影を GREEN にする。
- [x] T007 [Acceptance] 同じ `userName` での再作成が 409 になることを観測する。
- [x] T008 [Verify] `mise run verify`。

## Verification

- 削除した id に対する `GET` / `PUT` / `PATCH` / `DELETE` が 404 の `ScimProtocolError` を返す。
- 削除した User が `ListScimUsers` の `Resources` と `totalResults` の双方から消える。
- 削除した User が Group の `members` から消え、member および `manager` として指定すると `invalidValue` になる。
- 無効化 (`active: false`) した User は、いずれの経路からも消えない。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **`Disabled` まで消す。** 判定を `Lifecycle.Status != Active` と書くと無効化した User まで SCIM から消え、外部 IdP は無効化を削除と読む。判定は削除の 2 状態だけを対象にし、無効化した User が消えないことをテストで固定する。
- **判定が 1 箇所に収まらない。** 参照、変更、一覧、member 射影、member 解決、manager 解決の 6 経路それぞれが User を読む。どれか 1 つで判定を忘れると、その経路だけ削除済みの User が見え続ける。述語を 1 つ置き、経路ごとにテストを対応付ける。
- **`FindBySub` の既定が変わったと読む。** 本項目は idmanagement の読み取りを変えない。管理側は削除済み User を読めるままであり、変わるのは SCIM の射影だけである。
- **再作成の 409 を欠陥として読む。** これは §3.6 の SHOULD を採らないという判断であり、行の `Statement` に明記する。判断を変えるには索引と認証側の利用者名解決を含む別の work item が要る。
- **Group の member 射影の追加読み取り。** member 1 件ごとに User を引くので、member 数に比例した読み取りが増える。SCIM の同期単位を超える規模の Group が現れた場合は、判定の置き場所を割らずに一括取得へ替える。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は、`docs/contexts/sourcing/standards.md#RFC7644-DELETE-SEMANTICS` の追加と
  `REQ-SOURCING-002` の変更を返す。規範の差分はこの 2 件である。
  `REQ-SOURCING-002` の変更は `EX-SOURCING-002-05` の追加であり、既存の 4 例は動いていない。
  `mise run check-spec` は標準 156 行から 157 行、具体例 746 件から 747 件、テストが名指す id は
  281 件から 283 件になった。
  SCIM の入口から見て、削除した User が消えるようになった。削除した id への `GET` / `PUT` /
  `PATCH` / `DELETE` は 404 を返し、`ListScimUsers` の `Resources` と `totalResults` から消え、
  Group の `members` と他の User の `manager` にも現れない。削除済みの id を member または
  `manager` として送ると `invalidValue` で拒否され、対象の Group と User は変わらない。
  内部の User レコードは従来どおり soft delete のまま残る。
  **判定は述語 1 つに集めた。** `scimDeleted` は `userdomain.User` の `IsSoftDeleted` と `IsDeleted`
  を読む。状態符号を直接比較しない形にしたのは、`UserLifecycle.EffectiveStatus` が空の status を
  `Active` として扱う規則を idmanagement 側に残すためである。述語を通す経路は 7 つあり、そこへ
  `findLiveUserByScimID`（参照と変更の 4 経路）、`resolveLiveUserID`（member と manager の解決）、
  `liveUserByID`（member と manager の射影）の 3 つの入口から到達する。
  **読み取りの `manager` 射影を Scope へ入れた。** 起票時の Scope は書き込み側だけを挙げていたが、
  実装前に読み取り側も含める形へ広げた。`toScimUser` が出した `manager` を `ParseUserWrite` が
  読み戻すので、書き込みだけを拒否すると、SCIM クライアントが読んだ resource をそのまま PUT で
  送り返す経路が 400 `invalidValue` になる。この不一致は本項目が書き込み側を直したことで生まれる
  ものなので、本項目が引き取った。標準行の `Statement` にも `manager` を含めてある。
  **`DeleteUser` の戻り値を `ErrNotFound` へ揃えた。** これまで `errors.New("user not found")` を
  返しており、handler が delete の失敗をすべて 404 に落としていたので結果は同じだったが、
  判定を足すのに合わせて他の 3 経路と同じ型にした。
  **未リリースなので、リリースノートは移行の案内を持たない。** `documentation_impact` を `none` に
  することは検査が許さない（標準行の追加から `release_note` が導かれ、著者は強い側しか選べない）。
  したがって水準は `release_note` のまま、文面から既存の運用者へ向けた移行の段落を外し、
  何が変わったかだけを書いた。
- **Primary Use Case Evidence**:
  - id: scim-delete-removes-the-user
    unit_red: >-
      TestScimDeletedUserIsGoneFromEveryUsecasePath が 7 経路で失敗した。GetUser、UpdateUser、
      PatchUser、DeleteUser はいずれも err=<nil> で ErrNotFound を返さず、ListUsers は
      Total=2 len(Items)=2、Group の member 射影は削除済みを含む 2 件、CreateGroup と PatchUser の
      manager 解決は err=<nil>、manager の読み取り射影は削除済みの id を出した。区画
      "a disabled user is not deleted" だけが最初から通っており、無効化が消えていないことを
      RED の時点で確認できている。
    e2e_red: >-
      TestScimDeleteSemantics_DeletedUserIsGoneFromTheScimSurface が HTTP 境界で失敗した。
      削除後の GET / PUT / PATCH は 200 と本文、2 度目の DELETE は 204、一覧は totalResults = 2、
      Group の members は 2 件、削除済み member の追加は 400 のはずが 200、manager は削除済みの
      id を返し、削除済み manager の指定は 400 のはずが 200 だった。
    unit_fault_injection: >-
      findLiveUserByScimID から scimDeleted(user) の判定を外すと (宣言した fault model
      「判定を参照系にだけ置き、変更系と member / manager の解決を素通りさせる」に対応)、
      TestScimDeletedUserIsGoneFromEveryUsecasePath の 3 区画
      "the reading and writing paths report not found"、
      "member resolution refuses it and the group projection hides it"、
      "manager resolution refuses it and the user projection hides it" が落ちた。
    e2e_fault_injection: >-
      ListUsers から削除済みの読み飛ばしを外すと、
      TestScimDeleteSemantics_DeletedUserIsGoneFromTheScimSurface の区画
      "the collection omits it from Resources and totalResults" が totalResults = 2 と
      2 件の Resources で落ちた。応答の組み立てではなく HTTP の一覧本文を読んでいるので、
      射影の後で数え直す実装とも区別できる。
- **Change-Resistance Results**:
  判定を通す 5 つの地点それぞれで production を崩し、対応するテストが落ちることを観測した。
  注入のたびに `git diff --stat` で着弾を確かめ、観測後に元へ戻している。
  risk が `medium` なので契約上は代表 1 件で足りるが、判定が 1 箇所でも経路は 7 つあるため、
  経路ごとに独立して観測できることを示す必要があった。

  | 注入した故障 | 落ちたテストと観測 |
  |---|---|
  | `scimDeleted` を `!user.IsActive()` にする（判定が広すぎる） | 単体 `a disabled user is not deleted`: `GetUser err=SCIM resource not found`。受入 `a disabled user stays visible`: 200 のはずが 404。**無効化まで消えることを両境界が検出した** |
  | `findLiveUserByScimID` から `scimDeleted(user)` を外す | 単体 3 区画: 参照と変更の 4 経路が `err=<nil>`、member と manager の解決が `MutationError` を返さない |
  | `ListUsers` から読み飛ばしを外す | 単体 `the collection skips it without counting it`: `Total=2`。受入 `the collection omits it…`: `totalResults = 2` と 2 件の `Resources` |
  | `toScimGroup` から member の読み飛ばしを外す | 単体: `members` が 2 件。受入: `members` が 2 件、かつ拒否後の Group が 2 件（**拒否の効果側も落ちた**） |
  | `toScimUser` の `manager` 射影から判定を外す | 単体と受入の双方: `manager` が削除済みの id を返す |

  **無効化の側は 1 件目の注入でしか落ちない。** 残る 4 件は削除済みだけを対象にする判定なので、
  無効化を観測する区画は通ったままである。これは弱さではなく、判定の広さと置き場所が別の性質で
  あることの現れであり、1 件目が広さを、残りが置き場所を固定している。
- **Verification Results**:
  - `mise run verify` - passed（2026-09-12 に取得、exit 0、24.61s）
  - `mise run check-spec` - passed
    (`ok normative coverage (157 standard(s), 311 rule(s), 747 example(s), 283 id(s) named by a test)`)
  - `mise run check-contract-drift` - passed（0 finding(s)）
  - `mise run check-api-compat` - passed（no breaking changes vs the baseline）
  - `mise run check-status-drift` - passed（0 finding(s)）
  - `mise run check-work-items` - passed（533 件）
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-go-changed` - passed（`bootstrap`、`server_http`、`scim/handlers_http`、`scim/usecases`）
  - `mise run spec-diff` - `REQ-SOURCING-002` の変更と `RFC7644-DELETE-SEMANTICS` の追加
  - `mise run test-ui-e2e` - N/A: 変更は Go の 3 ファイル、テスト 2 ファイル、仕様文書と TypeSpec
    だけで、フロントエンドは 1 行も動いていない。SCIM は外部 IdP が呼ぶ経路であり、ブラウザーから
    到達しない。
