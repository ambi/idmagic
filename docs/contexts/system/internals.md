# System Internals

## Feature registry and resolution

`FeatureDefinition` は `FeatureID`、`FeatureVersion`、`FeatureMaturity`、`DefaultEnablement`、依存する `FeatureID`、`UpdatePolicy`、任意の `SpecificationReference` を持つ不変値である。composition root が静的な `FeatureRegistry` を渡し、設定ファイルやデータベースから機能の定義を増やさない。registry の検証は識別子と未版名の重複、存在しない依存、循環、不正な既定有効化をすべて集約して返す。

`ResolveFeatures(registry, explicitEnable, explicitDisable)` は時刻、乱数、永続化へ依存しない決定的な計算である。明示指定を既定値へ重ね、有効な機能の依存閉包を求める。存在しない識別子、同じ機能の有効化と無効化の併記、明示的に無効化した依存を必要とする選択は、すべての設定検証と同じく副作用のある初期化前に集約して拒否する。

解決結果は、有効な `FeatureDefinition`、起動警告、版付きの運用メタデータを一度に返す。設定リファレンスと `/health` はこの registry と解決結果から導出し、手書きの機能一覧を持たない。警告と運用メタデータは識別子、版、成熟度、更新方針だけを含み、環境変数の生値を含まない。

## Authorization transaction

OAuth の認可要求は、その内容を丸ごとサーバー側に保持する。ブラウザーへ渡すのは、短命な内部 UUID を載せたトランザクション Cookie (`HttpOnly`、`SameSite=Lax`、HTTPS では `Secure`) だけである。リダイレクト URI、PKCE 値、スコープ、クライアント識別子は、HTML にも URL にも JavaScript から読める状態にも現れない。これが、ログイン画面と同意画面を描く JavaScript を差し替えられても、認可要求そのものは書き換えられないことの根拠である。

SPA が `GET /api/auth/transaction` で取得できるのは、画面の種類、クライアント名、要求されたスコープといった表示用のデータに限る。ログインと同意のコマンドは、SPA が送る値ではなく Cookie から解決したトランザクションを正とする。同意では、現在のログインセッションの subject がトランザクションの subject と一致することを確かめる。

認可リクエストは 10 分で期限切れとなり、完了したリクエストは再利用できない。UI API のレスポンスには `Cache-Control: no-store` を付け、資格情報も内部のリクエスト ID も返さない。

## API boundary

入口は 4 つに分かれ、それぞれ別の認可を通る。ブラウザー向け認証 API は `/api/auth/*`、管理 API は `/api/admin/*`、セルフサービス API は `/api/account/*` に置き、OAuth / OIDC のプロトコルエンドポイントは各標準が定めるパスを保つ。管理 API とセルフサービス API の認可は、ログイントランザクション API とは独立している。ログインの途中であることが管理操作の権限に影響しない、という分離をこの配置が持つ。

## Admin console and account portal as OIDC RPs

管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) は IdP 自身の OIDC RP である。IdP の `/authorize` と `/token` に対する `authorization_code` + PKCE で認証し、管理用の `…0022` とアカウント用の `…0023` という固定 UUID の `client_id` を持つファーストパーティーのパブリッククライアントとして登録する。リソース所有者が IdP 自身のユーザーなので、同意画面は省略する。

純粋な SPA RP なので、アクセストークンはブラウザーの `sessionStorage` に保持し、`Authorization: Bearer` として `/api/{admin,account}/*` へ送る。バックエンドは RFC 9068 のリソースサーバーとして検証する。JavaScript からトークンへ到達できることは設計上受け入れたうえで、600 秒の短い有効期間、`Cache-Control: no-store`、そして URL・ログ・DOM にトークンを置かないことで露出の窓を狭める。

OIDC クライアントや鍵の設定を壊したときに直す手段そのものを失わないよう、ファーストパーティーのセッションログイン (`POST /api/auth/login`) を緊急経路として残す。
