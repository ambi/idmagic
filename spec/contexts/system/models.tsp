import "@typespec/http";

using TypeSpec.Http;

namespace IdMagic.Contract {

enum FeatureMaturity {
  experimental,
  preview,
  supported,
  deprecated,
}

enum FeatureUpdatePolicy {
  rolling,
  recreate_on_version_change,
  recreate_always,
}

@doc("このプロセスで有効になっている、実行時に選択できる機能の一つ。これらのフィールドは、設定や永続化ではなく、レジストリが管理する。")
model RuntimeFeatureMetadata {
  id: string;
  version: string;
  maturity: FeatureMaturity;
  update_policy: FeatureUpdatePolicy;
}

@doc("このプロセスで有効になっている、実行時に選択できる機能の、バージョン付きの運用メタデータ。")
model FeatureRuntimeMetadata {
  schema_version: "1";
  enabled: RuntimeFeatureMetadata[];
}

@doc("RFC 9457 の Problem Details。汎用の API のデフォルトのエラーボディである。`support_http.WriteProblem` が `Content-Type: application/problem+json` で出力する。形を標準が定める接点、すなわち OAuth 2.0 と OpenID Connect、Dynamic Client Registration（RFC 7591）、SCIM（RFC 7644）、SharedSignals の受信エンドポイント（RFC 8935）は、代わりにその標準のエラーボディを返す。ルートの仕様の HTTP エラーレスポンスの節を参照する。`is ProblemDetails` として宣言したすべての `*Error` モデルはこのボディを共有し、`type` の URN の接尾辞で区別する。")
model ProblemDetails {
  @doc("問題の種類を表す URI。IdMagic の問題の種類は `urn:idmagic:error:<code>` であり、`<code>` はサーバーが報告する安定したエラーコードである。この URI は参照解決できない。クライアントは URI を取得せず、URN の接尾辞で分岐する。")
  type: string;
  @doc("エラーコードから導く、人が読む短い要約。たとえば `invalid_role` は `Invalid role` になる。同じ `type` では変わらない。")
  title: string;
  @doc("レスポンスの HTTP ステータスコード。ボディにも繰り返す（RFC 9457 の 3.1 節）。")
  status: int32;
  @doc("この発生に固有の説明。常に英語で、ローカライズしない。ローカライズする UI は、代わりに `type` の URN の接尾辞を自身のメッセージへ対応付ける。")
  detail?: string;
  @doc("リクエストの相関 ID。レスポンスの `X-Request-ID` ヘッダーと同じ値である。リクエストに相関 ID がなかったときは省く。")
  instance?: string;
}


union AuthenticationRequiredResponseBody {
  AuthenticationRequiredError,
  InvalidAccessTokenError,
}


@doc("操作の前段のガードが、認証済みの主体を確立できなかったときに返す 401。使える資格情報のないリクエストには AuthenticationRequiredError を、提示したアクセストークンが検証に失敗したリクエストには InvalidAccessTokenError を返す。どちらも操作ごとの認可より前の認証の失敗なので、一つの共有のレスポンスとする。ルートの仕様の、宣言したステータスコードの節を参照する。")
model AuthenticationRequiredResponse {
  @statusCode statusCode: 401;
  @header contentType: "application/problem+json";
  @body body: AuthenticationRequiredResponseBody;
}


@doc("設定した一つの依存先への到達性。`verbose` を指定したとき readiness プローブが報告する。")
model DependencyStatus {
  @doc("依存先が応答したときは `healthy`、応答しなかったときは `unavailable`。")
  status: string;
  @doc("依存先を利用できない理由。健全なときは省く。")
  message?: string;
}


@doc("""
すべてのドメインイベントがとる送受信の形。`spec.MarshalDomainEvent` が作り、監査ストアが記録し、アカウントの
セキュリティ通知のディスパッチャーが購読し、`AdminAuditEventResponse` が管理者へ公開する。それ自体は HTTP の
リクエストやレスポンスのボディではない。これを受け付けたり返したりする操作はなく、呼び出し元へは監査の読み取りモデルを
通じてだけ届く。
""")
model DomainEventEnvelope {
  @doc("イベントの種類の名前。発行する Context の `EventType()` が返す値であり、`AuditEventQuery.type` が絞り込みに使う語彙である。")
  type: string;

  @doc("イベントが起きた時刻。UTC、ナノ秒精度の RFC 3339 で表す。記録を追記した時刻ではなく、発行する Context の時計による。")
  occurredAt: utcDateTime;

  @doc("イベントの本体。契約は下の公開フィールドだけで、それ以外は発行する Context に属する。")
  payload: DomainEventPayload;
}


@doc("""
ドメインイベントのペイロードの公表された言語。別の Context が名前で読むフィールドである。監査の検索属性の抽出器は
これらを検索軸へ射影し、アカウントのセキュリティ通知のディスパッチャーはこれらから受信者とガードを解決する。そのため
一つを改名すると、発行する Context はコンパイルできたまま、監査の絞り込みが黙って何も返さなくなる。すべてのイベントが
すべてのフィールドを含むわけではないので、どのフィールドも任意であり、フィールドを省いたイベントはその軸の下に現れない。
ペイロードは発行する Context に固有のフィールドも含むが、それらは契約ではなく、ここでは宣言しない。
`mise run check-event-contract` が、この宣言と読み取り箇所を同じ集合に保つ。
""")
model DomainEventPayload {
  @doc("イベントが属するテナント。これがないイベントは、テナントに限定した監査の検索から外れる。")
  tenantId?: string;

  @doc("イベントの対象となるサブジェクト。`actorUserId` がなく、エージェントが行うイベントでもないときは、操作した主体を表す。")
  userId?: string;

  @doc("管理操作を行った管理者。操作の対象のユーザーとは区別する。")
  actorUserId?: string;

  @doc("管理操作の対象のユーザー。`actor.id` と `target.id` は別の問いに答えるので、一つにまとめない。")
  targetUserId?: string;

  @doc("エージェント。エージェントが行うトークン交換と承認のイベントでは操作した主体を表し、それ以外ではエージェントが対象である。")
  agentId?: string;

  @doc("OAuth 2.0 のクライアント。")
  clientId?: string;

  @doc("ログインセッション。")
  sessionId?: string;

  @doc("認可のトランザクション。")
  transactionId?: string;

  @doc("一つの論理的な操作の記録に共通する相関 ID。")
  correlationId?: string;

  @doc("リクエストの相関 ID。レスポンスの `X-Request-ID` と同じ値である。")
  requestId?: string;

  @doc("送信されたままのアカウント名。平文で保持し、認証の失敗に限って使う。存在しなかったアカウントであっても追跡できる必要があるためである。")
  username?: string;

  @doc("クライアントの IP。平文で照合し、切り詰めもハッシュ化もしない。")
  ip?: string;

  @doc("クライアントの IP から解決した国。アカウントのセキュリティ通知の本文に書き出す。")
  countryCode?: string;

  @doc("クライアントのユーザーエージェント。通知のディスパッチャーは、ここからデバイスのラベルと、テナントごとのソルトを加えたデバイスのハッシュを導く。")
  userAgent?: string;

  @doc("遷移が起きた理由。セッションの失効をアカウントの保有者へ通知するのは、`self_revoke` と `admin_revoke` のときだけである。")
  reason?: string;

  @doc("ライフサイクルのワークフロー。")
  workflowId?: string;

  @doc("ライフサイクルのワークフローの実行。")
  runId?: string;

  @doc("ライフサイクルのワークフローの実行の中での、ステップの 0 から数えた位置。")
  stepIndex?: int32;

  @doc("委任の連鎖に現れる主体。両端だけでなく、どの参加者からもイベントにたどり着けるようにする。")
  actorChain?: string[];

  @doc("トークン交換が運んだ委任の段数。")
  delegationDepth?: int32;

  @doc("認可の判断が記録するときの名前で表した、同じ深さ。どちらも一つの検索軸へ射影する。")
  actorChainDepth?: int32;

  @doc("委任のモード。REQ-OAUTH2-049 が導く値であり、イントロスペクションと監査の記録が同じモードを示す。")
  delegationMode?: string;

  ...Record<unknown>;
}


}
