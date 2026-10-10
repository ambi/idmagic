namespace IdMagic.Contract {

@doc("リクエストに必須のパラメーターがないか不正な値を含む、または対象が他のテナントに属しており存在しないものとして扱う。OAuth2 モジュールが定める共有のエラーを、公開契約として写したもの。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_request` である。OAuth 2.0 のエンドポイントは、同じ条件を代わりに OAuthInvalidRequestError として報告する。")
model InvalidRequestError is ProblemDetails;


@doc("主体が操作のポリシーを満たさない。Authentication モジュールが定める共有のエラーを、公開契約として写したもの。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:access_denied` である。OAuth 2.0 のエンドポイントは、OAuth のエラーボディで答える箇所では同じ条件を OAuthAccessDeniedError として報告する。`/authorize` は、ブラウザーへ直接書くサインインのポリシーの拒否にこの形を使う。")
model AccessDeniedError is ProblemDetails;


@doc("提示した API アクセストークンのスコープが、求めた汎用 API の操作を満たさない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:insufficient_scope` である。OAuth 2.0 と OpenID Connect のエンドポイントは、同じ条件を代わりに OAuthInsufficientScopeError として報告する。")
model InsufficientScopeError is ProblemDetails;


@doc("ブラウザーのリクエストの Origin が製品のオリジンと一致しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_origin` である。")
model InvalidOriginError is ProblemDetails;


@doc("ブラウザーのリクエストが、一致する二重送信の CSRF の Cookie とヘッダーを含んでいない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:csrf_failed` である。")
model CsrfFailedError is ProblemDetails;


@doc("操作には最近の再認証が必要である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:step_up_required` である。")
model StepUpRequiredError is ProblemDetails;


@doc("連携した ID を外すとアカウントに許可されたサインインの方法がなくなるので、外せない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:unlink_denied` である。")
model UnlinkDeniedError is ProblemDetails;


@doc("テナントがリソースの作成の上限（Quota）に達した。Tenancy モジュールが定める共有のエラーを、公開契約として写したもの。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:quota_exceeded` であり、Dynamic Client Registration のエンドポイントを含むすべての経路で共有のエラーハンドラーが書く。")
model QuotaExceededError is ProblemDetails;


@doc("受信した SET の検証がフェイルクローズで失敗した。未登録の発行者、不正な署名、期限切れのトークン、オーディエンスの不一致、`jti` の重複、解決できないサブジェクト、またはこのテナントの受信用のストリームを指さない `stream_id` のいずれかである。受信エンドポイントは形を RFC 8935 の 2.3 節が定める接点なので、Problem Details を返さず、ボディは `err` と `description` を含む `application/json` である。ストリームが存在するかをレスポンスで明かさないように、どの原因でもこの一つのコードを報告する。")
model SecurityEventRejectedError {
  @doc("常に `security_event_rejected` である。")
  err: string;
  @doc("拒否の、人が読む説明。常に英語で、ローカライズしない。")
  description: string;
}


@doc("`event_types` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_stream_event_types_required` である。")
model SsfStreamEventTypesRequiredError is ProblemDetails;


@doc("`event_types` に、サーバーが知らない CAEP のイベントの種類が含まれる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_stream_event_type_invalid` である。")
model SsfStreamEventTypeInvalidError is ProblemDetails;


@doc("`delivery_endpoint` がないか、https の URL でない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_transmitter_delivery_endpoint_invalid` である。")
model SsfTransmitterDeliveryEndpointInvalidError is ProblemDetails;


@doc("`audience` がない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_transmitter_audience_required` である。")
model SsfTransmitterAudienceRequiredError is ProblemDetails;


@doc("`trusted_issuer` がないか、https の URL でない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_receiver_trusted_issuer_invalid` である。")
model SsfReceiverTrustedIssuerInvalidError is ProblemDetails;


@doc("`jwks_uri` も `jwks` も与えられていない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_receiver_jwks_required` である。")
model SsfReceiverJwksRequiredError is ProblemDetails;


@doc("`accepted_audiences` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ssf_receiver_accepted_audiences_required` である。")
model SsfReceiverAcceptedAudiencesRequiredError is ProblemDetails;


@doc("IdMagic から見たストリームの向き。`Transmit` は IdMagic が外部の受信者へ CAEP のイベントをプッシュし、`Receive` は IdMagic が外部の送信者からイベントを受け付ける。")
enum SsfStreamDirection {
  Transmit: "Transmit",
  Receive: "Receive",
}


@doc("ストリームが動いているかどうか。")
enum SsfStreamStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("実装済みの CAEP のイベントの種類。")
enum CaepEventType {
  session_revoked: "session-revoked",
  token_claims_change: "token-claims-change",
  credential_change: "credential-change",
  assurance_level_change: "assurance-level-change",
}


@doc("失効のエポックを進めたもの。")
enum RevocationReason {
  AgentKilled: "AgentKilled",
  AgentDisabled: "AgentDisabled",
  AgentCredentialUnbound: "AgentCredentialUnbound",
  OwnerDisabled: "OwnerDisabled",
  OwnerDeleted: "OwnerDeleted",
  ManualAdmin: "ManualAdmin",
  InboundSecurityEvent: "InboundSecurityEvent",
}


@doc("送信する SET の配送状態。")
enum SecurityEventDeliveryStatus {
  pending: "pending",
  delivered: "delivered",
  failed: "failed",
  dead_letter: "dead_letter",
}


@doc("受信した SET の検証の結果。")
enum SecurityEventVerificationResult {
  accepted: "accepted",
  rejected_signature: "rejected_signature",
  rejected_unknown_issuer: "rejected_unknown_issuer",
  rejected_replay: "rejected_replay",
  rejected_audience: "rejected_audience",
  rejected_subject_unresolved: "rejected_subject_unresolved",
}


@doc("CAEP のイベントの対象。IdMagic 自身が発行するアクセストークンのサブジェクトの表現（tenant_id と principal_id）と一対一に対応するので、WorkloadIdentity が保持するような別のサブジェクトの対応表はない。")
model SsfSubject {
  @doc("`Agent` または `User` のどちらか。")
  subject_type: string;
  tenant_id: string;
  principal_id: string;
}


@doc("署名して SET に包む前の、CAEP のイベントの意味。")
model CaepEvent {
  event_type: CaepEventType;
  subject: SsfSubject;
  reason?: RevocationReason;
  event_timestamp: utcDateTime;
  @doc("`policy`、`admin`、`user` のいずれか。")
  initiating_entity: string;
}


@doc("Security Event Token（RFC 8417）。署名したコンパクトな JWT と、復号した標準のクレームからなる。")
model SecurityEventToken {
  jti: string;
  iss: url;
  aud: string;
  iat: utcDateTime;
  event: CaepEvent;
  @doc("署名した JWT のコンパクト直列化。")
  compact: string;
}


@doc("管理者が `direction=Transmit` の SsfStream を登録するために送る JSON のリクエスト。")
model SsfTransmitterStreamRegisterRequest {
  event_types: CaepEventType[];
  delivery_endpoint: url;
  audience: string;
  delivery_authorization?: string;
  max_delivery_attempts?: int64;
}


@doc("管理者が `direction=Receive` の SsfStream を登録するために送る JSON のリクエスト。")
model SsfReceiverStreamRegisterRequest {
  event_types: CaepEventType[];
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
}


@doc("管理者が SsfStream の `event_types` を更新するために送る JSON のリクエスト。省いたフィールドは現在の値を保つ。")
model SsfStreamUpdateRequest {
  event_types?: CaepEventType[];
}


model RevocationEpochResponse {
  @doc("すでに進めたエポック。一度も失効したことのない Agent では省く。")
  epoch?: utcDateTime;
}


@doc("Agent の失効のエポックを保持する、一対一に付随する記録。`issued_at` がエポックより前のアクセストークンまたはセッションは無効として扱う（フェイルクローズ）。エポックは、KillAgent、DisableAgent、UnbindAgentCredential、所有者（`owner_user_id`）の Disable または Delete、受け付けた受信の SecurityEvent で進む。行のない Agent は一度も失効していない。")
model AgentRevocationEpoch {
  agent_id: string;
  tenant_id: string;
  epoch: utcDateTime;
  reason: RevocationReason;
  advanced_at: utcDateTime;
  @doc("エポックを進めたものを追跡するための参照。ドメインイベント、または受信した SET の `jti` である。")
  source_event_id?: string;
}


@doc("テナントが登録するすべての SSF のストリームに共通するメタデータ。向きに応じて、SsfTransmitterConfig か SsfReceiverConfig のちょうど一つが付随する。")
model SsfStream {
  id: string;
  tenant_id: string;
  direction: SsfStreamDirection;
  @doc("このストリームが送る、または受け付けるイベントの種類の部分集合。")
  event_types: CaepEventType[];
  status: SsfStreamStatus = SsfStreamStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("`direction=Transmit` の SsfStream に付随する配送の設定。ストリームごとに一つである。")
model SsfTransmitterConfig {
  stream_id: string;
  @doc("プッシュを受け付ける外部の受信者のエンドポイント。https でなければならない。")
  delivery_endpoint: url;
  @doc("このストリームで発行する SET の `aud` のクレーム。")
  audience: string;
  @doc("受信者へプッシュするときに Authorization ヘッダーで提示する、任意の bearer のシークレット。")
  delivery_authorization?: string;
  max_delivery_attempts: int64 = 8;
}


@doc("`direction=Receive` の SsfStream に付随する、外部の送信者に対する信頼の設定。ストリームごとに一つである。WorkloadTrustBundle と同じ、登録済みの発行者による検証を再利用する。`jwks_uri` とインラインの `jwks` の少なくとも一方が必要である。")
model SsfReceiverConfig {
  stream_id: string;
  @doc("外部の送信者の SET の `iss` のクレームが一致しなければならない発行者の URL。")
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
}


@doc("送信する SET の配送状態。プロジェクターが RevocationEpochAdvanced から作り、配送と再試行は少なくとも一回である。")
model SecurityEventDelivery {
  id: string;
  tenant_id: string;
  stream_id: string;
  set_jti: string;
  set: SecurityEventToken;
  status: SecurityEventDeliveryStatus = SecurityEventDeliveryStatus.pending;
  attempt_count: int64 = 0;
  next_attempt_at?: utcDateTime;
  last_error?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("受け付けた受信の SET の記録。`set_jti` はストリームの中で一意なので、リプレイを検出して拒否する。")
model ReceivedSecurityEvent {
  id: string;
  tenant_id: string;
  stream_id: string;
  set_jti: string;
  event_type: CaepEventType;
  subject: SsfSubject;
  verification_result: SecurityEventVerificationResult;
  received_at: utcDateTime;
  @doc("SET が検証を通り、ローカルの失効のエポックに適用された時刻。")
  reflected_at?: utcDateTime;
}


@doc("Agent の失効のエポックが進み、ローカルの失効が確定した。送信のプロジェクターはこのイベントを起点に、CaepEvent と SecurityEventDelivery を作る。")
model RevocationEpochAdvanced {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
  epoch: utcDateTime;
}


@doc("キルスイッチまたは所有者のオフボーディングにより、Agent のアクセスを即座に失効させたことを記録する、監査とガバナンスのための要約のイベント。")
model AgentAccessRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
}


@doc("管理者が新しい SsfStream を登録した。")
model SsfStreamRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  direction: SsfStreamDirection;
}


@doc("管理者が SsfStream のメタデータを更新した。")
model SsfStreamUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("管理者が SsfStream を無効にした。以降は何も配送せず、何も受け付けない。")
model SsfStreamDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("管理者が、無効にした SsfStream を再び有効にした。")
model SsfStreamEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("管理者が SsfStream を削除した。それに結び付く TransmitterConfig または ReceiverConfig も一緒に削除した。")
model SsfStreamDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
}


@doc("SecurityEventDelivery の配送に成功した。")
model SecurityEventTransmitted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
}


@doc("SecurityEventDelivery の試行が失敗した。失敗した試行ごとにこのイベントを発行し、再試行を予定するか、`max_delivery_attempts` を使い切ったときは SecurityEventDeliveryDeadLettered が続く。")
model SecurityEventDeliveryFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  attemptCount: int64;
}


@doc("失敗した SecurityEventDelivery を、次の試行のために pending へ戻した。")
model SecurityEventDeliveryRetried {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  attemptCount: int64;
}


@doc("SecurityEventDelivery が `max_delivery_attempts` を使い切り、dead_letter へ移った。")
model SecurityEventDeliveryDeadLettered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
}


@doc("受信した SET を検証し、ローカルの失効として適用した。")
model SecurityEventReceived {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: string;
  setJti: string;
  eventType: CaepEventType;
}


@doc("受信した SET の検証がフェイルクローズで失敗し、何も適用しなかった。")
model SecurityEventRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("発行者を解決する前に拒否したときは、ストリームを特定できないことがある。")
  streamId?: string;
  verificationResult: SecurityEventVerificationResult;
}


@doc("リクエストボディの Security Event Token が、受信エンドポイントの受け付ける大きさを超えている。受信エンドポイントは形を RFC 8935 の 2.3 節が定める接点なので、ボディは Problem Details ではなく、`err` と `description` を含む `application/json` である。")
model SecurityEventTokenTooLargeError {
  @doc("常に `security_event_token_too_large` である。")
  err: string;
  @doc("人が読む説明。常に英語で、ローカライズしない。")
  description: string;
}


@doc("パスの `stream_id` が、このテナントの SsfStream を指さない。受信エンドポイントが返すので、ボディは Problem Details ではなく、その接点の `application/json` の形であり、RFC 8935 の 2.3 節に従って `err` と `description` を含む。管理 API は、同じ条件を `type` が `urn:idmagic:error:ssf_stream_not_found` の Problem Details として報告する。")
model SecurityEventStreamNotFoundError {
  @doc("常に `ssf_stream_not_found` である。")
  err: string;
  @doc("人が読む説明。常に英語で、ローカライズしない。")
  description: string;
}

}
