# 信頼設定と関連付けの管理

## 概要

この文書は、テナント管理者が `WorkloadTrustBundle` と `AgentWorkloadBinding` を管理する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 信頼設定と関連付けの登録、更新、無効化と再有効化、削除、JWKS の再取得、発行者情報の読み込み、パターンの試行 |
| 行為者 | テナント管理者 |
| 扱わないもの | 登録した信頼設定によるトークンの検証は[アテステーションの交換](attestation-exchange.md)が扱う |

## モデル

信頼設定は、JWKS の取得元（`jwks_uri`）かインラインの `jwks` のどちらかを持つ。
信頼設定は、検証の種別として `oidc` か `spiffe` を持つ。
`oidc` の信頼設定は発行者を必ず持ち、トラストドメインは運用者のラベルである。
`spiffe` の信頼設定はトラストドメインを SPIFFE のトラストドメインとして持ち、発行者は任意である。
JWKS の再取得は到達確認を行い、成功時に取得時刻を記録する。

## 状態遷移

### WorkloadTrustBundleLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配下の関連付けを交換に使えない。再有効化すれば `enabled` に戻せる。削除はレコードと配下の関連付けを取り除く終端の操作であり、以後の操作は存在しない信頼設定として拒否する。

<!-- spec:state-machine backend/workloadidentity/testdata/WorkloadTrustBundleLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| enabled | initial | 配下の関連付けを交換に使える |
| disabled | — | 配下の関連付けを交換に使えない。再有効化できる |
| deleted | terminal | レコードと配下の関連付けを削除した |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |
| enabled | WorkloadTrustBundleDeleted | — | deleted |  |
| disabled | WorkloadTrustBundleDeleted | — | deleted |  |

| State | 無効化 | 再有効化 | 削除 | トークンの交換 |
| --- | --- | --- | --- | --- |
| enabled | → disabled | 何もしない | → deleted | 何もしない |
| disabled | 何もしない | → enabled | → deleted | 拒否：invalid_grant（理由 `trust_bundle_disabled`） |
| deleted | 拒否：404 workload_trust_bundle_not_found | 拒否：404 workload_trust_bundle_not_found | 拒否：404 workload_trust_bundle_not_found | 拒否：invalid_grant（理由 `unregistered_issuer`） |

<!-- /spec:state-machine -->

### AgentWorkloadBindingLifecycle

作成時は `enabled` とする。無効化すると `disabled` に遷移し、それ以降の交換には使えない。再有効化すれば `enabled` に戻せる。削除はレコードを取り除く終端の操作である。

<!-- spec:state-machine backend/workloadidentity/testdata/AgentWorkloadBindingLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| enabled | initial | 交換に使える |
| disabled | — | 交換に使えない。再有効化できる |
| deleted | terminal | レコードを削除した |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| enabled | AgentWorkloadBindingUpdated | 項目を指定 | enabled | パターンまたは Agent を更新 |
| disabled | AgentWorkloadBindingUpdated | 項目を指定 | disabled | パターンまたは Agent を更新 |
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |
| enabled | AgentWorkloadBindingDeleted | — | deleted |  |
| disabled | AgentWorkloadBindingDeleted | — | deleted |  |

| State | 更新 | 無効化 | 再有効化 | 削除 | トークンの交換 |
| --- | --- | --- | --- | --- | --- |
| enabled | → enabled（項目を指定）<br>何もしない（項目を省略） | → disabled | 何もしない | → deleted | 何もしない（ほかの有効な関連付けに一致しない）<br>拒否：invalid_grant（理由 `ambiguous_match`、ほかの有効な関連付けにも一致する） |
| disabled | → disabled（項目を指定）<br>何もしない（項目を省略） | 何もしない | → enabled | → deleted | 拒否：invalid_grant（理由 `no_binding_match`、ほかの有効な関連付けに一致しない）<br>何もしない（ほかの有効な関連付けの一つだけに一致する） |
| deleted | 拒否：404 agent_workload_binding_not_found | 拒否：404 agent_workload_binding_not_found | 拒否：404 agent_workload_binding_not_found | 拒否：404 agent_workload_binding_not_found | 拒否：invalid_grant（理由 `no_binding_match`、ほかの有効な関連付けに一致しない）<br>何もしない（ほかの有効な関連付けの一つだけに一致する） |

<!-- /spec:state-machine -->

## 操作

### 管理者による信頼設定の管理

#### REQ-WORKLOADIDENTITY-008 管理者は信頼設定を登録・無効化・再有効化できる

- 管理者が信頼設定を登録したとき、WorkloadIdentity は、名前、トラストドメイン、発行者を前後の空白を除いて保存し、種別を保存し、`enabled` の信頼設定を作り、201 と信頼設定を返し、`WorkloadTrustBundleConfigured` を発行する。
- 管理者が種別を指定せずに信頼設定を登録したとき、WorkloadIdentity は、種別を `oidc` にする。
- 管理者が最大 TTL を指定せずに信頼設定を登録したとき、WorkloadIdentity は、最大 TTL を 3,600 秒にする。
- 管理者が信頼設定の名前、`jwks_uri`、`jwks`、受理する audience、最大 TTL を更新したとき、WorkloadIdentity は、指定した項目だけを保存し、200 と信頼設定を返し、`WorkloadTrustBundleUpdated` を発行する。
- 管理者が信頼設定を更新したとき、WorkloadIdentity は、種別、発行者、トラストドメインを変えない。
- 管理者がどの項目も指定せずに信頼設定を更新したとき、WorkloadIdentity は、200 と信頼設定を返し、イベントを発行しない。
- 管理者が信頼設定を無効化または再有効化したとき、WorkloadIdentity は、状態を `disabled` または `enabled` にし、204 を返し、`WorkloadTrustBundleDisabled` または `WorkloadTrustBundleEnabled` を発行する。
- 信頼設定がすでにその状態の間、管理者が無効化または再有効化を要求したとき、WorkloadIdentity は、204 を返し、イベントを発行しない。
- 管理者が信頼設定を削除したとき、WorkloadIdentity は、配下の関連付けを先に消してから信頼設定を消し、204 を返し、`WorkloadTrustBundleDeleted` を発行する。
- 管理者が JWKS の再取得を要求したとき、WorkloadIdentity は、到達できたかと、署名の検証に使う鍵（[REQ-WORKLOADIDENTITY-003](attestation-exchange.md#req-workloadidentity-003-署名が不正なアテステーションを拒否する) の表で選んだ鍵）の数を 200 で返し、到達できたかを載せた `WorkloadTrustBundleJWKSRefreshed` を発行する。
- JWKS に到達できた間、管理者が JWKS の再取得を要求したとき、WorkloadIdentity は、再取得の時刻を `jwks_cached_at` に記録する。
- JWKS に到達できない場合、WorkloadIdentity は、再取得をエラーにせず、`reachable` を `false` として返す。
- 名前のない登録または更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_name_required` で拒否する。
- `oidc` の信頼設定では、発行者のない登録を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_issuer_required` で拒否する。
- `spiffe` の信頼設定では、トラストドメインが SPIFFE のトラストドメインの構文を満たさない登録を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_invalid_trust_domain` で拒否する。
- `jwks_uri` と `jwks` のどちらもない登録を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_jwks_required` で拒否する。
- 受理する audience が空の登録または更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_audiences_required` で拒否する。
- 0 以下の最大 TTL の更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_invalid_ttl` で拒否する。
- 同じテナントのほかの信頼設定と同じ名前を指定された場合、WorkloadIdentity は、409 と `workload_trust_bundle_name_conflict` で拒否する。
- 同じテナントのほかの信頼設定と同じ発行者を指定された場合、WorkloadIdentity は、409 と `workload_trust_bundle_issuer_conflict` で拒否する。
- `spiffe` の信頼設定では、同じテナントのほかの `spiffe` の信頼設定と同じトラストドメインを指定された場合、WorkloadIdentity は、409 と `workload_trust_bundle_trust_domain_conflict` で拒否する。
- 存在しないか別のテナントの信頼設定を指定された場合、WorkloadIdentity は、404 と `workload_trust_bundle_not_found` で拒否する。
- 登録と更新を拒否した場合、WorkloadIdentity は、信頼設定を変えず、イベントを発行しない。
- **例**：EX-WORKLOADIDENTITY-008-01、EX-WORKLOADIDENTITY-008-02、EX-WORKLOADIDENTITY-008-03

#### REQ-WORKLOADIDENTITY-010 信頼設定と関連付けの管理は管理者に限られる

- `admin` のロールを持たない利用者が信頼設定または関連付けの登録、一覧、取得、更新、無効化、再有効化、削除、JWKS の再取得、主体の試行、発行者情報の読み込みを要求した場合、WorkloadIdentity は、403 と `access_denied` で拒否し、信頼設定と関連付けを変えず、外部への取得を行わない。
- **例**：EX-WORKLOADIDENTITY-010-01、EX-WORKLOADIDENTITY-010-02

### 管理者による関連付けの管理

#### REQ-WORKLOADIDENTITY-009 管理者は他テナントの Agent への関連付けを作成できない

- 管理者が信頼設定に関連付けを作成したとき、WorkloadIdentity は、主体のパターンと Agent を前後の空白を除いて保存し、`enabled` の関連付けを作り、201 と関連付けを返し、`AgentWorkloadBindingCreated` を発行する。
- 管理者が関連付けを無効化、再有効化、削除したとき、WorkloadIdentity は、204 を返し、`AgentWorkloadBindingDisabled`、`AgentWorkloadBindingEnabled`、`AgentWorkloadBindingDeleted` を発行する。
- 関連付けがすでにその状態の間、管理者が無効化または再有効化を要求したとき、WorkloadIdentity は、204 を返し、イベントを発行しない。
- 存在しないか、別のテナントに属する Agent を指定された場合、WorkloadIdentity は、422 と `agent_workload_binding_agent_not_found` で拒否し、関連付けを作らない。
- 主体のパターンが空の作成を要求された場合、WorkloadIdentity は、422 と `agent_workload_binding_pattern_required` で拒否し、関連付けを作らない。
- 同じ信頼設定に同じ主体のパターンの関連付けがある場合、WorkloadIdentity は、409 と `agent_workload_binding_pattern_conflict` で拒否し、関連付けを作らない。
- 存在しないか別のテナントの関連付けを指定された場合、WorkloadIdentity は、404 と `agent_workload_binding_not_found` で拒否する。
- **例**：EX-WORKLOADIDENTITY-009-01

### 管理者による関連付けの更新

#### REQ-WORKLOADIDENTITY-013 関連付けは削除せずにパターンと Agent を更新する

- 管理者が関連付けの主体のパターンまたは Agent を更新したとき、WorkloadIdentity は、指定した項目の前後の空白を除いて保存し、200 と更新した関連付けを返し、`AgentWorkloadBindingUpdated` にテナント、信頼設定、関連付け、更新後の Agent を載せて発行する。
- 関連付けの更新では、WorkloadIdentity は、指定しなかった項目、ID、テナント、信頼設定、状態、作成時刻、無効化時刻を保ち、更新時刻を記録する。
- 管理者が項目を指定せずに更新したとき、WorkloadIdentity は、200 と現在の関連付けを返し、更新時刻を変えず、イベントを発行しない。
- 空の主体のパターンを指定された場合、WorkloadIdentity は、422 と `agent_workload_binding_pattern_required` で拒否し、関連付けを変えず、イベントを発行しない。
- 存在しないか別テナントの Agent を指定された場合、WorkloadIdentity は、422 と `agent_workload_binding_agent_not_found` で拒否し、関連付けを変えず、イベントを発行しない。
- 同じ信頼設定のほかの関連付けと同じパターンを指定された場合、WorkloadIdentity は、409 と `agent_workload_binding_pattern_conflict` で拒否し、関連付けを変えず、イベントを発行しない。
- 不正な glob のパターンを指定された場合、WorkloadIdentity は、400 と `invalid_request` で更新を拒否し、関連付けを変えず、イベントを発行しない。
- 存在しないか別テナントの関連付けを指定された場合、WorkloadIdentity は、404 と `agent_workload_binding_not_found` で拒否し、関連付けを変えず、イベントを発行しない。

### 管理者によるパターンの試行

#### REQ-WORKLOADIDENTITY-014 主体の試行は交換と同じ照合規則で一致を返す

- 管理者が信頼設定に対して主体の例を試行したとき、WorkloadIdentity は、交換と同じ glob の照合で一致する有効な関連付けの一覧を 200 で返す。
- 管理者が入力中のパターンと主体の例を指定したとき、WorkloadIdentity は、そのパターンが主体の例に一致するかを返す。
- 管理者が入力中のパターンを指定したとき、WorkloadIdentity は、同じ信頼設定の有効な関連付けのパターンと互いを入力例として照合し、どちらかが一致した関連付けを重なりの候補として返す。
- 管理者が編集中の関連付けの ID を指定したとき、WorkloadIdentity は、その関連付けを重なり候補から除く。
- 主体もパターンも指定されないかパターンの構文が不正な場合、WorkloadIdentity は、400 と `invalid_request` で試行を拒否し、関連付けを変えない。
- 存在しないか別テナントの信頼設定を指定された場合、WorkloadIdentity は、404 と `workload_trust_bundle_not_found` で試行を拒否し、関連付けを変えない。
- 試行では、WorkloadIdentity は、信頼設定と関連付けを変えず、イベントを発行しない。
- **判断**：重なりの候補は近似であり、すべての重なりを検出しない。登録の拒否は同一パターンの重複だけとし、交換時は複数の一致を拒否する。

### 管理者による発行者情報の読み込み

#### REQ-WORKLOADIDENTITY-015 発行者情報は公開 HTTPS の取得先から検証して返す

- 管理者が発行者の URL を指定して情報を読み込んだとき、WorkloadIdentity は、末尾の slash を除いた URL に `/.well-known/openid-configuration` を追加して取得し、入力と完全一致する `issuer` と HTTPS の `jwks_uri` を 200 で返す。
- 発行者情報の取得では、WorkloadIdentity は、HTTPS と Web PKI の検証、接続時の公開 IP の確認を JWKS の取得と共通の保護で行う。
- 取得先またはリダイレクト先が HTTPS でないか内部アドレスへ解決する場合、WorkloadIdentity は、400 と `workload_issuer_metadata_invalid` で拒否し、その接続先へ要求を送らない。
- 発行者の URL に userinfo、query、fragment があるか絶対 URL でない場合、WorkloadIdentity は、400 と `workload_issuer_metadata_invalid` で拒否し、外部への取得を行わない。
- 取得が 3 秒以内に完了しないか、リダイレクトが 3 回以上か、応答が 200 以外か、本文が 1 MiB を超えるか、不正な JSON か、issuer が一致しないか、jwks_uri が欠落または不正な場合、WorkloadIdentity は、400 と `workload_issuer_metadata_invalid` で拒否し、設定を保存しない。
- 発行者情報の読み込みでは、WorkloadIdentity は、信頼設定を保存せず、イベントを発行しない。

### 管理画面での設定支援

#### REQ-WORKLOADIDENTITY-016 管理画面は設定を保存する前に試行と警告を表示する

- 管理者が信頼設定の登録画面で発行者情報を読み込んだとき、WorkloadIdentity は、検証した JWKS URI を表示し、明示的な適用で入力欄へ反映する。
- 発行者情報の読み込みが失敗した場合、WorkloadIdentity は、入力を保持してエラーを表示し、手動の設定を継続する。
- 管理者が主体の例を試行したとき、WorkloadIdentity は、一致なし、一つ、複数を区別し、一致したパターンと Agent を表示する。
- 入力中のパターンに重なりの候補がある間、WorkloadIdentity は、該当するパターンと近似の警告を表示し、保存を許す。
- 関連付けの設定画面では、WorkloadIdentity は、重なりの検出が近似であり、候補がなくても重なりを否定できないことを表示する。
- 関連付けの Agent の選択では、WorkloadIdentity は、OAuth2Client に束縛されていない Agent を区別し、交換できないことを表示する。
- 管理者が関連付けを編集したとき、WorkloadIdentity は、現在のパターンと Agent から編集を始め、保存で同じ ID の関連付けを更新し、キャンセルで未保存の入力を破棄する。

## セキュリティ上の考慮

信頼設定と関連付けの登録、更新、無効化、削除は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
関連付けの対象にできる `Agent` も、同じテナントのものに限る。

発行者情報の入力支援は、サーバーから管理者が指定した URL へ接続する。
取得先とリダイレクト先を公開 HTTPS に限り、返された issuer を入力値に束縛する。

## 実現方式

関連付けのパターンと Agent の更新は、対応する列だけを書き、無効化と再有効化は状態の列だけを書く。
更新前に読み込んだ値からほかの変更を巻き戻さず、削除された関連付けを更新で作り直さない。
パターンの重複は保存先の一意性の制約でも拒否する。
