import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Shared Signals")
namespace IdMagic.SharedSignals {

namespace Operations {
model ListSsfStreamsHttpResponse {
  streams: IdMagic.Contract.SsfStream[];
}

model ListSsfStreamsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSsfStreamsHttpResponse;
}

union ListSsfStreamsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSsfStreamsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSsfStreamsError403Body;
}

@doc("管理者が、自身のテナントの SsfStream を一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("ListSsfStreams")
@route("/api/admin/v1/shared-signals/streams")
@get
op ListSsfStreams(

): ListSsfStreamsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListSsfStreamsError403;

model GetSsfStreamSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SsfStream;
}

union GetSsfStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetSsfStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetSsfStreamError400Body;
}

union GetSsfStreamError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSsfStreamError403Body;
}

@doc("管理者が SsfStream を一つ読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("GetSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@get
op GetSsfStream(
  @path stream_id: string
): GetSsfStreamSuccess_200 | GetSsfStreamError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetSsfStreamError403;

model RegisterSsfTransmitterStreamSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SsfStream;
}

union RegisterSsfTransmitterStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterSsfTransmitterStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfTransmitterStreamError400Body;
}

union RegisterSsfTransmitterStreamError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterSsfTransmitterStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfTransmitterStreamError403Body;
}

union RegisterSsfTransmitterStreamError422Body {
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.SsfStreamEventTypesRequiredError,
  IdMagic.Contract.SsfStreamEventTypeInvalidError,
  IdMagic.Contract.SsfTransmitterDeliveryEndpointInvalidError,
  IdMagic.Contract.SsfTransmitterAudienceRequiredError,
}

model RegisterSsfTransmitterStreamError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfTransmitterStreamError422Body;
}

@doc("管理者が、`direction=Transmit` の SsfStream を、その配送先（SsfTransmitterConfig）とともに登録する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("RegisterSsfTransmitterStream")
@route("/api/admin/v1/shared-signals/streams/transmitter")
@post
op RegisterSsfTransmitterStream(
  @body body: IdMagic.Contract.SsfTransmitterStreamRegisterRequest
): RegisterSsfTransmitterStreamSuccess_201 | RegisterSsfTransmitterStreamError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterSsfTransmitterStreamError403 | RegisterSsfTransmitterStreamError422;

model RegisterSsfReceiverStreamSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SsfStream;
}

union RegisterSsfReceiverStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterSsfReceiverStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfReceiverStreamError400Body;
}

union RegisterSsfReceiverStreamError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterSsfReceiverStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfReceiverStreamError403Body;
}

union RegisterSsfReceiverStreamError422Body {
  IdMagic.Contract.QuotaExceededError,
  IdMagic.Contract.SsfStreamEventTypesRequiredError,
  IdMagic.Contract.SsfStreamEventTypeInvalidError,
  IdMagic.Contract.SsfReceiverTrustedIssuerInvalidError,
  IdMagic.Contract.SsfReceiverJwksRequiredError,
  IdMagic.Contract.SsfReceiverAcceptedAudiencesRequiredError,
}

model RegisterSsfReceiverStreamError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterSsfReceiverStreamError422Body;
}

@doc("管理者が、`direction=Receive` の SsfStream を、外部の送信者に対する信頼の設定（SsfReceiverConfig）とともに登録する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("RegisterSsfReceiverStream")
@route("/api/admin/v1/shared-signals/streams/receiver")
@post
op RegisterSsfReceiverStream(
  @body body: IdMagic.Contract.SsfReceiverStreamRegisterRequest
): RegisterSsfReceiverStreamSuccess_201 | RegisterSsfReceiverStreamError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterSsfReceiverStreamError403 | RegisterSsfReceiverStreamError422;

model UpdateSsfStreamSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SsfStream;
}

union UpdateSsfStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateSsfStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateSsfStreamError400Body;
}

union UpdateSsfStreamError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateSsfStreamError403Body;
}

union UpdateSsfStreamError422Body {
  IdMagic.Contract.SsfStreamEventTypesRequiredError,
  IdMagic.Contract.SsfStreamEventTypeInvalidError,
}

model UpdateSsfStreamError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateSsfStreamError422Body;
}

@doc("管理者が、`event_types` を含む SsfStream のメタデータを更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("UpdateSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@patch
op UpdateSsfStream(
  @path stream_id: string,
  @body body: IdMagic.Contract.SsfStreamUpdateRequest
): UpdateSsfStreamSuccess_200 | UpdateSsfStreamError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateSsfStreamError403 | UpdateSsfStreamError422;

model DisableSsfStreamSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableSsfStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableSsfStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableSsfStreamError400Body;
}

union DisableSsfStreamError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableSsfStreamError403Body;
}

@doc("管理者が SsfStream を無効にする。以降は何も配送せず、何も受け付けない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("DisableSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/disable")
@post
op DisableSsfStream(
  @path stream_id: string,): DisableSsfStreamSuccess_204 | DisableSsfStreamError400 | DisableSsfStreamError403;

model EnableSsfStreamSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableSsfStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableSsfStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableSsfStreamError400Body;
}

union EnableSsfStreamError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableSsfStreamError403Body;
}

@doc("管理者が、無効にした SsfStream を再び有効にする。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("EnableSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/enable")
@post
op EnableSsfStream(
  @path stream_id: string,): EnableSsfStreamSuccess_204 | EnableSsfStreamError400 | EnableSsfStreamError403;

model DeleteSsfStreamSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteSsfStreamError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteSsfStreamError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteSsfStreamError400Body;
}

union DeleteSsfStreamError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteSsfStreamError403Body;
}

@doc("管理者が SsfStream を削除する。それに結び付く TransmitterConfig または ReceiverConfig も一緒に削除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("DeleteSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@delete
op DeleteSsfStream(
  @path stream_id: string
): DeleteSsfStreamSuccess_204 | DeleteSsfStreamError400 | DeleteSsfStreamError403;

model ListSecurityEventDeliveriesHttpResponse {
  deliveries: IdMagic.Contract.SecurityEventDelivery[];
}

model ListSecurityEventDeliveriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSecurityEventDeliveriesHttpResponse;
}

union ListSecurityEventDeliveriesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListSecurityEventDeliveriesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListSecurityEventDeliveriesError400Body;
}

union ListSecurityEventDeliveriesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSecurityEventDeliveriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSecurityEventDeliveriesError403Body;
}

@doc("管理者が、`direction=Transmit` の SsfStream の配送状態（pending / delivered / failed / dead_letter）を一覧し、運用中の配送の健全性を確かめる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("ListSecurityEventDeliveries")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/deliveries")
@get
op ListSecurityEventDeliveries(
  @path stream_id: string
): ListSecurityEventDeliveriesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListSecurityEventDeliveriesError400 | ListSecurityEventDeliveriesError403;

model ReceiveSecurityEventSuccess_202 {
  @statusCode statusCode: 202;
}

union ReceiveSecurityEventError400Body {
  IdMagic.Contract.SecurityEventRejectedError,
}

model ReceiveSecurityEventError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: ReceiveSecurityEventError400Body;
}

union ReceiveSecurityEventError404Body {
  IdMagic.Contract.SecurityEventStreamNotFoundError,
}

model ReceiveSecurityEventError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/json";
  @body body: ReceiveSecurityEventError404Body;
}

union ReceiveSecurityEventError413Body {
  IdMagic.Contract.SecurityEventTokenTooLargeError,
}

model ReceiveSecurityEventError413 {
  @statusCode statusCode: 413;
  @header contentType: "application/json";
  @body body: ReceiveSecurityEventError413Body;
}

@doc("外部の送信者がプッシュした SET を受け付ける。署名の検証、登録済みの発行者との照合、`jti` の重複の検出、サブジェクトの解決のどれかで失敗すると、イベントを適用せずにフェイルクローズで拒否する。検証を通ったイベントは、ローカルの失効（AdvanceRevocationEpoch）として適用する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ReceiveSecurityEvent")
@route("/ssf/streams/{stream_id}/events")
@post
op ReceiveSecurityEvent(
  @path stream_id: string,
  @doc("RFC 8935 の 2.1 節が、Security Event Token のプッシュ配送のメディアタイプを定める。サーバーは他のメディアタイプを拒否しないが、仕様に従う送信者はこれを送る。")
  @header contentType: "application/secevent+jwt",
  @doc("JWS のコンパクト直列化による Security Event Token そのもの。ボディはトークンそのもので、それを包む JSON オブジェクトではない。")
  @body token: string
): ReceiveSecurityEventSuccess_202 | ReceiveSecurityEventError400 | ReceiveSecurityEventError404 | ReceiveSecurityEventError413;
}

}
