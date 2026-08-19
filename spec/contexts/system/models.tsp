namespace IdMagic.Contract {

@doc("RFC 9457 Problem Details, the default error body for generic APIs. `support_http.WriteProblem` emits it with `Content-Type: application/problem+json`. Contact points whose shape a standard fixes — OAuth 2.0 and OpenID Connect, Dynamic Client Registration (RFC 7591), SCIM (RFC 7644), and the SharedSignals receive endpoint (RFC 8935) — return that standard's error body instead; see the HTTP error responses section of the root specification. Every `*Error` model declared as `is ProblemDetails` shares this body and is told apart by its `type` URN suffix.")
model ProblemDetails {
  @doc("Problem type URI. IdMagic problem types are `urn:idmagic:error:<code>`, where `<code>` is the stable error code the server reports. The URI is not dereferenceable; clients switch on the URN suffix rather than fetching it.")
  type: string;
  @doc("Short human-readable summary derived from the error code: `invalid_role` becomes `Invalid role`. Stable for a given `type`.")
  title: string;
  @doc("HTTP status code of the response, repeated in the body (RFC 9457 section 3.1).")
  status: int32;
  @doc("Explanation specific to this occurrence. Always English and never localized; a UI that localizes maps the `type` URN suffix to its own message instead.")
  detail?: string;
  @doc("Request correlation id, the same value the response carries in the `X-Request-ID` header. Absent when the request had no correlation id.")
  instance?: string;
}


@doc("Reachability of one configured dependency, as the readiness probe reports it when `verbose` is set.")
model DependencyStatus {
  @doc("`healthy` when the dependency answered, `unavailable` when it did not.")
  status: string;
  @doc("Why the dependency is unavailable. Absent when it is healthy.")
  message?: string;
}

}
