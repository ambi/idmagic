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

@doc("liveness プローブ。プロセスがリクエストを処理できる状態かどうかを報告する。依存先に一時的に到達できなくても失敗しない。")
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

model ReadinessProbeError503 {
  @statusCode statusCode: 503;
  @body body: ReadinessProbeHttpResponse;
}

@doc("readiness プローブ。起動が完了したか、プロセスがドレイン中か、設定した永続化の依存先に到達できるかを報告する。依存先ごとの詳細は `verbose` を指定したときだけ返す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ReadinessProbe")
@route("/readyz")
@get
op ReadinessProbe(
  @query verbose?: string
): ReadinessProbeSuccess_200 | ReadinessProbeError503;

model StartupProbeHttpResponse {
  status: string;
}

model StartupProbeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartupProbeHttpResponse;
}

model StartupProbeError503 {
  @statusCode statusCode: 503;
  @body body: StartupProbeHttpResponse;
}

@doc("startup プローブ。初期化が完了するまでは 503 の `starting`、完了後は 200 の `healthy` を返す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartupProbe")
@route("/startupz")
@get
op StartupProbe(

): StartupProbeSuccess_200 | StartupProbeError503;

@doc("コンポジションルートでメトリクスのハンドラーを配線していないときに、メトリクスのエンドポイントが返す応答。経路はどちらの場合も登録するので、設定していないデプロイメントでは、404 を返すのではなく、スクレイプが明確に失敗する。")
model MetricsUnavailableHttpResponse {
  status: string;
}

model MetricsExpositionSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "text/plain; version=0.0.4; charset=utf-8";
  @doc("Prometheus / OpenMetrics のテキスト形式のメトリクスファミリー。")
  @body body: string;
}

model MetricsExpositionError503 {
  @statusCode statusCode: 503;
  @body body: MetricsUnavailableHttpResponse;
}

@doc("集約したメトリクスを Prometheus / OpenMetrics のスクレイプ向けに公開する管理用のエンドポイント。idmagic-api は HTTP の RED（リクエスト数、エラー率、レイテンシー）と、認証ドメインのゴールデンシグナル（ログインの結果、トークンの発行、抑制の発動）を公開する。idmagic-worker は、実行レーンごとの Jobs のゴールデンシグナル（キューの深さ、引き受けまでのレイテンシー、成功と失敗、再試行）を公開し、レーンは有限の ExecutionLane の集合から選ぶ。両者は独立したエンドポイントのインスタンスであり、プロセスを共有しない。tenant_id、user_id、client_id、job_id、具体的なパスのようなカーディナリティの高い値はラベルに含めず、経路のテンプレート、結果の分類、レーンのような有限の集合だけをラベルにする。このエンドポイントはアプリケーション API のテナントのミドルウェアの外にあり、ループバックまたは管理用ネットワークへバインドするか、認証を求めるかによって、到達できる相手をデプロイのポリシーで絞る。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("MetricsExposition")
@route("/metrics")
@get
op MetricsExposition(

): MetricsExpositionSuccess_200 | MetricsExpositionError503;
}

}
