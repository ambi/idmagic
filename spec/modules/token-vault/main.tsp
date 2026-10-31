import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("TokenVault")
namespace IdMagic.TokenVault {

namespace Operations {

model AdminForbidden {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.CsrfFailedError | IdMagic.Contract.InsufficientScopeError | IdMagic.Contract.InvalidOriginError | IdMagic.Contract.AccessDeniedError;
}

model AccountForbidden {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.CsrfFailedError | IdMagic.Contract.InvalidOriginError | IdMagic.Contract.AccessDeniedError;
}

model InvalidRequest {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.InvalidRequestError;
}

model ConnectorNotFound {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.TokenVaultConnectorNotFoundError;
}

model ConnectorConflict {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.TokenVaultConnectorConflictError;
}

model ConnectorCreated {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.TokenVaultConnector;
}

model ConnectorOk {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TokenVaultConnector;
}

model NoContent {
  @statusCode statusCode: 204;
}

@doc("管理者がコネクタを作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["connectors:write"])
@TypeSpec.OpenAPI.operationId("CreateTokenVaultConnector")
@route("/api/admin/v1/token-vault/connectors")
@post
op CreateTokenVaultConnector(@body body: IdMagic.Contract.TokenVaultConnectorCreateRequest): ConnectorCreated | InvalidRequest | IdMagic.Contract.AuthenticationRequiredResponse | AdminForbidden | ConnectorConflict;

@doc("管理者がテナントのコネクタを一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["connectors:read"])
@TypeSpec.OpenAPI.operationId("ListTokenVaultConnectors")
@route("/api/admin/v1/token-vault/connectors")
@get
op ListTokenVaultConnectors(): { @statusCode statusCode: 200; @body body: IdMagic.Contract.TokenVaultConnectorList; } | IdMagic.Contract.AuthenticationRequiredResponse | AdminForbidden;

@doc("管理者がコネクタを参照する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["connectors:read"])
@TypeSpec.OpenAPI.operationId("GetTokenVaultConnector")
@route("/api/admin/v1/token-vault/connectors/{id}")
@get
op GetTokenVaultConnector(@path id: string): ConnectorOk | IdMagic.Contract.AuthenticationRequiredResponse | AdminForbidden | ConnectorNotFound;

@doc("管理者がコネクタを更新する。`status` で無効化と再開を行う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["connectors:write"])
@TypeSpec.OpenAPI.operationId("UpdateTokenVaultConnector")
@route("/api/admin/v1/token-vault/connectors/{id}")
@patch(#{ implicitOptionality: false })
op UpdateTokenVaultConnector(@path id: string, @body body: IdMagic.Contract.TokenVaultConnectorUpdateRequest): ConnectorOk | InvalidRequest | IdMagic.Contract.AuthenticationRequiredResponse | AdminForbidden | ConnectorNotFound | ConnectorConflict;

@doc("管理者がコネクタと、その接続と進行中の連携を削除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["connectors:write"])
@TypeSpec.OpenAPI.operationId("DeleteTokenVaultConnector")
@route("/api/admin/v1/token-vault/connectors/{id}")
@delete
op DeleteTokenVaultConnector(@path id: string): NoContent | IdMagic.Contract.AuthenticationRequiredResponse | AdminForbidden | ConnectorNotFound;

@doc("本人が連携できるコネクタと、自分の接続を一覧する。")
@TypeSpec.OpenAPI.operationId("ListMyTokenVaultConnections")
@route("/api/account/v1/token-vault/connections")
@get
op ListMyTokenVaultConnections(): { @statusCode statusCode: 200; @body body: IdMagic.Contract.TokenVaultMyConnectionList; } | IdMagic.Contract.AuthenticationRequiredResponse | AccountForbidden;

@doc("本人がブラウザーセッションで連携を開始し、上流の認可 URL を受け取る。")
@TypeSpec.OpenAPI.operationId("AuthorizeMyTokenVaultConnection")
@route("/api/account/v1/token-vault/connections/{connector_id}/authorize")
@post
op AuthorizeMyTokenVaultConnection(@path connector_id: string, @body body: IdMagic.Contract.TokenVaultAuthorizeRequest): { @statusCode statusCode: 200; @body body: IdMagic.Contract.TokenVaultAuthorizeResponse; } | { @statusCode statusCode: 400; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.InvalidRequestError | IdMagic.Contract.TokenVaultInvalidScopeError; } | IdMagic.Contract.AuthenticationRequiredResponse | AccountForbidden | ConnectorNotFound;

@doc("本人がブラウザーセッションで接続を解除する。")
@TypeSpec.OpenAPI.operationId("UnlinkMyTokenVaultConnection")
@route("/api/account/v1/token-vault/connections/{connector_id}")
@delete
op UnlinkMyTokenVaultConnection(@path connector_id: string): NoContent | IdMagic.Contract.AuthenticationRequiredResponse | AccountForbidden | { @statusCode statusCode: 404; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultConnectionNotFoundError; };

@doc("上流の認可サーバーからのコールバックを受け、結果をアカウントポータルの `result` へ載せて戻す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteTokenVaultConnection")
@route("/api/token-vault/v1/callback/{connector_id}")
@get
op CompleteTokenVaultConnection(@path connector_id: string, @query state?: string, @query code?: string, @query error?: string): { @statusCode statusCode: 302; @header location: string; };

@doc("代行トークンを持つ Agent が、本人の接続で外部 API を呼ぶ。トークンは応答に現れない。")
@TypeSpec.OpenAPI.operationId("ExecuteTokenVaultRequest")
@route("/api/token-vault/v1/connectors/{connector_id}/execute")
@post
op ExecuteTokenVaultRequest(@path connector_id: string, @body body: IdMagic.Contract.TokenVaultExecuteRequest): { @statusCode statusCode: 200; @header cacheControl: "no-store"; @body body: IdMagic.Contract.TokenVaultExecuteResponse; } | InvalidRequest | IdMagic.Contract.AuthenticationRequiredResponse | { @statusCode statusCode: 403; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultConnectorAccessDeniedError; } | { @statusCode statusCode: 404; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultConnectorNotFoundError | IdMagic.Contract.TokenVaultConnectionNotFoundError; } | { @statusCode statusCode: 409; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultConnectionReauthorizationRequiredError; } | { @statusCode statusCode: 413; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultRequestTooLargeError; } | { @statusCode statusCode: 502; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultUpstreamUnreachableError | IdMagic.Contract.TokenVaultUpstreamResponseTooLargeError; } | { @statusCode statusCode: 503; @header contentType: "application/problem+json"; @body body: IdMagic.Contract.TokenVaultTemporarilyUnavailableError; };

}

}
