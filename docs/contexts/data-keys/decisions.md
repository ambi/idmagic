# DataKeys Decisions

- DEK のライフサイクル操作 (`bootstrap`、`rotate`、`disable`、`destroy`) は HTTP に公開しない。同じプロセス内の内部インターフェースとしてのみ呼べるため、テナント管理者がこれらを直接起動する経路は存在しない。
- 唯一の管理 API である `ListTenantDataKeyHealth` はテナント横断の運用情報を返すため、`system_admin` ロールを持つユーザーだけが呼べる。応答には鍵素材を一切含めず、`active_version`、`status`、プロバイダーへの到達性に限る。この操作は対話セッション限定であり、API アクセストークンからはどのスコープを持っていても到達できない。`ApiTokenScope` は DEK に対応する語彙を持たないためである。
