# WorkloadIdentity

## 責務と境界

自律エージェントの実行環境に対するワークロードアイデンティティフェデレーションを担う。
テナントが登録した外部のアテステーションの発行者の信頼設定（`WorkloadTrustBundle`）と、外部の主体を既存の `Agent` に対応付ける `AgentWorkloadBinding` を扱う。
主なアテステーションの形式は、Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID などの OIDC 互換の JWT である。

IdMagic は SPIRE のサーバーやエージェントを同梱、運用せず、外部のアテステーションを検証する RP として動作する。
検証したアテステーションは OAuth2 の Token Exchange のグラント（RFC 8693）の subject として扱い、長期のシークレットを配布する専用の資格情報の経路は設けない。

| 扱わないもの | 担当 |
| --- | --- |
| `Agent` とその `OAuth2Client` への束縛 | `IdManagement` |
| Token Exchange のグラントとアクセストークンの発行 | `OAuth2` |
| JWT の署名の検証と JWKS の取得の部品 | 技術的な共有アダプター `backend/shared/security/tokens_jose` |

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `WorkloadTrustBundle` | 名前、トラストドメイン、発行者、JWKS の取得元またはインラインの JWKS、受理する audience、外部のトークンの最大 TTL、状態 | `Tenant` を `tenant_id` で参照する。発行者はテナントの中で一意である |
| `AgentWorkloadBinding` | 外部の主体の glob パターン、対応先の `Agent`、状態 | `WorkloadTrustBundle` の配下にあり、同じテナントの `Agent` を参照する。主体のパターンは信頼設定の中で一意である |

- **判断**：関連付けは信頼設定の配下にあるが、独立して有効と無効を切り替えるため、信頼設定の Aggregate の内側には置かない。信頼設定を削除すると、配下の関連付けも削除する。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Workload Identity` のタグが定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `WorkloadTokenVerifier` の `VerifyWorkloadToken` | `OAuth2` の Token Exchange | このモジュールが実装する。ポートは `OAuth2` が定める | `subject_token` を検証し、対応先の `Agent` と束縛先の `client_id` を `WorkloadIdentityGrant` として返す |
| `AgentRepository` の読み取り | `IdManagement` が提供する | このモジュールが使う | 対応先の `Agent` の状態と、`OAuth2Client` への束縛を読む |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `WorkloadTrustBundle…`、`AgentWorkloadBinding…`、`WorkloadAttestationRejected`。`WorkloadTokenExchanged` は `OAuth2` がトークンの発行の後に発行する |

## 機能

| 機能 | 内容 |
| --- | --- |
| [信頼設定と関連付けの管理](trust-configuration/README.md) | 信頼設定と関連付けの登録、更新、無効化と再有効化、削除、JWKS の再取得 |
| [アテステーションの交換](attestation-exchange/README.md) | 外部のアテステーションの検証と、対応先の `Agent` の一意な特定 |

| 文書 | 内容 |
| --- | --- |
| [WorkloadIdentity の用語集](glossary.md) | このモジュールでの語義 |
| [WorkloadIdentity の設計](design/README.md) | 話題ごとの設計と重要な判断 |
