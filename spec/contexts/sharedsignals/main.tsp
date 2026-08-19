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
  IdMagic.Contract.AccessDeniedError,
}

model ListSsfStreamsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSsfStreamsError403Body;
}

@doc("An administrator lists the SsfStreams of their tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("ListSsfStreams")
@route("/api/admin/v1/shared-signals/streams")
@get
op ListSsfStreams(

): ListSsfStreamsSuccess_200 | ListSsfStreamsError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model GetSsfStreamError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSsfStreamError403Body;
}

@doc("An administrator reads one SsfStream.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("GetSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@get
op GetSsfStream(
  @path stream_id: string
): GetSsfStreamSuccess_200 | GetSsfStreamError400 | GetSsfStreamError403;

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

@doc("An administrator registers an SsfStream with `direction=Transmit` together with its delivery target (SsfTransmitterConfig).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("RegisterSsfTransmitterStream")
@route("/api/admin/v1/shared-signals/streams/transmitter")
@post
op RegisterSsfTransmitterStream(
  @body body: IdMagic.Contract.SsfTransmitterStreamRegisterRequest
): RegisterSsfTransmitterStreamSuccess_201 | RegisterSsfTransmitterStreamError400 | RegisterSsfTransmitterStreamError403 | RegisterSsfTransmitterStreamError422;

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

@doc("An administrator registers an SsfStream with `direction=Receive` together with the trust settings for the external transmitter (SsfReceiverConfig).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("RegisterSsfReceiverStream")
@route("/api/admin/v1/shared-signals/streams/receiver")
@post
op RegisterSsfReceiverStream(
  @body body: IdMagic.Contract.SsfReceiverStreamRegisterRequest
): RegisterSsfReceiverStreamSuccess_201 | RegisterSsfReceiverStreamError400 | RegisterSsfReceiverStreamError403 | RegisterSsfReceiverStreamError422;

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

@doc("An administrator updates the metadata of an SsfStream, `event_types` among it.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:write"])
@TypeSpec.OpenAPI.operationId("UpdateSsfStream")
@route("/api/admin/v1/shared-signals/streams/{stream_id}")
@patch
op UpdateSsfStream(
  @path stream_id: string,
  @body body: IdMagic.Contract.SsfStreamUpdateRequest
): UpdateSsfStreamSuccess_200 | UpdateSsfStreamError400 | UpdateSsfStreamError403 | UpdateSsfStreamError422;

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

@doc("An administrator disables an SsfStream. It neither delivers nor accepts anything afterwards.")
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

@doc("An administrator re-enables a disabled SsfStream.")
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

@doc("An administrator deletes an SsfStream. The TransmitterConfig or ReceiverConfig attached to it is deleted with it.")
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
  IdMagic.Contract.AccessDeniedError,
}

model ListSecurityEventDeliveriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSecurityEventDeliveriesError403Body;
}

@doc("An administrator lists the delivery state (pending / delivered / failed / dead_letter) of an SsfStream with `direction=Transmit`, to check delivery health in operation.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["shared-signals:read"])
@TypeSpec.OpenAPI.operationId("ListSecurityEventDeliveries")
@route("/api/admin/v1/shared-signals/streams/{stream_id}/deliveries")
@get
op ListSecurityEventDeliveries(
  @path stream_id: string
): ListSecurityEventDeliveriesSuccess_200 | ListSecurityEventDeliveriesError400 | ListSecurityEventDeliveriesError403;

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

@doc("Accepts a SET pushed by an external transmitter. Signature verification, the match against a registered issuer, duplicate `jti` detection, and subject resolution each reject the event without applying it, fail-closed. An event that passes verification is applied as a local revocation (AdvanceRevocationEpoch).")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ReceiveSecurityEvent")
@route("/ssf/streams/{stream_id}/events")
@post
op ReceiveSecurityEvent(
  @path stream_id: string,
  @doc("RFC 8935 section 2.1 fixes the media type for the push delivery of a Security Event Token. The server does not reject other media types, but a conforming transmitter sends this one.")
  @header contentType: "application/secevent+jwt",
  @doc("The Security Event Token itself, in JWS compact serialization. The body is the token, not a JSON object wrapping it.")
  @body token: string
): ReceiveSecurityEventSuccess_202 | ReceiveSecurityEventError400 | ReceiveSecurityEventError404 | ReceiveSecurityEventError413;
}

}
