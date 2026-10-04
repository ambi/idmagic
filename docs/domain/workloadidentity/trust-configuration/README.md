# 信頼設定と関連付けの管理

## 概要

この文書は、テナント管理者が `WorkloadTrustBundle` と `AgentWorkloadBinding` を管理する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 信頼設定の登録、更新、無効化と再有効化、削除、JWKS の再取得。関連付けの作成、無効化と再有効化、削除 |
| 行為者 | テナント管理者 |
| 扱わないもの | 登録した信頼設定によるトークンの検証は[アテステーションの交換](../attestation-exchange/README.md)が扱う |

## モデル

信頼設定は、JWKS の取得元（`jwks_uri`）かインラインの `jwks` のどちらかを持つ。
JWKS の再取得は、保存する鍵素材を差し替えるので、変更の操作として扱う。

## 状態遷移

### WorkloadTrustBundleLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配下の関連付けを交換に使えない。再有効化すれば `enabled` に戻せる。削除はレコードと配下の関連付けを取り除く終端の操作であり、以後の操作は存在しない信頼設定として拒否する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配下の関連付けを交換に使える |
| disabled | — | 配下の関連付けを交換に使えない。再有効化できる |
| deleted | terminal | レコードと配下の関連付けを削除した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |
| enabled | WorkloadTrustBundleDeleted | — | deleted |  |
| disabled | WorkloadTrustBundleDeleted | — | deleted |  |

| State | 無効化 | 再有効化 | 削除 | トークンの交換 |
|---|---|---|---|---|
| enabled | → disabled | 何もしない | → deleted | 何もしない |
| disabled | 何もしない | → enabled | → deleted | 拒否：invalid_grant（理由 `trust_bundle_disabled`） |
| deleted | 拒否：404 workload_trust_bundle_not_found | 拒否：404 workload_trust_bundle_not_found | 拒否：404 workload_trust_bundle_not_found | 拒否：invalid_grant（理由 `unregistered_issuer`） |

### AgentWorkloadBindingLifecycle

作成時は `enabled` とする。無効化すると `disabled` に遷移し、それ以降の交換には使えない。再有効化すれば `enabled` に戻せる。削除はレコードを取り除く終端の操作である。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 交換に使える |
| disabled | — | 交換に使えない。再有効化できる |
| deleted | terminal | レコードを削除した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |
| enabled | AgentWorkloadBindingDeleted | — | deleted |  |
| disabled | AgentWorkloadBindingDeleted | — | deleted |  |

| State | 無効化 | 再有効化 | 削除 | トークンの交換 |
|---|---|---|---|---|
| enabled | → disabled | 何もしない | → deleted | 何もしない（ほかの有効な関連付けに一致しない）<br>拒否：invalid_grant（理由 `ambiguous_match`、ほかの有効な関連付けにも一致する） |
| disabled | 何もしない | → enabled | → deleted | 拒否：invalid_grant（理由 `no_binding_match`、ほかの有効な関連付けに一致しない）<br>何もしない（ほかの有効な関連付けの一つだけに一致する） |
| deleted | 拒否：404 agent_workload_binding_not_found | 拒否：404 agent_workload_binding_not_found | 拒否：404 agent_workload_binding_not_found | 拒否：invalid_grant（理由 `no_binding_match`、ほかの有効な関連付けに一致しない）<br>何もしない（ほかの有効な関連付けの一つだけに一致する） |

## 操作

### 管理者による信頼設定の管理

#### REQ-WORKLOADIDENTITY-008 管理者は信頼設定を登録・無効化・再有効化できる

- 管理者が信頼設定を登録したとき、WorkloadIdentity は、名前と発行者を前後の空白を除いて保存し、`enabled` の信頼設定を作り、201 と信頼設定を返し、`WorkloadTrustBundleConfigured` を発行する。
- 管理者が最大 TTL を指定せずに信頼設定を登録したとき、WorkloadIdentity は、最大 TTL を 3,600 秒にする。
- 管理者が信頼設定の名前、`jwks_uri`、`jwks`、受理する audience、最大 TTL を更新したとき、WorkloadIdentity は、指定した項目だけを保存し、200 と信頼設定を返し、`WorkloadTrustBundleUpdated` を発行する。
- 管理者が信頼設定を更新したとき、WorkloadIdentity は、発行者とトラストドメインを変えない。
- 管理者がどの項目も指定せずに信頼設定を更新したとき、WorkloadIdentity は、200 と信頼設定を返し、イベントを発行しない。
- 管理者が信頼設定を無効化または再有効化したとき、WorkloadIdentity は、状態を `disabled` または `enabled` にし、204 を返し、`WorkloadTrustBundleDisabled` または `WorkloadTrustBundleEnabled` を発行する。
- 信頼設定がすでにその状態の間、管理者が無効化または再有効化を要求したとき、WorkloadIdentity は、204 を返し、イベントを発行しない。
- 管理者が信頼設定を削除したとき、WorkloadIdentity は、配下の関連付けを先に消してから信頼設定を消し、204 を返し、`WorkloadTrustBundleDeleted` を発行する。
- 管理者が JWKS の再取得を要求したとき、WorkloadIdentity は、到達できたかと鍵の数を 200 で返し、到達できたかを載せた `WorkloadTrustBundleJWKSRefreshed` を発行する。
- JWKS に到達できた間、管理者が JWKS の再取得を要求したとき、WorkloadIdentity は、再取得の時刻を `jwks_cached_at` に記録する。
- JWKS に到達できない場合、WorkloadIdentity は、再取得をエラーにせず、`reachable` を `false` として返す。
- 名前のない登録または更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_name_required` で拒否する。
- 発行者のない登録を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_issuer_required` で拒否する。
- `jwks_uri` と `jwks` のどちらもない登録を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_jwks_required` で拒否する。
- 受理する audience が空の登録または更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_audiences_required` で拒否する。
- 0 以下の最大 TTL の更新を要求された場合、WorkloadIdentity は、422 と `workload_trust_bundle_invalid_ttl` で拒否する。
- 同じテナントのほかの信頼設定と同じ名前を指定された場合、WorkloadIdentity は、409 と `workload_trust_bundle_name_conflict` で拒否する。
- 同じテナントのほかの信頼設定と同じ発行者を指定された場合、WorkloadIdentity は、409 と `workload_trust_bundle_issuer_conflict` で拒否する。
- 存在しないか別のテナントの信頼設定を指定された場合、WorkloadIdentity は、404 と `workload_trust_bundle_not_found` で拒否する。
- 登録と更新を拒否した場合、WorkloadIdentity は、信頼設定を変えず、イベントを発行しない。
- **例**：EX-WORKLOADIDENTITY-008-01、EX-WORKLOADIDENTITY-008-02、EX-WORKLOADIDENTITY-008-03

#### REQ-WORKLOADIDENTITY-010 信頼設定と関連付けの管理は管理者に限られる

- `admin` のロールを持たない利用者が信頼設定または関連付けの登録、一覧、取得、更新、無効化、再有効化、削除、JWKS の再取得を要求した場合、WorkloadIdentity は、403 と `access_denied` で拒否し、信頼設定と関連付けを変えない。
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

## セキュリティ上の考慮

信頼設定と関連付けの登録、更新、無効化、削除は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
関連付けの対象にできる `Agent` も、同じテナントのものに限る。
