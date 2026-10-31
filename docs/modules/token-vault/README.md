# TokenVault

## 責務と境界

利用者が外部サービスへ与えた OAuth のトークンを暗号化して保管し、更新し、代行トークンを持つ Agent の要求を評価して、Vault が代理で外部 API を呼ぶ。
Agent に上流のトークンを渡す経路は持たない。

| 扱わないもの | 担当 |
| --- | --- |
| ログインのための上流 IdP との連携 | `Authentication` |
| IdMagic が発行するトークンと代行の付与、失効の判定 | `OAuth2` |
| Agent のライフサイクル | `IdManagement` |
| DEK のライフサイクルと項目の暗号化 | `DataKeys` |

OAuth2 のイントロスペクションの結果を、利用側の `CallerToken` へ変換して読む。トークンの検証と失効の規則は OAuth2 が所有する。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| コネクタの管理 | 管理 API への要求 | `api` | [コネクタの管理](features/connector.md) |
| 連携と解除 | アカウントポータルからの要求と上流からのコールバック | `api` | [接続](features/connection.md) |
| 代理実行と更新 | Agent からの要求 | `api` が同期的に上流を呼ぶ | [代理実行](features/execution.md) |
| 再暗号化 | `data_key_reencryption` のジョブ | `worker` | [接続](features/connection.md#req-tokenvault-003-保管するシークレットはレコードに束縛して暗号化する) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| Vault が代理で実行し、トークンを渡さない | 呼び出しのたびに失効と規則を評価でき、トークンが外へ出ない。詳細は[判断](decisions.md#vault-が代理で実行する) |
| コネクタと接続を別の Aggregate にする | 管理者が定義する設定と、本人の資格情報は寿命も更新者も異なる |
| 更新の単一飛行を行のリースで実現する | 回転する更新トークンの競合を全プロセスで防ぐ。詳細は[判断](decisions.md#更新の単一飛行を行のリースで実現する) |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [コネクタの管理](features/connector.md) | 管理者によるコネクタの作成、更新、無効化、削除 |
| [接続](features/connection.md) | 本人による連携、一覧、解除と、保管の暗号化 |
| [代理実行](features/execution.md) | 認可の評価、外向きの要求の制限、更新と再認可 |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [TokenVault のデータ](data.md#集約) | Aggregate とその関係 |
| セキュリティ設計 | [代理実行](features/execution.md#セキュリティ上の考慮) | 認可の論理積と外向きの制限 |

| 文書 | 内容 |
| --- | --- |
| [TokenVault の用語集](glossary.md) | このモジュールでの語義 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 受容したリスク |
