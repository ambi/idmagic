# Frontend

## 責務と境界

ブラウザーの画面と、それを配信して公開 API へ中継するゲートウェイを扱う。
表示言語、認可トランザクションの表示、管理コンソールとアカウントポータルの OIDC RP、入力とエラーの表示を担当する。
実装は `frontend/` にあり、サーバーとは公開 HTTP 契約で接続する。
業務状態を保存する Go パッケージとデータベースを共有しない。

| 扱わないもの | 担当 |
| --- | --- |
| サーバーの構築と経路の結線、共通の HTTP 応答 | [System](../system/README.md) |
| 認可トランザクションの保持と Cookie からの解決 | [API の境界](../system/features/api-boundary.md#認可トランザクション) |
| 本人確認とログインセッション | [Authentication](../authentication/README.md) |
| OAuth の認可とトークン発行 | [OAuth2](../oauth2/README.md) |
| 業務状態の更新と許可、拒否の判断 | 公開 API を提供する各所有モジュールと[認可設計](../../design/security/authorization.md) |

SPA とゲートウェイはサーバーと同じリリースで提供する。
別イメージの構築とブラウザーのキャッシュによるバージョンのずれがあっても、任意のバージョンの組み合わせをサポートする独立したリリース系列は保証しない。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 認証画面の表示 | ログインまたは同意へ遷移する | SPA が認証 API から表示用データを取得する | [ホステッド UI とポータル](features/hosted-ui.md) |
| ポータルの利用 | 管理者または本人が画面を開く | SPA が OIDC RP として認可を受け、Bearer トークンで所有者の API を呼ぶ | [ホステッド UI とポータル](features/hosted-ui.md) |
| 表示言語の解決 | 画面を開く、または言語を選ぶ | ブラウザー内で辞書と書式を選び、既知のエラーを翻訳する | [表示言語](features/localization.md) |
| HTTP の中継 | API とプロトコルの経路への要求 | ゲートウェイが許可した経路を API へ中継する | [API の境界](../system/features/api-boundary.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| ポータルを IdP 自身の OIDC RP にする | 認可とトークン発行の経路を再利用する。[設計判断](decisions.md#管理コンソールとアカウントポータルを-bff-ではなく-spa-の-oidc-rp-にする)を参照する |
| 画面の単位で機能を分ける | 一つの画面が複数の所有者の API を利用する。サーバーの内部構造への対応は求めない |
| テナントと制御面を経路で分ける | 表示の切り替えと API の拒否を [REQ-SYSTEM-020](../system/features/api-boundary.md#req-system-020-テナント横断の操作はシステム経路とシステムコンソールだけに置く)に従わせる |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [表示言語](features/localization.md) | 表示言語の解決、フォールバックとエラーの翻訳 |
| [ホステッド UI とポータル](features/hosted-ui.md) | デモの入口、失効セッションからの復帰、認可トランザクションの表示と OIDC RP |

画面は `frontend/src/features/`、HTTP の変換は `frontend/src/api/`、辞書とロケールの解決は `frontend/src/lib/`、中継設定は `frontend/Caddyfile` と `frontend/vite.config.ts` にある。
System から配置を移した要件と例は、既存の `REQ-SYSTEM-*` と `EX-SYSTEM-*` を維持する。
混在する画面と API の経路の保証、および中継の照合は、System の [API の境界](../system/features/api-boundary.md)を一次情報とする。

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [フロントエンド設計](../../design/application/frontend.md#全体構成) | 画面の構成、状態、ルーティング、ビルドと配信 |
| セキュリティ設計 | [ホステッド UI とポータル](features/hosted-ui.md#セキュリティ) | ブラウザー内のポータルのトークン |

| 文書 | 内容 |
| --- | --- |
| [Frontend の用語集](glossary.md) | 表示言語とデモの入口の語義 |
| [設計判断](decisions.md) | OIDC RP としてポータルを提供する判断 |
