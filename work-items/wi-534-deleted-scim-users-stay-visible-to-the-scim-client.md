---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/sourcing/scenarios.feature.md, requirement: REQ-SOURCING-002 }
  - { path: docs/contexts/sourcing/standards.md, requirement: RFC7644-RESOURCE-OPERATIONS }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.DeleteScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.GetScimUser }
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
  - Enterprise 拡張の `manager` が削除済みの User の id を指すときは `invalidValue` で拒否する。
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

呼ぶ場所は次の 6 箇所である。

| 場所 | 削除済みだったときの振る舞い |
|---|---|
| `Usecases.GetUser` | `ErrNotFound` |
| `Usecases.UpdateUser` | `ErrNotFound` |
| `Usecases.PatchUser` | `ErrNotFound` |
| `Usecases.DeleteUser` | `ErrNotFound`（二重の削除は 404） |
| `Usecases.ListUsers` | 射影せず読み飛ばす |
| `Usecases.resolveMemberUserIDs` と `resolveManagerSub` | `invalidValue` の `MutationError` |

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
| RFC7644-DELETE-SEMANTICS | partial | MUST | 削除した User の id に対する以後の操作は 404 を返し、コレクションの照会結果と Group の `members` にも現れない。内部の User レコードは soft delete として残す。削除済みの User は `userName` の一意性判定には残るため、同じ `userName` での再作成は 409 の `uniqueness` になる。 |

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
6. Group の member 射影と member / manager の解決を GREEN にする。
7. 再作成が 409 になることを、採用していない側の観測として固定する。

## Tasks

- [ ] T001 [Spec] `RFC7644-DELETE-SEMANTICS` の行と `REQ-SOURCING-002` の Example を足す。
- [ ] T002 [Spec] TypeSpec の 404 と `@doc` を宣言に合わせる。
- [ ] T003 [Acceptance] 削除後の `GET` / `PUT` / `PATCH` / `DELETE` が 404 になり、一覧から消えることを HTTP 境界で観測する検査を書き、RED を確認する。
- [ ] T004 [Domain] `scimDeleted` を置き、参照系を GREEN にする。
- [ ] T005 [Use Cases] 変更系を GREEN にする。
- [ ] T006 [Use Cases] Group の member 射影と、member / manager の解決を GREEN にする。
- [ ] T007 [Acceptance] 同じ `userName` での再作成が 409 になることを観測する。
- [ ] T008 [Verify] `mise run verify`。

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
