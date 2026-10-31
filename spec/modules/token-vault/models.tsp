import "@typespec/http";

using TypeSpec.Http;

namespace IdMagic.Contract {

@doc("コネクタの状態。`disabled` は接続を保持したまま連携と代理実行を止める。")
enum TokenVaultConnectorStatus {
  active: "active",
  disabled: "disabled",
}

@doc("接続の状態。`reauthorization_required` はトークンを消してあり、本人の再連携を待つ。")
enum TokenVaultConnectionStatus {
  active: "active",
  reauthorization_required: "reauthorization_required",
}

@doc("上流のトークンエンドポイントでのクライアント認証。")
enum TokenVaultTokenEndpointAuthMethod {
  client_secret_basic: "client_secret_basic",
  client_secret_post: "client_secret_post",
}

@doc("コネクタが代理実行を許す要求の組。")
model TokenVaultExecutionRule {
  @doc("イベントに記す、コネクタの中で一意の規則の ID。")
  id: string;
  @doc("`GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE` のどれか。")
  method: string;
  @doc("`https://host[:port]` の形のオリジン。")
  origin: string;
  @doc("`/` で始まるパスの接頭辞。セグメントの境界で照合する。")
  path_prefix: string;
  @doc("接続の付与スコープに含まれるべきスコープ。`allowed_scopes` の部分集合である。")
  required_scopes: string[];
}

@doc("外部サービスのコネクタ。クライアントシークレットは応答に現れない。")
model TokenVaultConnector {
  id: string;
  tenant_id: string;
  name: string;
  status: TokenVaultConnectorStatus;
  authorization_endpoint: url;
  token_endpoint: url;
  revocation_endpoint?: url;
  client_id: string;
  token_endpoint_auth_method: TokenVaultTokenEndpointAuthMethod;
  client_secret_configured: boolean;
  allowed_scopes: string[];
  rules: TokenVaultExecutionRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}

model TokenVaultConnectorCreateRequest {
  name: string;
  authorization_endpoint: url;
  token_endpoint: url;
  revocation_endpoint?: url;
  client_id: string;
  token_endpoint_auth_method?: TokenVaultTokenEndpointAuthMethod;
  client_secret: string;
  allowed_scopes: string[];
  rules: TokenVaultExecutionRule[];
}

model TokenVaultConnectorUpdateRequest {
  name?: string;
  status?: TokenVaultConnectorStatus;
  authorization_endpoint?: url;
  token_endpoint?: url;
  revocation_endpoint?: url;
  client_id?: string;
  token_endpoint_auth_method?: TokenVaultTokenEndpointAuthMethod;
  @doc("省略すると保存したシークレットを維持する。")
  client_secret?: string;
  allowed_scopes?: string[];
  rules?: TokenVaultExecutionRule[];
}

model TokenVaultConnectorList {
  items: TokenVaultConnector[];
}

@doc("本人から見たコネクタと接続。トークンと有効期限は現れない。")
model TokenVaultMyConnection {
  connector_id: string;
  connector_name: string;
  connector_status: TokenVaultConnectorStatus;
  @doc("接続がなければ省く。")
  status?: TokenVaultConnectionStatus;
  granted_scopes: string[];
  last_used_at?: utcDateTime;
}

model TokenVaultMyConnectionList {
  items: TokenVaultMyConnection[];
}

model TokenVaultAuthorizeRequest {
  @doc("省略するとコネクタの `allowed_scopes` を要求する。")
  scopes?: string[];
}

model TokenVaultAuthorizeResponse {
  authorization_url: url;
}

model TokenVaultExecuteRequest {
  method: string;
  url: url;
  headers?: Record<string>;
  body_base64?: string;
}

model TokenVaultExecuteResponse {
  @doc("上流の HTTP ステータスコード。")
  status: int32;
  content_type?: string;
  body_base64: string;
}

@doc("コネクタが存在しないか、別のテナントのものである。`type` は `urn:idmagic:error:connector_not_found` である。")
model TokenVaultConnectorNotFoundError is ProblemDetails;

@doc("同じ名前のコネクタがテナントにある。`type` は `urn:idmagic:error:connector_conflict` である。")
model TokenVaultConnectorConflictError is ProblemDetails;

@doc("本人の接続がない。`type` は `urn:idmagic:error:connection_not_found` である。")
model TokenVaultConnectionNotFoundError is ProblemDetails;

@doc("接続は本人の再連携を待っている。`type` は `urn:idmagic:error:connection_reauthorization_required` である。")
model TokenVaultConnectionReauthorizationRequiredError is ProblemDetails;

@doc("代理実行の評価が拒否した。理由はイベントだけに記す。`type` は `urn:idmagic:error:connector_access_denied` である。")
model TokenVaultConnectorAccessDeniedError is ProblemDetails;

@doc("上流へ到達できないか、時間内に応答しない。`type` は `urn:idmagic:error:upstream_unreachable` である。")
model TokenVaultUpstreamUnreachableError is ProblemDetails;

@doc("上流の応答本文が上限を超えた。`type` は `urn:idmagic:error:upstream_response_too_large` である。")
model TokenVaultUpstreamResponseTooLargeError is ProblemDetails;

@doc("要求の本文が上限を超えた。`type` は `urn:idmagic:error:request_too_large` である。")
model TokenVaultRequestTooLargeError is ProblemDetails;

@doc("要求したスコープをコネクタが許していない。`type` は `urn:idmagic:error:invalid_scope` である。")
model TokenVaultInvalidScopeError is ProblemDetails;

@doc("失効かトークンの状態、更新を確かめられない。`type` は `urn:idmagic:error:temporarily_unavailable` である。")
model TokenVaultTemporarilyUnavailableError is ProblemDetails;

@doc("管理者がコネクタを作成した。")
model TokenVaultConnectorCreated { occurredAt: utcDateTime; tenantId: string; connectorId: string; }

@doc("管理者がコネクタを更新した。")
model TokenVaultConnectorUpdated { occurredAt: utcDateTime; tenantId: string; connectorId: string; status: TokenVaultConnectorStatus; }

@doc("管理者がコネクタを削除した。接続も消える。")
model TokenVaultConnectorDeleted { occurredAt: utcDateTime; tenantId: string; connectorId: string; }

@doc("本人が接続を連携した。")
model TokenVaultConnectionLinked { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId: string; }

@doc("コールバックのコード交換に失敗した。")
model TokenVaultConnectionLinkFailed { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId: string; }

@doc("本人が接続を解除した。`upstreamRevocation` は `succeeded`、`failed`、`not_configured` のどれかである。")
model TokenVaultConnectionUnlinked { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId: string; upstreamRevocation: string; }

@doc("上流の拒否により接続が再認可必須になった。")
model TokenVaultConnectionReauthorizationRequired { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId: string; }

@doc("代理実行を上流へ送った。パス、クエリ、本文、トークンを載せない。上流へ到達できなければ `upstreamStatus` は 0 である。")
model TokenVaultExecutionPerformed { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId: string; agentId: string; ruleId: string; method: string; origin: string; upstreamStatus: int32; }

@doc("代理実行を拒否した。上流へは送っていない。")
model TokenVaultExecutionDenied { occurredAt: utcDateTime; tenantId: string; connectorId: string; userId?: string; agentId?: string; reason: string; }

}
