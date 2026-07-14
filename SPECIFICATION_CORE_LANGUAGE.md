# Specification Core Language (SCL)

Specification Core Language (SCL) は Regenerative Architecture の第 1 層
*Specification Core* を記述する YAML 言語である。SCL は契約、型、状態、認可、受け入れ例、SLO、
UI navigation の単一上流ソースであり、コード、schema、API 文書、図、test、monitoring rule は
SCL から派生する。

本書は SCL 3.0 の規範仕様である。`MUST` / `MUST NOT` / `SHOULD` は機械検証または review の
合否基準を表す。

## 1 目的

SCL は次の性質を持つ。

- 実装言語、framework、database、runtime、policy engine、monitoring vendor に依存しない。
- 人間と AI が同じ上流仕様を読み書きできる。
- JSON Schema による shape 検証と、section 間の意味検証ができる。
- bounded context ごとに局所化でき、context map で公開言語と依存だけを接続する。
- 規則をそれを所有・実現する model、interface、state、authorization 等へ置く。

SCL は構成や技術選択を所有しない。それらは ADR または `ARCHITECTURE.md` に置く。

## 2 文書構造

SCL 3.0 文書は次の top-level field だけを持つ。

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

| field | 必須 | 意味 |
| --- | --- | --- |
| `system` | MUST | システム識別子。非空文字列。 |
| `spec_version` | MUST | SCL meta-spec version。SCL 3.0 文書では厳密に `"3.0"`。 |
| `context` | MAY | context-local 文書が所有する bounded context 名。 |
| `annotations` | MAY | 意味を変更しない補助 metadata (`map[string, any]`)。 |
| その他の section | MAY | 後続節で定義する名前付き要素の map。 |

未知の top-level field は拒否する。SCL 3.0 は `invariants`、`permissions`、
`user_experience` を持たず、互換 alias や自動変換も持たない。

### 2.1 standards — 外部標準との対応

`standards` は外部標準と採用する requirement を宣言する。

```yaml
standards:
  RFC7636:
    title: Proof Key for Code Exchange by OAuth Public Clients
    version: RFC 7636
    url: https://www.rfc-editor.org/rfc/rfc7636.html
    roles: [AuthorizationServer]
    scope: Authorization Code Grant の横取り攻撃対策
    requirements:
      - id: RFC7636-S256
        section: "§4.2"
        strength: MUST
        adoption: required
        statement: code_challenge_method は S256 を使用する。
        refs: [interfaces.Authorize, interfaces.Token]
```

requirement は `id`、`strength`、`adoption`、`statement` を必須とする。`strength` は
`MUST | SHOULD | MAY | MUST NOT | SHOULD NOT`、`adoption` は
`required | optional | excluded` である。`excluded` は `reason` を必須とする。

`refs` は唯一の汎用参照 field であり、各値を `section.name` として解決する。未解決参照、
旧 `relates_to`、work item / ADR / commit 番号は拒否する。

### 2.2 context_map — bounded context の対応

`context_map` は通常 root 文書だけに置く。各 entry は `description`、任意の `path`、
`publishes`、`depends_on`、`annotations` を持つ。

```yaml
context_map:
  TaskAuthoring:
    path: contexts/task-authoring.yaml
    description: Task の定義を所有する。
    publishes: [Task, TaskCreated]
  TaskExecution:
    path: contexts/task-execution.yaml
    description: Task の実行を所有する。
    depends_on:
      TaskAuthoring:
        uses: [Task, TaskCreated]
        via: published_language
```

`depends_on.<context>.uses` の各名前は参照先の `publishes` に存在しなければならない。
依存 graph は循環してはならない。`via` は `shared_kernel | published_language |
customer_supplier | conformist | anticorruption_layer` のいずれかである。

## 3 セクションリファレンス

変更時は次の所有規則を用いる。

| 変更の意味 | 見直す section / field |
| --- | --- |
| 用語、別名、翻訳 | `glossary` |
| 型、同一性、field 制約、model 内整合 | `models` |
| 入出力、error、event、事前・事後条件 | `interfaces` |
| lifecycle と遷移 | `states` |
| principal、policy、interface access | `authorization` と `interfaces.access` |
| 集計可能な性能・信頼性目標 | `objectives` |
| black-box の成功、境界、失敗、拒否 | `scenarios` |
| view 間 navigation | `flows` |
| 外部標準 requirement | `standards` |

### 3.1 glossary — 意味の語彙

`glossary` は曖昧語、alias、翻訳、外部標準語を説明する。model、event、interface を台帳として
重複登録してはならない。

```yaml
glossary:
  Principal:
    definition: 認可判断の主体。
    aliases: [Subject]
    not_to_confuse_with:
      - term: User
        reason: User は principal になり得る model の一つである。
```

entry は `definition` を必須とし、任意で `description`、`aliases`、`context`、
`not_to_confuse_with`、`annotations` を持つ。

### 3.2 models — domain model

`models` の各要素は `kind` を必須とする。

| kind | 必須 field | 意味 |
| --- | --- | --- |
| `entity` | `identity`, `fields` | 継続する同一性を持つ model。 |
| `value_object` | `fields` | field 値で同一性が決まる model。 |
| `event` | なし (`payload` MAY) | 発生済みの事実。 |
| `enum` | `values` | 閉じた値集合。 |
| `error` | なし (`fields` MAY) | interface が返す typed error。 |

```yaml
models:
  Tenant:
    kind: entity
    identity: id
    fields:
      id: { type: UUID }
      name: { type: String, constraints: [non_empty, { max_length: 80 }] }
      valid_from: { type: DateTime }
      valid_until: { type: DateTime, optional: true }
    constraints:
      - valid_until == null || valid_until > valid_from
```

`identity` は field 名または非空の field 名配列で、すべて `fields` に解決しなければならない。
field は `type` を必須とし、`optional`、`default`、`constraints`、`description`、
`annotations` を持てる。field type は組み込み型、model 名、または §4 の parametric 型に解決する。

field 単体の形式、範囲、長さは field `constraints` に置く。複数 field にまたがる規則は model
`constraints` に非空 CEL 文字列の配列として置く。model-level CEL binding はその model の field 名である。

### 3.3 interfaces — 外部・内部契約

interface は操作の black-box 契約を所有する。

```yaml
interfaces:
  UpdateTenant:
    description: Tenant の表示名を更新する。
    input:
      tenant_id: { type: UUID }
      name: { type: String }
    output:
      tenant: { type: Tenant }
    errors: [TenantNotFound, InvalidTenant]
    emits: [TenantUpdated]
    requires:
      - input.name != ""
      - context.authenticated
    ensures:
      - output.tenant.id == input.tenant_id
      - emitted.exists(e, e is TenantUpdated)
    access:
      policies: [TenantAdministrator]
      resource: { type: Tenant, id: input.tenant_id }
    idempotent: true
    bindings:
      - kind: http
        method: PATCH
        path: /tenants/{tenant_id}
```

`input` / `output` は field map、`errors` は `kind: error` model、`emits` は `kind: event`
model を参照する。`requires` / `ensures` は非空 CEL 文字列の配列である。

すべての interface は `access` を明示しなければならない。

- `public`: 認証なしで外部から利用でき、policy 評価をしない。
- `internal`: 外部 binding を持たない内部操作で、policy 評価をしない。
- object: protected 操作。非空 `policies` と `resource.type` / `resource.id` を必須とする。

`bindings.kind` は `http | grpc | cli | event | graphql | sdk | schedule`。kind 固有 field は
JSON Schema が検証する。SCL 3.0 は既存 binding 契約を維持する。

### 3.4 states — lifecycle

```yaml
states:
  TenantLifecycle:
    target: Tenant
    initial: Pending
    terminal: [Deleted]
    transitions:
      - from: Pending
        event: TenantActivated
        to: Active
        guard: input.approved && status == Pending
        effect: [TenantBecameActive]
```

state machine は `target`、`initial`、`transitions` を必須とする。transition は `from`、`event`、
`to` を必須とし、任意で `guard` と `effect` を持つ。`target` は model、`event` と `effect` は
event model に解決する。`terminal` に到達した state からの transition を定義してはならない。
`guard` の CEL binding は target model の field 名と `input` である。

### 3.5 authorization — principal と policy

`authorization` は singleton resource、再利用可能な principal 集合、policy を所有する。

```yaml
authorization:
  resources:
    System:
      description: system 全体を表す singleton resource。
  principals:
    TenantAdministrator:
      type: User
      matches:
        - context.authenticated
        - principal.status == "Active"
        - principal.tenant_id == context.tenant_id
  policies:
    TenantAdministrator:
      effect: permit
      principal: TenantAdministrator
      when: resource.tenant_id == context.tenant_id
```

`resources` の key は model では表さない singleton authorization resource type を宣言する。
principal は model を参照する `type` と非空 `matches` を必須とする。policy は
`effect: permit | forbid` と principal 参照を必須とし、任意で `when` を持つ。

protected interface の `resource.type` は model または `authorization.resources` に解決し、
`policies` はすべて `authorization.policies` に解決しなければならない。複数 policy は
matching permit が 1 件以上かつ matching forbid が 0 件の場合だけ allow し、それ以外は implicit deny
とする。interface 名を action 名とし、policy に action 一覧を重複記載しない。

### 3.6 objectives — SLO

`objectives` は観測可能な SLI に対する SLO だけを表す。

```yaml
objectives:
  AuthorizeLatency:
    description: 応答の 99% 以上を 500ms 未満にする。
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

`indicator`、`target`、`window` は必須。`0 < target <= 1`。`budgeting` は
`occurrences | timeslices` で、省略時は `occurrences`。`timeslices` の場合だけ `slice` を必須とする。
`interface` を書く場合は既知 interface に解決する。metric query、alert routing、retention、TTL、
security/runtime configuration は objective に置かない。

### 3.7 scenarios — 受け入れ例

scenario は主成功経路とそこからの extension の単一形を持つ。

```yaml
scenarios:
  管理者はテナントを作成できる:
    actor: SystemAdministrator
    given:
      - realm "acme" は未登録である
    main_success:
      - 管理者が CreateTenant を実行する
      - TenantCreated が発行される
    extensions:
      - at: 1
        condition: realm "acme" が既に登録されている
        steps:
          - InvalidRequestError が返る
          - Tenant は追加されない
```

`actor` と非空 `main_success` は必須。actor は glossary または authorization principal に解決する。
extension は 1 始まりの `at`、`condition`、非空 `steps` を必須とし、`at` は
`main_success` の範囲内でなければならない。`steps`、`goal`、`scope`、`level`、
`primary_actor`、`success_guarantees` を scenario 直下に置く旧形は拒否する。

scenario の自然文中に現れる既知の model、interface、event、error 名は意味 link として扱う。
通常語を未解決参照とは扱わない。機械的に必ず解決すべき参照は意味 field または `refs` に置く。

### 3.8 flows — view navigation

```yaml
flows:
  TenantManagement:
    entry: TenantList
    transitions:
      - { from: TenantList, action: edit, to: TenantEdit }
      - { from: TenantEdit, action: submit, interface: UpdateTenant, to: TenantList }
      - { from: TenantList, action: documentation, external: true }
```

flow は `entry` と非空 `transitions` を必須とする。transition は `from` と `action` を必須とし、
任意で `interface`、`to`、`external: true` を持つ。`to` の省略は終了を表す。`external` と `to` は
同時に指定できない。interface 参照は解決しなければならず、すべての `from` view は `entry` から
到達可能でなければならない。

flow は screen 台帳、route、goal、actor、loading/error state、横断 UX requirement を持たない。

## 4 型システム

### 4.1 組み込み型

標準 scalar は `String`、`Bool` / `Boolean`、`Int` / `Integer`、`Float`、`Decimal`、
`Number`、`Bytes`、`UUID`、`URI` / `URL`、`Date`、`DateTime`、`Duration`、`JSON`、`Any`。

### 4.2 parametric 型

collection と optionality は `List<T>` / `T[]`、`Set<T>`、`Map<K, V>`、`Optional<T>` で表せる。
参照された user-defined type は同じ文書の `models` または context map の公開言語として解決する。

### 4.3 field constraint

field `constraints` は文字列または単一 key object の配列で表す。標準 constraint は
`non_empty`、`min_length`、`max_length`、`minimum`、`maximum`、`pattern`、`format`、
`unique`。generator が知らない constraint を黙って捨ててはならない。

## 5 CEL expression と binding

SCL 3.0 の model constraint、interface contract、state guard、authorization、objective indicator、
resource id は非空の CEL 文字列で表す。構造化 expression object と boolean literal は受理しない。

| 位置 | 利用できる root binding |
| --- | --- |
| `models.<name>.constraints` | model の field 名 |
| `states.<name>.transitions[].guard` | target model の field 名、`input` |
| `interfaces.<name>.requires` | `input`, `resource`, `subject`, `context` |
| `interfaces.<name>.ensures` | requires の binding、`output`, `response`, `emitted` |
| `authorization.principals.*.matches` | `principal`, `resource`, `context` |
| `authorization.policies.*.when` | `principal`, `resource`, `context` |
| `interfaces.*.access.resource.id` | `input`, `resource`, `subject`, `context` |
| `objectives.*.indicator` | `request`, `response`, `event`, `measurement` |

validator は少なくとも root binding の範囲を検証し、利用不能な root を拒否する。完全な型検査を
提供する validator は field / model 型も検証しなければならない。

## 6 意味参照と検証順序

validator は次の順序で検証する。

1. `spec_version: "3.0"` を確認する。
2. JSON Schema で閉じた shape、必須性、enum、条件付き必須を検証する。
3. model type / identity、interface error / event、state target / event、authorization、objective、
   flow、scenario、standard refs を解決する。
4. 全 interface の access coverage と位置別 CEL binding を検証する。
5. context map graph と公開言語を検証する。

SCL 3.0 以外の version と廃止 field は拒否する。shape と意味検証は別 finding として報告できるが、
いずれか一つでも error があれば文書は invalid である。

## 7 記法と保存形式

- UTF-8 YAML を正本とし、拡張子は `.yaml` とする。
- map key は同一 map 内で一意でなければならない。
- element 名は参照の identity であり、rename は破壊的変更として扱う。
- `description` と scenario step は自然文、expression と identifier は機械可読文字列として扱う。
- SCL に work item、ADR、commit、実装ファイルへの trace id を書かない。
- 派生 artifact は SCL 更新後に再生成し、単独開発 / integration / main では同期させる。

## 8 versioning

`spec_version` は schema 自体の version であり、製品 version ではない。現在の workspace と validator は
`"3.0"` だけを受理し、欠落、未知 version、過去 version、廃止 field を拒否する。互換 alias、自動変換、
fallback は持たない。SCL 3.0 への移行判断と旧 field の所有先は [ADR-103](decisions/ADR-103-scl-3-localized-specification-structure.md) に記録する。
