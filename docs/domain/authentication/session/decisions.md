# ログインセッションの設計判断

- PostgreSQL の `authentication_sessions` を `LoginSession` の唯一の情報源とする。失効時もレコードを削除せず、失効済みであることを記録する。
