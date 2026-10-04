---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 大文字と小文字だけが異なるユーザー名の作成と、ほかの User と同じメールアドレスでの作成と更新を、管理 API と CSV が拒否するようになる。JIT と CSV で作った User が動的グループに所属するようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-83020-unify-principal-names-and-user-creation-paths.md }
initial_context:
  specification:
    - docs/domain/identity-management/README.md
    - docs/domain/identity-management/user/README.md#REQ-IDMANAGEMENT-042
    - docs/domain/identity-management/user/README.md#REQ-IDMANAGEMENT-043
    - docs/domain/identity-management/agent/README.md#REQ-IDMANAGEMENT-073
    - docs/domain/identity-management/group/README.md#REQ-IDMANAGEMENT-060
    - docs/domain/identity-management/user-csv/README.md#REQ-IDMANAGEMENT-056
    - docs/domain/identity-management/design/data.md
  typespec: [IdMagic.IdManagement.Operations.CreateAdminUser, IdMagic.IdManagement.Operations.UpdateAdminUser]
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/usecases/federated_user.go
    - backend/idmanagement/user/usecases/user_import_apply.go
    - backend/idmanagement/user/usecases/user_import_planner.go
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/shared/http/server_http/routes.go
    - infra/schema/postgres.sql
  tests:
    - backend/idmanagement/user/usecases/user_rules_test.go
    - backend/authentication/federation/handlers_http/federated_login_e2e_test.go
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-089 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-090 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-056 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-075 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-072 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.CreateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAdminUser }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.EmailTakenError }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UsernameConflictError }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupNameConflictError }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AgentNameConflictError }
primary_use_cases:
  - id: admin-create-case-variant-username
    requirement: REQ-IDMANAGEMENT-042
    observable_result: テナントに alice がいるとき、管理者による Alice の作成が username_conflict で拒否され、User が増えない。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_rules_test.go, name: TestCreateUserTrimsTheUsernameAndComparesItCaseInsensitively, task: test-go-race }
    fault_model: ユーザー名の検索が表記どおりの完全一致のままで、大文字と小文字だけが異なる User を二人作る。
  - id: admin-create-duplicate-email
    requirement: REQ-IDMANAGEMENT-042
    observable_result: ほかの User と大文字と小文字だけが異なるメールアドレスでの管理者による作成が、409 と email_taken で拒否される。
    boundary: adapter
    test: { path: backend/idmanagement/user/handlers_http/admin_user_handler_test.go, name: TestCreateAdminUserRejectsAnEmailAnotherUserHas, task: test-go-race }
    fault_model: 作成がメールアドレスの重複を調べない、または衝突を 409 email_taken に変換しない。
  - id: jit-joins-dynamic-group
    requirement: REQ-IDMANAGEMENT-089
    observable_result: 正式な入口で組み立てた JIT が、規則に一致する属性の User を動的グループに所属させる。
    boundary: adapter
    test: { path: backend/idmanagement/user/handlers_http/federated_user_provisioner_test.go, name: TestFederatedUserProvisionerEvaluatesDynamicGroups, task: test-go-race }
    fault_model: JIT の依存に Group のリポジトリが配線されず、動的グループの評価が黙って省かれる。
  - id: csv-create-joins-dynamic-group
    requirement: REQ-IDMANAGEMENT-004
    observable_result: User の CSV の適用で作った、規則に一致する User が動的グループに所属する。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_import_apply_test.go, name: TestApplyUserImportEvaluatesDynamicGroups, task: test-go-race }
    fault_model: CSV の適用が確定した User を動的グループの規則で評価しない。
  - id: csv-rejects-duplicate-email
    requirement: REQ-IDMANAGEMENT-056
    observable_result: ほかの User と同じメールアドレスの行が email_taken、前の行と同じメールアドレスの行が duplicate_email で rejected になる。
    boundary: unit
    test: { path: backend/idmanagement/user/usecases/user_import_planner_test.go, name: TestPlanUserImportRejectsDuplicateEmails, task: test-go-race }
    fault_model: 計画器がメールアドレスの重複を調べない。
  - id: postgres-name-key-unique
    requirement: REQ-IDMANAGEMENT-042
    observable_result: PostgreSQL のリポジトリが、大文字と小文字だけが異なるユーザー名の二人目の User の保存を拒否し、FindByUsername が表記によらず同じ User を返す。
    boundary: adapter
    test: { path: backend/idmanagement/user/db_postgres/contract_test.go, name: TestPersistenceContract, task: test-go-race }
    fault_model: 比較キーの列を保存しない、または一意索引が表記の列に残る。
---

# User、Group、Agent の名前を一つの値オブジェクトで比較し、User を作るすべての経路で同じ検証と動的グループの評価を行う

## 動機

IdManagement では、同じ種類の値と操作が、Aggregate や経路ごとに異なる振る舞いをしている。

| 対象 | 今の挙動 |
| --- | --- |
| 名前の比較 | User のユーザー名は大文字と小文字を区別して一意性を判定する。Group と Agent の名前は `strings.EqualFold` で区別せずに判定する。グループのメンバーシップの CSV は、ユーザー名を区別せずに照合する |
| メールアドレスの重複 | フェデレーションの JIT とメールアドレスの変更は重複を拒否し、管理者による作成と更新、CSV は拒否しない |
| 動的グループの評価 | 管理者による作成と更新は、User を動的グループの規則で評価する。JIT と CSV の適用は評価しないので、それらで作った User は次の全件の再評価まで動的グループに所属しない |

主要な製品と規格は、ユーザー名を大文字と小文字を区別せずに比較する。

| 製品・規格 | ユーザー名 | 名前を識別子とするグループ |
| --- | --- | --- |
| Okta | ログイン名は大文字と小文字を区別しない | 一意。大文字と小文字の扱いは公開文書にない |
| Microsoft Entra ID | UPN は大文字と小文字を区別しない | 表示名は一意でなくてよい |
| Google Workspace | メールアドレスを小文字にそろえる | グループのメールアドレスで識別し、区別しない |
| AWS IAM | 区別しない | 区別しない（`ADMINS` と `admins` を両方は作れない） |
| Keycloak | 保存時に小文字へ変換する | 同じ親の下で一意。完全一致で判定し、区別する |
| SCIM（RFC 7643） | `userName` は `caseExact: false` | `displayName` は一意性を持たない |

メールアドレスの一意性は、Google Workspace、Keycloak（デフォルト）、Auth0 が強制し、Okta と Entra ID は強制しない。
動的な所属は、Okta のグループ規則と Entra ID の動的グループが、作成の経路によらず評価する。
判断の分かれる点は、利用者の指示により、主要な IdP の多数派に合わせる。

## 対象範囲

- User のユーザー名、Group の名前、Agent の名前を、一つの値オブジェクト（名前）で正規化し、比較する。
  - 前後の空白を除いて保存する。
  - 入力された表記のまま保存し、表示する。
  - Unicode の case folding をした値で比較し、大文字と小文字を区別しない。
  - 同じテナントの削除されていない同じ種類の Aggregate の間で一意にする。
- メールアドレスの重複を、User を作るすべての経路（管理者による作成、CSV の適用、フェデレーションの JIT）と、メールアドレスの変更（管理者による更新、CSV による更新、本人による変更）で拒否する。
- User を作るすべての経路で、作った User を動的グループの規則で評価する。CSV の適用は、更新した User も評価する。
- 値オブジェクトの定義を IdManagement の `README.md` のモデルの節へ移し、各機能の要件はその定義を参照する。
- PostgreSQL の一意性の制約を、比較キーを保存した列への一意索引に置き換える。

## 対象外

- 既存データの移行。
  製品は未リリースなので、スキーマの宣言を変えるだけにする。
- 名前とメールアドレスの Unicode の正規化（NFKC、幅の変換など）。
- SCIM による User の作成と更新（`Sourcing`）。
  ユーザー名の一意性はリポジトリの検索を通して同じ比較になるが、メールアドレスの重複の拒否、使用量の上限、動的グループの評価は加えない。
  SCIM は外部の権威からの取り込みであり、拒否の仕方を SCIM のエラー応答として決める別の作業になる。
- メールアドレスの一意性を PostgreSQL の制約で守ること。
  SCIM の取り込みが重複を書けるので、制約を置くと取り込みが保存の時点で失敗する。アプリケーションの判定だけで守る。
- 本人のメールアドレスの変更（`RequestEmailChange`）の TypeSpec に 409 がない食い違い。
- 管理画面の部分一致検索の、ASCII 以外の大文字と小文字の扱い。DB のロケールに依存したままにする。

## 設計

### 比較キー

`strings.EqualFold` と `strings.ToLower` は Unicode の一部の文字で結果が異なり、PostgreSQL の `lower()` はデータベースのロケールで結果が変わる。
名前とメールアドレスの比較キーは Go の一つの関数だけで作り、アプリケーションの判定とデータベースの制約が同じキーを使う。

```go
// backend/idmanagement/domain/principal_name.go
func NameKey(name string) string   // 前後の空白を除き、cases.Fold() で case folding した値
func EmailKey(email string) string // NameKey と同じ変換
```

`cases.Fold()` は Unicode のデフォルトの caseless matching を行い、`ß` と `SS`、語末の `ς` と `Σ` を一致させる。
`cases.Caser` はゴルーチンの間で共有できないので、呼び出しごとに作る。

### 永続化

| 表 | 追加する列 | 一意性 |
| --- | --- | --- |
| `users` | `preferred_username_key`、`email_key` | `(tenant_id, preferred_username_key)` の削除されていない行の部分一意索引。`email_key` は検索の索引だけ |
| `groups` | `name_key` | `(tenant_id, name_key)` の一意制約 |
| `agents` | `name_key` | `(tenant_id, name_key)` の一意制約 |

キーを書くのは `SaveUser`、`SaveGroup`、`SaveAgent` の三つの SQL だけで、Go のリポジトリが `NameKey` と `EmailKey` で値を渡す。
`FindByUsername` と `FindByEmail` はキーの列で検索する。
メモリのリポジトリも同じ関数で比較する。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| `lower()` の式索引 | ロケールで結果が変わり、Go の比較と食い違うと、アプリケーションの判定を通った保存が DB の制約で 500 になる |
| Keycloak と同じく小文字へ変換して保存する | 利用者が入力した表記を表示できなくなる。Okta、Entra ID は表記を保つ |
| User だけ大文字と小文字を区別したまま残す | 同じ製品の中で名前の扱いが Aggregate ごとに異なり、CSV の照合とも食い違う |

### User の作成の共有

管理者による作成と JIT は、一つの関数で作成を確定する。

```go
// backend/idmanagement/user/usecases/user_creation.go
type newUser struct {
    User                *userdomain.User // ID、時刻、状態を除いた作成する User
    ActorUserID         string
    PasswordHistoryHash string // 空なら履歴を作らない
}
func createUser(ctx context.Context, deps AdminUserDeps, in newUser, now time.Time) (*userdomain.User, error)
// ユーザー名とメールアドレスの一意性、属性スキーマ、使用量の上限を検証し、ID を採番して確定し、
// 動的グループを評価し、パスワードの履歴、UserCreated、下流への通知を行う。
func ensureUserIdentityAvailable(ctx context.Context, repo userports.UserRepository, tenantID, selfID, username string, email *string) error
// UpdateUser も使う。selfID の User 自身との一致は衝突にしない。
func syncDynamicGroups(ctx context.Context, deps AdminUserDeps, user *userdomain.User, now time.Time) error
```

CSV の適用は、プレビューと同じ計画器で全行を計画し、一行ずつ不可分に確定するので、`createUser` を通さない。
計画器が `NameKey` と `EmailKey` で索引を引いて一意性を判定し、確定した後に `ApplyUserImport` が作成と更新の行の User を動的グループで評価する。
評価の依存は `UserImportApplyDeps.DynamicGroups`（`groupusecases.DynamicGroupDeps`）として渡す。

時刻はユースケースの入力、ID は `spec.NewUUIDv4`、永続化はリポジトリとコミッターのポートで行う。

### JIT の配線

`routes.go` は JIT の依存を、管理者向けハンドラーとは別に手で組み立てていたため、Group のリポジトリが渡っていなかった。
`userhttp.FederatedUserProvisioner(d Deps)` を公開し、管理者の操作と同じ `adminUserDeps(d)` から依存を作る。
`routes.go` は IdManagement のルートに渡す `Deps` をこの関数にも渡す。

### 要件の差分

| 要件 | 差分 |
| --- | --- |
| IdManagement の値オブジェクト（新設） | 名前とメールアドレスの正規化、比較、一意性の範囲を定義する。User の機能仕様にあった値オブジェクトの表を移す |
| REQ-IDMANAGEMENT-089（新設） | User の作成は、経路によらず、名前またはメールアドレスがほかの User と同じ User を作らず、作った User を動的グループの規則で評価する |
| REQ-IDMANAGEMENT-042 | 大文字と小文字だけが異なるユーザー名を `username_conflict` で、同じメールアドレスを 409 `email_taken` で拒否する。メールアドレスを拒否しないとしていた文を消す |
| REQ-IDMANAGEMENT-043 | 題を「ロールとパスワードの履歴を持たず、操作者を記録する」に変え、メールアドレスの重複の拒否と、動的グループを評価しない文を消す（REQ-IDMANAGEMENT-089 へ） |
| REQ-IDMANAGEMENT-090（新設） | 管理者による User の更新は、ほかの User と名前が同じユーザー名を `username_conflict` で、メールアドレスが同じメールアドレスを 409 `email_taken` で拒否し、自分の表記だけを変える更新を受け付ける |
| REQ-IDMANAGEMENT-004 | CSV の適用は、作成または更新した User を動的グループの規則で評価する |
| REQ-IDMANAGEMENT-056 | ユーザー名を名前の定義で照合する。ほかの User と同じメールアドレスの行を `email_taken`、前の行と同じメールアドレスの行を `duplicate_email` で拒否する |
| REQ-IDMANAGEMENT-060、073、072 | 「大文字と小文字を区別せずに同じ名前」を名前の値オブジェクトへの参照に変える |
| REQ-IDMANAGEMENT-075 | 更新でほかの Agent と同じ名前を `agent_name_conflict` で拒否する文を加える |
| `EmailTakenError`（TypeSpec、新設） | `CreateAdminUser` と `UpdateAdminUser` の 409 に加える |

例は、EX-IDMANAGEMENT-042-02 を「大文字と小文字だけが異なるユーザー名は衝突する」に改め、EX-IDMANAGEMENT-043-01 と EX-IDMANAGEMENT-043-03 を REQ-IDMANAGEMENT-089 の EX-IDMANAGEMENT-089-01、EX-IDMANAGEMENT-089-02 に置き換える。

### 仕様にない振る舞いの分類

| 振る舞い | 分類 | 対応 |
| --- | --- | --- |
| JIT と CSV の適用が動的グループを評価しない | (c) | 評価する |
| Agent の更新が、ほかの Agent と同じ名前を拒否する | (a) | REQ-IDMANAGEMENT-075 に書く |
| 管理者による更新が、ほかの User と同じメールアドレスを受け付ける | (c) | 拒否する |
| CSV の適用が下流への通知と `UserCreated` の発行をしない | (b) | 変えない。CSV は監査記録を確定と同じトランザクションで書く別の設計である |
| SCIM の作成がメールアドレスの重複と使用量を調べない | 対象外 | 対象外に記録した |
| CSV の適用で動的グループの評価が失敗する | (a) | 確定した行を残して適用を失敗させ、変更なしの行も評価して再適用で回収する（REQ-IDMANAGEMENT-004） |
| 作成の検証の順序（パスワードポリシーと一意性のどちらを先に返すか） | (b) | 固定しない。複数の違反があるときに返すエラーだけが変わる |
| Authentication の検証済みメールアドレスの照合が `strings.EqualFold` で二重に確かめる | (b) | 比較キーより厳しい側に倒れるだけなので変えない |

## 計画

1. 値オブジェクトと要件の差分を仕様に反映する。
2. 比較キーとリポジトリを変え、ユーザー名の比較を変える。
3. User を作る経路をまとめ、メールアドレスの重複と動的グループの評価を加える。
4. CSV の計画器と適用を変える。
5. Group と Agent の名前の比較を値オブジェクトへ置き換える。
6. スキーマの一意性の制約を更新する。

## タスク

- [x] T001 [Spec] 値オブジェクトと要件の差分を書く。`mise run check-spec` で、新しい例 EX-IDMANAGEMENT-089-01、EX-IDMANAGEMENT-089-02 を引くテストがないことの失敗を観測した。
- [x] T002 [Acceptance] 大文字と小文字だけが異なるユーザー名の作成が拒否されることの RED を確認する。`mise run test-go-package -- ./backend/idmanagement/user/usecases` で、TestCreateUserTrimsTheUsernameAndComparesItCaseInsensitively（REQ-IDMANAGEMENT-042）が、Alice の作成を受け付けて（`err=<nil>, want ErrUsernameConflict`）失敗した。
- [x] T003 [App] 作成の経路を共有の仕組みにまとめ、値オブジェクトで比較する。各振る舞いの RED と GREEN は `mise run test-go-test -- <package> <test>`、パッケージ単位は `mise run test-go-package`、DB を通す確認は `dangerouslyDisableSandbox` で行った。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run verify`

## リスク

- 作成の経路をまとめると、CSV の適用やフェデレーションの JIT の、ほかの差まで変わる。
  経路ごとの差を観点表で洗い出し、意図した差だけを要件として残す。

## 完了

- **Completed At**: 2026-10-05
- **Summary**:
  `mise run spec-diff -- main` は、REQ-IDMANAGEMENT-089 と REQ-IDMANAGEMENT-090 の追加、REQ-IDMANAGEMENT-004、042、043、056、060、072、073、075 の変更、`EmailTakenError` の追加、`CreateAdminUser`、`UpdateAdminUser`、`UsernameConflictError`、`GroupNameConflictError`、`AgentNameConflictError` の変更を示す。
  User のユーザー名、Group と Agent の名前は、IdManagement の値オブジェクト（名前）として、前後の空白を除き、表記を保ったまま Unicode の case folding で比較する。
  メールアドレスは同じテナントの削除されていない User の間で一意になり、管理者による作成と更新は 409 と `email_taken`、User CSV は `email_taken` と `duplicate_email` で重複を拒否する。
  管理者による作成とフェデレーションの JIT は一つの作成の関数を通り、JIT と User CSV の適用で作った User も動的グループの規則で評価する。CSV の適用は、更新と変更なしの行も評価し、評価に失敗した適用を再適用で回収できる。
  PostgreSQL は `users`、`groups`、`agents` に比較キーの列を持ち、名前の一意性をその列の一意索引で守る。
- **Primary Use Case Evidence**:
  - id: admin-create-case-variant-username
    red: 実装前に TestCreateUserTrimsTheUsernameAndComparesItCaseInsensitively が、Alice の作成を受け付けて（`err=<nil>, want ErrUsernameConflict`）失敗した。
    fault_injection: メモリのリポジトリが比較キーではなく表記で引いていた変更前の状態がこの RED であり、PostgreSQL 側は下の postgres-name-key-unique の注入で確かめた。
  - id: admin-create-duplicate-email
    red: 実装前に TestCreateAdminUserRejectsAnEmailAnotherUserHas が `second status=500` で失敗した。
    fault_injection: 管理 API のエラー変換 `writeAdminUserError` から `ErrEmailTaken` の分岐を消すと、同じテストが `second status=500` で失敗した。
  - id: jit-joins-dynamic-group
    red: 実装前に TestProvisionFederatedUserEvaluatesDynamicGroups が `members=[]` で失敗し、TestFederatedUserProvisionerEvaluatesDynamicGroups は入口の関数がなくビルドできなかった。
    fault_injection: 依存の組み立て `adminUserDeps` から `GroupRepo` を外すと TestFederatedUserProvisionerEvaluatesDynamicGroups が `members=[]` で、`createUser` から `syncDynamicGroups` を外すと TestProvisionFederatedUserEvaluatesDynamicGroups が `members=[]` で失敗した。
  - id: csv-create-joins-dynamic-group
    red: 実装前に TestApplyUserImportEvaluatesDynamicGroups が `members=[]` で失敗した。
    fault_injection: 適用の評価を無効にすると `members=[]` で、変更なしの行の評価を外すと `user-uma` が欠けて、同じテストが失敗した。
  - id: csv-rejects-duplicate-email
    red: 実装前に TestPlanUserImportRejectsDuplicateEmails が `row[0] action=created code="", want rejected "email_taken"` で失敗した。
    fault_injection: 計画器から `claimImportEmail` の呼び出しを外すと、同じテストが同じ失敗をした。
  - id: postgres-name-key-unique
    red: 契約の副テスト `names and emails compare by their case-folded keys` は、比較キーの列を加える前の SQL ではキーの列と引数がなくビルドできなかった。
    fault_injection: 保存の関数 `internalSaveUser` が比較キーではなく表記を `preferred_username_key` に書くと、TestPersistenceContract が `FindByUsername(STRASSE) = (<nil>, <nil>)` で失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idmanagement/domain` は 50 件を検出し、`principal_name.go` に生き残った変異はなかった（生き残った 8 件は変更していない `csv.go` などにある）。
  `mise run test-go-mutation -- backend/idmanagement/user/usecases` は 336 件を検出し、35 件が生き残った。変更した部分の生き残りは `user_creation.go:35` の `len(user.Attributes) > 0` を `>= 0` にする変異だけで、空の属性を検証してもスキーマを一度余計に読むだけなので等価である。ほかの生き残りは、変更していない変更項目の計算、ページングの防御、`UpdateUser` の評価の失敗の経路にある。
  変異器が表せない配線の除去は、上の各 `fault_injection` に記録した。Agent の名前は、置き換え前の `strings.EqualFold` で TestAgentNamesConflictUnderCaseFolding が `straße-bot` と `STRASSE-BOT` の衝突を検出できずに失敗することを確かめた。
  Group と Agent の PostgreSQL の `name_key` の一意索引には、大文字と小文字だけが異なる名前の保存を拒否する契約テストを加えていない。アプリケーションの判定がその前に拒否する。
- **Verification Results**:
  - `mise run test-go-changed` - 成功
  - `mise run check-spec`、`mise run check-api-compat`、`mise run check-contract-drift`、`mise run check-status-drift`、`mise run check-unspecified-vocabulary` - 成功
  - `mise run check-schema-tables` - 成功
  - `mise run check-schema` - 未実行。Docker デーモン（colima）が起動していない。
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 39 件中 38 件が成功した。失敗した 1 件は MCP リソースサーバーの画面の遷移待ちのタイムアウトで、テスト基盤が既知の WebKit の不具合（oven-sh/bun#43412）と判定した。`mise run test-ui-e2e-file -- tests/e2e/ui-scenario-actions.spec.ts` の再実行では 19 件すべてが成功した。
