# wi-573-agent-admin-api-answers-with-undeclared-statuses

エージェントの管理 API について、実際に返していた拒否を API 契約へ追加した。

- 存在しないエージェントへの操作は 404 `agent_not_found` を返す。別テナントのエージェントも同じ応答になる。
- 停止済みエージェントへの変更は 409 `agent_killed`、別のエージェントに束縛済みのクライアントのバインドは 409 `agent_client_already_bound` を返す。
- 無効化、再有効化、停止、削除の API 操作に、401 と CSRF・Origin・スコープの 403 を宣言した。

[IdMagic.IdManagement.Operations.BindAgentCredential](../../../spec/contexts/identity-management/main.tsp) で参照先のクライアントが見つからないときの `client_not_found` は、404 から 422 へ変わった。
別テナントのクライアントは、存在しないクライアントと同じ応答になる。
404 で分岐していた呼び出し元は、422 の `client_not_found` で分岐する。
