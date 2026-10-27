---
status: completed
---

# Agent に束縛したクライアントは共有シークレットで認証できないようにする

- `standards.md` へ `draft-ietf-wimse-aims-00` の節と AIMS00-AGENT-CLIENT-AUTH を追加した。Agent に束縛したクライアントは `private_key_jwt` か `tls_client_auth` で認証したときだけトークンを得られる。
- REQ-OAUTH2-055 を追加した。トークンエンドポイントはクライアント認証の後、グラントの分岐の前で一度だけ判定し、共有シークレットと `none` の Agent クライアントを 401 `invalid_client` で拒否して `AgentClientAuthenticationRejected` を発行する。Agent を解決しない認可コード、リフレッシュトークン、`device_code` も同じ判定を通る。束縛後の認証方式の変更もこの判定で拒否されるため、要求ごとの判定を正とした。
- REQ-IDMANAGEMENT-074 を変更し、`BindAgentCredential` は鍵の所有を証明しないクライアントを 422 `agent_client_auth_method_not_allowed` で拒否する。起票時の 400 ではなく `client_not_found` と同じ 422 に置いた。
- IdManagement の `ClientLookup` は `LookupBindableClient` へ置き換え、許可集合は FAPI 2.0 と共有する `OAuth2Client.ProvesKeyPossession` が持つ。
- Agent のトークンを発行させるテストのために、OAuth2 にテスト入力の生成専用の公開パッケージ `testing_clientauth` を置いた。
- CIBA のバックチャネル認証の開始では拒否しない（承認後のトークンはトークンエンドポイントで拒否される）。
