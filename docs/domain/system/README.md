# System

## 責務と境界

システムの入口を扱う。
ブラウザーが最初に触れる面（ホステッドの認証画面であるログイン、同意、デバイス認証、管理コンソール、アカウントポータル）と、それらを支える認可トランザクション、API の経路の分け方、表示言語の解決、起動時設定、過負荷のときの受け付けの制御、運用の資産がここに属する。

| 扱わないもの | 担当 |
| --- | --- |
| どのユーザーに何ができるか | 記録の正を持つ各 Context と[認可設計](../../design/security/authorization.md) |
| プロダクト全体が従う外部規範 | [全体の標準仕様](../standards.md) |
| Context をまたぐ語 | [用語集](../glossary.md) |
| 実行の単位 | [ランタイムアーキテクチャ](../../design/architecture/runtime.md) |
| 信頼境界 | [脅威モデル](../../design/security/threat-model.md) |
| 実行の手順と検証のコマンド | リポジトリの `README.md` |

この Context が決めるのは、どの経路へどの資格情報で到達できるかである。
業務データの認可は判断しない。

## モデル

この Context は Aggregate を定義しない。
起動と経路の組み立てだけを担い、記録の正はすべて各 Context に残る。

| 概念 | 内容 |
| --- | --- |
| 経路の種類 | ブラウザー向けの認証 API `/api/auth/*`、管理 API `/api/admin/*`、セルフサービス API `/api/account/*`、各標準が定めるパスの OAuth と OIDC のプロトコルのエンドポイント |
| 優先度クラス | `interactive_auth`、`management`、`management_bulk`、`infrastructure` |
| `FeatureRegistry` | 起動処理が静的に組み立てる、実行時に選べる機能の唯一の一覧 |
| 表示言語 | `ja` と `en`。フォールバックは `en` |

- **判断**：管理とアカウントのポータルの固定の `client_id` を含む、内部で生成する ID のカラムは、`TEXT` ではなく `UUID` 型とする。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `System` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| ドメインイベントの配信点 | すべての Context のユースケースの `Emit` | この Context が提供する | 発行と同じプロセスで、監査の記録とアカウントのセキュリティ通知へ写す。判断は[専用のイベントの基盤を持たない](design/decisions.md#専用のイベントの基盤を持たない) |
| 経路の優先度クラスの分類 | すべての Context の経路の登録 | この Context が定める | 経路の登録と同じ場所でクラスを宣言し、`ROUTE_PRIORITY.md` を生成する |
| 起動時設定とその参照文書 | すべての Context の起動時設定 | この Context が定める | 設定の定義から検証と `CONFIGURATION.md` を導く |

## 機能

| 機能 | 内容 |
| --- | --- |
| [運用](operations/README.md) | 運用の資産による SLO の検証、プローブ、クエリの期限 |
| [表示言語](localization/README.md) | 表示言語の解決、フォールバック、翻訳、API のエラーの言語 |
| [ホステッド UI とポータル](hosted-ui/README.md) | デモのログインの入口、失効したセッションからの復帰、認可トランザクション、ファーストパーティーの RP |
| [起動時設定](startup-configuration/README.md) | 起動時設定の検証、設定の参照文書、機能のレジストリ |
| [アドミッションコントロール](admission-control/README.md) | 飽和したときの優先度による拒否と、経路の優先度の参照文書 |
| [API の境界](api-boundary/README.md) | 非推奨のインターフェース、テナント横断の操作の入口、ゲートウェイの許可リスト |

| 文書 | 内容 |
| --- | --- |
| [System の用語集](glossary.md) | この Context での語義 |
| [System の設計](design/README.md) | 話題ごとの設計と重要な判断 |
