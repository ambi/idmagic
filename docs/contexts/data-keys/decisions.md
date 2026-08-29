# DataKeys Decisions

- DEK のライフサイクル操作 (`bootstrap`、`rotate`、`disable`、`destroy`) は HTTP に公開しない。同じプロセス内の内部インターフェースとしてのみ呼べるため、テナント管理者がこれらを直接起動する経路は存在しない。
- 唯一の管理 API である `ListTenantDataKeyHealth` はテナント横断の運用情報を返すため、`system_admin` ロールを持つユーザーだけが呼べる。応答には鍵素材を一切含めず、`active_version`、`status`、プロバイダーへの到達性に限る。この操作は対話セッション限定であり、API アクセストークンからはどのスコープを持っていても到達できない。`ApiTokenScope` は DEK に対応する語彙を持たないためである。
- この Context を `Generic` に分類する。エンベロープ暗号は確立した形であり、AEAD と鍵セットの処理を Tink へ委ねる判断（`docs/database.md`）は、`Generic` の区分が指示する「自作しない」をこの製品で最初に適用した例である。
