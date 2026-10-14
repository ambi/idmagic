---
status: completed
archived_from: c365abe38e4cde569aa58a1e449b36672da1649e
---

# 共有の HTTP 支援パッケージから Context 固有の処理を外す

`mise run spec-diff` は規範の差分なしを報告する。
`backend/shared/http/support_http` から、Tenancy 以外の Context と `tokens_jose` への import をなくした。
管理者の認可と Bearer の検証は `support_http` の `Authenticator` に残し、セッション、アクセストークン、API トークン、主体の照会をインターフェースとして受け取る形にした。
各 Context の側に置いたアダプターを、組み立て地点の `server_http` が結ぶ。
アプリケーションの割り当てのゲートは Application の `usecases.AccessGate` へ、クライアントの表示名の解決は `backend/application` へ、同意のエラーの写像は呼び出す handler へ移した。
境界の負債の台帳は 854 件から 510 件になり、`support_http` 経由の `shared-detour` は 337 件から Tenancy への 47 件になった。
HTTP の応答、認可の判定、発行するイベント、永続状態は変えていない。

全文は `git show c365abe38e4cde569aa58a1e449b36672da1649e:<このファイルのパス>` で読む。
