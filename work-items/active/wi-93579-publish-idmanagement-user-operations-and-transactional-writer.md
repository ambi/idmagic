---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
affected_spec:
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.GetAccountSecurity, impact: modifies }
---

# IdManagement のユーザー操作とトランザクションに参加する保存を公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 5 件は、Authentication と IdGovernance から IdManagement の非公開パッケージへの依存である。

| 利用側 | 参照先 | 使っているもの |
| --- | --- | --- |
| Authentication の `deps_http` と handler | `idmanagement/usecases`、`idmanagement/user/usecases` | アカウントのプロファイルの取得（`GetUserProfile`、`AccountProfileDeps`、`ErrUserNotFound`） |
| Authentication の `password/db_postgres` | `idmanagement/user/db_postgres` | パスワードの再設定と同じトランザクションでのユーザーの保存（`SaveUserTx`） |
| IdGovernance の `internal/db_postgres` | `idmanagement/user/db_postgres` | ライフサイクルの手順と同じトランザクションでのユーザーの保存（`SaveUserTx`） |

## 対象範囲

- アカウントのセキュリティ設定の取得が、IdManagement の公開契約だけで自分のユーザーを読む。
- 呼び出し側のトランザクションに参加するユーザーの保存を、IdManagement が公開し、組み立て地点で結ぶ。
- 解消した違反 ID を台帳から消す。
- `GetAccountSecurity` が既に返している 404 `user_not_found` を TypeSpec に宣言する。

## 対象外

- ユーザーの不変条件と保存表現の変更。
- Authentication の自己サービスの経路で、`authusecases.ErrUserNotFound` が `WriteAccountError` で 404 に写らず既定の扱いになる経路（WebAuthn、MFA、回復コード、追加認証）。応答を変えるため、別の作業で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D2：ユーザーの不変条件の検証と更新は IdManagement に残す。
- D7：パスワードの再設定とライフサイクルの手順は、ユーザーの保存と同じトランザクションで確定する必要がある。別のトランザクションで呼ぶ案は、片方だけが確定する窓を作るので採らない。
- [テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)で Tenancy の使用量に使った形（`pgx.Tx` を受け取る構築関数を所有者が公開し、利用側は構築関数を必須の引数として受け取り、組み立て地点が渡す）を、ユーザーの保存にも使う。

### プロファイルの取得（D3）

入力：`handleGetAccountSecurity` が使うのは、`GetUserProfile` の戻り値のうちユーザーだけである。
属性定義は捨てている。
ユーザーの取得は `FindBySub` と、見つからない、またはテナントが違うときの `ErrUserNotFound` だけである。

| 案 | 判断 |
| --- | --- |
| IdManagement が `GetUserProfile` 相当を公開パッケージへ置く | 採らない。使わない属性定義の読み取りと、使用例の型（`AccountProfileDeps`）を公開範囲へ広げる |
| Authentication の既存の `authusecases.LoadSelfUser` を使う。これは公開済みの `userports.UserRepository` の読み取りを合成している | 採る。IdManagement の公開契約表にある「`UserRepository` の読み取り」で足り、WebAuthn、MFA、回復コードの経路と同じ取得になる。公開範囲を広げない |

- 見つからないときの応答（404 `user_not_found`）は、ハンドラーが `authusecases.ErrUserNotFound` を同じ問題へ写して保つ。
  `WriteAccountError` へ写しを足すと、対象外に記した経路の応答も変わるので足さない。
- 属性スキーマの読み取りがなくなるので、その読み取りの失敗でセキュリティ設定の取得が失敗する経路は消える。
  利用者の観測する正常な応答は変わらない。
- `deps_http.AccountProfileDeps` は唯一の利用者がなくなるので消す。

### 仕様の差分

`GetAccountSecurity` は、変更前からユーザーが見つからないとき 404 `user_not_found` を返していた。
TypeSpec は 200、401、403 だけを宣言していた。
404 は `WriteAccountError` を経由して書かれていたので、状態コードの静的検査はこのずれを検出していなかった。
ハンドラーが 404 を直接書くと、検査がずれを報告する。

分類は (a) 要件にする（利用者の判断）。
現在の応答を契約として認め、`GetAccountSecurity` の応答へ `GetAccountSecurityError404`（本文は `IdMagic.Contract.UserNotFoundError`）を加える。
実装の応答は変えない。
宣言の追加は仕様差分に現れるので、`affected_spec` は `impact: modifies` にする。

### トランザクションに参加する保存（D2、D7）

```go
// backend/idmanagement/user/ports
type UserWriter interface {
    Save(ctx context.Context, user *userdomain.User) error
}

// backend/idmanagement/user/db_postgres
func UserWriterInTx(tx pgx.Tx) userports.UserWriter

// backend/authentication/password/db_postgres
func NewPasswordResetTokenStore(pool sharedpg.DB, usersInTx func(pgx.Tx) userports.UserWriter) *PasswordResetTokenStore

// backend/idgovernance/internal/db_postgres
func NewUserWorkflowCapture(pool sharedpg.DB, usersInTx func(pgx.Tx) userports.UserWriter) *UserWorkflowCapture

// backend/idgovernance
func NewPostgresModule(pool postgres.DB, users userports.UserRepository, usersInTx func(pgx.Tx) userports.UserWriter) Module
```

| 案 | 判断 |
| --- | --- |
| 保存だけの `UserWriter` ポートを公開し、`pgx.Tx` から作る構築関数を組み立て地点で渡す | 採る。トランザクションの所有者は利用側のまま変わらず、保存表現と sqlc の型は IdManagement に残る |
| `SaveUserTx` をそのまま公開パッケージへ移す | 採らない。sqlc の `DBTX` が利用側へ漏れる |
| `UserRepository` 全体を `pgx.Tx` から作る | 採らない。利用側が使うのは保存だけであり、読み取りまでトランザクションへ結ぶ契約を増やす |

- 構築関数は Tenancy の `QuotaRepositoryInTx` と同じく、組み立て地点（`bootstrap/postgres.go`）が渡す。
  IdManagement は公開方式が `legacy` なので、構築関数は `user/db_postgres` に置き、組み立て地点が直接参照する。
  `internal` へ移すときにルートパッケージへ移す。
- 構築関数を必須の引数にし、未設定のままトランザクションの外で保存する配線を作れないようにする。
- `SaveUserTx` は IdManagement の内側（メールアドレスの変更）だけが使うので、非公開の `internalSaveUser` へ置き換えて消す。
- 変更シナリオ：保存表現（列、JSONB の形）を変えても、利用側は変わらない。
  利用側の手順に新しい書き込みを足しても、IdManagement は変わらない。

残る仮定：利用側のトランザクションの確定と巻き戻しは利用側が所有し、`UserWriter` はコミットしない。

## タスク

- [x] T001 [Characterize] 変更する保存の経路が、トランザクションの巻き戻しをテストで固定しているかを確かめ、欠けていればテストを置く。
  - IdGovernance：`TestUserWorkflowCaptureCommitsUserAndRunsTogether`（EX-IDGOVERNANCE-004-01）が、ユーザーの保存の後の失敗で巻き戻ることを確かめている。
  - パスワードの再設定：`TestPasswordResetTokenStoreRollsBackWhenTheEffectFails` は、ユーザーの保存そのものを失敗させるので、保存がトランザクションの外へ出た誤りを検出しない。ユーザーの保存の後（履歴の追加）で失敗させ、ユーザーが変わらないことを確かめるテストを足す。
- [x] T002 [App] 公開ポートとトランザクションに参加する保存を置き、利用側と組み立て地点を書き換える。
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-test -- ./backend/authentication/password/db_postgres '.*'`、`./backend/idgovernance/internal/db_postgres`：保存を呼び出し側のトランザクションの外で行う変異を手で入れ、検出を確かめる。
- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`：組み立て地点とモジュールの構築関数の引数を変えるので、逆依存の範囲を絞らない。

## リスク

- 保存を呼び出し側のトランザクションの外で行う誤りは、通常の経路のテストでは見えない。
  書き込みの失敗で全体が巻き戻ることをテストで確かめ、トランザクションの外で保存する変異を手で入れて検出を確かめる。

## 要約

IdManagement が `userports.UserWriter` と `pgx.Tx` から作る `userpostgres.UserWriterInTx` を公開した。
パスワードの再設定と IdGovernance のワークフローの記録は、構築関数を必須の引数として受け取り、組み立て地点が渡す。
アカウントのセキュリティ設定の取得は、公開済みの `UserRepository` の読み取りを合成する `LoadSelfUser` へ移し、IdManagement に新しい公開操作を作らなかった（D3）。
`private-import` の負債は 9 件から 4 件になった。

仕様差分は `GetAccountSecurity` の TypeSpec 宣言だけである。
変更前から返していた 404 `user_not_found` が宣言されていなかったので、`GetAccountSecurityError404` を加えて契約を応答に合わせた。

パスワードの再設定には、User の保存の後で書き込みが失敗したときに User の変更も巻き戻るテストを加えた。
両方の経路で、User を別のトランザクションで確定させる変異を手で入れ、テストが検出することを確かめた。
