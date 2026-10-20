---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/user.lifecycle.md, requirement: REQ-IDMANAGEMENT-010, impact: conforms }
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-024, impact: conforms }
primary_use_cases:
  - id: login-keeps-a-concurrent-disable
    requirement: REQ-IDMANAGEMENT-010
    observable_result: ログインが User を読んでから最終ログインを記録するまでの間に管理者が User を無効化すると、ログインの後も User は Disabled のまま残る。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/concurrent_user_writes_e2e_test.go, name: TestLoginKeepsADisableMadeWhileItRecordsTheLogin, task: test-go-race }
    fault_model: ログインが先に読んだ User を丸ごと保存し、その間の無効化を書き戻す。
  - id: password-change-keeps-a-concurrent-disable
    requirement: REQ-AUTHENTICATION-024
    observable_result: パスワードの変更が新しいハッシュを保存するまでの間に管理者が User を無効化すると、変更の後も User は Disabled のまま残り、update_password は外れる。
    boundary: acceptance
    test: { path: backend/shared/http/server_http/concurrent_user_writes_e2e_test.go, name: TestPasswordChangeKeepsADisableMadeWhileItSavesTheHash, task: test-go-race }
    fault_model: パスワードの変更が先に読んだ User を丸ごと保存し、その間の無効化を書き戻す。
---

# Authentication は User の変える項目だけを更新する

## 動機

Authentication は、ログイン、パスワードの変更、TOTP の登録のたびに、IdManagement の `UserRepository.Save` で User を丸ごと保存する。
変える値は最終ログイン時刻、パスワードのハッシュと変更時刻、`update_password` の付与と解除、`mfa_enrolled` だけである。
`SaveUser` は全列を上書きし、版も確かめない。
そのため、ログインの処理が User を読んでから保存するまでの間に管理者が User を無効化すると、ログインの保存が `lifecycle` ごと `Active` を書き戻し、REQ-IDMANAGEMENT-010 の無効化が失われ得る。
同じ競合で、管理者による属性とロールの更新も失われ得る。

## 対象範囲

- `backend/authentication/internal/signin/handlers_http/login.go:recordLoginAndRequiredAction`、`internal/password/usecases/change_password.go`、`internal/mfa/usecases/account_mfa.go` の `UserRepo.Save` を、IdManagement が公開する、変える項目だけを確定する操作へ置き換える。
- 置き換えた操作が、同時に行われた User の無効化、属性とロールの更新を書き戻さないことを検査する。
- 修正後に、[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)の適用状況と、IdManagement のデータとリスクの文書から、Authentication による User の保存を除く。

## 対象外

- SCIM による User の保存（[wi-11577](wi-11577-route-scim-writes-through-identity-management-use-cases.md)）、seed の保存（[wi-24943](wi-24943-route-seed-writes-through-context-use-cases.md)）。
- 管理 API の楽観的な排他制御（[wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md)）。

## 設計

IdManagement が、User の一部の項目だけを書く操作を公開し、Authentication はそれを呼ぶ。
必須操作の付与と解除は、読んだ時点の User から計算した一覧で上書きせず、操作の中で現在の値に対して行う。

### 着手時の調査

Authentication から User を丸ごと保存する書き手は、起票時の 3 箇所ではなく 5 箇所あった。

| 書き手 | 変える値 |
| --- | --- |
| ログインの記録（`login.go:recordLoginAndRequiredAction`） | 最終ログイン時刻。期限切れなら `update_password` を足す |
| パスワードの変更（`change_password.go`） | ハッシュ、更新時刻、変更時刻、`update_password` を外す |
| パスワードの再設定（`PasswordResetTokenStore.ConsumeAndApply`。トークンの使用済み化と同じトランザクション） | 同上 |
| TOTP の登録（`account_mfa.go`） | `mfa_enrolled` を true にする |
| 第二要素の増減の同期（`SyncMfaEnrolled`。WebAuthn の登録と削除、TOTP の削除、管理者のリセット） | `mfa_enrolled` を計算し直す |

最終ログイン時刻、変更時刻、必須操作は、無効化の状態（`status`）と同じ JSONB の列 `lifecycle` にある。

### 型と操作

列だけを書く更新にすると、`lifecycle` の中のキーを `jsonb_set` で書く SQL を 3 種類持ち、期限切れの判定も SQL の中で行うことになる。
代わりに、行を固定して読み直した最新の User へ Go の関数で変更を当てる操作を一つ設ける。
判定は Go に残り、変更を当てる値は常に最新なので、その間の無効化や編集を書き戻さない。
行を固定する読み取りは SAML の IdP プロファイル（`GetSamlIDPProfileForUpdate`）に前例がある。

```go
// package userports
type UserRepository interface {
    // ...既存の操作
    // Update は sub の User の行を固定して読み直し、change をその最新の値に当てて保存する。
    // change が false を返すと保存しない。User がない (削除済みを含む) ときは change を呼ばずに nil を返す。
    // change の中で外部の作用を起こさない。
    Update(ctx context.Context, sub string, change func(*userdomain.User) (bool, error)) (*userdomain.User, error)
}

type UserWriter interface {
    Save(ctx context.Context, user *userdomain.User) error
    // Update は呼び出し側のトランザクションの中で、UserRepository.Update と同じことを行う。
    Update(ctx context.Context, sub string, change func(*userdomain.User) (bool, error)) (*userdomain.User, error)
}
```

`PasswordResetCommit` は保存済みの User の代わりに、`UserID` と、最新の User に新しいパスワードを当てる `Apply` を持つ。
パスワードの変更と再設定は、新しいパスワードを当てる同じ関数を使う。
ハッシュの計算と履歴の照合は、行を固定する前に行う。

設計ガイドラインの[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)は「その列だけを書く更新」と書いているので、行を固定して最新の値へ変更を当てる操作も同じ表の形として加える。

### 検証の境界

ログインとパスワードの変更は、正式な組み立てを通る E2E で、保存の直前に無効化を挟んで観測する。
TOTP の登録、第二要素の同期、再設定のストアは usecase とストアの単位で同じ故障を観測する。
`Update` の意味はメモリと PostgreSQL の共有の契約テストで確かめる。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| 現在のパスワードのハッシュを Authentication の集約へ移す | User の作成と CSV の取り込みが、二つのモジュールの保存を同じトランザクションで行う形になる。競合の解消には不要で、境界は単純にならない |
| `SaveUser` に版の比較を加える | 呼び出し元すべてに再試行が要り、関係のない項目どうしの競合でも失敗する |

## 計画

1. 着手時に、Authentication からの User の保存と、各操作が読んで判定する値を列挙し、操作のシグネチャを設計へ記す。
2. ログインの保存の直前に無効化を挟むテストで、競合の故障を RED にする。
3. 操作を実装し、Authentication の呼び出しを置き換えて GREEN にする。

## タスク

- [x] T001 [Design] 保存と判定を列挙し、操作のシグネチャを記す。
- [x] T002 [Acceptance] ログイン、パスワードの変更、TOTP の登録が、同時の無効化を書き戻す故障を RED にする。
- [x] T003 [App] IdManagement の操作を実装し、Authentication の保存を置き換える。
- [x] T004 [Verify] 変更を検証する。

## 検証

- メモリと PostgreSQL の両方の User の保存で、競合の検査を行う。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`、`mise run check-boundaries`。

## リスク

読んだ User から判定した値（パスワードの期限切れによる必須操作の付与など）を書く処理は、項目を絞っても同じ競合を残し得る。
判定と確定を同じ操作の中で行う。

## 要約

規範仕様の差分はない。
Authentication が User を丸ごと保存していた書き手は、起票時の 3 箇所ではなく、ログインの記録、パスワードの変更と再設定、TOTP の登録、第二要素の増減の同期の 5 箇所あった。
受け入れテストで、ログインとパスワードの変更が、保存の直前に行われた無効化を書き戻すことを RED にしてから直した。

変える値が無効化の状態と同じ JSONB の列 `lifecycle` にあり、期限切れの判定も最新の値から行う必要があるので、列だけを書く更新ではなく、行を `FOR UPDATE` で固定して読み直した最新の User へ変更を当てる `UserRepository.Update` と `UserWriter.Update` を IdManagement に設けた。
5 箇所の書き手はすべてこれを使い、パスワードの再設定はトークンの使用済み化と同じトランザクションで `UserWriter.Update` を使う。
パスワードの変更と再設定は、新しいパスワードを当てる同じ関数を共有する。
データ設計の同時の更新の規則にこの形を加え、適用状況と IdManagement のデータとリスクの文書から Authentication の保存を除いた。

`Update` の意味はメモリと PostgreSQL の共有の契約テストで、再設定の確定が無効化を書き戻さないことはパスワードの永続化の契約テストで確かめた。
TOTP の登録と解除のテストは修正の後に書いたので、2 つの書き手を修正前の版に戻して失敗することを確かめた。
管理者の編集による丸ごとの保存が Authentication の記録を書き戻す逆向きの競合は、wi-605 の版による検出に残る。
