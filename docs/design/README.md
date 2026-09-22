# 設計文書

このディレクトリには、IdMagic が要求を実現する構造と仕組みを記載する。
製品の目的と要求は[要求文書](../requirements/README.md)を参照する。

| 文書 | 内容 |
| --- | --- |
| [アーキテクチャ](architecture/README.md) | システム境界、論理構成、実行構成、配置、全体に関わる判断 |
| [アプリケーション設計](application/README.md) | 機能、API、UI、モジュールの実装方針 |
| [データ設計](data/README.md) | データの正、整合性、保存、移行、廃棄 |
| [インフラストラクチャ設計](infrastructure/README.md) | コンピューティング、ストレージ、ネットワーク |
| [セキュリティ設計](security/README.md) | 資産、信頼境界、セキュリティ制御 |
| [信頼性設計](reliability/README.md) | 障害、冗長性、縮退、復元 |
| [性能設計](performance/README.md) | 負荷、資源、待ち行列、拡張 |
| [オブザーバビリティ設計](observability/README.md) | メトリクス、ログ、トレース、検知 |
| [検証設計](verification/README.md) | 要求を受け入れるための証拠 |

## 用語

| 用語 | 意味 | 詳細 |
| --- | --- | --- |
| デプロイ、デプロイメント | アプリケーションを実行環境へ配置し、稼働させること | [デプロイメントアーキテクチャ](architecture/deployment.md) |
| ランタイム | 稼働中のプロセス、その役割、プロセス間の接続 | [ランタイムアーキテクチャ](architecture/runtime.md) |
| プラットフォーム | コンピューティング、ストレージ、環境差、構成管理、IaC の責任範囲 | [プラットフォーム設計](infrastructure/platform.md) |
| コンピューティング | 構成要素を動かす CPU とメモリの提供元 | [プラットフォーム設計](infrastructure/platform.md#コンピューティング) |
| 構成ファイル | `infra/` の Docker Compose ファイル、Kubernetes マニフェスト、Terraform | [デプロイメントアーキテクチャ](architecture/deployment.md) |
| 共通トポロジー | デプロイ先によらず共通する実行単位と接続関係 | [デプロイメントアーキテクチャ](architecture/deployment.md#共通トポロジー) |
| デプロイプロファイル | 共通トポロジーを特定の製品とリソースで実現する構成例。実際に稼働中の環境を示すものではない | [デプロイメントアーキテクチャ](architecture/deployment.md#デプロイプロファイル) |
| シークレット | 認証情報、暗号鍵、トークンなど、漏えいを防ぐ必要があり、リポジトリへ保存しない設定値 | [シークレット設計](security/secrets.md) |
| キャパシティ | 処理能力とその算出。ストレージの量は「保存容量」と呼ぶ | [キャパシティ設計](performance/capacity.md) |
| 想定ワークロード | キャパシティ算出に用いる利用規模と負荷 | [キャパシティ設計](performance/capacity.md#想定ワークロード) |
| サイジング計算式 | レプリカ数と接続数を求める式 | [キャパシティ設計](performance/capacity.md#サイジング計算式) |
| ロードシェディング順序 | 飽和時に優先度の低い経路から受け付けを落とす順序 | [キャパシティ設計](performance/capacity.md#ロードシェディング順序) |
| アドミッションコントロール | 過負荷時にハンドラーの手前で受け付けを止める仕組み | [System の内部設計](../domain/system/internals.md#アドミッションコントロール) |
| オブザーバビリティ | メトリクス、ログ、トレースと、それらの相関 | [オブザーバビリティ設計](observability/) |
| ガイドライン | 設計時に参照する判断基準 | [API ガイドライン](application/api-guidelines.md)、[設計ガイドライン](application/design-guidelines.md) |
