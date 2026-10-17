# System

## 責務と境界

システムの入口を扱う。
ブラウザーが最初に触れる面（ホステッドの認証画面であるログイン、同意、デバイス認証、管理コンソール、アカウントポータル）と、それらを支える認可トランザクション、API の経路の分け方、表示言語の解決、起動時設定、過負荷のときの受け付けの制御、運用の資産がここに属する。

| 扱わないもの | 担当 |
| --- | --- |
| どのユーザーに何ができるか | 記録の正を持つ各モジュールと[認可設計](../../design/security/authorization.md) |
| プロダクト全体が従う外部規範 | [全体の標準仕様](../../requirements/standards.md) |
| モジュールをまたぐ語 | [共通用語集](../glossary.md) |
| 実行の単位 | [ランタイムアーキテクチャ](../../design/architecture/runtime.md) |
| 信頼境界 | [脅威モデル](../../design/security/threat-model.md) |
| 実行の手順と検証のコマンド | リポジトリの `README.md` |

このモジュールが決めるのは、どの経路へどの資格情報で到達できるかである。
業務データの認可は判断しない。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 起動 | プロセスの起動 | 各プロセスが、設定を集約して検証し、失敗すれば部分的に起動せずに止まる | [起動時設定](features/startup-configuration.md) |
| 要求の受け付け | HTTP の要求 | `api` が、ルーティングの後、ハンドラーの前に入場を判定する | [アドミッションコントロールの設計](features/admission-control.md#実現方式) |
| ドメインイベントの配信 | 各モジュールの `Emit` | 発行元と同じプロセス、同じ goroutine | [判断](decisions.md#専用のイベントの基盤を持たない) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 経路で認可の範囲を表す | テナント横断の能力を、画面の分岐ではなく経路で固定する。詳細は[判断](decisions.md#テナント横断の操作の入口を一つにする) |
| ポータルを IdP 自身の OIDC RP にする | 認可とトークンの発行の経路を二重化しない。詳細は[判断](decisions.md#管理コンソールとアカウントポータルを-bff-ではなく-spa-の-oidc-rp-にする) |
| 一つの API の Deployment で提供する | 詳細は[判断](decisions.md#api-のプレーンを分けない) |
| 一覧を散文に写さず、定義から生成する | 経路の優先度と起動時設定の参照文書を、コードから生成して乖離を検出する |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [運用](features/operations.md) | 運用の資産による SLO の検証、プローブ、クエリの期限 |
| [表示言語](features/localization.md) | 表示言語の解決、フォールバック、翻訳、API のエラーの言語 |
| [ホステッド UI とポータル](features/hosted-ui.md) | デモのログインの入口、失効したセッションからの復帰、認可トランザクション、ファーストパーティーの RP |
| [起動時設定](features/startup-configuration.md) | 起動時設定の検証、設定の参照文書、機能のレジストリ |
| [アドミッションコントロール](features/admission-control.md) | 飽和したときの優先度による拒否と、経路の優先度の参照文書 |
| [API の境界](features/api-boundary.md) | 非推奨のインターフェース、テナント横断の操作の入口、ゲートウェイの許可リスト |
| [HTTP API の共通の振る舞い](features/http-api.md) | エラーの形式、ページサイズとカーソルとページ送りのヘッダー、ボディと文字列の長さの上限、セキュリティヘッダー、レートリミットの拒否の形式 |

System の機能は機能スライスを持たず、組み立て地点と共有の部品が実装する。

| 機能 | 実装する部品 |
| --- | --- |
| [運用](features/operations.md) | プローブ、運用のマニフェスト、永続化の共通のアダプター |
| [表示言語](features/localization.md) | フロントエンドの辞書と表示言語の解決 |
| [ホステッド UI とポータル](features/hosted-ui.md) | 認可トランザクション、ポータルの OIDC RP、デモのログインの入口 |
| [起動時設定](features/startup-configuration.md) | 起動時設定の検証、`FeatureRegistry`、`CONFIGURATION.md` の生成 |
| [アドミッションコントロール](features/admission-control.md) | 優先度クラスの分類、入場の上限のミドルウェア、`ROUTE_PRIORITY.md` の生成 |
| [API の境界](features/api-boundary.md) | 経路の種類ごとの認可、非推奨のヘッダー、ゲートウェイの許可リスト |
| [HTTP API の共通の振る舞い](features/http-api.md) | `backend/shared/http/support_http`、`backend/shared/spec`、起動処理のボディ上限のミドルウェア |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [API の境界](features/api-boundary.md#概要) | API と UI の方式はシステムのアプリケーション設計が扱い、このモジュールは経路の分け方をAPI の境界で定める |
| データ設計 | [System のデータ](data.md#集約) | Aggregate とその関係。このモジュールは永続状態を保存しない |
| インフラストラクチャ設計 | [配置の設計](../../design/architecture/deployment.md#共通トポロジー) | システムのインフラストラクチャ設計と配置の設計が扱う |
| セキュリティ設計 | [ホステッド UI とポータルの設計](features/hosted-ui.md#セキュリティ)、[認可設計](../../design/security/authorization.md#主体の種類) | 認可トランザクションとポータルのトークンはホステッド UI とポータルの設計が、システム全体の認可は認可設計が扱う |
| 信頼性設計 | [運用](features/operations.md#概要) | プローブは運用が、システム全体の可用性は信頼性設計が扱う |
| 性能設計 | [アドミッションコントロールの設計](features/admission-control.md#アーキテクチャ)、[キャパシティ設計](../../design/performance/capacity.md#仮定値と実測値) | 過負荷のときの受け付けの制御はアドミッションコントロールの設計が、キャパシティの計画はキャパシティ設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [System の用語集](glossary.md) | このモジュールでの語義 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
