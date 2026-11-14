---
status: completed
---

# 単独のクライアントの削除も、Application の削除と同じく同意とトークンを片付ける

REQ-OAUTH2-035 の削除の文を三文に分け、管理者による単独のクライアントの削除が、リフレッシュトークン、同意、資格情報、セッションの関連付け、Agent の資格情報の束縛を同じトランザクションで消し、OAuth2 クライアントと消した同意の使用量を同じ確定で減らし、失敗時には何も変えないと定めた。
これまでは同意か未失効のリフレッシュトークンを持つクライアントの削除が外部キーの違反で 500 になり、消せても `consents` の使用量が残っていた。

OAuth2 は単独の削除の操作 `ClientDeletion.DeleteStandalone` を公開し、管理 API だけがそれを通る。
PostgreSQL ではクライアントを固定して組み込みと Application の所有を確かめ直し、`PrepareApplicationDeletionInTx` と同じ状態削除を共有する。
束縛の削除と Tx 内の使用量は組み立て地点が IdManagement と Tenancy から渡す。
Application の削除の経路と確定の順は変えていない。
起票時の候補 A（`DeleteForApplicationWithState` を両経路で共有する）は、PostgreSQL の Application の削除がその操作を通らないため採らなかった。
`ClientRepo.Delete` をポートから外す案は、CIMD の包装とテストの差し替えへの波及が大きいため保留した。
束縛を消しても `AgentCredentialUnbound` は発行しない（Application の削除と同じ）。
