---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-08-15
depends_on: [wi-273-unified-api-token-foundation]
change_kind: feature
initial_context:
  specification:
    - spec/contexts/api-tokens/SPECIFICATION.md#REQ-APITOKENS-002
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-003
    - spec/contexts/saml/SPECIFICATION.md#REQ-SAML-005
    - spec/contexts/ws-federation/SPECIFICATION.md#REQ-WSFEDERATION-001
    - spec/contexts/provisioning/SPECIFICATION.md#REQ-PROVISIONING-001
    - spec/contexts/application/SPECIFICATION.md#REQ-APPLICATION-004
  typespec:
    - IdMagic.Contract.ApiTokenScope
    - IdMagic.Contract.ApiTokenPrincipal
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/server_http/routes.go
    - backend/apitoken/domain/token.go
    - backend/apitoken/usecases/usecases.go
    - backend/sourcing/scim/handlers_http/handlers.go
  tests:
    - backend/shared/http/support_http
    - backend/oauth2/handlers_http
    - backend/shared/http/server_http
  stop_before_reading: [frontend, infra]
affected_spec:
  - { path: spec/contexts/api-tokens/models.tsp, symbol: IdMagic.Contract.ApiTokenScope }
  - { path: spec/contexts/api-tokens/scenarios.md, requirement: REQ-APITOKENS-004 }
  - { path: spec/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-025 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-025 }
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-003 }
  - { path: spec/contexts/saml/scenarios.md, requirement: REQ-SAML-005 }
  - { path: spec/contexts/ws-federation/scenarios.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-001 }
  - { path: spec/contexts/application/scenarios.md, requirement: REQ-APPLICATION-004 }
---

# 管理 API を API アクセストークンの粒度スコープで認可する

## Motivation

`ApiTokenScope` は 43 個のスコープを定義し、うち 33 個が管理 API に対応する粒度スコープ (`users:read`、`wsfed:write`、`saml:read`、`provisioning:write` など) である。管理コンソールのトークン発行画面 (`ApiTokenScopePicker.tsx`) はこれらを API の種類とリソースごとにまとめて提示し、REQ-APITOKENS-001 はその提示の仕方まで規範として定めている。しかし実際に検証されるのは 10 個 — SCIM の `scim:*` 4 個 (`backend/sourcing/scim/handlers_http/handlers.go`) と、アカウント API の `account:*` 6 個 (`backend/shared/http/support_http/auth.go` の `requiredAccountScope`) — だけである。

管理 API 198 本の認可は `RequireAdmin` に集約されており、ここが見るのは呼び出し元の認証状態、テナントの一致、アカウントが有効であること、実効ロールに `admin` を含むことの 4 点だけである。スコープ集合は認証時に解決されるが、その先で参照されない。ポータル境界のスコープ (`idmagic.admin` / `idmagic.account`) はクロスポータル利用を塞ぐが、粒度は与えない。

結果として 3 つの問題がある。

**最小権限が成立しない。** スコープを選ぶ UI は、実際には何も絞っていない選択肢を提示していることになる。(棚卸しの結果、実際の欠落はこれより深かった。T001 の結論を参照。)

**仕様が実装より先行している。** REQ-OAUTH2-003、REQ-SAML-005、REQ-WSFEDERATION-001、REQ-PROVISIONING-001、REQ-APPLICATION-004 はいずれも「`<resource>:read` だけで変更操作を要求する → `AccessDeniedError` で拒否される」を規範的な振る舞いとして宣言しているが、現在の実装は許可する。5 つの Context の normative scenario が実装と一致していない。

**漏洩時の影響が読めない。** 監査で「このトークンは何ができたか」に答えるとき、記録されたスコープ集合は答えにならない。発行時点の発行者のロールを再構成する必要がある。

なお [[wi-320-agent-management-api-scope-wiring]] は同じ欠落を `agents:read` / `agents:write` について切り出し、「配線するか、スコープを削除するか」の二択を立てたうえで、IdManagement Context 全体への配線を明示的に別 work item へ送っている。本 WI がその別 work item であり、Agent を含む管理 API 全体を対象とする。

## Scope

- **仕様**: 管理 API の operation と `ApiTokenScope` の対応を TypeSpec の `x-api-token-scopes` に宣言する。各 Context の `SPECIFICATION.md` の `Authorization Boundary` には、その Context が使うスコープ語彙と、`read` / `write` の割り当てが操作の名前から読み取れない場合の判断だけを書く。実装しないと決めたスコープは `ApiTokenScope` から削除する。
- **共有の強制点**: `backend/shared/http/support_http` に、ルートから契約の operation を解決して必要スコープをフェイルクローズに判定する仕組みを追加する。
- **ドリフト検出**: 宣言のない operation と、どの operation も要求しないスコープの両方を検出する検査を追加し、`just check` の対象にする。
- **エラー応答**: スコープ不足は RFC 6750 に従い `WWW-Authenticate` の `insufficient_scope` と、必要なスコープ名を返す。ロール不足の `access_denied` と区別する。

## Out of Scope

- ブラウザーセッションによる管理操作の振る舞い。セッション経由の呼び出しはスコープを持たないため、従来どおりロールだけで判定する。本 WI が変えるのは API アクセストークン経由の経路に限る。
- ロールモデルそのものの変更。`admin` / `system_admin` の 2 段構成と `actionRules` の規則表 (`backend/shared/spec/policy.go`) には触れない。スコープはロールとの論理積として重ねる。
- テナント単位のロール定義とカスタムロール。[[wi-320-agent-management-api-scope-wiring]] の Motivation が触れているテナント単位ロールのモデル化は引き続き別の課題とする。
- `authorization-model:*` など、現在 `ApiTokenScope` に存在しないスコープの追加。[[wi-371-authorization-rebac-admin-ui]] が Out of Scope としている ReBAC 管理 API のスコープもここでは足さない。既存の語彙を強制することに絞る。
- SCIM とアカウント API。すでに強制済みである。

## Design

### T001 の結論: 粒度スコープは「強制されていない」のではなく「到達できない」

棚卸しの最初に判明した事実として、`Motivation` の前提そのものが誤っていた。`resolveAuthnContext` のポータル境界は `/api/admin/v1/` に対して `idmagic.admin` をトークンのスコープ集合に要求する (`requiredPortalScope`) が、`IssueApiToken` が JWT の `scope` に載せるのは選択された粒度スコープだけで、`idmagic.admin` は付かない。したがって `saml:read` だけを持つ API アクセストークンは SAML の管理 API を通るのではなく、**管理 API のすべてを `insufficient_scope` で拒否される**。既存の `TestBearerWithoutPortalScopeIsForbidden` がこの分岐を固定している。

つまり 33 個の粒度スコープは「選べるが効かない」のではなく「選べるが何にも使えない」。本 WI は強制を足すだけでなく、粒度スコープを管理 API へ到達させる経路を同時に作る。これはアカウント API で既に成立している形と同じで、`hasRequiredAccountScope` はポータルスコープ `idmagic.account` の代わりにルートが要求する `account:*` を受け入れる。管理 API 側にも同じ規則を置き、**API アクセストークンではルートが宣言する粒度スコープがポータルスコープの代わりになる**。ブラウザーのポータルが提示する通常の OAuth アクセストークンは従来どおり `idmagic.admin` で判定し、粒度は要求しない。粒度スコープを持つのは API アクセストークンだけだからである。

### T001 の結論: 対応表と、削除するスコープ

管理 API は 210 本 (WI 起票時の 198 本から増えている)。19 個のリソースに対応する 37 個の管理粒度スコープは、いずれも 1 本以上のルートに対応づけられた。**`ApiTokenScope` から削除するスコープはない。**

- `sessions:read` / `sessions:write` は Authentication Context の管理セッション API (`ListUserSessions` / `GetUserSignInActivity` / `RevokeUserSession` / `RevokeAllUserSessions`) に対応する。
- `consents:read` / `consents:write` は制御面の操作ではなく、OAuth2 Context のテナント内同意管理 API (`ListAdminConsents` / `GetAdminConsent` / `RevokeAdminConsent`) に対応する。Design 起草時の「制御面なので到達できない」という見立ては誤りだった。
- `tenants:read` / `tenants:write` は制御面のテナント CRUD に対応する。`admin:tenants_manage` は `system_admin` かつデフォルト (制御面) テナント所属を要求するが、その条件を満たす利用者は自身のテナントで API アクセストークンを発行できるため、到達経路は存在する。ロールとの論理積は変わらず効く。

したがって `just check-api-compat` に破壊的変更は現れない。移行の猶予期間の議論も、既存トークンが管理 API で **既に** 全滅していた以上、失うものがない。強制の開始は回帰ではなく到達経路の新設である。

### T001 の結論: 語彙のない管理 API は対話セッション限定にする

既存のスコープ語彙に対応するものがない管理 API が 19 本ある。`Out of Scope` はスコープの追加を禁じているので、これらは **API アクセストークンからは到達できない (対話セッション限定)** と宣言する。既に `requiredAccountScope` がステップアップ経路に使っている `interactive_session` の形をそのまま使う。

| 管理 API | 本数 | 対話セッション限定にする理由 |
|---|---|---|
| API アクセストークン自身の発行・一覧・失効 | 3 | トークンからトークンを発行できると、任意のスコープ集合への昇格経路になる。発行は人間の管理者セッションに限る。 |
| 外部 IdP 接続 (`identity-providers`) | 9 | 攻撃者が管理する IdP を登録できることは、任意のユーザーとしてのサインイン、すなわち認証の迂回そのものである。`settings:write` に畳み込むと、設定変更のためのトークンが黙って認証迂回の能力を得る。 |
| ReBAC の認可モデルと関係タプル (`authorization`) | 6 | [[wi-371-authorization-rebac-admin-ui]] が語彙の追加を Out of Scope にしている。 |
| DEK の健全性 (`data-keys/health`) | 1 | `system_admin` 限定のテナント横断運用情報で、対応する語彙がない。 |

### T001 の結論: `KillAgent` は `agents:write` に含める

含める。`DELETE /api/admin/v1/agents/{agent_id}` は既に `agents:write` にあり、Agent の削除はキルより厳密に破壊的である。キルだけを外しても、同じスコープを持つトークンは削除で同じ結果に到達できるため、防御にはならない。一方で外すと、スコープ付きトークンの最も正当な用途である自動化されたインシデント対応 (侵害された Agent の即時停止) が対話セッションを要求することになる。したがって `agents:write` に含め、粒度は `agents:read` との分離で確保する。

### T001 の結論: 参照だけを行う POST は read スコープにする

`read` / `write` の割り当ては HTTP メソッドではなく、操作が対象リソースを変更するかで決める。次の POST は対象を変更しないため参照系とする。

- CSV エクスポートの一連の操作 (`users:read` / `groups:read`)。エクスポートジョブは対象データの読み出しの投影であり、`User` も `Group` も変更しない。`write` を要求すると、データを読むだけのために変更権限を渡すことになる。
- 動的グループ規則のプレビュー (`groups:read`)、LifecycleWorkflow のドライラン (`lifecycle-workflows:read`)、通知テンプレートのプレビュー (`settings:read`)。いずれも評価結果を返すだけで保存しない。

逆に通知テンプレートのテスト送信は実際にメールを送るため `settings:write` とする。

### 論理積として重ねる

スコープはロールを置き換えず、論理積として重ねる。API アクセストークンで到達できるのは「発行者が今もそのロールを持つ」かつ「トークンのスコープが対象操作を許す」場合に限る。ロール側を緩めると、スコープを付けたトークンが発行者より強くなる経路ができる。逆にスコープをロールの代替にすると、発行後にロールを外された利用者のトークンが生き続ける。どちらも避ける。

セッション経由の呼び出しにはスコープが存在しない。ここでスコープを必須にするとブラウザーからの管理操作がすべて落ちるので、**強制するのはトークン経由の呼び出しに限る**。判定は「呼び出し元がスコープを持つ主体か」で分岐させ、持つ主体には必ず要求する。持たない主体を素通しさせる分岐は、スコープなしのトークンで迂回できてはならないので、`ApiTokenPrincipal` が解決できたかどうかで判定する。空のスコープ集合は REQ-APITOKENS-002 のとおり認証段階で既に落ちる。

### ルート → スコープの対応をどこに置くか

3 案を検討した。

**(a) ハンドラー内で個別に宣言する。** SCIM が採る形で、`h.authenticate(c, apitokendomain.ScopeScimUsersWrite)` のように呼び出しの先頭で要求する。分かりやすいが、198 本のうち 1 本でも書き忘れると、そのルートだけスコープなしで通る。書き忘れが検出されない。

**(b) ルーティング登録時に宣言する。** `g.GET(path, handler)` を、スコープを取るラッパーへ置き換える。登録が対応の正本になるので、宣言のないルートを起動時に検出できる。ルートとスコープが同じ行に並ぶため読みやすい。

**(c) TypeSpec の operation に宣言し、ランタイム契約から解決する。** すでに `generate-contract` が TypeSpec から実行時のルートメタデータを生成し、`just check-generated-contract` が乖離を検出している。Discovery Metadata と同じく、契約を単一の正本にできる。

**(c) を採る。** 起票時は (b) を採り (c) を別 WI へ送る想定だったが、(c) の実装費用の見積もりが誤っていた。新しいデコレーターは不要で、`@TypeSpec.OpenAPI.extension("x-api-token-scopes", #[...])` が operation ごとの拡張として OpenAPI に出る。`generate-contract` は既に OpenAPI から `operations_gen.go` を生成しており、この拡張を読んで `Operation.Scopes` を足すだけで済む。`just check-generated-contract` が既にランタイム契約と TypeSpec の乖離を検出するため、ドリフト検出も新たに作る部分が少ない。

(c) の利点は費用対効果だけではない。

- **正本が 1 つになる。** 対応表が実装側と仕様側に二重化しない。(b) では Go の登録行が正本になり、TypeSpec と `SPECIFICATION.md` は追随する写しになる。
- **配線が不要になる。** ルートの登録側を 20 個のファイルにわたって書き換える必要がなく、判定は契約からの解決 1 か所に閉じる。ルートを足した人が宣言を書き忘れる経路は、契約に operation を足すときの検査で塞ぐ。
- **API 利用者から見える。** 生成した OpenAPI に載るため、どの endpoint がどのスコープを要求するかが公開する API 文書の一部になる。

**(a) と (b) を却下する理由は変わらない。** (a) は書き忘れが検出されない。(b) は正本を実装側に置くことになる。

いずれの案でも、**宣言のないルートは拒否する (フェイルクローズ)**。(c) では「契約に対応する operation がない」「operation が `x-api-token-scopes` を宣言していない」のどちらも拒否に倒す。

対話セッション限定の operation は、空の配列ではなく `#["interactive_session"]` を宣言する。空配列と「宣言がない」を JSON と Go の両方で取り違えずに区別するのは脆く、また `interactive_session` は既に `requiredAccountScope` がステップアップ経路の `WWW-Authenticate` に使っている値なので、応答の語彙とも揃う。

いずれの案でも、**宣言のないルートは拒否する (フェイルクローズ)**。許可へ倒すと、新しい管理 API を足した人がスコープを宣言し忘れたときに、そのルートだけ全スコープで通るようになる。この失敗は静かに起きるため、起動時に落とすほうがよい。

### 対応させられないスコープをどうするか

現在の 33 個のうち、対応する管理 API が存在しないものがある。`sessions:read` / `sessions:write` は管理 API のセッション操作 (`RevokeSession` / `RevokeUserSessions`) に対応しうるが、`consents:read` / `consents:write` と `tenants:read` / `tenants:write` は制御面の操作であり、`admin:tenants_manage` が `system_admin` かつ制御面テナント所属を要求するため、テナント単位のトークンからは到達できない。

配線の作業中に、対応するルートが存在しないスコープが確定した時点で、そのスコープは `ApiTokenScope` から削除する。選べるのに何も許可しないスコープを残すと、利用者は最小権限を設定したつもりで実際には別のスコープに依存し続けることになる。削除は破壊的変更なので、`just check-api-compat` のベースライン差分として明示的に扱う。

*(棚卸しの結果、削除対象は 1 つも出なかった。上の「T001 の結論: 対応表と、削除するスコープ」を参照。この規則は将来の追加スコープに対して有効なままなので、どのルートも要求しないスコープを検出する検査を `just check` に載せる。)*

### 既存トークンへの影響

現在発行済みのトークンは、スコープを持ってはいるが、それが強制されたことがない。強制を始めた時点で、必要なスコープを持たないトークンは 403 になる。これは意図した是正であって回帰ではないが、無告知で切り替えると自動化が止まる。

移行の猶予期間は設けない。**猶予期間を設けると、その間はスコープが強制されないため、本 WI の目的である最小権限が成立しない期間が残る。** 代わりに、切り替え前に「どのトークンがどのスコープ不足で落ちるか」を発行済みトークンのスコープ集合と対応表から静的に算出できるようにし、リリースノートで影響するトークンを提示する。トークンは再発行が唯一の復旧手段 (本文を保存しないため) なので、影響範囲を事前に示せることが重要になる。

*(T001 の結論により、この節が想定した回帰は起こらない。管理 API へ到達できていた API アクセストークンは 1 本も存在しないため、強制の開始で失われる能力はない。影響が出るのは、対話セッション限定と決めた 19 本を API アクセストークンから呼ぼうとしていた場合だけで、それも現状すべて拒否されている。)*

### 却下した案

- **スコープを全廃してロールだけにする。** 発行 UI とスコープの語彙をすべて削除する案。最小権限を諦めることになり、SCIM 側で既に成立している粒度も失う。
- **警告だけを出して通す (監査モード)。** 上記のとおり、強制しない期間は最小権限が成立しない。ログだけが増える。
- **ロールをスコープで置き換える。** 発行後にロールを外された利用者のトークンが生き続ける。

## Plan

1. 管理 API 198 本を Context ごとに棚卸しし、ルートとスコープの対応表を作る。対応するスコープがないルートと、対応するルートがないスコープを両方洗い出す。
2. 対応表を各 Context の `SPECIFICATION.md` の `Authorization Boundary` に記述する。削除するスコープを確定し、TypeSpec の `ApiTokenScope` と Go の `domain.Scope` から外す。
3. 共有の強制点を実装し、宣言のないルートを起動時に検出して落とす検査を入れる。
4. Context ごとに配線する。1 Context 1 コミットとし、その Context の normative scenario をテストの自己証跡にする。
5. ドリフト検出を `just check` に載せる。
6. 影響するトークンの算出手順を確認し、リリースノートの下書きを添える。

**未解決の点はない。** 起票時の未解決点 (`KillAgent` の扱い) は手順 1 の棚卸しで確定した。Design の「T001 の結論: `KillAgent` は `agents:write` に含める」を参照。

## Tasks

- [x] T001 [Design] 管理 API 210 本のルートとスコープの対応表を作り、対応のないルートとスコープを洗い出した。`KillAgent` の扱いを確定し、Design に 5 つの結論として追記した。
- [x] T002 [Spec] 対応表を各 Context の `SPECIFICATION.md` の `Authorization Boundary` へ記述する。削除するスコープはないので `ApiTokenScope` は変更しない。REQ-APITOKENS-004、REQ-IDMANAGEMENT-025、REQ-AUTHENTICATION-025 を追加する。`just check-spec` と `just check-api-compat` を通す。
- [x] T003 [Spec] 管理 API 210 本の operation へ `@TypeSpec.OpenAPI.extension("x-api-token-scopes", ...)` を宣言する。対話セッション限定は `interactive_session`。
- [x] T004 [App] `generate-contract` が拡張を `Operation.ApiTokenScopes` として生成し、`RuntimeContract.OperationForRoute` がルートテンプレートから operation を解決する。テスト: `TestEveryAdminRouteResolvesDeclaredApiTokenScopes`。
- [x] T005 [App] `backend/shared/http/support_http/admin_scope.go` に、契約からの解決とフェイルクローズ判定を実装する。テスト: `TestAdminApiTokenScopeEnforcement` (REQ-APITOKENS-004, REQ-SAML-005, REQ-OAUTH2-003)、`TestAdminPortalTokenSkipsGranularScopes`。
- [x] T006 [App] スコープ不足時に `insufficient_scope` と必要スコープ名を `WWW-Authenticate` へ載せ、ロール不足の `access_denied` と区別する。テスト: `TestAdminApiTokenInsufficientScopeChallenge` (RFC6750-API-TOKEN-ERROR)。
- [x] T007 [Check] 宣言のない operation と、どの operation も要求しないスコープを検出する検査を実装し、`just check-admin-scopes` として `just check` に載せる。テスト: `tools/check/src/admin-scopes.test.ts`。
- [x] T008 [Verify] `just verify` を通し、影響するトークンの算出手順を確認する。

## Verification

- `just check-spec`
  - reason: 削除するスコープを含む TypeSpec が契約として通ること。
- `just check-api-compat`
  - reason: `ApiTokenScope` からの列挙値の削除が破壊的変更として検出されること。差分を見たうえで意図した削除であることを確認する。
- `just verify-go`
- `just verify`
- 手動: `saml:read` だけを持つトークンを発行し、SAML SP の参照が通り、SAML SP の登録・ユーザー削除・署名鍵ローテーションがいずれも `insufficient_scope` で落ちることを確認する。同じ操作を管理コンソールのブラウザーセッションから行い、従来どおり通ることを確認する。
- 手動: 発行者から `admin` ロールを外し、必要スコープを満たすトークンでも操作が落ちることを確認する (論理積が成立していること)。

## Risk Notes

**既存トークンが落ちる。** 強制を始めた時点で、必要スコープを持たない発行済みトークンは 403 になる。トークン本文は保存しないため復旧は再発行しかなく、自動化が止まった側は原因の特定に時間がかかりうる。`WWW-Authenticate` に必要スコープ名を載せること、切り替え前に影響するトークンを算出して提示することの 2 点で緩和する。

**配線漏れが静かに通る。** 198 本のうち 1 本でも宣言を忘れると、そのルートだけ全スコープで通る。宣言のないルートを拒否し、起動時と `just check` の両方で検出する。許可へ倒す実装にしない。

**破壊的操作を過大なスコープに含める。** `KillAgent`、`DeleteAdminUser`、`RotateTenantSigningKey` を粗いスコープへまとめると、漏洩したトークン 1 本の影響が広がる。棚卸しの段階で操作ごとに明示的に判断し、判断を Design へ記録する。

**スコープの削除が利用者の設定を壊す。** 現在 `tenants:*` や `consents:*` を選んで発行しているトークンがあれば、そのスコープが消えることで再発行が必要になる。`just check-api-compat` の差分として明示的に扱い、リリースノートに列挙する。*(削除は発生しなかったため、この risk は実現しなかった。)*

## Completion

- **Completed At**: 2026-08-15
- **Summary**:
  管理 API 210 本の operation に、API アクセストークンが到達するために必要な `ApiTokenScope` を TypeSpec の `x-api-token-scopes` として宣言し、ランタイム契約 (`Operation.ApiTokenScopes`) 経由でフェイルクローズに強制するようにした。

  意味の差分は 2 つある。第 1 に、**粒度スコープが初めて管理 API へ到達する**。従来、API アクセストークンはポータル境界の `idmagic.admin` を持たないため管理 API のすべてで `insufficient_scope` に落ちており、33 個の粒度スコープは発行画面で選べるだけで何も許可していなかった。アカウント API と同じ形に揃え、管理 API では operation が宣言する粒度スコープがポータル境界のスコープの代わりになる。第 2 に、**その到達が operation 単位に閉じる**。`saml:read` を持つトークンは SAML SP を参照できるが、SAML SP の登録もユーザーの削除も署名鍵のローテーションも通らない。宣言のない operation と、契約に対応のないルートは拒否する。ロールとの論理積は変わらず、発行者がロールを失えばスコープを満たすトークンでも `access_denied` になる。

  対話セッション限定として、API アクセストークンからは到達できないと宣言した operation が 19 本ある (API アクセストークン自身の管理 3、外部 IdP 接続 9、ReBAC 6、DEK 健全性 1)。いずれも既存の語彙に対応がなく、既存スコープへ畳み込むと昇格経路になる。

  `ApiTokenScope` の列挙値は 1 つも削除していない。棚卸しの結果、37 個の管理粒度スコープはすべて 1 つ以上の operation に対応づいた (`sessions:*`、`consents:*`、`tenants:*` を含む)。`just check-api-compat` に破壊的変更は現れない。発行済みトークンが失う能力もない — 管理 API へ到達できていたトークンが存在しないためで、影響は「到達できるようになる」方向にしかない。影響の算出は、保存済みのスコープ集合 (`api_tokens.scopes`) と契約の `x-api-token-scopes` の突き合わせで静的に行える。

  正本は TypeSpec 1 か所である。起票時の設計はルート登録側に対応表を持つ案 (b) だったが、`@TypeSpec.OpenAPI.extension` で新しいデコレーターなしに宣言でき、既存の `generate-contract` / `check-generated-contract` に乗ることが分かったため案 (c) を採った。ルート登録側 (20 ファイル) の書き換えは不要になり、宣言はレンダリングした API 文書にも載る。

  **正本を 1 か所に保つための後続の是正。** 最初の実装では、`Scope` の記述どおり operation とスコープの対応表を各 Context の `Authorization Boundary` にも置いた。これは TypeSpec の宣言と同じ対応を散文側に複製したもので、照合する検査がないまま片方が古びる。表を外し、各 Context の境界文には「どのスコープ語彙を使うか」と「`read` / `write` の割り当てが操作の名前から読み取れない場合の判断とその理由」だけを残した。CSV エクスポートを参照系に置く理由、`KillAgent` を `agents:write` に含める理由、デフォルトサインインポリシーを `settings:*` に対応させる理由、外部 IdP 接続を対話セッション限定にする理由がそれにあたり、いずれも注釈からは読み取れない durable な判断である。網羅的な対応は TypeSpec だけが持つ。
- **Verification Results**:
  - `just check-spec` - passed (25 document(s), 322 operation(s))
  - `just check-api-compat` - passed (no breaking changes vs the frozen baseline)
  - `just check-admin-scopes` - passed (210 operation(s); 宣言のない operation なし、どの operation も要求しないスコープなし)
  - `just check` - passed
  - `just verify` - passed
  - 未実施 (手動): 実サーバーでトークンを発行しての確認。`TestAdminApiTokenScopeEnforcement` が `saml:read` での参照の許可、`saml:write` を要する登録の拒否、別リソース (`users:read`) の拒否、対話セッション限定の拒否を、`TestAdminPortalTokenSkipsGranularScopes` がブラウザーセッション側の非回帰を、それぞれ同じ判定経路で固定している。ロールとの論理積は `RequireAdmin` の既存判定が変わっていないことで担保する。
