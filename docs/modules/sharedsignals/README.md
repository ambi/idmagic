# SharedSignals

## 責務と境界

Shared Signals Framework（SSF）と Continuous Access Evaluation Profile（CAEP）による、エージェントのほぼリアルタイムな失効を扱う。
IdMagic は SSF の送信側と受信側の両方として振る舞う。

| 扱わないもの | 担当 |
| --- | --- |
| `Agent` の強制終了、無効化、資格情報の束縛の解除 | `IdManagement` |
| アクセストークンの `issued_at` と失効エポックの比較 | `OAuth2` の `Introspect` と保護された API |
| SET の署名の鍵 | `SigningKeys` |
| ストリームの数の上限 | `Tenancy` のリソース上限 |

## モデル

中心となるのは、`Agent` ごとの失効エポックである。
`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者のオフボーディング、検証した受信の Security Event Token（SET、RFC 8417）のいずれかを契機に、単調に前進する。
`OAuth2` の `Introspect` は、この値をアクセストークンの `issued_at` と比べ、即時の失効へ反映する（`LocalRevocation`）。

確定した失効は、SSF のストリームを通じて CAEP のイベントとして外部の受信側へ伝える（`EcosystemPropagation`）。
外部の送信側から受け取った検証済みのイベントも、同じ失効エポックへ反映する。

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `AgentRevocationEpoch` | `Agent` ごとの失効エポックと、最後に進めた理由 | `Agent` を参照する |
| `SsfStream` | 向き（`Transmit` または `Receive`）、状態、送信側の設定または受信側の設定 | `Tenant` を `tenant_id` で参照する |
| `SecurityEventDelivery` | 送信するストリーム、SET、状態、試行の回数、次の試行の時刻 | `SsfStream` を参照する |
| `ReceivedSecurityEvent` | 受信したストリーム、`jti`、検証の結果 | `SsfStream` を参照する |

- **判断**：失効を一つの時刻に畳む理由は、[失効を Agent ごとの一つの時刻に集約する](decisions.md#失効を-agent-ごとの一つの時刻に集約する)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Shared Signals` のタグが定める。
SET の受信のエンドポイントは、[SharedSignals の標準仕様](standards.md)が定める範囲で動く。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `IdManagement` の Agent のイベント | `IdManagement` が発行する | このモジュールが購読する | `AgentKilled`、`AgentDisabled`、資格情報の束縛の解除、所有者の無効化と削除を契機に失効エポックを進める |
| `CheckRevocationEpoch` | `OAuth2` の `Introspect` | このモジュールが提供する | Agent の失効エポックを返す |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `RevocationEpochAdvanced`、`AgentAccessRevoked`、`SecurityEventReceived`、`SecurityEventRejected`、`SecurityEventTransmitted`、`SecurityEventDelivery…`、`SsfStream…` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [失効エポック](features/revocation.md) | 失効エポックの前進と、イントロスペクションへの即時の反映 |
| [SSF ストリームの管理](features/stream.md) | 送信側と受信側のストリームの登録、更新、無効化と再有効化、削除 |
| [SET の受信](features/receiver.md) | 外部の送信側からの SET の検証、主体の解決、失効エポックへの反映 |
| [SET の配送](features/transmitter.md) | 外部の受信側への CAEP のイベントの配送、再試行、配信不能 |

## 文書

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 委譲 | [標準仕様](standards.md#security-event-token-set) | 管理 API はシステムのアプリケーション設計に従い、受信のエンドポイントの形は標準仕様が定める |
| データ設計 | 委譲 | [モデル](#モデル) | Aggregate とその関係は仕様のモデルが、テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | 委譲 | [SET の受信](features/receiver.md#セキュリティ上の考慮)、[失効エポック](features/revocation.md#セキュリティ上の考慮) | 受信の検証はSET の受信が、エポックの公開の範囲は失効エポックのセキュリティ上の考慮の節が扱う |
| 信頼性設計 | 委譲 | [SET の配送の設計](features/transmitter.md#信頼性) | 配送の再試行と配信不能はSET の配送の設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [SharedSignals の用語集](glossary.md) | このモジュールでの語義 |
| [SharedSignals の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
