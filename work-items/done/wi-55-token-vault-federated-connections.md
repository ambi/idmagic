---
status: completed
---

# エージェントが外部 API を代行するための Token Vault を設ける

新しいモジュール TokenVault を追加した。管理者がコネクタを定義し、本人がブラウザーセッションで外部サービスを連携、解除し、Agent は Vault の代理実行だけで外部 API を呼ぶ（REQ-TOKENVAULT-001〜015、`ConnectionLifecycle`、`connectors:read`、`connectors:write`）。上流のトークンを呼び出し元へ渡す経路は持たない。
許可は呼び出しごとの論理積である。`/introspect` と同じ失効判定、`aud` が `<発行者>/api/token-vault/v1` を含むこと、Token Exchange の `act`、`authorization_details`（`token_vault_connection`）、コネクタの規則、接続の付与スコープをすべて求める。`act` を必須にしたのは、代行付与の撤回の判定がそのトークンにだけ働くと実装中に分かったためで、起票時の想定にはなかった。テナントは `McpResourceServer` と `AuthorizationDetailType` の登録が要る。
更新は行のリースと `token_version` の条件付き更新で全プロセスの単一飛行にし、`invalid_grant` と更新トークンのない期限切れだけが接続を `reauthorization_required` へ落とす。暗号文はテナント、接続（コネクタと本人の組）、項目に束縛し、`FieldMigrator` を二つ登録した。
起票時から変えた点：外部の主体の保存と本人向けの利用履歴を対象外にし、`last_used_at` と Audit で代える。呼び出しごとの人の承認は wi-30707 などの後続へ残す。
実装をテストより先に書いたため、RED の代わりに認可、失効の配線、state の束縛、単一飛行、SSRF、リダイレクト、AAD などへ故障を注入して検出を確かめた。単一飛行と連携のトークン拒否のテストは故障を検出できなかったので強化した。
確認の限界：実在の外部プロバイダとの接続、ブラウザーでの操作（E2E）は未実施で、上流は通信の層で演じた。再暗号化が読んだ後の更新を上書きしないことは SQL の条件だけで、競合のテストはない。上流の応答本文のトークン反射、User の物理削除時の上流の失効、上流のレート制限への適応は `docs/modules/token-vault/risks.md` に残した。
