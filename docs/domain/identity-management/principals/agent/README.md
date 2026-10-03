# エージェント

## 概要

この文書は、非人間のプリンシパル `Agent` を、管理者が登録し、資格情報を束縛し、停止し、削除する機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `Agent` の登録、所有者、`OAuth2Client` への資格情報の束縛、更新、無効化と再有効化、停止（キル）、削除 |
| 行為者 | 管理者 |
| コードの機能スライス | `backend/idmanagement/agent` |
| 扱わないもの | Agent がトークン交換の actor として振る舞う委譲と、Agent のトークンの発行の可否は `OAuth2` が扱う。資格情報そのもの（クライアントの秘密情報と鍵）は `OAuth2Client` が持つ |

## モデル

`Agent` は、テナントに属する Aggregate root である。
自身の資格情報を持たず、`AgentCredentialBinding` で既存の `OAuth2Client` に束縛してトークンを得る。

| 項目 | 内容 |
| --- | --- |
| `id` | URL セーフなスラッグ |
| `name`、`description` | 名前と説明。名前はテナントの中で大文字と小文字を区別せずに一意である |
| `kind` | `autonomous`（人の都度の承認なしに動く）または `supervised`（人の監督の下で動く） |
| `owner_sub` | 所有者の User |
| `roles` | ロール |
| `status` | `Active`、`Disabled`、`Killed` |
| 束縛 | 束縛した `OAuth2Client` の `client_id` の集合。一つの `OAuth2Client` は一つの Agent にだけ束縛できる |

- **判断**：`Agent` を `User` と `OAuth2Client` に並ぶ第 3 のプリンシパルにする理由は、[Agent を OAuth2Client に束縛する第 3 のプリンシパルとする](../../design/decisions.md#agent-を-oauth2client-に束縛する第-3-のプリンシパルとする)。
- **判断**：すべての Agent に所有者を求める。誰も責任を持たない非人間のアイデンティティを残さないためである。

## 状態遷移

### AgentLifecycle

`Active` は通常の稼働、`Disabled` は元に戻せる運用停止、`Killed` は一方向の緊急停止である。
`Active` 以外の Agent には、新しいトークンを発行しない。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。新しいトークンを発行できる唯一の状態である |
| Disabled | — | 復元可能な運用停止 |
| Killed | terminal | 一方向の緊急停止。復元できない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentDisabled | — | Disabled |  |
| Disabled | AgentEnabled | — | Active |  |
| Active | AgentKilled | — | Killed |  |
| Disabled | AgentKilled | — | Killed |  |

## 操作

### 登録

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 名前、説明、区分、任意の所有者、ロール |
| 成功時の作用 | `Active` の Agent を作り、テナントの Agent の使用量を一つ増やし、`AgentRegistered` を発行する |
| 拒否 | 区分の欠落と未知の区分、空の名前、同名の Agent、`Active` でない所有者。どの拒否も Agent を作らず、使用量を変えない |

#### REQ-IDMANAGEMENT-009 管理者はエージェントを登録しクライアント資格情報をバインドできる

- 管理者は、区分 `kind` を指定して Agent を登録でき、Agent はその区分で登録される。
- 管理者は、Agent に同じテナントの `OAuth2Client` の資格情報を束縛できる。
- 管理者は、Agent を無効化し、再有効化できる。再有効化した Agent は一覧に現れる。
- `kind` を指定しない登録は AgentKindRequiredError で拒否し、デフォルト値で補わない。
- 既知のどの値でもない `kind` の登録は InvalidAgentKindError で拒否し、既知の値へ丸めない。
- 別のテナントの `client_id` の束縛は、存在しない `client_id` と同じ OAuth2ClientNotFoundError で拒否し、束縛を作らない。
- **判断**：区分は、実行時にトークンを発行するかを決める（REQ-OAUTH2-050）。補ったり丸めたりすると、管理者が意図しない区分でトークンが発行される。
- **担保手段**：`usecases.RegisterAgent`、`usecases.BindCredential`、`usecases.SetAgentDisabled`
- **例**：EX-IDMANAGEMENT-009-01、EX-IDMANAGEMENT-009-04

#### REQ-IDMANAGEMENT-073 Agent の登録の名前と所有者

- 名前は前後の空白を除いて保存する。空白を除いて空になる名前は、422 と `agent_name_required` で拒否する。
- 同じテナントのほかの Agent と大文字と小文字を区別せずに同じ名前は、409 と `agent_name_conflict` で拒否する。
- 所有者を指定しない登録は、登録した管理者を所有者とする。
- 所有者は、同じテナントの `Active` の User でなければならない。それ以外の所有者は、422 と `agent_owner_not_found` で拒否する。
- 拒否した登録は Agent を作らず、テナントの Agent の使用量を変えない。
- **担保手段**：`usecases.RegisterAgent`

### 資格情報の束縛と解除

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | `client_id` |
| 成功時の作用 | 束縛を作るか外し、`AgentCredentialBound` または `AgentCredentialUnbound` を発行する |
| 拒否 | 見つからない `client_id`、ほかの Agent に束縛済みの `OAuth2Client`、`Killed` の Agent への束縛 |
| 冪等性 | 同じ束縛の繰り返しと、束縛していない `OAuth2Client` の解除は、成功を返しイベントを発行しない |

#### REQ-IDMANAGEMENT-074 Agent の資格情報の束縛は一つの Agent に限り、同じ束縛には何もしない

- 束縛する `client_id` は、前後の空白を除いて同じテナントの OAuth2Client から探す。空の値と見つからない値は、422 と `client_not_found` で拒否する。
- ほかの Agent に束縛済みの OAuth2Client の束縛は、409 と `agent_client_already_bound` で拒否する。
- 同じ Agent に束縛済みの OAuth2Client の束縛と、束縛していない OAuth2Client の解除は成功を返し、イベントを発行しない。
- 束縛の解除は、`Killed` の Agent にもできる。
- **担保手段**：`usecases.BindCredential`、`usecases.UnbindCredential`

### 更新

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 名前、説明、区分、所有者、ロール |
| 成功時の作用 | 値が変わった項目だけを保存し、`AgentUpdated` を発行する。所有者を変えたときは、続けて `AgentOwnerChanged` を発行する |
| 拒否 | `Killed` の Agent（409 `agent_killed`） |
| 冪等性 | 同じ値での更新は、`updated_at` を進めず、イベントを発行しない |

#### REQ-IDMANAGEMENT-075 Agent の更新は値が変わった項目だけを記録し、所有者の変更を別に記録する

- `AgentUpdated` の `changed_fields` には、`name`、`description`、`kind`、`owner_sub`、`roles` のうち値が変わった項目だけを載せる。
- どの項目の値も変わらない更新は成功を返し、`updated_at` を進めず、イベントを発行しない。
- 所有者を変える更新は、`AgentUpdated` に続けて、変更前と変更後の所有者を載せた `AgentOwnerChanged` を発行する。
- **担保手段**：`usecases.UpdateAgent`

### 無効化と再有効化

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 成功時の作用 | `Disabled` または `Active` にし、`AgentDisabled` または `AgentEnabled` を発行する |
| 拒否 | `Killed` の Agent（409 `agent_killed`）。所有者の User が `Active` でない Agent の再有効化（409 `agent_owner_inactive`） |
| 冪等性 | なし。すでにその状態でも時刻を進め、イベントを発行する |

#### REQ-IDMANAGEMENT-076 Agent の無効化と再有効化は、すでにその状態でも記録し直す

- `Disabled` の Agent の無効化は成功を返し、`disabled_at` と `updated_at` を操作の時刻に進め、`AgentDisabled` を発行する。
- `Active` の Agent の再有効化は成功を返し、`updated_at` を進め、`AgentEnabled` を発行する。
- **担保手段**：`usecases.SetAgentDisabled`
- **要判断**：User の無効化と再有効化は、すでにその状態なら何もしない。Agent では時刻を進めてイベントを重ねて発行するため、`disabled_at` は最初に止めた時刻を示さない。User と同じにするかを決める。

#### REQ-IDMANAGEMENT-082 所有者の User が止まっている Agent は再有効化できない

- 所有者の User が同じテナントの `Active` でない Agent の再有効化は、409 と `agent_owner_inactive` で拒否し、Agent を変えず、イベントを発行しない。
- **判断**：所有者の停止に伴う無効化（REQ-IDMANAGEMENT-081）を、再有効化ですぐに打ち消せないようにする。所有者を `Active` に戻すか、`Active` の別の User へ所有者を変えてから再有効化する。
- **担保手段**：`usecases.SetAgentDisabled`
- **例**：EX-IDMANAGEMENT-082-01

### 所有者の停止に伴う無効化

| 項目 | 内容 |
| --- | --- |
| 行為者 | 所有者の User を止めた操作（管理 API、ライフサイクルワークフロー、SCIM の取り込み） |
| 入力 | 止めた User |
| 成功時の作用 | その User が所有する `Active` の Agent を `Disabled` にし、Agent ごとに `AgentDisabled` を発行する |
| 冪等性 | `Disabled` と `Killed` の Agent は変えない。止まっている User への再実行は、残っている `Active` の Agent だけを無効化する |

#### REQ-IDMANAGEMENT-081 所有者の User が止まると、その User が所有する Agent を無効化する

- 所有者の User を無効化する、削除を予約する、完全削除すると、その User が所有する `Active` の Agent をすべて `Disabled` にし、Agent ごとに `AgentDisabled` を発行する。
- 管理 API、ライフサイクルワークフロー、SCIM の取り込みのどの経路で User を止めても、同じように無効化する。期限切れの削除予約の自動の完全削除も含む。
- `Disabled` と `Killed` の Agent は変えない。
- すでに `Disabled` の User をもう一度無効化する、またはすでに削除予約中の User の削除をもう一度予約すると、User は変えず、残っている `Active` の Agent を無効化する。
- 所有者の User を再有効化または復元しても、Agent は `Disabled` のまま残る。
- **判断**：すべての Agent に所有者を求めるのは、誰も責任を持たない非人間のアイデンティティを残さないためである。所有者が組織を去った後も Agent が動き続けると、その判断が成り立たない。
- **判断**：所有者の再開で Agent を自動で再開しない。所有者の停止より前から止めていた Agent まで再開してしまうからである。
- **担保手段**：`usecases.DisableAgentsOwnedBy`、`usecases.SetUserDisabled`、`usecases.SoftDeleteUser`、`usecases.DeleteUser`
- **判断**：伝播は User の確定とは別に行う。途中で失敗したときは、同じ操作の再実行で残った Agent を回収する。
- **例**：EX-IDMANAGEMENT-081-01、EX-IDMANAGEMENT-081-02、EX-IDMANAGEMENT-081-03、EX-IDMANAGEMENT-081-04、EX-IDMANAGEMENT-081-05

### 停止

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 成功時の作用 | `Killed` にし、`AgentKilled` を発行する。以後、新しいトークンを発行しない |
| 拒否 | `Killed` の Agent の停止と、`Killed` の Agent への更新、無効化、再有効化、束縛（409 `agent_killed`）。Agent を変えず、イベントを発行しない |

#### REQ-IDMANAGEMENT-077 停止した Agent は変更できない

- `Killed` の Agent の更新、無効化、再有効化、停止、資格情報の束縛は、409 と `agent_killed` で拒否し、Agent を変えず、イベントを発行しない。
- **担保手段**：`usecases.UpdateAgent`、`usecases.SetAgentDisabled`、`usecases.KillAgent`、`usecases.BindCredential`

### 削除

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 成功時の作用 | Agent の記録と束縛を消し、テナントの Agent の使用量を一つ減らし、`AgentDeleted` を発行する |
| 拒否 | `Killed` の Agent（409 `agent_killed`）。記録を残す |

#### REQ-IDMANAGEMENT-078 Agent の削除は束縛ごと記録を消し、停止した Agent は削除できない

- 削除は Agent の記録と、その Agent の資格情報の束縛を消す。
- 削除は、テナントの Agent の使用量を一つ減らし、`AgentDeleted` を発行する。
- `Killed` の Agent の削除は、409 と `agent_killed` で拒否し、記録を残す。
- **担保手段**：`usecases.DeleteAgent`
- **要判断**：停止した Agent は削除できないため、緊急停止した Agent の記録と使用量は残り続ける。停止した Agent の削除を許すかを決める。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| トークンの発行 | `Active` 以外の Agent には新しいトークンを発行しない。判定は束縛した `OAuth2Client` のトークンの発行の境界で行い、判定に曖昧さがあれば発行しない側へ倒す |
| テナント境界 | 束縛できるのは同じテナントの `OAuth2Client` だけである。別のテナントの `client_id` は、存在しないものと区別できない応答で拒否する |
| 認可 | Agent の操作は `agents:*` のスコープで認可する。規則は[管理 API の認可](../../common/admin-access/README.md)が定める |
