# WorkloadIdentity のデータ

この文書は、WorkloadIdentity の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `WorkloadTrustBundle` | 名前、トラストドメイン、発行者、JWKS の取得元またはインラインの JWKS、受理する audience、外部のトークンの最大 TTL、状態 | `Tenant` を `tenant_id` で参照する。発行者はテナントの中で一意である |
| `AgentWorkloadBinding` | 外部の主体の glob パターン、対応先の `Agent`、状態 | `WorkloadTrustBundle` の配下にあり、同じテナントの `Agent` を参照する。主体のパターンは信頼設定の中で一意である |

- **判断**：関連付けは信頼設定の配下にあるが、独立して有効と無効を切り替えるため、信頼設定の Aggregate の内側には置かない。信頼設定を削除すると、配下の関連付けも削除する。
