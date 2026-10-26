# SharedSignals

## 責務と境界

Shared Signals Framework（SSF）と Continuous Access Evaluation Profile（CAEP）による、エージェントのほぼリアルタイムな失効を扱う。
SSF の送受信に必要なストリームと Agent ごとの失効エポックを所有し、IdManagement の通知と外部の SET を失効エポックの更新へ変換する。

| 扱わないもの | 担当 |
| --- | --- |
| `Agent` の強制終了、無効化、資格情報の束縛の解除 | `IdManagement` |
| アクセストークンの `issued_at` と失効エポックの比較 | `OAuth2` の `Introspect` と保護された API |
| SET の署名の鍵 | `SigningKeys` |
| ストリームの数の上限 | `Tenancy` のリソース上限 |

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 失効エポックの前進 | `IdManagement` の Agent と User のイベント | イベントを発行した操作の中で、発行の後に反応を呼ぶ。管理 API の操作は `api` で、管理 API の外から User を止める操作（ライフサイクルワークフローの `disable_user` は `worker`、SCIM の取り込みは `api`）はそれを実行するプロセスで呼ぶ。反応の組み立ては `usecases.NewAgentRevocationReactor` の一つである | [失効エポック](features/revocation.md) |
| 配送の作成 | `AgentAccessRevoked` | 失効を進めた同じプロセスが、5 秒の期限で配送を作る。SET の `iss` には、どのプロセスでも共有の設定の `ISSUER` を使う | [SET の配送の設計](features/transmitter.md#実現方式) |
| 配送の試行 | 一定の間隔 | `worker` の配送のループ | [SET の配送の設計](features/transmitter.md#実現方式) |
| SET の受信 | 外部の送信側の `/ssf/streams/{stream_id}/events` への要求 | `api` が同期的に検証し、失効エポックを進める | [SET の受信](features/receiver.md) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/shared-signals/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [SSF ストリームの管理](features/stream.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 失効を Agent ごとの一つの時刻に集約する | 発行済みのトークンを列挙せずに、一度にすべてを無効にできる。詳細は[判断](decisions.md#失効を-agent-ごとの一つの時刻に集約する) |
| ローカルの失効を先に確定させ、外部への伝播を切り離す | 受信側の障害や遅延がローカルの失効を妨げない。詳細は[判断](decisions.md#ローカルの失効を外部への伝播より先に確定させる) |
| 受信のエンドポイントを SET の署名で認証する | 外部の送信側はブラウザーのセッションを持たない。詳細は[判断](decisions.md#受信のエンドポイントを管理-api-の認証ではなく-set-の検証で守る) |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [失効エポック](features/revocation.md) | 失効エポックの前進と、イントロスペクションへの即時の反映 |
| [SSF ストリームの管理](features/stream.md) | 送信側と受信側のストリームの登録、更新、無効化と再有効化、削除 |
| [SET の受信](features/receiver.md) | 外部の送信側からの SET の検証、主体の解決、失効エポックへの反映 |
| [SET の配送](features/transmitter.md) | 外部の受信側への CAEP のイベントの配送、再試行、配信不能 |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#security-event-token-set) | 管理 API はシステムのアプリケーション設計に従い、受信のエンドポイントの形は標準仕様が定める |
| データ設計 | [SharedSignals のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | [SET の受信](features/receiver.md#セキュリティ上の考慮)、[失効エポック](features/revocation.md#セキュリティ上の考慮) | 受信の検証はSET の受信が、エポックの公開の範囲は失効エポックのセキュリティ上の考慮の節が扱う |
| 信頼性設計 | [SET の配送の設計](features/transmitter.md#信頼性) | 配送の再試行と配信不能はSET の配送の設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [SharedSignals の用語集](glossary.md) | このモジュールでの語義 |
| [SharedSignals の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
