# OAuth2 のアーキテクチャ

この文書は、OAuth2 の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `Authentication` | 認可が、ログインセッションと認証の強度を読む | 相手のセッションの解決を使う |
| `Application` | 認可が、割り当てとサインインポリシーを評価する | 相手の評価を呼ぶ |
| `ClaimMapping` | ID トークンと UserInfo が、クレームを組み立てる | 相手の `IssueClaimsWithFloor` を呼ぶ |
| `SigningKeys` | トークンの署名と JWKS | 相手の `KeyStore` を使う |
| `SharedSignals` | イントロスペクションが、Agent の失効エポックを読む | 相手の `CheckRevocationEpoch` を呼ぶ |
| `IdManagement` | Token Exchange と承認が、Agent とその束縛を読む | 相手の Agent の Repository を使う |
| `Jobs` | バックチャネルのログアウトの配信 | 相手の永続キューを使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 曖昧さがなく列挙できる、プロトコルの上で重要な振る舞いは、仕様で一度だけ宣言し、ユースケースやアダプターで作り直さない | 状態遷移、認可の規則、Discovery Metadata、デバイスフローの遷移が、実装のたびにずれない。詳細は[判断](decisions.md#認可とデバイスのライフサイクルを宣言的な遷移表で表す) |
| 認可の判断を AuthZEN 型のポートに通す | 詳細は[判断](decisions.md#ポリシーの境界をまたぐ認可の判断を-authzen-型のポートに通す) |
| 委譲は権限を広げない | Token Exchange は元の権限の部分集合だけを渡す。詳細は[判断](decisions.md#ユーザーの代理の行為を-token-exchange-による委任で表す) |

## 構成要素

コードは、7 つの機能スライスと、モジュールの直下の互換のファサードからなる。
モジュールの直下の `domain`、`ports`、`usecases` は機能スライスの上に置く互換のファサードであり、起動の組み立ては `module.go` だけである。

| 機能仕様 | 機能スライス |
| --- | --- |
| [クライアント](../client/README.md) | `client` |
| [認可](../authorization/README.md) | `authorization` |
| [同意](../consent/README.md) | `consent` |
| [デバイス認可](../device/README.md) | `device` |
| [トークン](../token/README.md) | `token` |
| [承認リクエスト](../approval/README.md) | `approval` |
| [ログアウト](../logout/README.md) | `logout` |
| [プロトコルのエンドポイント](../protocol-endpoints/README.md) | 機能スライスはなく、ルートの `handlers_http` が Discovery と流量の制限を扱う |
| [管理 API の認可](../admin-access/README.md) | 機能スライスはなく、ルートの AuthZEN の規則表と `policy_tenancy` が扱う |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 認可 | ブラウザーの `/authorize` | `api` が、セッション、割り当て、ポリシー、同意を確かめて認可コードを発行する | [認可の設計](../authorization/design.md) |
| トークン | クライアントの `/token`、`/introspect`、`/revoke`、`/userinfo` | `api` が、クライアントを認証してから同期的に処理する | [トークンの設計](../token/design.md) |
| 承認 | クライアントのバックチャネル認可要求と、本人の承認の画面 | `api` が、判断とポーリングをストアで直列化する | [承認リクエストの設計](../approval/design.md) |
| バックチャネルのログアウト | セッションの失効 | `worker` が配信のジョブを実行する | [ログアウトの設計](../logout/design.md) |
