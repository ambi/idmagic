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

@doc("プロセスが要求を処理できることを示す liveness probe。依存サービスの一時障害では失敗させない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("LivenessProbe")
@route("/livez")
@get
op LivenessProbe(

): LivenessProbeSuccess_200;

model ReadinessProbeHttpResponse {
  status: string;
  dependencies?: unknown;
}

model ReadinessProbeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ReadinessProbeHttpResponse;
}

@doc("起動完了、drain 状態、構成された永続化依存の到達性を確認する readiness probe。verbose 指定時だけ依存詳細を返す。")
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

@doc("初期化完了までは 503 starting、完了後は 200 healthy を返す startup probe。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartupProbe")
@route("/startupz")
@get
op StartupProbe(

): StartupProbeSuccess_200;

model MetricsExpositionHttpResponse {
  @doc("OpenMetrics text format")
  body: string;
}

model MetricsExpositionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: MetricsExpositionHttpResponse;
}

@doc("Prometheus/OpenMetrics 形式でスクレイプ可能な集約メトリクスを公開する management endpoint。\nidmagic-api は HTTP RED (リクエスト数・エラー率・レイテンシ) と認証ドメインのゴールデンシグナル\n(login 成否、token 発行、throttle ヒット) を、idmagic-worker は Jobs の実行レーン別ゴールデンシグナル\n(queue depth、claim latency、成功・失敗、retry。lane は ExecutionLane の有限集合) を公開する\nindependent な endpoint インスタンスであり、両者は同一プロセスを共有しない。tenant_id、user_id、\nclient_id、job_id、path 実値など高カーディナリティな値は label に含めず、route template や\noutcome class、lane などの有限集合だけを label とする。application API の tenant middleware から\n分離し、deploy policy で loopback/management network または認証のいずれかにより公開範囲を制限する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("MetricsExposition")
@route("/metrics")
@get
op MetricsExposition(

): MetricsExpositionSuccess_200;
}

}
