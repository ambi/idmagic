import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("System")
namespace IdMagic.System {

namespace Operations {
model LivenessProbeHttpResponse {
  status: string;
}

model LivenessProbeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: LivenessProbeHttpResponse;
}

@doc("Liveness probe: reports whether the process can serve requests at all. A dependency that is temporarily unreachable does not fail it.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("LivenessProbe")
@route("/livez")
@get
op LivenessProbe(

): LivenessProbeSuccess_200;

model ReadinessProbeHttpResponse {
  status: string;
  dependencies?: Record<IdMagic.Contract.DependencyStatus>;
}

model ReadinessProbeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ReadinessProbeHttpResponse;
}

@doc("Readiness probe: reports whether startup has finished, whether the process is draining, and whether the configured persistence dependencies are reachable. The per-dependency detail is returned only when `verbose` is set.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ReadinessProbe")
@route("/readyz")
@get
op ReadinessProbe(
  @query verbose?: string
): ReadinessProbeSuccess_200;

model StartupProbeHttpResponse {
  status: string;
}

model StartupProbeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartupProbeHttpResponse;
}

@doc("Startup probe: 503 `starting` until initialization has finished, 200 `healthy` afterwards.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartupProbe")
@route("/startupz")
@get
op StartupProbe(

): StartupProbeSuccess_200;

model MetricsExpositionSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "text/plain; version=0.0.4; charset=utf-8";
  @doc("The metric families in Prometheus / OpenMetrics text exposition format.")
  @body body: string;
}

@doc("Management endpoint exposing aggregated metrics for a Prometheus / OpenMetrics scrape.\nidmagic-api exposes HTTP RED (request count, error rate, latency) together with the golden signals of the\nauthentication domain (login outcome, token issuance, throttle hits); idmagic-worker exposes the golden signals\nof Jobs per execution lane (queue depth, claim latency, success and failure, retry, where the lane is drawn from\nthe finite ExecutionLane set). They are independent endpoint instances and never share a process. High-cardinality\nvalues -- tenant_id, user_id, client_id, job_id, the concrete path -- are kept out of the labels; only finite sets\nsuch as the route template, the outcome class, and the lane become labels. The endpoint sits outside the tenant\nmiddleware of the application API, and deploy policy narrows who can reach it, either by binding it to a loopback\nor management network or by requiring authentication.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("MetricsExposition")
@route("/metrics")
@get
op MetricsExposition(

): MetricsExpositionSuccess_200;
}

}
