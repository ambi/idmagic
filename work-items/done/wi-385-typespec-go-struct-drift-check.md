---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-19
change_kind: bugfix
priority: p1
depends_on: []
evidence_policy: risk-based-v2
initial_context:
  source:
    - tools/check/src/security-controls.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/src/event-contract.ts
    - tools/generate-contract/src/main.ts
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/handlers_http/handlers.go
    - backend/shared/spec/operations_gen.go
  tests:
    - tools/check/src
  stop_before_reading: [frontend, docs/contexts, spec/contexts]
affected_spec:
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.UpdateAuthorizationDetailTypeRequest }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.AdminSettingsUpdateRequest }
  - { path: spec/contexts/saml/models.tsp, symbol: IdMagic.Contract.ClaimMappingPolicy }
  - { path: spec/contexts/api-tokens/main.tsp, symbol: IdMagic.ApiTokens.Operations.IssueApiToken }
---

# TypeSpec が宣言する要求・応答本体と Go が実際に読み書きするものを機械的に突き合わせる

## Motivation

`wi-381` は `UserAttributeDef` が 10 フィールド中 2 つしか宣言していないことを、`wi-382` はそれが TypeSpec 全体に広がっていることを直した。2 件とも人間が 1 つずつ handler と突き合わせて見つけたもので、同じ食い違いが明日また入っても誰も気づかない。`wi-382` の Out of Scope はこの検査を「今回の作業を将来の再発から守る唯一の手段」と書いている。

`wi-382` の作業中に、突き合わせが機械化できることは実証されている。Go の route 登録から handler を引き、handler が到達する `NoStoreJSON` / `c.JSON` / `c.Bind` / `DecodeJSON` の引数を読めば、封筒のキーと decode 先の構造体が取れる。317 operation のうち 271 でこれが成功した。残りは間接呼び出しで、そこは手で解いた。

## Scope

- operation ごとに、TypeSpec の `@body` が指す型と Go 側の実際の本体を突き合わせる検査を `tools/check` に足す。少なくとも次を検出する。
  - 応答の 1 段目のキー集合の不一致 (契約が封筒を持つ / 持たない)。
  - 要求本体のプロパティ集合と decode 先構造体の JSON タグの差。
  - path / query パラメータが要求本体のプロパティとして重複して宣言されていること。
- 突き合わせられなかった operation を沈黙させず、未解決として報告する。検査が「何を見ていないか」が読めることを優先する。
- `mise run check` に組み込む。

## Out of Scope

- ステータスコード集合の突き合わせ。`wi-386` が扱う。
- Go 側の構造体を TypeSpec から生成する方向。生成は既存 handler の全面書き換えを伴い、検査より桁違いに重い。まず差分を見えるようにする。
- 未解決を 0 件にすること。追えない operation を減らすのは読み取り側の作業で、別 work item として切り出す。**上限値も置かない** —— 追えない形の handler を次に書く人の build を落とすことになり、それはその人の欠陥ではなく読み取り側の欠陥だからである。毎回の出力に被覆を明示することで沈黙を避ける。
- 型の互換性 (文字列か数値か、必須か省略可か)。見るのはキー集合の一致までとする。型まで見るには Go の型解決が要り、正規表現の射程を超える。
- **既知の食い違いを許可リストに載せること。** 検査が見つけた 14 件はすべて本 work item で解決した。許可リストを置けば、それは「宣言と実装が食い違ったままでよい」と記録することになり、この検査を足す理由そのものと矛盾する。

## Design

### 突き合わせの鎖

`operation` ごとに次の順で解決する。どの段で切れたかを未解決の理由として持つ。

1. **契約側**: 生成済み OpenAPI から `operationId` → 要求本体の第 1 段プロパティ集合、応答本体の第 1 段キー集合、path / query パラメータ名。
2. **経路**: `operationId` の `method` + `path` (`{id}` 形) を、Go の route 登録 (`g.POST("/...:id...", d.handleX)`, echo の `:id` 形) と正規化して照合し、handler の名前を得る。
3. **handler**: `func ... <name>(c *echo.Context) error {` から括弧の対応で本体を切り出す。
4. **要求**: 本体の `DecodeJSON(c.Request(), &x)` / `c.Bind(&x)` から `x` を得て、同じ本体の `var x T` で `T` を引き、`type T struct` の JSON タグを集める。
5. **応答**: 本体の `NoStoreJSON(c, <status>, <expr>)` / `c.JSON(<status>, <expr>)` の `<expr>` が、その場の複合リテラル (`map[string]any{...}` か `T{...}`) のときだけキー集合を取る。変数を渡している場合は解決しない。

### 検出する 3 つ

- **D1 要求本体のプロパティ差**: 契約のプロパティ集合と Go の JSON タグ集合の対称差。
- **D2 path / query の二重宣言**: 契約が parameter として宣言した名前が、同じ operation の要求本体のプロパティにも現れる。
- **D3 応答の封筒差**: 契約の応答第 1 段キー集合と Go の応答リテラルのキー集合の対称差。`wi-382` が消した架空のラッパー (`{"request": {...}}`) はここに出る。

### 未解決を沈黙させない

解決できなかった operation は理由付きで数え、**毎回の出力に被覆を書く** (`compared 184/333 operation(s), 149 not followed: response-not-a-literal=125, ...`)。`--list-unresolved` で個々の名前も出る。

**閾値は置かない。** 追えない operation で build を落とすと、次にその形の handler を書いた人が落ちる。それはその人の欠陥ではなく、この読み取り側が追えていないという欠陥である。Risk Notes が求める「合格に数えない」は、出力に被覆を書くことで満たす。

### 検出した食い違いは解決する

検査を実ツリーに当てて 14 件の食い違いが出た。**すべて本 work item で解決した。** どちらを直すかは 1 件ずつ次の順で判断した。

1. **標準や `docs/api-rules.md` が答えを決めるか。** 決めるならそちらが正で、いまどちらが一致しているかは関係ない。
2. **どちらの形がワイヤーに存在したことがあるか。** サーバーが一度も受理・生成していない項目を書いた契約は fiction なので、契約を直す。逆にサーバーが実際に送っているなら、契約を直すほうが安い。
3. **差分が機能の欠落を隠していないか。** 契約が宣言する項目を Go が読んでいないとき、「宣言が余計」か「機能が未実装」のどちらかである。

14 件の内訳と判断は Completion に記録した。

検討した代替案:

- **Go の型検査器 (`go/packages`) を使う**: 変数に入れた応答も解決でき、未解決は激減する。ただし検査が Go の tool chain を要求するようになり、`tools/` の他の検査 (すべて Bun) と実行基盤が分かれる。まず正規表現で差分を見えるようにし、未解決の実数を見てから判断する。本 work item では採らない。
- **既知の食い違いを許可リストに載せて `mise run check` へ入れる**: 検査だけを足して製品の契約には触れない、という小さい変更にできる。しかしそれは「宣言と実装が食い違ったままでよい」と記録することで、この検査を足す理由と矛盾する。14 件を解決した。

## Tasks

- [x] T001 [Test] 突き合わせの各段 (経路・handler・要求・応答) と 3 つの検出規則を単体で RED に置いた。
  `tools/check/src/contract-drift.test.ts` 35 件。
- [x] T002 [Tooling] `contract-drift.ts` (純関数) と `check-contract-drift.ts` (入力収集と終了コード) を実装した。
- [x] T003 [Acceptance] `wi-381` が直した形 (入れ子モデルが Go の一部項目しか宣言しない) を戻すと
  検査が落ちることを実測した。
- [x] T004 [Spec] 検査が見つけた 14 件の食い違いを、1 件ずつ判断して解決した。
- [x] T005 [Tooling] `mise run check` に `check-contract-drift` を組み込んだ。
- [x] T006 [Verify] `mise run verify`、`mise run check-api-compat` (ベースライン更新後)。

## Verification

- `mise run check`
- 手動確認: `wi-381` と `wi-382` が直した食い違いを意図的に 1 件戻すと、検査が落ちる。
- 手動確認: 突き合わせられなかった operation が未解決として報告され、沈黙しない。

## Risk Notes

Go の静的解析だけでは間接呼び出しを追い切れない。追えなかった operation を「合格」に数えると、検査があるのに守られていない状態になる。未解決を明示的に報告し、その件数を減らすこと自体を作業として扱う。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `added TypeSpec declarations: UpdateAuthorizationDetailTypeRequest, AdminSettingsUpdateRequest` を返す。`mise run check-contract-drift` を足し、`mise run check` に組み込んだ。operation ごとに、生成 OpenAPI の要求・応答本体と、Go の route → handler → decode 先構造体を突き合わせ、要求プロパティの差 (D1)、path/query と本体の二重宣言 (D2)、応答の第 1 段キーの差 (D3) を落とす。入れ子モデルへも降りる —— `wi-381` の欠陥 (`UserAttributeDef` が 10 項目中 2 つしか宣言しない) は入れ子にあり、第 1 段だけを見る検査では再発を捕まえられない。
  **検査が見つけた 14 件はすべて解決した。** 許可リストも閾値も置いていない。実ツリーは 333 operation 中 184 を突き合わせ、149 は追えていない (大半は応答を変数で返す handler)。この件数は毎回の出力に書く。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-contract-drift` を、`wi-381` が直した形へ戻したツリーに対して実行した。
  - **Requirement**: N/A: リポジトリの検査ツールであり、対応する規範的な製品要件を持たない。代わりに失敗したのは、本 work item の Verification が名指しする「`wi-381` と `wi-382` が直した食い違いを意図的に 1 件戻すと検査が落ちる」という条件である。
  - **Observed Failure**: `AdminUserUpdateRequest` から `given_name` と `family_name` を外すと `fail  D1 UpdateAdminUser: adminUserUpdateRequest decodes request properties the contract does not declare: family_name, given_name` (exit 1)。戻すと exit 0。
  - **Detection Reason**: 契約側を実際に壊して、検査が落ちることを見る。壊したのは第 1 段ではなく `AdminUserUpdateRequest` の項目で、これは `wi-381` の欠陥と同じ「宣言が実装より狭い」形である。最初の実装は第 1 段しか見ておらず、この確認で入れ子を見ていないことが分かった (下記)。検査を足しただけで受け入れとせず、壊して落ちることまで見たのはそのためである。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/contract-drift.test.ts` (35 件)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。
  - **Observed Failure**: 最初は `error: Cannot find module './contract-drift.ts'` (実装が存在しない)。以降は 1 挙動ずつ RED を置いた。
  - **Detection Reason**: 突き合わせの鎖は段が多く、どこか 1 段が黙って外れると検査全体が「差分なし」を返す。そこで段ごとに、このリポジトリで実際に現れる形を固定した —— 委譲クロージャによる経路登録、`func handleX(d Deps, c *echo.Context)` の signature、括弧の対応による handler 本体の切り出し、入れ子 map の第 1 段だけを読むこと、2xx の書き込みだけを読むこと。**追えない場合を「合格」にしないことも同じ強さで主張する**: 経路が無い、decode 先が追えない、応答が変数、成功の書き込みが 2 通りで食い違う、の 4 つは未解決として報告されることを検査で固定した。
- **Change-Resistance Results**:
  検査そのものを実ツリーに当てる過程が、方法として機能した。**最初の版が返した 15 件の指摘のうち 5 件は偽陽性で、いずれも実装の欠陥だった。**
  - 入れ子 map のキーを第 1 段と区別せず全部拾っていた → `CheckAccess` と `ListAccessibleResources` に 5 件の架空の差分。深さを見るよう直した。
  - `map[string]any` しか読まず `map[string]string` を無視していた → 追えない operation が 10 件多く出ていた。
  - handler が書く最初の応答を読んでいた → `ReadinessProbe` が 503 の本体と比較され、差分が無いのに指摘された。2xx だけを読み、成功の書き込みが食い違う場合は未解決にするよう直した。
  逆に、**検査を足しただけでは `wi-381` の欠陥を捕まえられなかった**ことも受け入れ確認で判明した (入れ子を見ていなかった)。この 4 点はいずれも、検査を実物に当てて初めて出たもので、単体テストだけでは出ていない。
  解決した 14 件の判断:
  - **標準・規約が決めた (1 件)**: `RegisterClient` の `id_token_signed_response_alg` / `require_pkce` —— DCR は `PS256` を固定し、`require_pkce` はサーバー方針である。標準行 `RFC7591-REGISTER` は `optional/MAY` で、この 2 つを受理する義務は無い。選べない設定を宣言していたので契約から外した。
  - **ワイヤーの実績が決めた (9 件)**: `ClaimMappingPolicy` (契約は `configuration` という置き石、サーバーは `name_id`/`rules`)、`DeprovisionPolicy` の `percentage` と `percent` の綴り違い (frontend も `percent` を使う)、`CompleteStepUpAuthentication` の `assertion`、`SubmitBrowserDevice` の `action`、`UpdateAdminUser` の `given_name`/`family_name`/`attributes`、`CreateAdminApplication` の SAML 5 項目、`GetAdminApplication` の `sign_in_policy`、`RegisterWsFedRelyingParty` (集約そのものを要求本体にしていた。正しい要求モデルは既に在ったのに operation が指していなかった)、`IssueApiToken` の `client_id` (常に組み込みクライアントで、呼び出し側は選べない)。
  - **1 モデルを 2 operation で共用していた (2 件)**: `UpdateAuthorizationDetailType` は `type` を path と本体の両方で宣言し、`UpdateAdminSettings` は `default_locale` を受けるのに共用相手の `UpdateTenant` は受けない。受け付ける項目が違うので、それぞれに要求モデルを与えた。
  - **宣言が機能の欠落を隠していた (1 件)**: `ListLifecycleWorkflowRuns` の `next_after`。カーソルを宣言しながら handler は書かず、送り返す `after` パラメータも無い。実装されていない paging を約束していたので、宣言を外し、その旨を `@doc` に書いた。
  - **副次的に 1 件**: `WsFedRelyingPartyRequest` の `display_name` を必須と宣言していたが、サーバーは空を受理する。契約を実測に合わせて省略可にした。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check` - passed (`check-contract-drift` を含む)
  - `mise run check-contract-drift` - `ok  contract body drift (0 finding(s); compared 184/333 operation(s), 149 not followed)`
  - `mise run check-api-compat` - 修正直後は 27 件の破壊的変更。すべて「サーバーが一度も受理・生成していない項目の削除」で、依存できたクライアントは存在しえない。該当する 15 schema と 3 operation だけをベースラインへ反映して `no breaking changes` (差分 174 行)。ベースラインは本件と無関係な追加分でも既にずれており、全面更新はそれを飲み込むため行っていない。
  - `mise run test-tools` / `lint-tools` / `typecheck-tools` - passed (contract-drift の単体 35 件を含む)
  - `mise run spec-diff` - `added TypeSpec declarations: UpdateAuthorizationDetailTypeRequest, AdminSettingsUpdateRequest`

## Follow-up

追えていない 149 operation のうち 125 は「応答を変数で返す handler」である。減らすには Go の型解決 (`go/packages`) を入れて変数の型を辿る必要があり、`tools/` の実行基盤が Bun から分かれる。費用と効果を比べる別 work item として切り出す。

`wi-386` はこの work item に依存していた。突き合わせの鎖 (operationId → 経路 → handler → 本体) はそのまま使えるので、ステータスコード集合の突き合わせはこの上に乗る。
