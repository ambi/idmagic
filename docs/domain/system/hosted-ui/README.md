# ホステッド UI とポータル

## 概要

この文書は、ホステッドの認証画面、管理コンソール、アカウントポータルの入口の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | デモのログインの入口の表示、失効したセッションからの同じ画面への復帰、認可トランザクションの保持、ファーストパーティーの OIDC RP としてのポータル |
| 行為者 | EndUser、Administrator、Operator |
| 扱わないもの | 認証の手段そのものは `Authentication` が、トークンの発行は `OAuth2` が扱う |

## モデル

`DemoLoginAffordance` は、HomePage に表示する、ローカルのデモの資格情報による `authorization_code` の流れへの近道である。
資格情報は Seeding の `development` のプロファイルが作る。
Vite の開発サーバーではデフォルトで表示し、それ以外のビルドでは起動時設定 `VITE_DEMO_LOGIN_ENABLED=true` の場合だけ表示する。
表示の時に、`development` のプロファイルの適用の状態は検査しない。

管理コンソール（`/admin/*`）とアカウントポータル（`/account/*`）は、IdP 自身の OIDC RP である。
管理用の `…0022` とアカウント用の `…0023` という固定の UUID の `client_id` を持つ、ファーストパーティーのパブリッククライアントとして登録する。
リソースの所有者が IdP 自身のユーザーなので、同意の画面は省く。

- **判断**：ポータルを BFF の背後に置かない理由は、[管理コンソールとアカウントポータルを BFF ではなく SPA の OIDC RP にする](../design/decisions.md#管理コンソールとアカウントポータルを-bff-ではなく-spa-の-oidc-rp-にする)。

## 操作

### 運用者によるデモのログインの入口の有効化

#### REQ-SYSTEM-006 起動時設定により Vite 開発サーバー以外でも DemoLoginAffordance が表示される

- Vite の開発サーバー以外のビルドでは、`VITE_DEMO_LOGIN_ENABLED=true` の間、利用者が HomePage を開いたとき、System は、`DemoLoginAffordance` を表示する。
- Vite の開発サーバー以外のビルドでは、`VITE_DEMO_LOGIN_ENABLED` が未設定か `true` 以外の間、利用者が HomePage を開いたとき、System は、`DemoLoginAffordance` を表示しない。
- 利用者が `DemoLoginAffordance` を使ったとき、System は、`development` のプロファイルが作ったデモの資格情報で `authorization_code` の流れを進め、プロファイルの適用の状態を表示の時に検査しない。
- `development` のプロファイルを適用していない場合、System は、デモの資格情報がないので認可を失敗させる。
- **例**：EX-SYSTEM-006-01、EX-SYSTEM-006-02、EX-SYSTEM-006-03

### 利用者による開発サーバーでのデモのログイン

#### REQ-SYSTEM-007 Vite 開発サーバーでの実行時は設定なしで DemoLoginAffordance が表示される

- Vite の開発サーバーでは、利用者が HomePage を開いたとき、System は、`VITE_DEMO_LOGIN_ENABLED` によらず `DemoLoginAffordance` を表示する。
- **例**：EX-SYSTEM-007-01

### 管理者によるポータルの再読み込み

#### REQ-SYSTEM-015 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する

- 管理コンソールまたはアカウントポータルの API が 401 を返したとき、System は、保持していたアクセストークン、リフレッシュトークン、OIDC のコールバックの `state` を捨て、直前の画面への同一オリジンの相対の `return_to` を保って再認可を 1 回だけ始め、再ログインの後に元の画面へ戻す。
- 再認可から復旧できない場合、System は、再ログインの導線を表示する。
- **例**：EX-SYSTEM-015-01、EX-SYSTEM-015-02

## セキュリティ上の考慮

ブラウザーの認証の流れ（`/api/auth/*`）は、認可トランザクションの Cookie で解決する。
トランザクションの内容は常にサーバーの側に保持し、HTML、URL、JavaScript から読めるアプリケーションの状態には含めない。

ファーストパーティーのセッションのログイン（`POST /api/auth/login`）を、緊急の経路として残す。
OIDC のクライアントや鍵の設定を壊すと、直す手段そのものが失われるからである。
