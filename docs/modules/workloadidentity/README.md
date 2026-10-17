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

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 管理 API | 解決済みのテナントの `/api/admin/v1/workload-identity/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [信頼設定と関連付けの管理](features/trust-configuration.md) |
| アテステーションの交換 | `OAuth2` のトークンエンドポイントへの Token Exchange の要求 | `api` の `OAuth2` のユースケースが、このモジュールの検証を同期的に呼ぶ | [アテステーションの交換の設計](features/attestation-exchange.md#実現方式) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 外部のアテステーションを検証する RP に徹する | SPIRE などの基盤を同梱せずに、既存の実行環境の資格情報を使える。詳細は[判断](decisions.md#外部のアテステーションを検証する-rp-に徹し長期のシークレットを配らない) |
| 交換の結果を、登録した関連付けだけで決める | トークンの内容で対応先が変わらない |
| 複数の一致を拒否する | 関連付けの追加が既存のワークロードの対応先を変えない。詳細は[判断](decisions.md#複数の関連付けに一致した主体は優先順位で選ばず拒否する) |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [信頼設定と関連付けの管理](features/trust-configuration.md) | 信頼設定と関連付けの登録、更新、無効化と再有効化、削除、JWKS の再取得 |
| [アテステーションの交換](features/attestation-exchange.md) | 外部のアテステーションの検証と、対応先の `Agent` の一意な特定 |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [WorkloadIdentity のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | [アテステーションの交換](features/attestation-exchange.md#セキュリティ上の考慮) | 交換の経路が管理者の権限を通らないことと、テナントの境界はアテステーションの交換のセキュリティ上の考慮の節が扱う |
| 信頼性設計 | [アテステーションの交換の設計](features/attestation-exchange.md#信頼性) | JWKS の取得の失敗の扱いはアテステーションの交換の設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [WorkloadIdentity の用語集](glossary.md) | このモジュールでの語義 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
