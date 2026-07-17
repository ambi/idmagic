---
status: accepted
authors: [tn]
created_at: 2026-07-14
---

# ADR-103: SCL 3.0 は規則を所有要素へ局所化し認可・SLO・UI 仕様を用途別に単純化する

> **注記（役割の境界）**: SCL 3.0 の**規範仕様は `SPECIFICATION_CORE_LANGUAGE.md`（および `spec/contexts/*.yaml`）が所有する**（影響節の後続 work item で移管済み）。本 ADR は 3.0 採用の**決定と理由の記録**であり、以下の構造定義・YAML 例は決定時点のもの。現行の規範形が本 ADR と食い違う場合は `SPECIFICATION_CORE_LANGUAGE.md` を正とする。

## コンテキスト

SCL 2.0 は `models`、`interfaces`、`states`、`scenarios` に加えて、横断分類として
`invariants`、`permissions`、`objectives`、`user_experience` を持つ。IdMagic の仕様を
この形で拡張した結果、後者の分類は次の問題を抱えた。

- `invariants` にはモデル制約、状態遷移規則、interface の事前・事後条件、認可、セキュリティ、
  UI 品質、運用要件、アーキテクチャ判断が混在している。同じ意味が model/interface の
  `description`、scenario、UX requirement にも重複する。
- `objectives` は SLO だけでなく retention、TTL、security configuration、ログ形式、runtime
  設定まで `kind` union に収容しており、「測定可能な目標」という共通意味を失っている。
- `permissions` は同じ認証済み・有効・role・tenant 条件を各ルールの `allow_when` に反復し、
  どの interface を保護するかも `protects` の省略により曖昧になり得る。
- `user_experience` は screen 台帳、route、view state、画面遷移、セキュリティ要件、ローカライズ、
  accessibility を一か所に集め、UI 実装と scenario の双方と重複している。
- 関連は `relates_to`、`protects`、UX 内の section 別配列、自然文の自動リンクに分散し、
  参照整合性を一つの規則で検査できない。
- scenario は単純な `steps` と Cockburn use case 形式を併用できるため、同じ粒度の仕様でも
  読み方と記述方針が一定しない。

wi-183 では SCL を Markdown、TypeSpec、Cedar、OpenSLO、Mermaid 等の複数正本へ分解する案を
試したが、一覧性、検索性、型忠実度、機械検証性、パターンの一貫性が SCL 単一形式より悪化し、
変更を撤回した。この経験から、SCL という単一の論理正本は維持し、その内部構造を簡潔にする。

認可については ADR-010 の AuthZEN 形式 `{subject, action, resource, context}` を維持する。
Cedar は生成先として引き続き想定するが、SCL を Cedar の保存構文の複製にはしない。Cedar が
推奨する principal group と action group の意味を保ちながら、SCL では共通 principal 条件と
interface からの policy 参照を正本にする。

## 決定

### 1. SCL 3.0 とトップレベル構造

本変更を後方互換性のない SCL 3.0 とする。SCL は引き続き context-local な YAML 文書と、
複数 context を束ねる context map から構成する。

```yaml
system: TaskTracker
spec_version: "3.0"
context: TaskExecution

annotations: { ... }
standards: { ... }
context_map: { ... }
glossary: { ... }
models: { ... }
interfaces: { ... }
states: { ... }
authorization: { ... }
objectives: { ... }
scenarios: { ... }
flows: { ... }
```

`invariants` と `user_experience` は廃止する。旧形式を受理する互換 alias、deprecated field、
自動 fallback は設けない。移行は後続 work item で全 context と tooling を一括して行う。

### 2. invariant を所有要素の契約へ移す

普遍的規則を独立した一覧へ置かず、規則を実現・検証する要素へ置く。

| 規則の意味 | SCL 3.0 の正本 |
| --- | --- |
| 単一 field の形式・範囲・長さ | `models.<name>.fields.<field>.constraints` |
| model 内の複数 field にまたがる整合 | `models.<name>.constraints` |
| 操作の事前条件 | `interfaces.<name>.requires` |
| 操作の事後条件・応答・event 発行 | `interfaces.<name>.ensures` |
| 状態遷移の許可条件・効果・終端性 | `states.transitions.guard/effect`、`states.terminal` |
| 認可判断 | `authorization` と `interfaces.<name>.access` |
| ブラックボックスで観測する具体例 | `scenarios` |
| 集計可能な性能・信頼性目標 | `objectives` |
| UI の view 間遷移 | `flows` |
| 構成・技術選択・実装方針 | ADR または `ARCHITECTURE.md` |

model-level `constraints`、interface の `requires` / `ensures` は CEL 文字列の配列とする。
個々の式に名前や補助 object を持たせない。意図は親要素の `description`、具体的な反例は
scenario に置く。

```yaml
models:
  TenantFooterLink:
    kind: value_object
    fields:
      label:
        type: String
        constraints: [non_empty, { max_length: 80 }]
      url:
        type: String
        constraints: [non_empty, { pattern: "^https://" }]

interfaces:
  UploadTenantBrandingAsset:
    requires:
      - context.upload.content_type in ["image/png", "image/jpeg", "image/webp", "image/gif"]
      - context.upload.magic_byte_matches_content_type
      - size(input.file) <= 262144
    ensures:
      - emitted.exists(e, e is TenantBrandingUpdated)

  GetTenantBrandingAsset:
    ensures:
      - response.headers["Content-Type"] in ["image/png", "image/jpeg", "image/webp", "image/gif"]
      - response.headers["X-Content-Type-Options"] == "nosniff"
```

CEL の binding は位置で決める。

- model `constraints`: model の field 名
- `states.guard`: 現行どおり target model の field 名と `input`
- interface `requires`: `input`、`resource`、`subject`、`context`
- interface `ensures`: `requires` の binding に加え、`output`、`response`、`emitted`
- authorization principal/policy: `principal`、`resource`、`context`
- objective indicator: `request`、`response`、`event`、`measurement`

### 3. 認可を principal、policy、interface access に分ける

トップレベルの `permissions` を `authorization` に置き換える。配下には明示的な singleton resource
`resources`、再利用可能な主体集合 `principals`、認可規則 `policies` を置く。`permission` は許可された能力または判定結果を指し、
permit/forbid と条件を含む宣言は policy であるため、規則名には `policies` を用いる。

```yaml
authorization:
  resources:
    System:
      description: システム全体を表す singleton resource

  principals:
    TenantAdministrator:
      type: User
      matches:
        - context.authenticated
        - principal.status == Active
        - '"admin" in principal.effective_roles'
        - principal.tenant_id == context.tenant_id

    SystemAdministrator:
      type: User
      matches:
        - context.authenticated
        - principal.status == Active
        - '"system_admin" in principal.effective_roles'
        - principal.tenant_id == "default"

  policies:
    TenantAdministrator:
      effect: permit
      principal: TenantAdministrator

    SystemAdministrator:
      effect: permit
      principal: SystemAdministrator
```

`resources.<name>` は domain model では表さない singleton authorization resource type を明示する。
`principals.<name>` は `type` と非空の `matches` を持つ。これは Cedar entity group の保存形式ではなく、
principal scope と condition へ展開する SCL の authoring abstraction である。role、認証状態、主体の
lifecycle、request context に依存する共通条件を一度だけ宣言する。

`policies.<name>` は `effect: permit | forbid`、`principal`、任意の `when` を持つ。`principal` は
`authorization.principals` を参照する。`when` は所有者一致、resource classification、MFA、時刻など、
その policy 固有で request ごとに変わる ABAC/ReBAC 条件に限定する。

interface は `access` により enforcement を宣言する。

```yaml
interfaces:
  ResolveTenant:
    access: internal

  GetTenantBranding:
    access: public

  UpdateTenantBranding:
    access:
      policies: [TenantAdministrator]
      resource:
        type: TenantBranding
        id: context.tenant_id

  UpdateTenant:
    access:
      policies: [SystemAdministrator]
      resource:
        type: Tenant
        id: input.tenant_id
```

`access` の形は次のいずれかだけを許す。

- `public`: 外部から認証なしで呼べる。policy 評価を行わない。
- `internal`: 外部 binding を持たない内部 interface。policy 評価を行わない。
- object: protected interface。非空の `policies` と `resource.type` / `resource.id` を必須とする。

`resource.type` は model または明示された singleton authorization resource、`resource.id` は CEL 式とする。
interface 名を AuthZEN/Cedar の action 名とする。同じ policy を参照する interface 群から Cedar action
group と action `memberOf` を派生し、action の `appliesTo` は principal type と resource type から生成する。
policy 側に action 一覧を重ねて書かない。

複数 policy を参照した場合は Cedar と同じ合成規則を使う。matching `permit` が1件以上あり、matching
`forbid` が0件なら allow、それ以外は implicit deny とする。明示的 forbid は permit より優先する。
validator は全 interface の `access` 明示、protected interface の policy/resource 完備、参照先、
resource ID 式、principal/resource/action type の整合を検査する。

この決定は ADR-010 を置き換えない。AuthZEN request model と policy adapter 境界を維持し、その上流に
ある SCL authoring model を精密化する。

### 4. objectives を SLO 専用にする

`objectives` は OpenSLO を参考に、観測可能な SLI に対する SLO だけを表す。各 objective は boolean
`indicator`、達成比 `target`、評価窓 `window`、budget 計算方法 `budgeting` を持つ。

```yaml
objectives:
  AuthorizeLatency:
    description: Authorize 応答の99%以上を500ms未満にする。
    interface: Authorize
    indicator: measurement.latency < duration("500ms")
    target: 0.99
    window: 30d
    budgeting: occurrences

  WorkerAvailability:
    indicator: measurement.healthy
    target: 0.999
    window: 30d
    budgeting: timeslices
    slice: 5m
```

`target` は `0 < target <= 1`、`budgeting` は `occurrences | timeslices`、既定は `occurrences` とする。
`timeslices` の場合だけ `slice` を必須とする。metric backend、PromQL、monitoring vendor、alert routing は
SCL コアに含めず派生先の設定とする。

旧 objective は次のように移す。

- `kind: slo`: 新 objective へ変換する。
- retention: model の lifecycle、削除/archive interface、state、scenario へ移す。
- lifetime / TTL: model field constraint、state transition、expiry interface、scenario へ移す。
- security configuration: model constraint、interface contract、authorization、scenario へ移す。
- runtime/logging/architecture configuration: interface contract、ADR、`ARCHITECTURE.md` のうち所有場所へ移す。

### 5. flows を UI ナビゲーション図に限定する

`user_experience` を廃止し、`flows` は view 間のナビゲーションだけを表す。flow に `goal`、`actor`、
screen 台帳、route、loading/error state、横断 requirement を持たせない。ユーザーの達成目標と結果は
scenario、認可は authorization、外部標準は standards、測定可能な UI 品質は objective が所有する。

```yaml
flows:
  TenantManagement:
    entry: TenantList
    transitions:
      - from: TenantList
        action: create
        to: TenantCreate

      - from: TenantCreate
        action: submit
        interface: CreateTenant
        to: TenantList

      - from: TenantList
        action: edit
        to: TenantEdit

      - from: TenantEdit
        action: submit
        interface: UpdateTenant
        to: TenantList
```

flow は `entry` と非空の `transitions` を持つ。transition は `from` と `action` を必須、`interface` と
`to` を任意とする。`to` を省略した transition は終了を表す。外部遷移は `external: true` とし、
`to` と `external` は同時に指定しない。view は `entry`、`from`、`to` から推論するため別台帳を持たない。
小さな flow を許容し、CRUD 一式を一つの人工的な goal にまとめない。

### 6. scenarios を主成功経路と拡張に一本化する

scenario は map key 自体を受け入れ基準の見出しとし、`actor`、任意の `given`、必須の
`main_success`、任意の `extensions` だけを中核構造とする。単独 `steps`、`goal`、`scope`、`level`、
`success_guarantees` は廃止する。scope は context 文書から、goal と保証は見出しと主成功経路から分かる。

```yaml
scenarios:
  管理者はテナントを作成できる:
    actor: SystemAdministrator
    given:
      - realm "acme" は未登録である
    main_success:
      - 管理者が realm "acme" のテナントを作成する
      - 作成されたテナントは status "Active" で返る
      - "TenantCreated" が発行される
    extensions:
      - at: 1
        condition: realm "acme" が既に登録されている
        steps:
          - エラー "InvalidRequestError" が返る
          - テナントは追加されない
```

`actor` は glossary または authorization principal、`at` は1始まりの主成功 step 番号を参照する。
extension は `condition` と非空の `steps` を必須とし、主成功経路から分岐する失敗・拒否・代替経路を
記述する。scenario は引き続き interface を通したブラックボックス仕様とする。

### 7. 関連参照を意味フィールドから導く

model field type、interface input/output/error/event、state target/event、authorization principal/policy/resource、
objective interface、flow interface、scenario step に現れる既知名を意味参照として検査・リンクする。
`relates_to`、`protects`、UX の section 別参照配列は廃止する。

汎用参照は外部標準 requirement の `refs` にだけ許可し、値は `section.name` 形式とする。

```yaml
standards:
  RFC7636:
    requirements:
      - id: RFC7636-S256
        strength: MUST
        adoption: required
        statement: code_challenge_method は S256 を使用する。
        refs:
          - interfaces.Authorize
          - interfaces.Token
```

### 8. Tenancy context の代表移行

`spec/contexts/tenancy.yaml` の旧 invariant は次へ移す。

| SCL 2.0 | SCL 3.0 |
| --- | --- |
| `TenantBrandingSafeTokens` | branding model/request の field constraints と更新 interface 契約 |
| `TenantBrandingSafeAssetServing` | upload `requires` と asset GET `ensures` |
| `TenantBrandingAssetTenantIsolation` | authorization principal/resource と cross-tenant scenario |
| `TenantBrandingUploadedAssetIsDisplayable` | upload→GET→UI 表示の主成功 scenario |
| `TenantBrandingFailsOpen` | `GetTenantBranding.ensures` と fallback scenario |

旧 `AdminTenantsManage`、`AdminSettingsRead/Update`、`BrandingUpdate` の反復条件は
`authorization.principals.TenantAdministrator/SystemAdministrator` へ集約する。各 interface が参照する
policy と resource を宣言する。旧 `UX-ADMIN-CONTROL-PLANE` は system administrator policy、system
console flow、通常 administrator の拒否 scenario に分解する。

Tenancy に置かれている監査ログの参照・export scenario は Tenancy interface を使わないため Audit
context へ移す。`objectives.PasswordPolicy` は SLO ではないため、所有 context の password policy
model/configuration contract へ移す。

### 9. 参照した外部設計

- OpenSLO: <https://openslo.com/>
- OpenID AuthZEN Authorization API 1.0: <https://openid.github.io/authzen/>
- Cedar basic policy syntax: <https://docs.cedarpolicy.com/policies/syntax-policy.html>
- Cedar RBAC with principal groups: <https://docs.cedarpolicy.com/bestpractices/bp-implementing-roles.html>
- Cedar action `memberOf` / `appliesTo` schema: <https://docs.cedarpolicy.com/schema/json-schema-grammar.html>

## 却下した代替案

- `invariants` を `guarantees` に改名して残す: 名前を変えても異なる所有責務を横断 bucket に集める
  問題は解消しない。
- 形式検証対象だけ `invariants` に残す: 何が「形式検証対象」かで分類が揺れ、model/interface/state
  契約との重複が残る。
- 全要素に共通 `rules` 配列を置く: 記法は揃うが、事前条件・事後条件・状態 guard の意味の違いを
  隠し、生成先を判断しにくくする。
- SCL を複数の仕様形式へ分解する: wi-183 の試行で一覧性、機械検証性、型忠実度、一貫性が低下した。
- `objectives` に retention、TTL、security、runtime 設定を残す: 共通の測定・評価意味を定義できず、
  kind union が無制限な設定 bucket へ戻る。
- トップレベル名を `permissions` のままにする: permit/forbid と条件を持つ規則集合には `policies` が
  より正確であり、principal 定義まで含む全体には `authorization` が適切である。
- policy が action 一覧を所有する: interface と policy の双方に操作名を書き、追加・rename 時に
  drift が生じる。
- Cedar action group と `appliesTo` を SCL に直接手書きする: Cedar との一対一性は高いが、SCL の
  局所性を失い、interface 名を中央台帳へ再列挙する。action group は interface の policy 参照から
  決定的に生成できる。
- interface に policy を置かず、参照関係を自動推論する: 認可漏れと意図的 public の区別ができない。
- `user_experience` の screen 台帳を縮小して残す: route/view state と UI 実装の同期コストが残り、
  scenario と flow の責務が重なる。
- flow ごとに `goal` を必須にする: 管理画面のナビゲーションを「参照・作成・更新・削除する」のような
  人工的な複合 goal にしやすい。goal は scenario の見出しが所有する。
- UI flow を scenario に統合する: UI topology と受け入れ挙動が再び混在し、画面遷移図を局所的に
  取得しにくい。
- 単純 `steps` と Cockburn use case の二形式を維持する: 同じ種類の仕様に二つの読み方が残る。
- SCL 2.x と3.0を段階併存させる: validator、renderer、AI authoring rule が二重化し、廃止対象の
  曖昧さを移行期間中も増幅する。

## 影響

- 本 ADR の採用時点では `spec/scl.yaml`、`spec/contexts/*.yaml`、runtime、外部 API を変更せず、
  後続 work item で段階的に移行する。
- 採用後は別 work item で `SPECIFICATION_CORE_LANGUAGE.md` を SCL 3.0 に改訂し、JSON Schema、
  yaml validator、HTML renderer、OpenAPI/JSON Schema generator、型、fixture、全 context を一括移行する。
- SCL 変更後は HTML、model JSON Schema、OpenAPI 等の派生物を再生成する。
- RA の SCL section 対応、SCL-first skill、work item scope 指針、agent instructions、
  `REGENERATIVE_ARCHITECTURE.md`、必要に応じて `ARCHITECTURE.md` を同期する。
- authorization generator は interface の policy 参照から AuthZEN request schema、Cedar action schema、
  action group、permit/forbid policy を生成できるようにする。
- validator は全 interface の access classification、protected action の policy coverage、参照整合、
  CEL binding/type、scenario/flow/objective の新しい単一形を検査する。
- 全 context の移行では、旧 invariant/objective/UX requirement を機械的に名前変更せず、上記の
  所有規則に従って一件ずつ再分類する。移行前後で規範的挙動を変える必要がある場合は、その変更を
  別 ADR または work item に分離する。
