---
status: completed
---

# AD から取り込んだ User のパスワードの確認を Directory Connector の LDAP の bind に委ね、ログインと WS-Trust の usernamemixed で使えるようにする

Directory Connector による AD のパスワード照合を追加し、ブラウザーのログインと WS-Trust の usernamemixed から同じ認証操作を使う。
委譲対象はローカルのパスワードのハッシュを持たず、AD の拒否や配送障害でもローカル照合へ退避しない。
署名鍵とは別の P-256 鍵で JWE を作り、外向き HTTPS の 20 秒のポーリングで配送する。
要求は発行から 5 秒で失効し、単回配信で暗号文を消し、結果は最初の確定と同じ再送だけを認める。
管理者向けの委譲設定と診断を追加し、公開の認証応答では障害と資格情報の誤りを区別しない。
複数 Agent の振り分け、AD への書き戻し、実 Windows と Entra の PRT は対象外とする。
実 AD と Entra の動作および本番の認証遅延は未検証であり、WAL とバックアップからの暗号文の物理消去は保証しない。
