# SharedSignals のアーキテクチャ

この文書は、SharedSignals の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 反応が Agent と User のイベントを受け、所有者の Agent を探す。受信が主体を Agent として解決する | 相手のドメインイベントの型と、`agent` の `ports.AgentRepository` を使う。依存の向きは、このモジュールから相手への一方向である |
| `SigningKeys` | 配送が SET に署名する | `sign_jose` が相手の `KeyStore` で署名する |
| `Tenancy` | ストリームの登録と削除が、使用量を増減する | 相手の `QuotaRepository` を使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 失効を Agent ごとの一つの時刻に集約する | 発行済みのトークンを列挙せずに、一度にすべてを無効にできる。詳細は[判断](decisions.md#失効を-agent-ごとの一つの時刻に集約する) |
| ローカルの失効を先に確定させ、外部への伝播を切り離す | 受信側の障害や遅延がローカルの失効を妨げない。詳細は[判断](decisions.md#ローカルの失効を外部への伝播より先に確定させる) |
| 受信のエンドポイントを SET の署名で認証する | 外部の送信側はブラウザーのセッションを持たない。詳細は[判断](decisions.md#受信のエンドポイントを管理-api-の認証ではなく-set-の検証で守る) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。
機能仕様に対応するコードは、要件の ID を `//spec:covers` で名指すテストからたどる。

| 層 | 責務 |
| --- | --- |
| `domain` | 失効エポック、ストリームと送信側と受信側の設定、配送、受信した SET、CAEP のイベント、ドメインイベント |
| `ports` | 各 Aggregate の Repository、SET の署名、検証、送信 |
| `usecases` | 失効の反応、ストリームの管理、受信、配送の投影と試行 |
| `verify_jose`、`sign_jose` | SET の検証と署名 |
| `push_http` | 受信側のエンドポイントへの HTTP の送信 |
| `handlers_http` | テナント単位の管理 API と、受信のエンドポイント |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 失効エポックの前進 | `IdManagement` の Agent と User のイベント | イベントを発行した操作の中で、発行の後に反応を呼ぶ。管理 API の操作は `api` で、管理 API の外から User を止める操作（ライフサイクルワークフローの `disable_user` は `worker`、SCIM の取り込みは `api`）はそれを実行するプロセスで呼ぶ。反応の組み立ては `usecases.NewAgentRevocationReactor` の一つである | [失効エポック](../revocation/README.md) |
| 配送の作成 | `AgentAccessRevoked` | 失効を進めた同じプロセスが、5 秒の期限で配送を作る。SET の `iss` には、どのプロセスでも共有の設定の `ISSUER` を使う | [SET の配送の設計](../transmitter/design.md) |
| 配送の試行 | 一定の間隔 | `worker` の配送のループ | [SET の配送の設計](../transmitter/design.md) |
| SET の受信 | 外部の送信側の `/ssf/streams/{stream_id}/events` への要求 | `api` が同期的に検証し、失効エポックを進める | [SET の受信](../receiver/README.md) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/shared-signals/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [SSF ストリームの管理](../stream/README.md) |
