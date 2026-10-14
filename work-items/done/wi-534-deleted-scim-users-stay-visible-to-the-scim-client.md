---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SCIM で削除した User を、以後の SCIM 操作と一覧から消す

`mise run spec-diff` は、`docs/domain/sourcing/standards.md#RFC7644-DELETE-SEMANTICS` の追加と
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

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
