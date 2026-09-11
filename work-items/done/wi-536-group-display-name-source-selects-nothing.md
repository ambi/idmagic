---
depends_on: [wi-537-api-compat-drops-keywords-written-beside-a-ref]
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 取得元が実際に効くようになり、`display_name_source` の値域が自由な文字列から 3 つの列挙へ狭まる。管理画面の入力も選択に変わるので、リリースの読み手に見える差分がある。未リリースなので移行の案内は要らない。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-536.md }
initial_context:
  specification:
    - docs/contexts/provisioning/standards.md
  typespec:
    - IdMagic.Contract.GroupPushConfig
    - IdMagic.Contract.ProvisioningGroupDisplayNameSource
  source:
    - backend/provisioning/domain/connection.go
    - backend/provisioning/usecases/deliver.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/source_idmanagement/group_attribute_source.go
    - backend/provisioning/handlers_http/handlers.go
    - frontend/src/features/admin-applications/AdminApplicationProvisioningSettings.tsx
    - frontend/src/features/admin-applications/AdminApplicationProvisioning.i18n.ts
    - frontend/src/types.ts
  tests:
    - backend/provisioning/usecases/deliver_test.go
    - backend/provisioning/e2e_capture_delivery_test.go
  stop_before_reading:
    - backend/provisioning/db_postgres
    - backend/provisioning/client_scim
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-GROUP-RESOURCES }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.GroupPushConfig }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ProvisioningGroupDisplayNameSource }
primary_use_cases:
  - id: group-display-name-follows-the-connection
    requirement: RFC7643-OUT-GROUP-RESOURCES
    observable_result: 取得元に `email` を選んだ接続では、下流が受け取る Group の `displayName` が Group のメールアドレスになる。取得元を変えていない接続では Group の名前のままである。
    unit_test: { path: backend/provisioning/usecases/deliver_test.go, name: TestDeliverGroup_DisplayNameFollowsTheConfiguredSource, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_capture_delivery_test.go, name: TestE2E_GroupChange_DisplayNameFollowsTheConfiguredSource, task: test-go-race }
    unit_fault_model: 取得元を読まず、`display_name` を常に Group の名前から解決する（現状の欠陥そのもの）。
    e2e_fault_model: 選んだ属性を Group が持たないときに配信を失敗させる。表示名を fail-closed の判断として扱い、設定の打ち間違いで Group の送出全体が止まる。
---

# Group の表示名の取得元は保存されるだけで、送出する `displayName` を何も選んでいない

## Motivation

`docs/contexts/provisioning/standards.md` の `RFC7643-OUT-GROUP-RESOURCES` は「`displayName` の取得元は
`GroupPushConfig.display_name_source` が選び、既定は Group の名前である」と宣言する。TypeSpec の
`GroupPushConfig.display_name_source` も「下流の `displayName` へ射影する IdMagic 側の Group 属性キー」と
説明し、管理画面は自由入力の欄として書き込める。

**しかし、この値を読む production の経路が 1 つも無い。** `GroupAttributeSource.ResolveAttributes` は接続を
参照せず、`display_name` を常に `Group.Name` から解決する。管理者が `email` や `description` を入れても、
下流へ届く `displayName` は Group の名前のままである。設定は保存され、画面は入力を受け付け、配信は成功し、
観測できる差だけが無い。

`GroupAttributeSource` の注釈は「`name` が既定であり今日の時点で意味を持つ唯一の値なので、未設定や未知の
取得元は配信を失敗させずに `Group.Name` へ落ちる」と書く。未知の値を既定へ落とす判断そのものは妥当である
（表示名は fail-closed の判断ではない）。成立していないのはその手前で、**既知の値でも選べない**。
`name` 以外に意味を持つ値が無いのは実装がそう決めたからであって、解決済みの属性表は `name`、`description`、
`email` を既に持っている。

[[wi-441-push-groups-produces-no-delivery]] は Scope に「`GroupPushConfig` の表示名の取得元を反映する」を
挙げ、Design にも同じ文を書いたうえで、反映しない実装を残した。宣言だけが先に進み、実装が追いつかなかった
箇所である。

[[wi-507-back-provisioning-standards-rows-with-tests]] が `RFC7643-OUT-GROUP-RESOURCES` にテストを対応付ける
途中でこれを見つけた。同項目は行の他の節（`schemas`、対応付けが解決した属性、メンバーシップを本文に載せず
増分 `add` の PATCH で送ること、除去を送らないこと）を観測して消化しており、この節だけが観測を持てない。

## Scope

- `GroupPushConfig.display_name_source` が選んだ属性キーから `display_name` を解決する。
- 未設定と未知のキーは `Group.Name` へ落とす。表示名は fail-closed の判断ではないので、配信は失敗させない。
- 選べる値の集合を決め、TypeSpec の説明と管理画面の入力をその集合に合わせる。自由入力のままにするか列挙に
  するかは、この項目が決める設計の判断である。
- `RFC7643-OUT-GROUP-RESOURCES` の「取得元を選ぶ」節を区別できるテストを対応付ける。既定とは違うキーを選んだ
  接続で、下流が受け取る `displayName` がそのキーの値になることを観測する。

## Out of Scope

- `RFC7643-OUT-GROUP-RESOURCES` の他の節の観測。[[wi-507-back-provisioning-standards-rows-with-tests]] が持つ。
- User 側の属性解決。`UserAttributeSource` は接続ごとの取得元の選択を持たない。
- 属性対応付け (`AttributeMappingRule`) の表現力。拡張スキーマや多段のフィルターパスは
  `RFC7643-OUT-SCHEMA-EXTENSIONS` が宣言する範囲であり、この項目は触れない。
- メンバーシップの送出。

## Design

**選べる値は列挙にする。** 自由入力のまま任意の属性キーを通す道もあったが、選ばない。打ち間違いが
「保存され、画面は受け付け、配信は成功し、観測できる差だけが無い」という結果になり、この欠陥そのものの形を
再生産する。契約の側で閉じた集合にすれば、その状態は管理 API の境界で止まる。IdMagic 側の Group が持つ
属性は `name`、`description`、`email` の 3 つなので、集合は実体と一致する。

`ProvisioningGroupDisplayNameSource` を TypeSpec と domain の両方に置く。`GroupPushConfig.DisplayNameSource`
の型はその列挙になる。

**設定を読む場所は配送のユースケースである。** `ports.AttributeSource` の
`ResolveAttributes(ctx, tenantID, sourceType, sourceID)` は接続を受け取らない。接続ごとの選択を属性源へ
届けるには port の署名を変える必要があり、User 側にも波及する。一方 `deliverGroup` は既に接続を手にして
いるので、そこで解決すれば境界を増やさずに済む。

したがって責務をこう分ける。

- `GroupAttributeSource` は Group の事実だけを解決する —— `id`、`name`、`description`、`email`。
  `display_name` は返さない。どれを表示名にするかは接続ごとの判断であって、Group の事実ではない。
- `deliverGroup` が接続の選択を読み、`display_name` を組み立ててから client へ渡す。

主要な型と操作。

```go
type ProvisioningGroupDisplayNameSource string   // name | description | email
func (s ProvisioningGroupDisplayNameSource) Valid() bool
func (c *GroupPushConfig) DisplayNameSourceKey() string   // nil と未知は "name"
func groupDisplayName(attrs map[string]any, config *domain.GroupPushConfig) any
```

`DisplayNameSourceKey` は nil レシーバーを受ける。`GroupPush` が未設定の接続でも既定へ落ちる必要があり、
呼ぶ側に nil 検査を撒かないためである。

**未設定と未知と空は、いずれも Group の名前へ落ちる。** 表示名は fail-closed の判断ではない。`description`
と `email` は Group が持たないことがあるので、選んだ属性が空の配信を失敗させると、設定の打ち間違い 1 つで
その Group の送出全体が止まる。空文字を送るのも下流に拒否されうるので選ばない。

## Verification

- 既定とは違う取得元を選んだ接続で、下流が受け取る Group リソースの `displayName` がそのキーの値になる。
- 未設定と未知のキーは `Group.Name` になり、配信は成功する。
- `RFC7643-OUT-GROUP-RESOURCES` を名指すテストが、この節を区別する入力と観測を持つ。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **既定値で通るテストだけを書く。** 既定は現状の実装でも通るので、既定だけを観測しても選択が働いている
  証拠にならない。既定とは違うキーを必ず 1 つ通す。
- **未知のキーを失敗にする。** 表示名は fail-closed の判断ではない。未知のキーで配信を止めると、設定の
  打ち間違いが Group の送出全体を落とす。
- **選べる値の集合を決めずに実装する。** 自由入力のまま任意の属性キーを通すと、解決済みの属性表に無い
  キーの扱い（空文字にするのか既定へ落とすのか）が暗黙になる。TypeSpec の説明と画面の入力を、決めた
  集合に合わせる。

## Tasks

- [x] T001 [Spec] `ProvisioningGroupDisplayNameSource` を TypeSpec に足し、`display_name_source` の型を
  その列挙にする。`RFC7643-OUT-GROUP-RESOURCES` の該当節に、選べる集合と落ち方を書く。
- [x] T002 [Prereq] `check-api-compat` が `$ref` の傍らの keyword を捨てるために誤報を出すので、
  [[wi-537-api-compat-drops-keywords-written-beside-a-ref]] へ切り出して先に直す。
- [x] T003 [Domain] `ProvisioningGroupDisplayNameSource` と `GroupPushConfig.DisplayNameSourceKey()`。
  集合の外を `ProvisioningConnection.Validate()` が拒否する。recipe:
  `mise run test-go-test -- ./backend/provisioning/domain TestProvisioningConnection_ValidateRefusesAnUnknownDisplayNameSource`。
- [x] T004 [Use Cases] `deliverGroup` が接続の選択を読み、`display_name` を組み立てる。recipe:
  `mise run test-go-test -- ./backend/provisioning/usecases TestDeliverGroup_DisplayNameFollowsTheConfiguredSource`。
- [x] T005 [Adapters] `GroupAttributeSource` は Group の事実だけを解決し、`display_name` を返さない。
- [x] T006 [UI] 取得元の欄を自由入力から選択に変える。辞書に 3 つの選択肢を足す。recipe:
  `mise run test-ui-unit-file -- src/features/admin-applications/AdminApplicationProvisioningSettings.test.tsx`。
- [x] T007 [Docs] `docs/releases/changes/wi-536.md`。
- [x] T008 [Verify] `mise run verify` と `mise run test-ui-e2e`。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `RFC7643-OUT-GROUP-RESOURCES` の変更と、TypeSpec への
  `ProvisioningGroupDisplayNameSource` の追加を報告する。`group_push.display_name_source` が、下流へ送る
  Group の `displayName` に実際に反映されるようになった。それまでこの設定を読む production の経路は
  **1 つも無く**、管理者が何を入力しても下流が受け取る `displayName` は常に Group の名前だった。
  **値域を閉じた。** `display_name_source` は任意の文字列から `name`、`description`、`email` の 3 つへ狭まり、
  集合の外は `ProvisioningConnection.Validate()` が拒否する。管理画面の欄も自由入力から選択になった。
  自由入力のままにする道もあったが、打ち間違いが「保存され、配信も成功し、下流だけが変わらない」結果に
  なる —— **この欠陥そのものの形を再生産する**ので選ばなかった。
  **責務を 2 つに分けた。** `ports.AttributeSource` は接続を受け取らないので、取得元の選択を属性源へ届けるには
  port の署名を変えることになり、User 側にも波及する。代わりに `GroupAttributeSource` は Group の事実
  (`id`、`name`、`description`、`email`) だけを解決し、`display_name` を返さなくなった。どれを表示名にするかは
  接続ごとの判断であって Group の事実ではないので、接続を既に手にしている `deliverGroup` が組み立てる。
  **未設定、集合の外、選んだ属性が空 —— いずれも Group の名前へ落ちる。** 集合の外は管理 API が拒否するので
  今後は入らないが、`DisplayNameSourceKey()` は nil レシーバーと未知の値を受け続ける。`description` と
  `email` は Group が持たないことがあり、そこで配信を失敗させると設定の打ち間違い 1 つでその Group の
  送出全体が止まる。空文字を送るのも下流の検証に拒否されうるので選ばない。
  **検査器の欠陥を 1 件見つけ、先に直した。** `check-api-compat` は `$ref` の傍らに書かれた keyword を
  捨てるので、`display_name_source` を列挙にしただけで「default が `"name"` から undefined へ変わった」と
  5 件を誤報した。生成物はその位置に `"default": "name"` を持っている。baseline を更新すれば報告は消えるが、
  同じ読み落としは逆向き（傍らの `default` が変わっても報告ゼロ）にも効くので、
  [[wi-537-api-compat-drops-keywords-written-beside-a-ref]] として切り出し、先に直してから本項目へ戻った。
- **Primary Use Case Evidence**:
  - id: group-display-name-follows-the-connection
    unit_red: >-
      TestDeliverGroup_DisplayNameFollowsTheConfiguredSource は
      `domain.GroupDisplayNameSourceEmail` が存在せずビルドできなかった。型を入れた後は
      「説明を選ぶ」「メールアドレスを選ぶ」の 2 区画が
      `display_name = engineering, want The engineering group` と
      `want engineering@example.com` で落ちた。
    e2e_red: >-
      TestE2E_GroupChange_DisplayNameFollowsTheConfiguredSource は
      `POST /Groups body = map[displayName:engineering ...], want displayName=engineering@example.com`
      で落ちた。取得元に `email` を選んだ接続で、下流には Group の名前が届いていた。
    unit_fault_injection: >-
      `groupDisplayName` を取得元ではなく常に `attrs["name"]` から解決するようにすると、同テストの
      「説明を選ぶ」「メールアドレスを選ぶ」が落ちた。既定と「名前を選ぶ」の 2 区画は通るので、
      既定だけを観測しても選択の証拠にならないことがこの注入で読める。
    e2e_fault_injection: >-
      選んだ属性を Group が持たないときに配信を失敗させると、
      TestE2E_GroupChange_DisplayNameFallsBackToTheNameWhenTheSourceIsEmpty が落ちた
      (`ExecuteDelivery() error = provisioning: group has no email to use as displayName`)。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-api-compat`（TypeSpec を列挙にした直後）
  - **Requirement**: N/A: この行は `REQ` 番号を持たない標準行 `RFC7643-OUT-GROUP-RESOURCES` に対応する。
  - **Observed Failure**: exit 1、`5 breaking change(s)`。ただしこれは**検査器の誤報**であり、製品側の
    RED ではない。切り分けて [[wi-537-api-compat-drops-keywords-written-beside-a-ref]] で直した後は
    `ok API compatibility (no breaking changes vs spec/idmagic.openapi.baseline.json)` を返す。
  - **Detection Reason**: 製品の受入境界は Primary Use Case Evidence の E2E が持つ。ここに記録したのは、
    仕様を先に変えた時点で最初に落ちた検査が何で、なぜそれが製品の欠陥ではなかったかである。
- **Unit RED Evidence**:
  - **Test**: `TestProvisioningConnection_ValidateRefusesAnUnknownDisplayNameSource`
  - **Requirement**: N/A: 値域を閉じることは標準行 `RFC7643-OUT-GROUP-RESOURCES` の実装上の帰結であり、`REQ` 番号を持たない。
  - **Observed Failure**: `Validate() = nil, want an error for display_name_source="nickname"` と
    `…="displayName"`。列挙を足しただけでは実行時に何も変わらず、集合の外が保存できていた。
  - **Detection Reason**: TypeSpec の列挙は Go の検証を生成しないので、契約だけを狭めても値域は閉じない。
    閉じたことの担保は、拒否を観測するテストしかない。
- **Change-Resistance Results**:
  4 件の故障を注入し、すべて検出された。生存はゼロ。注入は Edit で当て、毎回テストの落ち方で着弾を
  確かめている。

  | 注入した故障 | 落ちたテストと観測 |
  |---|---|
  | M1 取得元を読まず常に Group の名前を使う（欠陥そのもの） | 単体の 2 区画と E2E。既定と「名前を選ぶ」は通る |
  | M2 空でも落とさず、選んだ属性をそのまま送る | 単体の「持たないとき」「空のとき」が `<nil>` と空文字で落ち、E2E は `displayName:<nil>` を送る |
  | M3 取得元が無いときに配信を失敗させる（fail-closed にする） | E2E の落ち先テストと単体の「持たないとき」。**非提供と拒否を分けている観測がこれで落ちる** |
  | M4 集合の外の取得元を `Validate()` が受け入れる | `…ValidateRefusesAnUnknownDisplayNameSource` の 2 区画。配信側は名前へ落ちるので、この注入は配信のテストを 1 件も落とさない —— 境界が 2 つあることが読める |

  **方法の限界。** 注入は取得元の解決と検証だけを崩しており、属性対応付け (`AttributeMappingRule`) の側は
  動かしていない。`displayName ← display_name` の既定の対応付けを外す注入は、この行ではなく
  `RFC7643-OUT-CORE-RESOURCES` の被覆が扱う範囲である。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (27 件)
