# System のアーキテクチャ

この文書は、System の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

この Context は業務の Context ではなく、それらを組み立てる起動処理、経路、UI の合成を扱う。
コードは一つの `backend/system` を持たず、起動処理と経路の組み立てに分かれて置かれる。

| 場所 | 責務 |
| --- | --- |
| `backend/cmd/idmagic`、`backend/cmd/idmagic-worker`、`backend/cmd/idmagic-batch`、`backend/cmd/idmagic-seed` | プロセスの入口と起動時設定の読み込み |
| `backend/cmd/internal/bootstrap` | 起動時設定の検証、依存の組み立て、ドメインイベントの配信点、機能のレジストリ |
| `backend/shared/http/server_http` | 経路の組み立て、優先度クラスの分類、アドミッションコントロール、ゲートウェイの許可リスト |
| `backend/cmd/idmagic-route-reference` | `ROUTE_PRIORITY.md` の生成 |
| `frontend` | ホステッドの認証画面、管理コンソール、アカウントポータル、表示言語 |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 経路で認可の範囲を表す | テナント横断の能力を、画面の分岐ではなく経路で固定する。詳細は[判断](decisions.md#テナント横断の操作の入口を一つにする) |
| ポータルを IdP 自身の OIDC RP にする | 認可とトークンの発行の経路を二重化しない。詳細は[判断](decisions.md#管理コンソールとアカウントポータルを-bff-ではなく-spa-の-oidc-rp-にする) |
| 一つの API の Deployment で提供する | 詳細は[判断](decisions.md#api-のプレーンを分けない) |
| 一覧を散文に写さず、定義から生成する | 経路の優先度と起動時設定の参照文書を、コードから生成して乖離を検出する |

## 構成要素

| 機能仕様 | 主な構成要素 |
| --- | --- |
| [運用](../operations/README.md) | プローブ、運用のマニフェスト、永続化の共通のアダプター |
| [表示言語](../localization/README.md) | フロントエンドの辞書と表示言語の解決 |
| [ホステッド UI とポータル](../hosted-ui/README.md) | 認可トランザクション、ポータルの OIDC RP、デモのログインの入口 |
| [起動時設定](../startup-configuration/README.md) | 起動時設定の検証、`FeatureRegistry`、`CONFIGURATION.md` の生成 |
| [アドミッションコントロール](../admission-control/README.md) | 優先度クラスの分類、入場の上限のミドルウェア、`ROUTE_PRIORITY.md` の生成 |
| [API の境界](../api-boundary/README.md) | 経路の種類ごとの認可、非推奨のヘッダー、ゲートウェイの許可リスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 起動 | プロセスの起動 | 各プロセスが、設定を集約して検証し、失敗すれば部分的に起動せずに止まる | [起動時設定](../startup-configuration/README.md) |
| 要求の受け付け | HTTP の要求 | `api` が、ルーティングの後、ハンドラーの前に入場を判定する | [アドミッションコントロールの設計](../admission-control/design.md) |
| ドメインイベントの配信 | 各 Context の `Emit` | 発行元と同じプロセス、同じ goroutine | [判断](decisions.md#専用のイベントの基盤を持たない) |
