# Specification Core Language (SCL)

Specification Core Language (SCL) は Regenerative Architecture の第 1 層
*Specification Core* を記述する YAML 言語である。SCL は契約、型、状態、認可、受け入れ例、SLO、
UI navigation の単一上流ソースであり、コード、schema、API 文書、図、test、monitoring rule は
SCL から派生する。

本書は SCL 3.0 の規範仕様である。`MUST` / `MUST NOT` / `SHOULD` は機械検証または review の
合否基準を表す。本書の各 field 表は `tools/yaml-check/schemas/scl-v3.schema.json`(shape)と
`tools/yaml-check/src/scl-semantics.ts` / `tools/yaml-check/src/context-map.ts`(意味検証)が
実際に強制する制約と一対一で対応する。本書はそれらが検証しない制約を追加しない。

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
| `context` | MAY | context-local 文書が所有する bounded context 名。非空文字列。 |
| `annotations` | MAY | 意味を変更しない補助 metadata (object)。 |
| `standards` | MAY | §2.1 参照。 |
| `context_map` | MAY | §2.2 参照。 |
| `glossary` | MAY | §3.1 参照。 |
| `models` | MAY | §3.2 参照。 |
| `interfaces` | MAY | §3.3 参照。 |
| `states` | MAY | §3.4 参照。 |
| `authorization` | MAY | §3.5 参照。単一 object (map の map ではない)。 |
| `objectives` | MAY | §3.6 参照。 |
| `scenarios` | MAY | §3.7 参照。 |
| `flows` | MAY | §3.8 参照。 |

トップレベルはここに列挙した field だけを許可する (JSON Schema `additionalProperties: false`)。
SCL 3.0 は `invariants`、`permissions`、`user_experience` を持たず、互換 alias や自動変換も
持たない。

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

`Standard` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `title` | MUST | 非空文字列。 |
| `url` | MUST | URI 形式の文字列。 |
| `version` | MAY | 文字列。 |
| `roles` | MAY | 文字列配列。 |
| `scope` | MAY | 文字列。 |
| `requirements` | MAY | 下表の requirement 配列。 |

requirement の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `id` | MUST | 非空文字列。 |
| `strength` | MUST | `MUST \| SHOULD \| MAY \| MUST NOT \| SHOULD NOT`。 |
| `adoption` | MUST | `required \| optional \| excluded`。 |
| `statement` | MUST | 非空文字列。 |
| `section` | MAY | 文字列。 |
| `reason` | `adoption: excluded` のとき MUST、それ以外 MAY | 文字列。 |
| `refs` | MAY | `section.name` 形式の一意な文字列配列 (§3 冒頭表参照)。 |

`refs` は唯一の汎用参照 field であり、各値を `section.name` として解決する。未解決参照、
旧 `relates_to`、work item / ADR / commit 番号は拒否する。

### 2.2 context_map — bounded context の対応

`context_map` は通常 root 文書だけに置く。

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

`ContextMapEntry` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `description` | MUST | 非空文字列。 |
| `path` | MAY | 非空文字列。 |
| `publishes` | MAY | 文字列配列。 |
| `depends_on` | MAY | 依存先 context 名をキーとする map。 |
| `annotations` | MAY | object。 |

`depends_on.<context>` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `uses` | MUST | 非空文字列配列。各値は参照先 context の `publishes` に存在しなければならない。 |
| `via` | MAY | `shared_kernel \| published_language \| customer_supplier \| conformist \| anticorruption_layer`。 |
| `reason` | MAY | 文字列。 |

`depends_on` の各キーは既知の context 名に解決しなければならず、依存 graph は循環しては
ならない (`tools/yaml-check/src/context-map.ts` が DFS で検証する)。`via: shared_kernel` かつ
`uses` が 3 件を超える関係は warning として報告される (`published_language` または
`anticorruption_layer` への narrowing を推奨する信号であり、文書を invalid にはしない)。

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
| view が見せるもの・利用者の操作・view 間 navigation | `flows` |
| 外部標準 requirement | `standards` |

`refs` (§2.1) が解決する `section.name` の `section` は `glossary` / `models` / `interfaces` /
`states` / `objectives` / `scenarios` / `flows` のいずれか、または `authorization` (principal /
policy / resource いずれかの名前) である。

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

`GlossaryEntry` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `definition` | MUST | 非空文字列。 |
| `description` | MAY | 文字列。 |
| `aliases` | MAY | 文字列配列。 |
| `context` | MAY | 文字列。 |
| `not_to_confuse_with` | MAY | `{term, reason}` 配列 (両方 MUST)。 |
| `annotations` | MAY | object。 |

### 3.2 models — domain model

`models` の各要素は `kind` を必須とする。

| kind | 必須 field (kind 以外) | 意味 |
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

`Model` の field 全体:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `kind` | MUST | `entity \| value_object \| event \| enum \| error`。 |
| `identity` | kind: entity で MUST | 非空文字列、または非空文字列配列。すべて `fields` に解決する。 |
| `fields` | kind: entity/value_object で MUST | `FieldMap` (下記)。 |
| `constraints` | MAY | 非空 CEL 文字列の配列 (`ExpressionList`)。 |
| `values` | kind: enum で MUST | 非空文字列配列。 |
| `payload` | MAY (kind: event で主に使用) | `FieldMap`。 |
| `description` | MAY | 文字列。 |
| `annotations` | MAY | object。 |

`FieldMap` の各 field (`FieldDef`) は次を持つ。

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `type` | MUST | 組み込み型名・model 名・§4 の parametric 型表記のいずれかを含む識別子。 |
| `optional` | MAY | boolean。 |
| `default` | MAY | 任意値。 |
| `constraints` | MAY | 文字列または単一 key object の配列 (§4.3)。 |
| `description` | MAY | 文字列。 |
| `annotations` | MAY | object。 |

field 単体の形式、範囲、長さは field `constraints` に置く。複数 field にまたがる規則は model
`constraints` に非空 CEL 文字列の配列として置く。model-level CEL binding はその model の
field 名である (§5)。

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
    read_only: false
    bindings:
      - kind: http
        method: PATCH
        path: /tenants/{tenant_id}
```

`Interface` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `access` | MUST | §3.3.2 参照。 |
| `input` | MAY | `InterfaceFieldMap` (下記)。 |
| `output` | MAY | `InterfaceFieldMap`。 |
| `errors` | MAY | 文字列配列。各名は `kind: error` の model に解決する。 |
| `emits` | MAY | 文字列配列。各名は `kind: event` の model に解決する。 |
| `requires` | MAY | 非空 CEL 文字列の配列 (`ExpressionList`)。 |
| `ensures` | MAY | 非空 CEL 文字列の配列。 |
| `idempotent` | MAY | boolean。 |
| `read_only` | MAY | boolean。 |
| `bindings` | MAY | §3.3.1 の `Binding` 配列。 |
| `description` | MAY | 文字列。 |
| `steps` | MAY | 文字列配列。パラメータの組み合わせが多い操作 (RFC が定める endpoint 等) を
  自然文テンプレートで例示する注釈で、機械検証の対象にはならない。 |
| `annotations` | MAY | object。 |

#### 3.3.0 InterfaceField — input/output の要素

`input`/`output` の各要素 (`InterfaceField`) は次のいずれかの形を取る。

1. `FieldDef` そのもの (単一 field)。
2. ネストした field group: `{ fields: FieldMap, description?, annotations? }`。
   HTTP body のように複数 field を一つの入力単位 (例: `body`) へまとめる場合に使う。

```yaml
input:
  body:
    fields:
      username: { type: String }
      password: { type: String, annotations: { sensitive: true } }
  return_to: { type: String, optional: true }
```

#### 3.3.1 bindings — kind 別必須 field

`Binding` は `kind` を必須とし、`kind` ごとに次の field を追加で必須・任意とする。

| kind | 必須 | 任意 |
| --- | --- | --- |
| `http` | `method`, `path` | `successful_status_codes`, `request_form` (`body \| query \| form`), `headers` |
| `grpc` | `service`, `method` | `streaming` (`unary \| client \| server \| bidi`) |
| `cli` | `command` | `args`, `flags`, `exit_codes` |
| `event` | `channel`, `direction` (`produce \| consume`) | `delivery` (`at_most_once \| at_least_once \| exactly_once`), `ordering` (`none \| per_key \| global`), `partition_key` |
| `graphql` | `operation` (`query \| mutation \| subscription`), `field` | — |
| `sdk` | `function` | — |
| `schedule` | `cron` または `every` のいずれか一方 | — |

すべての `kind` で `description` は任意。

#### 3.3.2 access — interface の公開範囲

すべての interface は `access` を明示しなければならない。

- `public`: 認証なしで外部から利用でき、policy 評価をしない。
- `internal`: 外部 binding を持たない内部操作で、policy 評価をしない (`bindings` を持つ
  `internal` interface は invalid)。
- object: protected 操作。非空 `policies` と `resource.type` / `resource.id` を必須とする。
  `resource.type` は model または `authorization.resources` に解決し、`resource.id` は非空
  CEL 文字列 (§5)、`policies` の各名は `authorization.policies` に解決しなければならない。

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

`StateMachine` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `target` | MUST | model 名に解決する非空文字列。 |
| `initial` | MUST | 非空文字列。`target` が enum 型 field を持つ場合はその値集合に解決する。 |
| `transitions` | MUST | 下表の配列。 |
| `terminal` | MAY | 非空文字列配列。 |
| `description` | MAY | 文字列。 |
| `annotations` | MAY | object。 |

`transitions[]` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `from` | MUST | 非空文字列。`terminal` に含まれる値からの transition は invalid。 |
| `event` | MUST | `kind: event` の model に解決する非空文字列。 |
| `to` | MUST | 非空文字列。 |
| `guard` | MAY | 非空 CEL 文字列。 |
| `effect` | MAY | `kind: event` の model に解決する文字列配列。 |

`terminal` に到達した state からの transition を定義してはならない。`guard` の CEL binding は
**target model の field 名を無接頭辞で束縛する唯一の位置**である (`input` も利用できる)。他の
すべての CEL 位置が `object.field` 形式で束縛するのに対し、`guard` だけは
`status == Pending` のように bare な field 名を直接参照する (§5.1 参照)。

### 3.5 authorization — principal と policy

`authorization` は singleton resource、再利用可能な principal 集合、policy を所有する
(section 自体は map の map ではなく単一 object)。

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

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `resources` | MAY | key = singleton resource 名、値は `{ description? }`。model では表さない
  authorization resource type を宣言する。 |
| `principals` | MAY | key = principal 名。`type` (model 名、MUST) と `matches` (`ExpressionList`、MUST)、
  `description` (MAY) を持つ。 |
| `policies` | MAY | key = policy 名。`effect` (`permit \| forbid`、MUST) と `principal` (`principals` の
  キーに解決する非空文字列、MUST)、`when` (非空 CEL 文字列、MAY)、`description` (MAY) を持つ。 |

複数 policy は matching permit が 1 件以上かつ matching forbid が 0 件の場合だけ allow し、
それ以外は implicit deny とする。interface 名を action 名とし、policy に action 一覧を重複記載
しない。

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

`Objective` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `indicator` | MUST | 非空 CEL 文字列 (§5)。 |
| `target` | MUST | number、`0 < target <= 1`。 |
| `window` | MUST | 非空文字列。 |
| `budgeting` | MAY | `occurrences \| timeslices`、既定 `occurrences`。 |
| `slice` | `budgeting: timeslices` のとき MUST、それ以外は指定不可 | 非空文字列。 |
| `interface` | MAY | 既知 `interfaces.*` に解決する非空文字列。 |
| `description` | MAY | 文字列。 |
| `annotations` | MAY | object。 |

metric query、alert routing、retention、TTL、security/runtime configuration は objective に
置かない。

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

`Scenario` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `actor` | MUST | `glossary` または `authorization.principals` に解決する非空文字列。 |
| `main_success` | MUST | 非空文字列配列。 |
| `given` | MAY | 非空文字列配列。 |
| `extensions` | MAY | 下表の配列。 |

`extensions[]` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `at` | MUST | 1 始まりの整数。`main_success` の長さ以下でなければならない。 |
| `condition` | MUST | 非空文字列。 |
| `steps` | MUST | 非空文字列配列。 |

`steps`、`goal`、`scope`、`level`、`primary_actor`、`success_guarantees` を scenario 直下に
置く旧形は拒否する。scenario の自然文中に現れる既知の model、interface、event、error 名は
意味 link として扱う。通常語を未解決参照とは扱わない。機械的に必ず解決すべき参照は意味 field
または `refs` に置く。

### 3.8 flows — view の内容と navigation

flow は「利用者が何を見て (`sees`)、何をするか (`does`)」を view ごとに表し、`does` の遷移先
(`to`) で view 間の navigation を表す。

```yaml
flows:
  TenantManagement:
    entry: TenantList
    views:
      TenantList:
        sees: テナント一覧画面(realm、表示名、ステータス)
        does:
          - action: edit
            does: 編集ボタンをクリックする
            to: TenantEdit
      TenantEdit:
        sees: テナント編集画面(表示名入力フォーム)
        does:
          - action: submit
            does: 表示名を入力して、保存ボタンをクリックする
            interface: UpdateTenant
            to: TenantList
          - action: documentation
            does: ドキュメントへのリンクをクリックする
            external: true
```

`Flow` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `entry` | MUST | 非空文字列。`views` のいずれかのキーであることを想定する (§3.8.1)。 |
| `views` | MUST | 非空 map。key = view 名、値は `FlowView` (下記)。 |

`FlowView` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `sees` | MUST | 非空文字列。その view で利用者が見る内容の一言要約 (画面名、表示項目、入力項目)。 |
| `does` | MAY | 非空配列。利用者がその view から取れる操作 (`FlowAction`、下記)。 |

`FlowAction` の field:

| field | 必須 | 型 / 制約 |
| --- | --- | --- |
| `action` | MUST | 非空文字列。同一 view 内で一意でなければならない。 |
| `does` | MUST | 非空文字列。利用者の操作の一言要約 (例: 「入力して、送信ボタンをクリックする」)。 |
| `interface` | MAY | 既知 `interfaces.*` に解決する非空文字列。 |
| `to` | MAY | 遷移先 view 名。省略は同一 flow 内での終了を表す。他の flow の `entry` や、
  そこから到達可能な view を指してもよく、その場合は解決検証の対象外とする
  (flow をまたぐ画面遷移を表現するため)。 |
| `external` | MAY | `true` のみ許容。SCL が定義しない外部遷移を表す。`to` と同時指定不可。 |

#### 3.8.1 到達可能性

`entry` から `does[].to` を辿って、同一 flow の `views` に宣言されたすべてのキーへ到達
できなければならない (到達できない view は invalid)。`interface` を指定した場合は
`interfaces.*` に解決しなければならない。

flow は screen 台帳、route、goal、actor、loading/error state、横断 UX requirement を持たない。
`sees`/`does` は画面/操作の一言要約であり、全量の UI 仕様ではない。アクセシビリティや
ローカライズ等の横断 UI 要件は `standards` の requirement として、`refs` で `flows.<Name>` を
参照する形で表す。

## 4 型システム

### 4.1 組み込み型

型検証は type 文字列から識別子を抽出し、次の組み込み識別子集合と `models` のキーのいずれかに
解決することだけを検査する (完全な文法解析ではない)。組み込み識別子:
`Any`、`Bool`、`Boolean`、`Bytes`、`Date`、`DateTime`、`Decimal`、`Duration`、`Float`、`Int`、
`Integer`、`JSON`、`Map`、`Number`、`Optional`、`String`、`UUID`、`URI`、`URL`、`List`、`Set`。

### 4.2 parametric 型

collection と optionality は `List<T>` / `T[]`、`Set<T>`、`Map<K, V>`、`Optional<T>` で表せる。
`List`/`Set`/`Map`/`Optional` はラッパー識別子として組み込み型集合に含まれ、`T`/`K`/`V` に
現れる識別子は同じ文書の `models` または組み込み型に解決しなければならない。

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
| `states.<name>.transitions[].guard` | target model の field 名 (無接頭辞)、`input` |
| `interfaces.<name>.requires` | `input`, `resource`, `subject`, `context` |
| `interfaces.<name>.ensures` | requires の binding、`output`, `response`, `emitted` |
| `authorization.principals.*.matches` | `principal`, `resource`, `context` |
| `authorization.policies.*.when` | `principal`, `resource`, `context` |
| `interfaces.*.access.resource.id` | `input`, `resource`, `subject`, `context` |
| `objectives.*.indicator` | `request`, `response`, `event`, `measurement` |

validator は少なくとも root binding の範囲を検証し、利用不能な root を拒否する
(`tools/yaml-check/src/scl-semantics.ts` の `validateExpressions`)。完全な型検査を提供する
validator は field / model 型も検証しなければならない。

### 5.1 root binding の実体

上表は「どの root 識別子が使えるか」だけを定める。各 root の実体は次の通り。

- **`input`**: その interface が宣言する `input` の field map そのもの。field 名がそのまま
  binding される (例: `input.tenant_id`)。
- **`output`**: その interface が宣言する `output` の field map。`ensures` でのみ利用できる。
- **`response`**: HTTP 応答そのものを指す暗黙 binding (例: `response.headers["Content-Type"]`)。
  `output` が表す値オブジェクトとは別に、応答全体 (header 等) を参照する場合に使う。
- **`resource`**: `access.resource.type` が指す model のインスタンス。`type` が
  `authorization.resources` の singleton を指す場合は `resource.id` だけが意味を持つ
  (例: `id: context.tenant_id`)。
- **`subject`**: `interfaces.requires`/`ensures`/`access.resource.id` で束縛される、操作の
  呼び出し元。実務では `subject.id` の形でのみ用いる。
- **`principal`**: `authorization.principals.*.matches`/`policies.*.when` で束縛される、
  操作の呼び出し元。`subject` と同じ「呼び出し元」を指すが束縛される節が異なる。
  `glossary` の `Subject` (存在する場合、OIDC の `sub` claim 等) とは別概念であることに注意する。
- **`context`**: 実行時の暗黙コンテキスト。正準フィールドは spec 全体の実例から次を認める:
  `authenticated`、`tenant_id`、`client_id`、`client_authenticated`、`grant_type`、
  `dpop_proof`、`code_challenge`、`login_throttled`、`token_active`、`token_client_id`、
  `token_scopes`、`transaction_cookie`、`transaction_id`、`upload.content_type`、
  `upload.magic_byte_matches_content_type`。新しい `context` field を使う場合はこの一覧に
  追記する。
- **`emitted`**: 現在の interface 呼び出しで発行済みのイベントの集合。個々の要素は CEL の
  `is` マクロを通してのみ型判定できる (例: `emitted.exists(e, e is TenantUpdated)`)。
  `ensures` でのみ利用できる。
- **`measurement`**: `objectives.*.indicator` で観測値を束縛する root。実務では
  `measurement.latency`、`measurement.status_code`、`measurement.available`、
  `measurement.requests_per_second` 等、SLI の測定値を指す。
- **`request`**/**`event`**: `objectives.*.indicator` で許可される root だが、本書時点で
  実例はない (将来の SLI 表現のための予約)。

`states.*.guard` は上記と異なり、`target` model の field 名を **`object.field` 形式ではなく
無接頭辞で** 束縛する唯一の位置である (例: `guard: input.approved && status == Pending` の
`status` は target model の `status` field)。

## 6 意味参照と検証順序

validator は次の順序で検証する。

1. `spec_version: "3.0"` を確認する。
2. JSON Schema で閉じた shape、必須性、enum、条件付き必須を検証する。
3. model type / identity、interface error / event、state target / event、authorization、objective、
   flow (view 到達可能性・action 一意性・interface 参照)、scenario、standard refs を解決する。
4. 全 interface の access coverage と位置別 CEL binding を検証する。
5. context map の参照解決・循環検出・shared_kernel サイズ警告を検証する。

SCL 3.0 以外の version と廃止 field は拒否する。shape と意味検証は別 finding として報告できるが、
いずれか一つでも error があれば文書は invalid である。

## 7 記法と保存形式

- UTF-8 YAML を正本とし、拡張子は `.yaml` とする。
- map key は同一 map 内で一意でなければならない。
- element 名は参照の identity であり、rename は破壊的変更として扱う。
- `description` と scenario step、flow の `sees`/`does` は自然文、expression と identifier は
  機械可読文字列として扱う。
- SCL に work item、ADR、commit、実装ファイルへの trace id を書かない。
- 派生 artifact は SCL 更新後に再生成し、単独開発 / integration / main では同期させる。

## 8 versioning

`spec_version` は schema 自体の version であり、製品 version ではない。現在の workspace と validator は
`"3.0"` だけを受理し、欠落、未知 version、過去 version、廃止 field を拒否する。互換 alias、自動変換、
fallback は持たない。SCL 3.0 への移行判断と旧 field の所有先は
[ADR-103](decisions/ADR-103-scl-3-localized-specification-structure.md) に、flows の
views/sees/does 記法と CEL root binding の形式定義は
[ADR-112](decisions/ADR-112-scl-flow-view-registry-and-cel-binding-reference.md) に記録する。
