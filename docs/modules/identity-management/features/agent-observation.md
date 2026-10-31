# エージェントの観測

## 概要

この文書は、外部の発見器が見つけたエージェントの観測を取り込み、管理者が管理済みの `Agent` と突き合わせる機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `AgentObservation` の取り込み、候補の計算、一致の確認、除外、再開、鮮度と保持 |
| 行為者 | 取り込み元（API アクセストークンの提示者）、管理者 |
| コードの機能スライス | `backend/idmanagement/agent` |
| 扱わないもの | SaaS ごとのクローラー、端末の走査、ネットワークの傍受、外部アカウントの停止は扱わない。すべての Agent を発見できることは保証しない |

## モデル

`AgentObservation` は、外部の主体を見たという未確認の報告である。
`Agent` の第二の正本ではなく、観測も確認も `Agent`、資格情報の束縛、ロールを変えない。

| 項目 | 内容 |
| --- | --- |
| `id` | 観測の識別子 |
| `issuer`、`subject` | 外部の発行者と主体。組がテナントの中で一意である。前後の空白を除き、大文字と小文字を区別して比較する |
| `external_registration_id` | 外部の登録 ID。任意 |
| `display_name` | 表示名。任意。主体の照合には使わない |
| `resources` | 利用する資源の名前 |
| `responsible_candidate` | 責任者の候補。任意。所有者には反映しない |
| `observed_at` | 取り込み元が観測した時刻 |
| `source_token_id` | 最新の属性を報告した API アクセストークンの ID |
| 決定 | 未決定、確認（`confirmed_agent_id`）、除外のどれか |

一覧が返す状態は、決定と候補から次の表で決まる。

| `state` | 条件 |
| --- | --- |
| `unregistered` | 未決定で、候補がない |
| `candidate` | 未決定で、候補が一つ以上ある |
| `confirmed` | 管理者が Agent との一致を確認した |
| `excluded` | 管理者が除外した |

古さは状態とは別の `stale` で示す。

## 操作

### 取り込み

#### REQ-IDMANAGEMENT-095 観測の取り込みは、専用のスコープの API アクセストークンだけを受け付ける

- `agent-observations:import` のスコープの API アクセストークンで取り込みを要求されたとき、IdManagement は、要求先のテナントへ観測を保存し、そのトークンの ID を観測元として記録する。
- API アクセストークンでない認証で取り込みを要求された場合、IdManagement は、AccessDeniedError で拒否し、観測を保存しない。
- `agent-observations:import` を持たないトークンで取り込みを要求された場合、IdManagement は、InsufficientScopeError で拒否し、観測を保存しない。
- `agent-observations:import` だけを持つトークンで観測の一覧、確認、除外、再開を要求された場合、IdManagement は、InsufficientScopeError で拒否する。
- **判断**：取り込みと確認の権限を包含させない。発見器の資格情報が漏れても、偽の観測を一致として確定できないようにするためである。
- **判断**：観測元は要求の本文で指定させない。本文の自己申告では出所を偽装できる。

#### REQ-IDMANAGEMENT-096 観測の取り込みは、件数と長さの上限の内にあるバッチを全件保存するか一件も保存しない

- 取り込みを受理したとき、IdManagement は、新しく作った件数、更新した件数、変更しなかった件数を返す。
- 取り込みで観測を作るか更新したとき、IdManagement は、件数を載せた `AgentObservationsImported` を一件発行する。
- 取り込みで何も変えなかったとき、IdManagement は、イベントを発行しない。
- 項目が 1 から 100 の範囲にないバッチを要求された場合、IdManagement は、422 と `agent_observation_batch_invalid` で拒否し、一件も保存しない。
- `issuer` か `subject` が空であるか、512 文字を超える項目を要求された場合、IdManagement は、422 と `agent_observation_invalid` で拒否し、一件も保存しない。
- `external_registration_id`、`display_name`、`responsible_candidate`、各資源の名前のどれかが 256 文字を超えるか、資源が 20 件を超える項目を要求された場合、IdManagement は、422 と `agent_observation_invalid` で拒否し、一件も保存しない。
- `observed_at` が欠けているか、受信の時刻より 5 分を超えて未来である項目を要求された場合、IdManagement は、422 と `agent_observation_invalid` で拒否し、一件も保存しない。
- 保存するとテナントの観測が 10,000 件を超える場合、IdManagement は、409 と `agent_observation_limit_exceeded` で拒否し、一件も保存しない。
- **判断**：未来の時刻を拒否するのは、未来の時刻を一度受け入れると、以後の正しい観測が古いものとして無視されるからである。
- **判断**：一部だけ保存しない。発見器が部分的な失敗を見落とすと、未確認の観測が欠けた台帳を完全なものとして読ませてしまう。

#### REQ-IDMANAGEMENT-097 再送と順序が逆転した観測は、保存した値と決定を変えない

- 保存値より `observed_at` が新しい観測を受けたとき、IdManagement は、観測の属性と観測元を更新する。
- 保存値と `observed_at` が同じか古い観測を受けたとき、IdManagement は、観測を変えず、変更しなかった件数に数える。
- 確認または除外した観測をより新しい時刻で再び受けたとき、IdManagement は、属性だけを更新し、決定を変えない。
- 一つのバッチに同じ `issuer` と `subject` の項目が複数あるとき、IdManagement は、順に適用し、`observed_at` が最も新しい項目の属性を残す。

### 突き合わせ

#### REQ-IDMANAGEMENT-098 候補は登録済みの対応だけから計算し、候補の数によらず一致を自動で確定しない

- 管理者が観測を一覧したとき、IdManagement は、各観測の状態、候補の Agent と根拠、発見元、最終の観測時刻、古さを返す。
- 観測の `issuer` を持つ信頼設定の `Enabled` の Workload binding の `subject_pattern` が `subject` に一致するとき、IdManagement は、その Agent を根拠 `workload_binding` の候補にする。
- 観測の `external_registration_id` と同じ `client_id` のクライアントが Agent に束縛されているとき、IdManagement は、その Agent を根拠 `client_binding` の候補にする。
- 複数の Agent が候補になったとき、IdManagement は、候補をすべて返し、未確認のままにする。
- 決定で絞り込む一覧を要求されたとき、IdManagement は、その決定の観測だけを返す。
- 未知の決定で絞り込む一覧を要求された場合、IdManagement は、400 と `invalid_request` で拒否する。
- 候補の材料のどれかを読めない場合、IdManagement は、候補を欠いた一覧を返さずに要求を失敗させる。
- **判断**：名前と表示名は根拠にしない。同名の別の主体を結ぶと、偽の観測から所有者の責任が移るからである。
- **判断**：候補が一つでも確定しない。観測元は未確認の情報であり、一意な一致も偽造できるからである。

#### REQ-IDMANAGEMENT-099 一致の確認は未決定の観測を同じテナントの Agent に結び、先の決定を上書きしない

- 管理者が未決定の観測の確認を要求したとき、IdManagement は、指定した Agent との一致を確定し、`AgentObservationConfirmed` を発行する。
- 管理者が候補でない Agent を指定して確認したとき、IdManagement は、一致を確定し、候補でなかったことをイベントに記録する。
- 同じ Agent にすでに確認した観測の確認を要求されたとき、IdManagement は、何もせず、確認済みの観測を返す。
- 別の Agent に確認した観測か、除外した観測の確認を要求された場合、IdManagement は、409 と `agent_observation_already_decided` で拒否し、決定を変えない。
- 同じテナントに存在しない観測を指定された場合、IdManagement は、404 と `agent_observation_not_found` で拒否する。
- 同じテナントに存在しない Agent を指定された場合、IdManagement は、404 と `agent_not_found` で拒否し、決定を変えない。
- 確認した Agent が削除されたとき、IdManagement は、その観測を未決定へ戻す。
- **判断**：`Killed` の Agent への確認も受け付ける。止めたはずの Agent が外部で動いている事実は記録する価値があるからである。

#### REQ-IDMANAGEMENT-100 除外と再開は決定だけを変える

- 管理者が未決定の観測の除外を要求したとき、IdManagement は、観測を除外し、`AgentObservationExcluded` を発行する。
- すでに除外した観測の除外を要求されたとき、IdManagement は、何もしない。
- 確認した観測の除外を要求された場合、IdManagement は、409 と `agent_observation_already_decided` で拒否する。
- 管理者が確認または除外した観測の再開を要求したとき、IdManagement は、観測を未決定へ戻し、`AgentObservationReopened` を発行する。
- 未決定の観測の再開を要求されたとき、IdManagement は、何もしない。
- 同じテナントに存在しない観測を指定された場合、IdManagement は、404 と `agent_observation_not_found` で拒否する。

#### REQ-IDMANAGEMENT-101 観測と決定は権限を付与しない

- 観測を取り込み、確認し、除外し、再開したとき、IdManagement は、Agent の登録、状態、所有者、ロール、資格情報の束縛と Workload binding を変えない。
- 観測にどの Agent も対応しないとき、IdManagement は、Agent を作らない。
- **判断**：観測を管理対象にするには、管理者が既存の登録の操作で有効な所有者を持つ Agent を登録してから確認する。所有者の検証を取り込みの経路に複製しないためである。
- **判断**：責任者の候補を所有者へ反映しない。外部の情報から所有者の責任を移すと、未確認の報告で権限と責任が動くからである。

### 鮮度と保持

#### REQ-IDMANAGEMENT-102 古い観測を区別し、確認していない観測は保持期限で消す

- 最終の観測時刻から 30 日を過ぎた観測を一覧するとき、IdManagement は、`stale` を真にする。
- 保持期限の削除が動いたとき、IdManagement は、最終の観測時刻から 180 日を過ぎた未決定と除外の観測を消す。
- 保持期限の削除が動いたとき、IdManagement は、確認した観測を消さない。
- **判断**：観測は外部システムの稼働状況を示すので、確認していないものを無期限に持たない。除外も同じ期限で消え、再び報告されれば未決定として現れる。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 権限の分離 | 取り込みは `agent-observations:import`、一覧は `agents:read`、確認、除外、再開は `agents:write` で認可する |
| 偽の観測 | 観測は未確認の情報として表示し、認可とトークンの発行はこの表を読まない |
| テナント境界 | トークンのテナントへだけ書く。別のテナントの観測と Agent は存在しないものと同じ応答で拒否する |
| 最小の属性 | 秘密情報の欄を持たず、モデルの実行内容を受けない。監査に属性の値を載せない |

## 実現方式

この機能の設計のうち、ほかの文書が扱う内容は次のとおりである。

- 設計判断：規則の判断の欄と、モジュールの[重要な設計判断](../decisions.md)に置く
- データ設計：モジュールの[データ](../data.md)に従う
- リスク：外部の発見器の網羅性には依存せず、見つからない主体は台帳に現れない

### アーキテクチャ

| 構成要素 | 責務 | 依存先 |
| --- | --- | --- |
| 取り込みのユースケース | 全件の検証、上限、順序の判定を保存先へ渡す | `AgentObservationRepository` |
| 候補の計算 | 根拠を読み、ドメインの計算で候補と状態を決める。状態を持たない | `AgentRepository`、`WorkloadSubjectMatcher`（組み立ての地点が `WorkloadIdentity` の公開操作を渡す） |
| 確認、除外、再開のユースケース | 条件付きの更新で決定を確定し、イベントを発行する | `AgentObservationRepository`、`AgentRepository` |
| 保持期限の削除 | Batch の `retention-sweep` から期限を過ぎた観測を消す | `AgentObservationRepository` |

取り込みはテナントの助言ロックの下で件数を確かめて保存する。
確認済みの状態は `agents` への外部キー（`ON DELETE SET NULL`）で表すので、Agent の削除と同じ確定で解除される。

### 信頼性

| 失敗の状況 | 結果 | 回収の方法 |
| --- | --- | --- |
| 取り込みの途中で保存に失敗する | トランザクションが戻り、一件も保存しない。取り込み元はエラーを受け取る | 同じバッチを再送する |
| 保存の後でイベントの発行に失敗する | 観測は残り、要求はエラーを返す | 同じバッチの再送は変更なしになる。件数のイベントは欠けるが、観測そのものは一覧で読める |
