import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.SharedSignals {

namespace Operations {
model ListSsfStreamsHttpResponse {
  response: IdMagic.Contract.SsfStream[];
}

model ListSsfStreamsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSsfStreamsHttpResponse;
}

union ListSsfStreamsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListSsfStreamsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSsfStreamsError403Body;
}

@doc("管理者が所属テナントの SsfStream を一覧する。")
@TypeSpec.OpenAPI.operationId("ListSsfStreams")
@route("/api/admin/v1/shared-signals/streams")
@get
op ListSsfStreams(

): ListSsfStreamsSuccess_200 | ListSsfStreamsError403;

model GetSsfStreamHttpResponse {
  response: IdMagic.Contract.SsfStream;
}

model GetSsfStreamSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetSsfStreamHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model GetSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSsfStreamError403Body;
}

@doc("管理者が個々の SsfStream を参照する。")
@TypeSpec.OpenAPI.operationId("GetSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@get
op GetSsfStream(
  @path stream_id: unknown
): GetSsfStreamSuccess_200 | GetSsfStreamError400 | GetSsfStreamError403;

model RegisterSsfTransmitterStreamHttpRequest {
  request: IdMagic.Contract.SsfTransmitterStreamRegisterRequest;
}

model RegisterSsfTransmitterStreamHttpResponse {
  response: IdMagic.Contract.SsfStream;
}

model RegisterSsfTransmitterStreamSuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterSsfTransmitterStreamHttpResponse;
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

@doc("管理者が direction=Transmit の SsfStream と配送先 (SsfTransmitterConfig) を登録する。")
@TypeSpec.OpenAPI.operationId("RegisterSsfTransmitterStream")
@route("/api/admin/v1/shared-signals/streams/transmitter")
@post
op RegisterSsfTransmitterStream(
  @body body: RegisterSsfTransmitterStreamHttpRequest
): RegisterSsfTransmitterStreamSuccess_201 | RegisterSsfTransmitterStreamError400 | RegisterSsfTransmitterStreamError403 | RegisterSsfTransmitterStreamError422;

model RegisterSsfReceiverStreamHttpRequest {
  request: IdMagic.Contract.SsfReceiverStreamRegisterRequest;
}

model RegisterSsfReceiverStreamHttpResponse {
  response: IdMagic.Contract.SsfStream;
}

model RegisterSsfReceiverStreamSuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterSsfReceiverStreamHttpResponse;
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

@doc("管理者が direction=Receive の SsfStream と外部 transmitter の信頼設定 (SsfReceiverConfig) を登録する。")
@TypeSpec.OpenAPI.operationId("RegisterSsfReceiverStream")
@route("/api/admin/v1/shared-signals/streams/receiver")
@post
op RegisterSsfReceiverStream(
  @body body: RegisterSsfReceiverStreamHttpRequest
): RegisterSsfReceiverStreamSuccess_201 | RegisterSsfReceiverStreamError400 | RegisterSsfReceiverStreamError403 | RegisterSsfReceiverStreamError422;

model UpdateSsfStreamHttpRequest {
  stream_id: unknown;
  request: IdMagic.Contract.SsfStreamUpdateRequest;
}

model UpdateSsfStreamHttpResponse {
  response: IdMagic.Contract.SsfStream;
}

model UpdateSsfStreamSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateSsfStreamHttpResponse;
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

@doc("管理者が SsfStream の event_types 等のメタデータを更新する。")
@TypeSpec.OpenAPI.operationId("UpdateSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@patch
op UpdateSsfStream(
  @path stream_id: unknown,
  @body body: UpdateSsfStreamHttpRequest
): UpdateSsfStreamSuccess_200 | UpdateSsfStreamError400 | UpdateSsfStreamError403 | UpdateSsfStreamError422;

model DisableSsfStreamHttpRequest {
  stream_id: unknown;
}

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

@doc("管理者が SsfStream を無効化する。以後配送/受理を行わない。")
@TypeSpec.OpenAPI.operationId("DisableSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/disable")
@post
op DisableSsfStream(
  @path stream_id: unknown,
  @body body: DisableSsfStreamHttpRequest
): DisableSsfStreamSuccess_204 | DisableSsfStreamError400 | DisableSsfStreamError403;

model EnableSsfStreamHttpRequest {
  stream_id: unknown;
}

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

@doc("管理者が無効化済みの SsfStream を再有効化する。")
@TypeSpec.OpenAPI.operationId("EnableSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/enable")
@post
op EnableSsfStream(
  @path stream_id: unknown,
  @body body: EnableSsfStreamHttpRequest
): EnableSsfStreamSuccess_204 | EnableSsfStreamError400 | EnableSsfStreamError403;

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

@doc("管理者が SsfStream を削除する。付随する Transmitter/ReceiverConfig は cascade で削除する。")
@TypeSpec.OpenAPI.operationId("DeleteSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@delete
op DeleteSsfStream(
  @path stream_id: unknown
): DeleteSsfStreamSuccess_204 | DeleteSsfStreamError400 | DeleteSsfStreamError403;

model ListSecurityEventDeliveriesHttpResponse {
  response: IdMagic.Contract.SecurityEventDelivery[];
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
  IdMagic.Contract.AccessDeniedError,
}

model ListSecurityEventDeliveriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSecurityEventDeliveriesError403Body;
}

@doc("管理者が direction=Transmit の SsfStream の配送状況 (pending/delivered/failed/dead_letter) を一覧する。運用の delivery health 確認に使う。")
@TypeSpec.OpenAPI.operationId("ListSecurityEventDeliveries")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/deliveries")
@get
op ListSecurityEventDeliveries(
  @path stream_id: unknown
): ListSecurityEventDeliveriesSuccess_200 | ListSecurityEventDeliveriesError400 | ListSecurityEventDeliveriesError403;

model ReceiveSecurityEventHttpRequest {
  stream_id: unknown;
  @doc("受理した SET の compact serialization。")
  token: string;
}

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

@doc("外部 transmitter からの SET を受理する。署名検証・登録済み issuer 照合・jti 重複検知・subject 解決のいずれかに失敗した場合は反映せず拒否する (fail-closed)。検証を通過したイベントはローカル revocation (AdvanceRevocationEpoch) として反映する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ReceiveSecurityEvent")
@route("/ssf/streams/{stream_id}/events")
@post
op ReceiveSecurityEvent(
  @path stream_id: unknown,
  @body body: ReceiveSecurityEventHttpRequest
): ReceiveSecurityEventSuccess_202 | ReceiveSecurityEventError400;
}

}
