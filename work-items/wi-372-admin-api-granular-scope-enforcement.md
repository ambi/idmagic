---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-15
depends_on: [wi-273-unified-api-token-foundation]
change_kind: feature
affected_spec:
  - { path: spec/contexts/api-tokens/models.tsp, symbol: IdMagic.Contract.ApiTokenScope }
  - { path: spec/contexts/api-tokens/SPECIFICATION.md, requirement: REQ-APITOKENS-002 }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-003 }
  - { path: spec/contexts/saml/SPECIFICATION.md, requirement: REQ-SAML-005 }
  - { path: spec/contexts/ws-federation/SPECIFICATION.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/contexts/provisioning/SPECIFICATION.md, requirement: REQ-PROVISIONING-001 }
  - { path: spec/contexts/application/SPECIFICATION.md, requirement: REQ-APPLICATION-004 }
---

# 管理 API を API アクセストークンの粒度スコープで認可する

## Motivation

`ApiTokenScope` は 43 個のスコープを定義し、うち 33 個が管理 API に対応する粒度スコープ (`users:read`、`wsfed:write`、`saml:read`、`provisioning:write` など) である。管理コンソールのトークン発行画面 (`ApiTokenScopePicker.tsx`) はこれらを API の種類とリソースごとにまとめて提示し、REQ-APITOKENS-001 はその提示の仕方まで規範として定めている。しかし実際に検証されるのは 10 個 — SCIM の `scim:*` 4 個 (`backend/sourcing/scim/handlers_http/handlers.go`) と、アカウント API の `account:*` 6 個 (`backend/shared/http/support_http/auth.go` の `requiredAccountScope`) — だけである。

管理 API 198 本の認可は `RequireAdmin` に集約されており、ここが見るのは呼び出し元の認証状態、テナントの一致、アカウントが有効であること、実効ロールに `admin` を含むことの 4 点だけである。スコープ集合は認証時に解決されるが、その先で参照されない。ポータル境界のスコープ (`idmagic.admin` / `idmagic.account`) はクロスポータル利用を塞ぐが、粒度は与えない。

結果として 3 つの問題がある。

**最小権限が成立しない。** `saml:read` だけを選んで発行したトークンで、テナント内のユーザー削除も署名鍵のローテーションも通る。発行者が `admin` ロールを持つ以上、そのトークンは発行者の全権を持つ。スコープを選ぶ UI は、実際には何も絞っていない選択肢を提示していることになる。

**仕様が実装より先行している。** REQ-OAUTH2-003、REQ-SAML-005、REQ-WSFEDERATION-001、REQ-PROVISIONING-001、REQ-APPLICATION-004 はいずれも「`<resource>:read` だけで変更操作を要求する → `AccessDeniedError` で拒否される」を規範的な振る舞いとして宣言しているが、現在の実装は許可する。5 つの Context の normative scenario が実装と一致していない。

**漏洩時の影響が読めない。** 監査で「このトークンは何ができたか」に答えるとき、記録されたスコープ集合は答えにならない。発行時点の発行者のロールを再構成する必要がある。

なお [[wi-320-agent-management-api-scope-wiring]] は同じ欠落を `agents:read` / `agents:write` について切り出し、「配線するか、スコープを削除するか」の二択を立てたうえで、IdManagement Context 全体への配線を明示的に別 work item へ送っている。本 WI がその別 work item であり、Agent を含む管理 API 全体を対象とする。

## Scope

- **仕様**: 管理 API のルートと `ApiTokenScope` の対応表を、それを所有する各 Context の `SPECIFICATION.md` の `Authorization Boundary` に記述する。実装しないと決めたスコープは `ApiTokenScope` から削除する。
- **共有の強制点**: `backend/shared/http/support_http` に、ルートと必要スコープの対応を解決してフェイルクローズに判定する仕組みを追加する。
- **ハンドラーの配線**: 管理 API 198 本に必要スコープを宣言する。
- **ドリフト検出**: 宣言のないルートと、どのルートも要求しないスコープの両方を検出する検査を追加し、`just check` の対象にする。
- **エラー応答**: スコープ不足は RFC 6750 に従い `WWW-Authenticate` の `insufficient_scope` と、必要なスコープ名を返す。ロール不足の `access_denied` と区別する。

## Out of Scope

- ブラウザーセッションによる管理操作の振る舞い。セッション経由の呼び出しはスコープを持たないため、従来どおりロールだけで判定する。本 WI が変えるのは API アクセストークン経由の経路に限る。
- ロールモデルそのものの変更。`admin` / `system_admin` の 2 段構成と `actionRules` の規則表 (`backend/shared/spec/policy.go`) には触れない。スコープはロールとの論理積として重ねる。
- テナント単位のロール定義とカスタムロール。[[wi-320-agent-management-api-scope-wiring]] の Motivation が触れているテナント単位ロールのモデル化は引き続き別の課題とする。
- `authorization-model:*` など、現在 `ApiTokenScope` に存在しないスコープの追加。[[wi-371-authorization-rebac-admin-ui]] が Out of Scope としている ReBAC 管理 API のスコープもここでは足さない。既存の語彙を強制することに絞る。
- SCIM とアカウント API。すでに強制済みである。

## Design

### 論理積として重ねる

スコープはロールを置き換えず、論理積として重ねる。API アクセストークンで到達できるのは「発行者が今もそのロールを持つ」かつ「トークンのスコープが対象操作を許す」場合に限る。ロール側を緩めると、スコープを付けたトークンが発行者より強くなる経路ができる。逆にスコープをロールの代替にすると、発行後にロールを外された利用者のトークンが生き続ける。どちらも避ける。

セッション経由の呼び出しにはスコープが存在しない。ここでスコープを必須にするとブラウザーからの管理操作がすべて落ちるので、**強制するのはトークン経由の呼び出しに限る**。判定は「呼び出し元がスコープを持つ主体か」で分岐させ、持つ主体には必ず要求する。持たない主体を素通しさせる分岐は、スコープなしのトークンで迂回できてはならないので、`ApiTokenPrincipal` が解決できたかどうかで判定する。空のスコープ集合は REQ-APITOKENS-002 のとおり認証段階で既に落ちる。

### ルート → スコープの対応をどこに置くか

3 案を検討した。

**(a) ハンドラー内で個別に宣言する。** SCIM が採る形で、`h.authenticate(c, apitokendomain.ScopeScimUsersWrite)` のように呼び出しの先頭で要求する。分かりやすいが、198 本のうち 1 本でも書き忘れると、そのルートだけスコープなしで通る。書き忘れが検出されない。

**(b) ルーティング登録時に宣言する。** `g.GET(path, handler)` を、スコープを取るラッパーへ置き換える。登録が対応の正本になるので、宣言のないルートを起動時に検出できる。ルートとスコープが同じ行に並ぶため読みやすい。

**(c) TypeSpec の operation に宣言し、ランタイム契約から解決する。** すでに `generate-contract` が TypeSpec から実行時のルートメタデータを生成し、`just check-generated-contract` が乖離を検出している。Discovery Metadata と同じく、契約を単一の正本にできる。

**(b) を採り、(c) への移行を残す。** (c) が最終的に望ましいが、`@scopes` に相当するデコレーターの追加、エミッターの拡張、ランタイム契約の形の変更を伴い、198 本の配線と同時に動かすと失敗の切り分けができない。(b) で先に対応表を実装側に確定させ、宣言のないルートを検出できる状態を作る。契約から生成する形への移行は、対応表が安定してから別 WI とする。

いずれの案でも、**宣言のないルートは拒否する (フェイルクローズ)**。許可へ倒すと、新しい管理 API を足した人がスコープを宣言し忘れたときに、そのルートだけ全スコープで通るようになる。この失敗は静かに起きるため、起動時に落とすほうがよい。

### 対応させられないスコープをどうするか

現在の 33 個のうち、対応する管理 API が存在しないものがある。`sessions:read` / `sessions:write` は管理 API のセッション操作 (`RevokeSession` / `RevokeUserSessions`) に対応しうるが、`consents:read` / `consents:write` と `tenants:read` / `tenants:write` は制御面の操作であり、`admin:tenants_manage` が `system_admin` かつ制御面テナント所属を要求するため、テナント単位のトークンからは到達できない。

配線の作業中に、対応するルートが存在しないスコープが確定した時点で、そのスコープは `ApiTokenScope` から削除する。選べるのに何も許可しないスコープを残すと、利用者は最小権限を設定したつもりで実際には別のスコープに依存し続けることになる。削除は破壊的変更なので、`just check-api-compat` のベースライン差分として明示的に扱う。

### 既存トークンへの影響

現在発行済みのトークンは、スコープを持ってはいるが、それが強制されたことがない。強制を始めた時点で、必要なスコープを持たないトークンは 403 になる。これは意図した是正であって回帰ではないが、無告知で切り替えると自動化が止まる。

移行の猶予期間は設けない。**猶予期間を設けると、その間はスコープが強制されないため、本 WI の目的である最小権限が成立しない期間が残る。** 代わりに、切り替え前に「どのトークンがどのスコープ不足で落ちるか」を発行済みトークンのスコープ集合と対応表から静的に算出できるようにし、リリースノートで影響するトークンを提示する。トークンは再発行が唯一の復旧手段 (本文を保存しないため) なので、影響範囲を事前に示せることが重要になる。

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

**未解決の点**: `KillAgent` を `agents:write` に含めるかどうか。[[wi-320-agent-management-api-scope-wiring]] が指摘したとおり、漏洩したトークン 1 本で全 Agent が停止しうる。含めない場合は人間の管理者セッション限定の操作として仕様に明記する。手順 1 の棚卸しで確定させる。

## Tasks

- [ ] T001 [Design] 管理 API 198 本のルートとスコープの対応表を作り、対応のないルートとスコープを洗い出す。`KillAgent` の扱いを確定し、本ファイルの Design に結論と根拠を追記する。
- [ ] T002 [Spec] 対応表を各 Context の `SPECIFICATION.md` の `Authorization Boundary` へ記述する。削除するスコープを TypeSpec の `ApiTokenScope` から外し、`just check-spec` と `just check-api-compat` を通す。
- [ ] T003 [App] `backend/shared/http/support_http` に、必要スコープの宣言・解決・判定を実装する。テスト: トークン経由で不足時に拒否、充足時に許可、セッション経由は従来どおり、スコープを持つ主体で宣言のないルートは拒否 (REQ-APITOKENS-002)。
- [ ] T004 [App] スコープ不足時に `insufficient_scope` と必要スコープ名を `WWW-Authenticate` へ載せ、ロール不足の `access_denied` と区別する。テスト: `backend/shared/http/support_http/auth_test.go` (RFC6750-API-TOKEN-ERROR)。
- [ ] T005 [App] OAuth2 の管理 API を配線する。テスト: `oauth-clients` / `authorization-detail-types` / `mcp-resource-servers` のスコープ境界 (REQ-OAUTH2-003)。
- [ ] T006 [App] SAML と WS-Federation の管理 API を配線する。テスト: `saml:read` / `saml:write`、`wsfed:read` / `wsfed:write` の境界 (REQ-SAML-005, REQ-WSFEDERATION-001)。
- [ ] T007 [App] Provisioning と Application の管理 API を配線する。テスト: `provisioning:*` と `applications:*` の境界 (REQ-PROVISIONING-001, REQ-APPLICATION-004)。
- [ ] T008 [App] IdManagement (User / Group / Agent) と IdGovernance の管理 API を配線する。テスト: `users:*` / `groups:*` / `agents:*` / `lifecycle-workflows:*` の境界。
- [ ] T009 [App] Tenancy、SigningKeys、Audit の管理 API を配線する。テスト: `settings:*` / `signing-keys:*` / `audit:read` の境界。
- [ ] T010 [Check] 宣言のないルートと、どのルートも要求しないスコープを検出する検査を実装し、`just check` に載せる。
- [ ] T011 [Verify] `just verify` を通し、影響するトークンの算出手順を確認する。

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

**スコープの削除が利用者の設定を壊す。** 現在 `tenants:*` や `consents:*` を選んで発行しているトークンがあれば、そのスコープが消えることで再発行が必要になる。`just check-api-compat` の差分として明示的に扱い、リリースノートに列挙する。
