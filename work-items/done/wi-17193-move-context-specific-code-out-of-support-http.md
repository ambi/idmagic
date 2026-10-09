---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-09
priority: p2
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 共有の HTTP 支援パッケージと各 Context の間の Go の依存の向きを変えるだけで、HTTP の応答、認可の判定、発行するイベントは変わらないので、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/admin_scope.go
    - backend/application/usecases/access_gate.go
    - backend/application/client_display_names.go
    - backend/shared/http/support_http/deps.go
    - backend/shared/http/support_http/pagination.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/http/server_http/routes.go
    - backend/application/module.go
    - backend/application/ports/subject_directory.go
    - backend/authentication/domain/authentication_context.go
    - backend/authentication/deps_http/deps.go
    - backend/idmanagement/deps_http/deps.go
    - backend/tenancy/context.go
    - tools/check/src/boundary-fitness.ts
    - tools/check/src/check-boundaries.ts
    - tools/check/src/boundary-debt-ratchet.ts
    - docs/design/architecture/logical.md
  tests:
    - backend/shared/http/support_http
    - backend/shared/http/testing_stack/stack.go
  stop_before_reading: [frontend, spec]
spec_impact: { kind: none, reason: "共有の HTTP 支援パッケージにある Context 固有の処理を、その Context の側の実装と組み立て地点へ移すだけである。管理 API と account API の 401、403 とその Problem Details の種別、Bearer と DPoP の検証結果、アプリケーションの割り当てのゲートの判定、同意の表示名、発行するドメインイベント、永続状態は変えない。" }
---

# 共有の HTTP 支援パッケージから Context 固有の処理を外す

## 動機

`tools/check/boundary-debt.json` の境界の負債 854 件のうち 363 件は、`backend/shared` を経由して別の Context へ到達する依存（`shared-detour`）である。
そのうち 337 件は、一つのパッケージ `backend/shared/http/support_http` が原因である。

`support_http` は、IdManagement（19）、Tenancy（14）、Application（10）、Authentication（8）、ApiTokens（5）、SharedSignals（2）の各 Context を import している（括弧内は import 文の数）。
管理者の認可、アプリケーションのゲート、同意、クライアントの表示名、テナントの Cookie など、特定の Context の語彙を扱う処理がここにあるためである。
`support_http` を使うすべての Context が、これらの Context へ間接に依存したことになり、Context 間の循環（`context-cycle`、118 件）の多くもここを通る。

`docs/domain/structure.md` は、`backend/shared/` を「Context 間の依存規則を迂回する中継点にはしない」と定めている。
この一つのパッケージを直せば、負債の大半が一度に消える。

着手時に実測した 337 件の行き先は次のとおりである。

| 行き先 | 件数 | 経路 |
| --- | --- | --- |
| Tenancy（`backend/tenancy`、`tenancy/domain`、`tenancy/ports`） | 47 | テナントの解決と、解決済みのテナントの読み取り |
| IdManagement、Application、Authentication、ApiTokens、OAuth2 | 222 | 管理者の認可、Bearer の検証、アプリケーションのゲート、表示名、同意のエラー |
| ClaimMapping、SigningKeys、OAuth2 の logout | 68 | DPoP の検証のために import する `backend/shared/security/tokens_jose` を経由した到達 |

## 対象範囲

- `support_http` から、Tenancy 以外の Context と `tokens_jose` への import をなくす。
  `support_http` が必要とする判断は、`support_http` が宣言するインターフェースとして受け取り、各 Context の側に置いたアダプターを組み立て地点（`backend/shared/http/server_http`）で結ぶ。
- Context の処理そのものは、その語彙を所有する Context へ移す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- `support_http` の Tenancy への依存（47 件）。
  wi-39119 が解決済みのテナントの受け渡しを Tenancy の公開言語へ移し、Context Map に Tenancy からの関係を書き足すと、`support_http` から `tenancy/ports` への依存は違反ではなくなる。
  本項目でテナントの部分を各 Context へ移すと、Context Map にない Tenancy への直接の依存が新しい違反 ID として現れ、負債の ratchet（`check-boundary-debt-ratchet`）が拒否する。
  そのため、テナントの解決ミドルウェア、`RequestTenantID` などの読み取り、`Deps.TenantRepo`、制御面の判定が使う `DefaultTenantID` は `support_http` に残す。
- `backend/shared/security/tokens_jose`（17 件）と `backend/shared/storage/db_memory`（9 件）の `shared-detour`。
  本項目で `support_http` の分け方が決まれば同じ方法で直せるが、残りの負債の棚卸し（wi-33994）で扱う。
- 共有の HTTP 支援パッケージの API の整理のうち、境界の違反を消すのに要らないもの。

## 設計

`support_http` が Context の型を受け取る箇所は、依存の向きを逆にして消す。
`support_http` は自分が必要とする最小のインターフェースと、それが運ぶ Context に依存しない値を宣言し、各 Context はそれを満たすアダプターを自分の側に置く。
`structure.md` が `WorkloadIdentity` から `OAuth2` への関係で採っている形と同じである。

アダプターを各 Context に置くのは、新しい違反 ID を生まないためでもある。
アダプターは自分の Context と `support_http` だけを import するので、Context 間の新しい依存は生まれない。
組み立て地点の `server_http` は System に属し、どの Context からも import されないので、ここで全 Context のアダプターを結んでも迂回の経路にならない。

### 認証と管理者の認可（`Authenticator`）

`support_http` に残すのは、Bearer と DPoP の Authorization ヘッダーの解釈、ポータルのスコープと account の粒度スコープ、管理 API の API トークンのスコープ、アクセストークンの audience とリクエスト先レルムの照合、ロールの判定、拒否の応答の書き出しである。
Context の語彙に触れる処理は、次のインターフェースの向こうへ移す。

```go
// support_http
type Authenticator struct {
    Sessions     SessionAuthenticator  // Authentication: authentication/deps_http.SessionAuthentications
    AccessTokens AccessTokenVerifier   // OAuth2: oauth2/handlers_http.NewResourceAccessTokens
    ApiTokens    ApiTokenAuthenticator // ApiTokens: apitoken/handlers_http.ResourceAuthenticator
    Principals   PrincipalDirectory    // IdManagement: idmanagement/deps_http.Principals
}

type Authentication interface {
    Subject() string
    AuthenticatedAt() int64
    Session() string
    Pending() bool
}
type SessionAuthenticator interface {
    ResolveSession(ctx context.Context, header http.Header) (Authentication, error)
    EndSession(ctx context.Context, sessionID string) error
}

type AccessToken struct {
    Subject, ClientID, Scope string
    Audience                 []string
    IssuedAt                 int64
    Managed                  bool
    DPoPBound                bool
    DPoPJKT                  string
}
type DPoPProof struct{ Header, Method, HTU, AccessToken string }
type AccessTokenVerifier interface {
    IntrospectAccessToken(ctx context.Context, token string) (*AccessToken, error) // 無効、失効なら nil
    VerifyDPoPProof(ctx context.Context, proof DPoPProof) (jkt string, err error)
}

type ApiTokenPrincipal struct {
    UserID, ClientID string
    Scopes           []string
}
type ApiTokenAuthenticator interface {
    AuthenticateApiToken(ctx context.Context, token string) (ApiTokenPrincipal, error)
}

type Principal struct {
    ID, TenantID string
    Roles        []string
    Active       bool
}
type PrincipalDirectory interface {
    FindPrincipal(ctx context.Context, subject string) (*Principal, error)
    EffectiveRoles(ctx context.Context, principal Principal) []string
}

func (a *Authenticator) Authenticate(c *echo.Context) (Authentication, error)
func (a *Authenticator) RequireAdmin(c *echo.Context) (*Principal, error)
func (a *Authenticator) RequireAdministrator(c *echo.Context) (*Principal, error)
func (a *Authenticator) ResolveAdminActor(c *echo.Context) (*Principal, error)
func (a *Authenticator) RequireControlPlaneUser(c *echo.Context) (*Principal, error)
```

- 認証の結果は、Authentication Context の `*authdomain.AuthenticationContext` がそのまま `Authentication` を満たす。
  Bearer で認証したときは `support_http` が主体と認証時刻だけを持つ値を作る。
  AMR、ACR、保留の目的を読む OAuth2 と Authentication は、`authdomain.ContextOf(authn)` で `*AuthenticationContext` へ戻す。
  `ContextOf` は Bearer の値から `UserID` と `AuthTime` だけを持つ文脈を作り、変更前に Bearer の経路が返していた値と一致させる。
- 管理者の主体は、`*userdomain.User` ではなく `support_http.Principal` で返す。
  呼び出し側が使うのは ID、所属テナント、ロールだけである。
  `RequireAdmin` が直接ロールを、ほかの三つが Group 由来を合成した実効ロールを返す違いは変えない。
- 外部の作用は、アダプターの側に置く。
  セッションの失効（`SessionManager.Store.Revoke` と現在時刻）は `EndSession`、DPoP の証明のリプレイ記録と現在時刻は `VerifyDPoPProof` が持つ。
- 検査の順序は変えない。
  イントロスペクションと失効、realm の audience、管理発行のトークンの照合、DPoP、スコープの順に判定する。
  DPoP の検証はリプレイ記録へ書き込むので、前の判定で拒否するトークンの証明を記録しない順序を保つ。
- 配線がないときの振る舞いを保つ。
  `AccessTokens` がなければ Bearer は未認証、`ApiTokens` がなければ管理発行のトークンは無効、`Sessions` がなければ Cookie は未認証、`Principals` がなければ利用者の照合を省く。
  制御面の経路の `Authenticator` に `ApiTokens` を結ばない現在の配線も、そのまま写す。

### アプリケーションの割り当てのゲート

`ApplicationGate` を Application の `usecases.AccessGate` へ移す。
判定結果の型 `ApplicationAccessDecision` は、Application を Supplier とする OAuth2、Saml、WsFederation が読めるように `application/domain` へ置く。
Group の所属は、`application/ports.GroupMemberships` として受け取り、IdManagement の Group の保存先から引くアダプターを `application/module.go` に置く。
`SubjectDirectory` と同じ形である。
`ClientIP` はゲートのホップ数で計算する現在の形のまま `AccessGate` へ移す。
消費する側（OAuth2、Saml、WsFederation の HTTP アダプター）は、`AccessGate` の具体型ではなく、自分が宣言する `ApplicationGate` インターフェースで受け取る。
OAuth2 のハンドラーが埋め込んだゲートから読んでいたサインインポリシーの保存先は、OAuth2 の `Deps` のフィールドにし、組み立て地点がゲートと同じ保存先を結ぶ。
本番の呼び出し元がない `ApplicationAccessAllowed` は削除し、テストは `EvaluateApplicationAccess` の `Allowed` を読む。

### クライアントの表示名と同意のエラー

`ClientDisplayNameResolver` を、組み立てている `backend/application` へ移す。
OAuth2 と Authentication は、自分が使うメソッドだけのインターフェースで受け取る。
`WriteConsentError` は、それを呼ぶ OAuth2 と Authentication の handler のパッケージ内の関数にする。
Authentication の `Deps` は別パッケージの型の別名なので、メソッドにはできない。

### 埋め込んだ `Authenticator` から読んでいたフィールド

ハンドラーの一部は、埋め込んだ `Authenticator` のフィールドを直接読んでいた。
読んでいた値は、Context ごとに次の形で同じ値を受け取る。
いずれも新しい違反 ID を生まない置き場所を選んだ。

| 消費する側 | 読んでいた値 | 変更後 |
| --- | --- | --- |
| OAuth2、Authentication | `SessionManager`、`AuthnResolver` | 各 `Deps` のフィールド。既存の import の範囲で型を名指せる |
| Saml、WsFederation | `SessionManager`、`AuthnResolver` | 各 HTTP アダプターの `Sessions` 構造体。`Module.Register` はこの型を受け取るので、Context の直下のパッケージが Authentication を import しない |
| IdManagement | `SessionManager.Store` | `userports.SessionPurger`（匿名化が使う `DeleteAllForSub` だけ）。`AdminUserDeps.SessionStore` もこの型にし、`user/usecases` から Authentication のセッションの port への依存も消えた |
| Authentication の account 文脈 | `EffectiveRoles(ctx, *User)` | `EffectiveRoles(ctx, support.Principal)` |
| IdManagement の step-up | `*AuthenticationContext` を受ける `mfausecases.StepUpSatisfied` | 引数を `authdomain.ResolvedAuthentication` にし、内部で `ContextOf` で戻す。既存の呼び出し元は `*AuthenticationContext` をそのまま渡せる |
| Tenancy のテスト送信 | 主体の表示名とメールアドレス | 主体は宛先を持たないので、Tenancy が既に持つ `UserRepo` で操作者を引き直す |

### 仕様にない振る舞いの分類

Tenancy のテスト送信で操作者を引き直すと、認可の後に記録が消えた場合の分岐と、引き直しの失敗の分岐が新しく生まれる。
前者は認可の主体がいない場合と同じ 403、後者はほかの保存先の失敗と同じく呼び出し元へ返す。
どちらも利用者が観測する結果を新しく定めるものではなく、認可の直後に記録が消える競合だけで起こるので、(b) 書かないに分類する。

採らない案は次のとおり。

| 案 | 採らない理由 |
| --- | --- |
| `support_http` を Context の数だけ複製する | 技術的な処理の重複が生まれ、修正が一方にしか入らない |
| 負債の理由を「意図した例外」へ書き換えて残す | `structure.md` の規則そのものに反し、境界の検査が `shared` 経由の越境を見逃す前例になる |
| `Authenticator` を Authentication または IdManagement の公開言語へ移す | 25 のパッケージが `Authenticator` を使い、多くは Context Map にその Context との関係を持たない。迂回の違反が直接の依存の違反へ形を変えるだけで、ratchet が新しい違反 ID を拒否する |
| 認証の結果を `support_http` の構造体で表し、AMR などの全フィールドを写す | Authentication の語彙（保留の目的、登録の期限）を共有のパッケージへ複製することになり、迂回を語彙の重複へ置き換えるだけである |

## 計画

1. 移す処理のうち、規範 ID を引くテストが固定していない振る舞いを、移しても変わらない境界で特性化テストに固定する。
   `Authenticator` は `server_http.Register` を通した HTTP、ゲートと表示名は `application.Module` の公開メソッドを境界にする。
2. 同意のエラー、表示名、ゲート、`Authenticator` の順に依存を逆転する。
3. `mise run check-boundaries` の報告に従い、消えた違反 ID を負債から消す。

## タスク

- [x] T001 [Inventory] `support_http` の Context 依存を関数単位で一覧にする。
  動機の表と設計の節に記録した。
- [x] T002 [Characterize] 移す処理のうち、規範 ID を引くテストが固定していない振る舞いを特性化テストで固定する。
  `mise run spec-review-candidates -- backend/shared/http/support_http` が挙げた行のうち、移す処理に当たるものを固定した。
  - `backend/shared/http/server_http/authenticator_characterization_test.go`: `TestCharacterizeAdminAccessThroughTheSessionCookie`、`TestCharacterizeAdminAccessThroughABearerToken`、`TestCharacterizeControlPlaneAccess`、`TestCharacterizeAdminActorResolution`
  - `backend/application/module_characterization_test.go`: `TestCharacterizeApplicationGateDecision`、`TestCharacterizeApplicationGatePropagatesStoreFailures`、`TestCharacterizeClientDisplayNamesResolveEachClientOnce`
  - 検査: `mise run test-go-test -- ./backend/shared/http/server_http 'TestCharacterize.*'`、`mise run test-go-test -- ./backend/application 'TestCharacterize.*'`
  - 特性化テストは対象と別のパッケージにあるので、`test-go-mutation` は対象への変異でそれらを走らせない。
    代わりに、特性化した分岐へ手で 19 件の変異（エラーの握りつぶし、セッションの失効の削除、主体の照合の削除、拒否の種類の差し替え、理由の文言の差し替え、重複の解決）を入れ、19 件すべてを特性化テストが検出した。
  - 固定しなかった分岐: Authentication の組み立てが型付きの nil を渡すため、セッションの解決器がない構成は現在も panic し、HTTP の入口から観測できない。応答の書き込み自体の失敗（`refused` の書き込みエラー）も HTTP からは起こせない。
- [x] T003 [App] 同意のエラー、表示名、ゲートを所有する Context へ移す。
  - 検査: `mise run -c test-go-package ./backend/application/... ::: test-go-package ./backend/oauth2/handlers_http ::: test-go-package ./backend/saml/... ::: test-go-package ./backend/wsfederation/... ::: test-go-package ./backend/shared/http/server_http`
  - 移したテスト: `application/usecases/access_gate_test.go`、`application/client_display_names_test.go`、`oauth2/handlers_http/consent_error_test.go`、`authentication/handlers_http/consent_error_test.go`
- [x] T004 [App] `Authenticator` の依存をインターフェースへ逆転し、各 Context にアダプターを置く。
  - アダプター: `authentication/deps_http/session_authentications.go`、`oauth2/handlers_http/resource_access_tokens.go`、`apitoken/handlers_http/resource_authenticator.go`、`idmanagement/deps_http/principals.go`。組み立ては `shared/http/server_http/authenticator.go`
  - 移したテスト: DPoP の束縛（REQ-OAUTH2-045）と失効（EX-OAUTH2-047-01〜03）は、OAuth2 のアダプター越しに `oauth2/handlers_http/resource_access_tokens_test.go` で確かめる。`support_http` のスコープと audience の判定はフェイクで、Group 由来のロールは IdManagement のアダプターで確かめる
  - 削除したテスト: `TestWithEffectiveRolesReturnsADistinctUserValue`。主体は照会のたびに作る値になり、複製の同一性を問う公開の操作がなくなった。合成したロールが元のスライスを共有しないことは `TestEffectiveRolesComposesGroupRolesIntoAFreshSlice` が引き続き確かめる
  - 検査: `mise run lint-go`、`mise run test-go-changed`。特性化テストは T002 のコミットから一字も変えずに通った
- [x] T005 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
  - 台帳の違反 ID は 854 件から 510 件になった。`shared-detour` は 363 件から 73 件で、残りは `support_http` から Tenancy への 47 件と、対象外の `tokens_jose`（17 件）と `db_memory`（9 件）である
  - 副次的に、Audit から IdManagement（監査の管理 API が `*userdomain.User` を名指さなくなった）と、IdManagement の `user/usecases` から Authentication のセッションの port への違反も消えた
  - 検査: `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet -- main`
- `mise run verify`

## リスク

- 管理者の認可や Bearer の判定を移すとき、配線を誤ると防護が外れる。
  移す前に `server_http.Register` を通した特性化テストを置き、移した後も同じ応答になることを確かめる。
  既存の `support_http` のテストは、組み立てだけを新しい形へ変え、表明は変えない。
- 認証の結果をインターフェースで運ぶので、nil の `*AuthenticationContext` を包んだインターフェースが nil でなくなる誤りが起こり得る。
  セッションのアダプターは、文脈がないときに型のない nil を返す。

## 完了

- **完了日**: 2026-10-09
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  `backend/shared/http/support_http` から、Tenancy 以外の Context と `tokens_jose` への import をなくした。
  管理者の認可と Bearer の検証は `support_http` の `Authenticator` に残し、セッション、アクセストークン、API トークン、主体の照会をインターフェースとして受け取る形にした。
  各 Context の側に置いたアダプターを、組み立て地点の `server_http` が結ぶ。
  アプリケーションの割り当てのゲートは Application の `usecases.AccessGate` へ、クライアントの表示名の解決は `backend/application` へ、同意のエラーの写像は呼び出す handler へ移した。
  境界の負債の台帳は 854 件から 510 件になり、`support_http` 経由の `shared-detour` は 337 件から Tenancy への 47 件になった。
  HTTP の応答、認可の判定、発行するイベント、永続状態は変えていない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries` と、`server_http.Register` を通す特性化テスト（`TestCharacterizeAdminAccessThroughTheSessionCookie` ほか 3 件）
  - **要件**: N/A: 規範上の振る舞いを変えない、依存の向きの変更であり、対応する REQ がない。
  - **観測した失敗**: 変更前の `main` では、`support_http` を経由した `shared-detour` が 337 件あった。境界の検査は台帳に載った負債として通すが、台帳から消すと 337 件の未記録の違反として失敗する。変更後は、消した違反 ID が台帳に残っていれば `boundary debt is resolved; remove it` として失敗し、台帳を刈り込んで通った。
  - **検出できる理由**: 依存の向きを変える作業なので、製品の振る舞いの RED は存在しない。代わりに、リファクタリングを必要にした構造ゲート（境界の検査と ratchet）が変更の前後で違いを観測し、移す前に置いた特性化テストが振る舞いの不変を固定した。特性化テストは T002 のコミットから一字も変えずに通った。
- **単体 RED の証拠**:
  - **テスト**: 手で入れた変異に対する特性化テストと、`support_http` の単体テスト
  - **要件**: N/A: 規範上の振る舞いを変えない、依存の向きの変更であり、対応する REQ がない。
  - **観測した失敗**: 該当なし。振る舞いを変えないので、実装前に失敗する単体テストはない。代替として、移す前のコードへ 19 件の変異を入れ、19 件すべてで特性化テストが失敗することを確かめた（T002）。
  - **検出できる理由**: 変異は、移した処理の各分岐（保存先のエラーの伝播、無効な主体のセッションの失効、管理発行のトークンの主体の照合、拒否の種類、ゲートの理由、表示名の重複の解決）を一つずつ壊したものであり、移した後の実装がどれかを取りこぼせば同じテストが失敗する。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/shared/http/support_http` は 394 件中 264 件を検出し、61 件が生き残った。
  今回変えた `auth.go` と `admin_scope.go` の生き残りは 3 件である。
  `authorizationToken` の長さの比較（`>` を `>=`）は、接頭辞だけのヘッダーが空のトークンになり、呼び出し元が空を未提示として扱うので等価である。
  `Authenticate` の、無効な主体のセッションを終わらせる条件の 2 件（`a.Sessions != nil` と `authn.Session() != ""` の否定）は、同じパッケージのテストでは観測しない。手で同じ変異を入れると、`server_http` の特性化テスト `inactive_user_ends_the_session` が 2 件とも失敗した。
  ほかの生き残りは今回変えていないファイルにある。
  ツールが表現できない配線の変異は手で入れた。組み立て地点から Group の保存先、失効リスト、API トークンの照合、セッションの失効をそれぞれ外すと、特性化テストの `admin_role_granted_through_a_group`、`revocation_lookup_fails`、`managed_token_issued_to_the_subject`、`inactive_user_ends_the_session` がそれぞれ失敗した。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
