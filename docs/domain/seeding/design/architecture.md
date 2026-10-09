# Seeding のアーキテクチャ

この文書は、Seeding の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールのコードは、ほかのモジュールに依存しない。
ほかのモジュールへの書き込みは、起動処理（`backend/cmd/internal/bootstrap`）が実装する `Contributor` が行う。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `OAuth2` | ファーストパーティーのクライアントとデモのクライアントを作る | `Contributor` が相手のクライアントの Repository を呼ぶ |
| `IdManagement` | デモと合成データの User と Group を作る | `Contributor` が相手の Repository を呼ぶ |
| `Authentication` | デモの User のパスワードの履歴と MFA の要素を作る | `Contributor` が相手の Repository を呼ぶ |
| `Application`、`SAML`、`WS-Federation` | デモのアプリケーションと信頼先を作る | `Contributor` が相手の Repository を呼ぶ |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 権限ではなく、実行環境と環境ポリシーで書き込みを制限する | テナントをまたぐ書き込みができる唯一の経路を、管理 API に載せない。詳細は[判断](decisions.md#seed-の計画と適用に-http-のエンドポイントを設けない) |
| プレビューと適用で一つの計画器を共有する | プレビューだけが通り、適用で落ちる差を作らない |
| 論理キーと ID を決定的に導き、進捗を保存しない | 再実行だけで部分的な失敗から収束する。詳細は[判断](decisions.md#進捗テーブルを使わず決定的な論理キーで冪等にする) |

## 構成要素

コードは機能スライスを持たない。

| 層 | 責務 |
| --- | --- |
| `domain` | `Request`、`Manifest`、`Plan` と、環境ポリシーとマニフェストの検証 |
| `usecases` | マニフェストの検証とシークレットの解決、計画と適用の手順、プロセスの中の排他 |
| `manifests_yaml` | マニフェストの厳密な読み込みと `include` の解決、`env` と `file` のシークレットの解決 |
| `backend/cmd/idmagic-seed` | コマンドの引数と設定から要求を組み立てる |
| `backend/cmd/internal/bootstrap` の `Contributor` | リソースの種類ごとの計画と、各モジュールの Repository への書き込み |

すべての要件は[seed の計画と適用](../seed-run/README.md)に属する。

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| プレビュー | 運用者が `idmagic-seed` を `dry_run` で起動する | コマンドのプロセスが計画だけを作り、書き込まない | [seed の計画と適用の設計](../seed-run/design.md) |
| 適用 | 運用者が `idmagic-seed` を `apply` で起動する | コマンドのプロセスが計画、適用、収束の確認を行う | [seed の計画と適用の設計](../seed-run/design.md) |
