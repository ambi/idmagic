# Agent のガードレール

## 概要

この文書は、管理者が Agent のトークンの発行に課す上限を、バージョン付きのポリシーとして宣言し、発行のたびに評価する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | ポリシーの保存とバージョン、発行の評価、超過時の拒否と承認への降格、評価の監査 |
| 行為者 | テナント管理者、Agent に束縛したクライアント |
| 扱わないもの | 実行時の操作の評価は[リソースアクセスの評価](resource-access.md)が、Agent のアイデンティティと停止は [IdManagement](../../identity-management/README.md) が扱う。発行済みのトークンは取り消さない |

## モデル

ポリシーはテナントのデフォルトと Agent 別の二層である。
未設定の項目は制限せず、ポリシーのない Agent の発行は変わらない。
`allowed_resources` を持つ層では、`aud` を指定しない発行も超過とする。許可した資源に収まると示せないからである。

| 項目 | 意味 | 超過の扱い |
| --- | --- | --- |
| `allowed_resources` | トークンの `aud` と ID-JAG の `resource` に許す資源 | `deny` か `require_approval` |
| `allowed_authorization_detail_types` | 許す `authorization_details` の種類 | `deny` か `require_approval` |
| `issuance_rate` | 窓（`window_seconds`）あたりの発行の回数（`max_issuances`） | `deny` か `require_approval` |
| `expires_at` | この時刻以降の発行を超過とする | `deny` か `require_approval` |
| `max_delegation_depth` | `act` の入れ子の上限 | `deny` のみ |
| `max_token_ttl_seconds` | アクセストークンの有効期間の上限 | 拒否せず短くする |
| `require_approval` | `true` なら常に承認を求める | 承認を求める |

超過の扱いは `on_violation` に種別ごとに宣言し、省略は `deny` である。

## 操作

### 発行の評価

#### REQ-OAUTH2-073 Agent が関わるトークンの発行は、グラントによらず関わるすべての Agent のポリシーで評価される

- 要求したクライアントに束縛した Agent があるか、発行するトークンの `agent_id` か `agent_chain` に Agent があるトークンを発行するとき、OAuth2 は、グラントの種類によらず、そのすべての Agent についてテナントのデフォルトと Agent 別のポリシーを評価する。
- 要求したクライアントに束縛した Agent があり、ID-JAG を発行するとき、OAuth2 は、同じ評価を署名の前に行う。
- 関わる Agent のない発行では、OAuth2 は、ポリシーを読まない。
- **判断**：評価する経路を限ると、残りのグラントが上限の回避に使える。送信者制約と同じく、全グラントが通る発行器で評価する。
- **例**：EX-OAUTH2-073-01、EX-OAUTH2-073-02

#### REQ-OAUTH2-074 ガードレールを超える発行は、宣言した扱いで拒否される

- どちらかの層の `deny` の項目を超える発行の要求を受けた場合、OAuth2 は、400 と `access_denied` で拒否し、トークンを発行しない。
- どちらの層の `deny` の項目も超えず、`require_approval` の項目を超えるか `require_approval` が `true` の発行の要求を受けた場合、OAuth2 は、REQ-OAUTH2-076 に従う。
- ポリシーか発行の計数を読めない場合、OAuth2 は、400 と `access_denied` で拒否し、トークンを発行しない。
- 発行を評価するとき、OAuth2 は、`issuance_rate` を、ほかの項目で拒否しない発行だけについて層ごとに数える。
- 実効の `max_token_ttl_seconds` か `expires_at` までの時間がデフォルトの有効期間より短いとき、OAuth2 は、短い方をトークンの `exp` と応答の `expires_in` に用いる。
- ガードレールで拒否するとき、OAuth2 は、応答に超えた項目とバージョンを含めない。
- **判断**：障害時に上限なしへ倒さない。理由を応答で示すと Agent が上限を探れるので、理由は監査にだけ残す。
- **例**：EX-OAUTH2-074-01、EX-OAUTH2-074-02、EX-OAUTH2-074-03、EX-OAUTH2-074-04

#### REQ-OAUTH2-075 ガードレールで拒否するリフレッシュはリフレッシュトークンを消費しない

- ガードレールを超えるリフレッシュの要求を受けた場合、OAuth2 は、ローテーションせずに拒否し、提示されたリフレッシュトークンを有効なまま残す。
- ガードレールを超える認可コード、デバイスコード、`auth_req_id` の交換を受けた場合、OAuth2 は、それらを消費した状態で拒否し、クライアントにフローのやり直しを求める。
- **判断**：ローテーションの後に拒否すると、再試行が再利用と判定されてファミリーごと失効する。一回限りの資格情報は、消費しても権限が広がらないので安全側の結果として受け入れる。
- **例**：EX-OAUTH2-075-01

#### REQ-OAUTH2-076 承認へ降格した上限は、Agent の所有者の承認だけが超えられる

- 降格した項目を超える発行が承認からの発行でない場合、OAuth2 は、400 と `unauthorized_client` で拒否し、トークンを発行しない。
- 主体が Agent の `Active` な所有者である CIBA の承認から発行するとき、OAuth2 は、降格した項目の超過を理由に拒否しない。
- 主体が Agent の所有者でない承認から、降格した項目を超えて発行する要求を受けた場合、OAuth2 は、400 と `access_denied` で拒否し、トークンを発行しない。
- **判断**：現在の CIBA の承認者は Agent が `login_hint` で選ぶので、誰の承認でも上限を超えられると統制にならない。上限に触れない代理の承認は正当なので、CIBA の開始では拒否しない。
- **例**：EX-OAUTH2-076-01、EX-OAUTH2-076-02

### 記録

#### REQ-OAUTH2-078 ガードレールの評価はバージョンと理由とともに監査に残る

- ポリシーのある Agent の発行を評価したとき、OAuth2 は、Agent、クライアント、グラントの種類、結果（`allowed`、`approved`、`approval_required`、`denied`）、理由、評価したバージョン、応答の `X-Request-ID` と同じ `requestId` を載せた `AgentGuardrailEvaluated` を発行する。
- 評価の後にトークンを発行したとき、OAuth2 は、`AgentGuardrailEvaluated` に `tokenId` を含める。
- 評価を記録するとき、OAuth2 は、`AgentGuardrailEvaluated` に `authorization_details` の値を含めない。
- ポリシーを保存か削除したとき、OAuth2 は、変更後の内容とバージョンを載せた `AgentGuardrailPolicyChanged` を発行する。
- **例**：EX-OAUTH2-078-01

### 管理

#### REQ-OAUTH2-077 ガードレールのポリシーの更新は読んだバージョンに対してだけ成立する

- 管理者がテナントのデフォルトか同じテナントの Agent のポリシーを、現在のバージョンと等しい `expected_version` で保存したとき、OAuth2 は、ポリシーを保存し、内容から導いた `version` とともに 200 で返す。
- ポリシーのない対象へ空文字列の `expected_version` で保存したとき、OAuth2 は、ポリシーを作成し、200 で返す。
- `expected_version` が現在のバージョンと異なる保存を受けた場合、OAuth2 は、409 と `agent_guardrail_conflict` で拒否し、保存しない。
- 検証を通らないポリシーを受けた場合、OAuth2 は、400 と `invalid_agent_guardrail` で拒否し、保存しない。
- 存在しないか別のテナントの Agent のポリシーの取得か保存を受けた場合、OAuth2 は、404 と `agent_not_found` で拒否する。
- 管理者が Agent のポリシーを取得したとき、OAuth2 は、Agent 別のポリシーとテナントのデフォルトを返す。
- 管理者がポリシーを削除したとき、OAuth2 は、204 を返す。
- ポリシーを更新したとき、OAuth2 は、発行済みのトークンを変えず、次の発行から新しいバージョンで評価する。
- Agent を削除したとき、OAuth2 は、その Agent のポリシーを削除する。
- 管理者の API アクセストークンで要求されたとき、OAuth2 は、`agents:read` で参照を、`agents:write` で保存と削除を許可する。
- **判断**：バージョンを内容から導くと、管理者の上げ忘れで監査が偽にならない。緊急の停止はポリシーではなく Agent の無効化とキルが担う。
- **例**：EX-OAUTH2-077-01、EX-OAUTH2-077-02

## 実現方式

### アーキテクチャ

判定はドメインの計算で、発行の計数だけをユースケースが編成する。
`/token` はグラントの分岐より前に、その要求に限った発行器を組み立て、`SignAccessToken` で評価する。
リフレッシュはローテーションの前に同じ評価を行い、発行器はその結果を再利用して二重に数えない。
層ごとに判定して結果を合成するので、値を合成する規則を持たない。

### データ

`agent_guardrail_policies` を OAuth2 が所有する。
テナントのデフォルトは `agent_id` のない行である。
計数は `shared/ratelimit` の固定窓に、要求ごとの閾値を渡して数える。
窓の変更は現在の窓が切れてから効く。
