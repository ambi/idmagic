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

- **判断**：`Agent` を `User` と `OAuth2Client` に並ぶ第 3 のプリンシパルにする理由は、[Agent を OAuth2Client に束縛する第 3 のプリンシパルとする](../design/decisions.md#agent-を-oauth2client-に束縛する第-3-のプリンシパルとする)。
- **判断**：すべての Agent に所有者を求める。誰も責任を持たない非人間のアイデンティティを残さないためである。

## 状態遷移

### AgentLifecycle

`Active` は通常の稼働、`Disabled` は元に戻せる運用停止、`Killed` は一方向の緊急停止である。
`Active` 以外の Agent には、新しいトークンを発行しない。
`Deleted` は記録を消した後の状態であり、以後の操作は存在しない Agent と同じに扱う。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。新しいトークンを発行できる唯一の状態である |
| Disabled | — | 復元可能な運用停止 |
| Killed | terminal | 一方向の緊急停止。復元できない |
| Deleted | terminal | 記録と束縛を消した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentUpdated | — | Active |  |
| Disabled | AgentUpdated | — | Disabled |  |
| Active | AgentDisabled | — | Disabled |  |
| Disabled | AgentDisabled | — | Disabled |  |
| Disabled | AgentEnabled | owner.status == 'Active' | Active |  |
| Active | AgentEnabled | owner.status == 'Active' | Active |  |
| Active | AgentKilled | — | Killed |  |
| Disabled | AgentKilled | — | Killed |  |
| Active | AgentDeleted | — | Deleted |  |
| Disabled | AgentDeleted | — | Deleted |  |
| Active | AgentCredentialBound | — | Active |  |
| Disabled | AgentCredentialBound | — | Disabled |  |
| Active | AgentCredentialUnbound | — | Active |  |
| Disabled | AgentCredentialUnbound | — | Disabled |  |
| Killed | AgentCredentialUnbound | — | Killed |  |

| State | 更新 | 無効化 | 再有効化 | 停止 | 削除 | 資格情報の束縛 | 束縛の解除 | 所有者の停止に伴う無効化 |
|---|---|---|---|---|---|---|---|---|
| Active | → Active（値が変わる）<br>何もしない（値が変わらない） | → Disabled | → Active（所有者が Active）<br>拒否：409 agent_owner_inactive（所有者が Active でない） | → Killed | → Deleted | → Active（束縛していない）<br>何もしない（同じ Agent に束縛済み） | → Active（束縛している）<br>何もしない（束縛していない） | → Disabled |
| Disabled | → Disabled（値が変わる）<br>何もしない（値が変わらない） | → Disabled | → Active（所有者が Active）<br>拒否：409 agent_owner_inactive（所有者が Active でない） | → Killed | → Deleted | → Disabled（束縛していない）<br>何もしない（同じ Agent に束縛済み） | → Disabled（束縛している）<br>何もしない（束縛していない） | 何もしない |
| Killed | 拒否：409 agent_killed | 拒否：409 agent_killed | 拒否：409 agent_killed | 拒否：409 agent_killed | 拒否：409 agent_killed | 拒否：409 agent_killed | → Killed（束縛している）<br>何もしない（束縛していない） | 何もしない |
| Deleted | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 拒否：404 agent_not_found | 何もしない |

## 操作

### 登録

#### REQ-IDMANAGEMENT-009 Agent の登録は、指定した `kind` で Agent を作り、`kind` の欠落と未知の値を拒否する

- 管理者が区分 `kind` を指定して Agent を登録したとき、その区分で Agent を作る。
- `kind` を指定しない登録を要求された場合は、AgentKindRequiredError で拒否し、デフォルト値で補わない。
- 既知のどの値でもない `kind` の登録を要求された場合は、InvalidAgentKindError で拒否し、既知の値へ丸めない。
- **判断**：区分は、実行時にトークンを発行するかを決める（REQ-OAUTH2-050）。補ったり丸めたりすると、管理者が意図しない区分でトークンが発行される。
- **例**：EX-IDMANAGEMENT-009-01、EX-IDMANAGEMENT-009-03

#### REQ-IDMANAGEMENT-073 Agent の登録は、名前を正規化し、同じテナントの `Active` の User だけを所有者に受け付ける

- 管理者が Agent を登録したとき、`Active` の Agent を作り、テナントの Agent の使用量を一つ増やし、`AgentRegistered` を発行する。
- Agent の名前は、前後の空白を除いて保存する。
- 管理者が所有者を指定せずに登録したとき、登録した管理者を所有者とする。
- 前後の空白を除くと空になる名前を指定された場合は、422 と `agent_name_required` で拒否する。
- 同じテナントのほかの Agent と大文字と小文字を区別せずに同じ名前を指定された場合は、409 と `agent_name_conflict` で拒否する。
- 同じテナントの `Active` の User でない所有者を指定された場合は、422 と `agent_owner_not_found` で拒否する。
- 登録を拒否した場合は、Agent を作らず、テナントの Agent の使用量を変えない。

### 資格情報の束縛と解除

#### REQ-IDMANAGEMENT-074 Agent の資格情報の束縛は、同じテナントの OAuth2Client を一つの Agent にだけ結び、同じ束縛には何もしない

- 管理者が束縛したとき、Agent に同じテナントの `OAuth2Client` の資格情報を結び、`AgentCredentialBound` を発行する。
- 管理者が束縛を解除したとき、束縛を消し、`AgentCredentialUnbound` を発行する。
- 束縛する `client_id` は、前後の空白を除いて同じテナントの OAuth2Client から探す。
- 管理者が同じ Agent に束縛済みの OAuth2Client を束縛した場合、または束縛していない OAuth2Client の束縛を解除した場合は、成功を返し、イベントを発行しない。
- `Killed` の間も、束縛を解除できる。
- 空の `client_id` と、見つからない `client_id` を指定された場合は、422 と `client_not_found` で拒否する。
- 別のテナントの `client_id` を指定された場合は、存在しない `client_id` と同じ OAuth2ClientNotFoundError で拒否し、束縛を作らない。
- ほかの Agent に束縛済みの OAuth2Client を指定された場合は、409 と `agent_client_already_bound` で拒否する。
- **例**：EX-IDMANAGEMENT-074-04、EX-IDMANAGEMENT-074-05

### 更新

#### REQ-IDMANAGEMENT-075 Agent の更新は値が変わった項目だけを記録し、所有者の変更を別に記録する

- 管理者が Agent を更新したとき、値が変わった項目だけを保存し、`name`、`description`、`kind`、`owner_sub`、`roles` のうち値が変わった項目だけを `changed_fields` に載せた `AgentUpdated` を発行する。
- 管理者が所有者を変えたとき、`AgentUpdated` に続けて、変更前と変更後の所有者を載せた `AgentOwnerChanged` を発行する。
- 管理者がどの項目の値も変えない更新を要求した場合は、成功を返し、`updated_at` を進めず、イベントを発行しない。
- 前後の空白を除くと空になる所有者を指定された場合は、422 と `agent_owner_required` で拒否し、Agent を変えない。

### 無効化と再有効化

#### REQ-IDMANAGEMENT-076 Agent の無効化は `Disabled` に、再有効化は `Active` にし、すでにその状態でも記録し直す

- 管理者が Agent を無効化したとき、Agent を `Disabled` にし、`AgentDisabled` を発行する。
- 管理者が Agent を再有効化したとき、Agent を `Active` に戻し、`AgentEnabled` を発行する。再有効化した Agent は一覧に現れる。
- 管理者が `Disabled` の Agent を無効化したとき、成功を返し、`disabled_at` と `updated_at` を操作の時刻に進め、`AgentDisabled` を発行する。
- 管理者が `Active` の Agent を再有効化したとき、成功を返し、`updated_at` を進め、`AgentEnabled` を発行する。
- **例**：EX-IDMANAGEMENT-076-01、EX-IDMANAGEMENT-076-02

#### REQ-IDMANAGEMENT-082 所有者の User が `Active` でない Agent の再有効化は拒否する

- 所有者の User が同じテナントの `Active` でない Agent の再有効化を要求された場合は、409 と `agent_owner_inactive` で拒否し、Agent を変えず、イベントを発行しない。
- **判断**：所有者の停止に伴う無効化（REQ-IDMANAGEMENT-081）を、再有効化ですぐに打ち消せないようにする。所有者を `Active` に戻すか、`Active` の別の User へ所有者を変えてから再有効化する。
- **例**：EX-IDMANAGEMENT-082-01

### 所有者の停止に伴う無効化

#### REQ-IDMANAGEMENT-081 所有者の User が止まると、その User が所有する Agent を無効化する

- 所有者の User を無効化する、削除を予約する、または完全削除したとき、その User が所有する `Active` の Agent をすべて `Disabled` にし、Agent ごとに `AgentDisabled` を発行する。
- 管理 API、ライフサイクルワークフロー、SCIM の取り込みのどの経路で User を止めたときも、同じように無効化する。期限切れの削除予約の自動の完全削除も含む。
- 所有者の User を止めたとき、`Disabled` と `Killed` の Agent は変えない。
- すでに `Disabled` の User をもう一度無効化したとき、またはすでに削除予約中の User の削除をもう一度予約したとき、User は変えず、残っている `Active` の Agent を無効化する。
- 所有者の User を再有効化または復元したとき、Agent は `Disabled` のまま残す。
- **判断**：すべての Agent に所有者を求めるのは、誰も責任を持たない非人間のアイデンティティを残さないためである。所有者が組織を去った後も Agent が動き続けると、その判断が成り立たない。
- **判断**：所有者の再開で Agent を自動で再開しない。所有者の停止より前から止めていた Agent まで再開してしまうからである。
- **判断**：伝播は User の確定とは別に行う。途中で失敗したときは、同じ操作の再実行で残った Agent を回収する。
- **例**：EX-IDMANAGEMENT-081-01、EX-IDMANAGEMENT-081-02、EX-IDMANAGEMENT-081-03、EX-IDMANAGEMENT-081-04、EX-IDMANAGEMENT-081-05

### 停止

#### REQ-IDMANAGEMENT-077 停止した Agent の更新、無効化、再有効化、停止、束縛は拒否する

- 管理者が `Active` または `Disabled` の Agent を停止したとき、Agent を `Killed` にし、`AgentKilled` を発行する。
- `Killed` の間は、更新、無効化、再有効化、停止、資格情報の束縛を 409 と `agent_killed` で拒否し、Agent を変えず、イベントを発行しない。

### 削除

#### REQ-IDMANAGEMENT-078 Agent の削除は束縛ごと記録を消し、停止した Agent の削除は拒否する

- 管理者が Agent を削除したとき、Agent の記録と、その Agent の資格情報の束縛を消し、テナントの Agent の使用量を一つ減らし、`AgentDeleted` を発行する。
- `Killed` の Agent の削除を要求された場合は、409 と `agent_killed` で拒否し、記録を残す。
- 削除した Agent、存在しない Agent、別のテナントの Agent を対象とする参照、更新、無効化、再有効化、停止、削除、束縛、束縛の解除を要求された場合は、404 と `agent_not_found` で拒否する。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| トークンの発行 | `Active` 以外の Agent には新しいトークンを発行しない。判定は束縛した `OAuth2Client` のトークンの発行の境界で行い、判定に曖昧さがあれば発行しない側へ倒す |
| テナント境界 | 束縛できるのは同じテナントの `OAuth2Client` だけである。別のテナントの `client_id` は、存在しないものと区別できない応答で拒否する |
| 認可 | Agent の操作は `agents:*` のスコープで認可する。規則は[管理 API の認可](../admin-access/README.md)が定める |
