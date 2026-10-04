# ホステッド UI とポータルの設計

この文書は、[ホステッド UI とポータル](README.md)の要件を保証する仕組みを扱う。

## セキュリティ

### 認可トランザクション

OAuth の認可の要求は、その内容を丸ごとサーバーの側に保持する。
ブラウザーへ渡すのは、短命な内部の UUID を載せたトランザクションの Cookie（`HttpOnly`、`SameSite=Lax`、HTTPS では `Secure`）だけである。
リダイレクト URI、PKCE の値、スコープ、クライアントの識別子は、HTML にも URL にも JavaScript から読める状態にも現れない。
これが、ログインの画面と同意の画面を描く JavaScript を差し替えられても、認可の要求そのものは書き換えられないことの根拠である。

SPA が `GET /api/auth/transaction` で取得できるのは、画面の種類、クライアントの名前、要求されたスコープといった表示用のデータに限る。
ログインと同意のコマンドは、SPA が送る値ではなく Cookie から解決したトランザクションを正とする。
同意では、現在のログインセッションの subject がトランザクションの subject と一致することを確かめる。

認可の要求は 10 分で期限切れとなり、完了した要求は再利用できない。
UI の API のレスポンスには `Cache-Control: no-store` を付け、資格情報も内部の要求の ID も返さない。

### ポータルのトークン

ポータルは IdP の `/authorize` と `/token` に対する `authorization_code` と PKCE で認証する。
純粋な SPA の RP なので、アクセストークンはブラウザーの `sessionStorage` に保持し、`Authorization: Bearer` として `/api/{admin,account}/*` へ送る。
バックエンドは RFC 9068 のリソースサーバーとして検証する。
JavaScript からトークンへ到達できることは設計の上で受け入れ、600 秒の短い有効期間、`Cache-Control: no-store`、URL とログと DOM にトークンを置かないことで、露出の窓を狭める。
