namespace IdMagic.Contract {

@doc("リクエストに必須パラメータの欠落・不正値がある、または対象が別テナントに属し未存在として扱われる。OAuth2 context が定義する共通 error の published language stub。")
model InvalidRequestError {

}


@doc("principal が対象操作の policy を満たさない。Authentication context が定義する共通 error の published language stub。")
model AccessDeniedError {

}


@doc("テナントのリソース作成上限 (Quota) に達している。Tenancy context が定義する共通 error の published language stub。")
model QuotaExceededError {

}


@doc("IdManagement が publish する Agent principal の最小 stub。")
model Agent {
  id: string;
  tenant_id: string;
  owner_user_id: string;
  status: string;
}


@doc("受信した SET の検証が fail-closed で失敗した (未登録 issuer・署名不正・期限切れ・audience 不一致・jti 重複・subject 未解決)。")
model SecurityEventRejectedError {

}


@doc("event_types が空である。")
model SsfStreamEventTypesRequiredError {

}


@doc("event_types に未知の CAEP event type が含まれる。")
model SsfStreamEventTypeInvalidError {

}


@doc("delivery_endpoint が未指定、または https URL でない。")
model SsfTransmitterDeliveryEndpointInvalidError {

}


@doc("audience が未指定である。")
model SsfTransmitterAudienceRequiredError {

}


@doc("trusted_issuer が未指定、または https URL でない。")
model SsfReceiverTrustedIssuerInvalidError {

}


@doc("jwks_uri と jwks のいずれも指定されていない。")
model SsfReceiverJwksRequiredError {

}


@doc("accepted_audiences が空である。")
model SsfReceiverAcceptedAudiencesRequiredError {

}


@doc("idmagic からみた stream の向き。Transmit は idmagic が外部 receiver へ CAEP イベントを push する、Receive は idmagic が外部 transmitter からのイベントを受理する。")
enum SsfStreamDirection {
  Transmit: "Transmit",
  Receive: "Receive",
}


@doc("stream の稼働状態。")
enum SsfStreamStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("実装する CAEP イベント種別。")
enum CaepEventType {
  session_revoked: "session-revoked",
  token_claims_change: "token-claims-change",
  credential_change: "credential-change",
  assurance_level_change: "assurance-level-change",
}


@doc("revocation epoch を前進させた起点。")
enum RevocationReason {
  AgentKilled: "AgentKilled",
  AgentDisabled: "AgentDisabled",
  AgentCredentialUnbound: "AgentCredentialUnbound",
  OwnerDisabled: "OwnerDisabled",
  OwnerDeleted: "OwnerDeleted",
  ManualAdmin: "ManualAdmin",
  InboundSecurityEvent: "InboundSecurityEvent",
}


@doc("outbound SET 配送の状態。")
enum SecurityEventDeliveryStatus {
  pending: "pending",
  delivered: "delivered",
  failed: "failed",
  dead_letter: "dead_letter",
}


@doc("inbound SET の検証結果。")
enum SecurityEventVerificationResult {
  accepted: "accepted",
  rejected_signature: "rejected_signature",
  rejected_unknown_issuer: "rejected_unknown_issuer",
  rejected_replay: "rejected_replay",
  rejected_audience: "rejected_audience",
  rejected_subject_unresolved: "rejected_subject_unresolved",
}


@doc("CAEP イベントが指す対象。idmagic 自身が発行する access token の subject 表現 (tenant_id + principal_id) とそのまま対応させ、WorkloadIdentity のような別テーブルでの subject mapping を新設しない。")
model SsfSubject {
  @doc("\"Agent\" または \"User\"。")
  subject_type: string;
  tenant_id: string;
  principal_id: string;
}


@doc("SET へ署名して包む前の、CAEP イベントの意味内容。")
model CaepEvent {
  event_type: CaepEventType;
  subject: SsfSubject;
  reason?: RevocationReason;
  event_timestamp: utcDateTime;
  @doc("\"policy\" | \"admin\" | \"user\" のいずれか。")
  initiating_entity: string;
}


@doc("RFC 8417 の Security Event Token。署名済み compact JWT と、そのデコード済み標準クレーム。")
model SecurityEventToken {
  jti: string;
  iss: url;
  aud: string;
  iat: utcDateTime;
  event: CaepEvent;
  @doc("署名済み JWT の compact serialization。")
  compact: string;
}


@doc("管理者が direction=Transmit の SsfStream を登録する JSON リクエスト。")
model SsfTransmitterStreamRegisterRequest {
  event_types: CaepEventType[];
  delivery_endpoint: url;
  audience: string;
  delivery_authorization?: string;
  max_delivery_attempts?: int64;
}


@doc("管理者が direction=Receive の SsfStream を登録する JSON リクエスト。")
model SsfReceiverStreamRegisterRequest {
  event_types: CaepEventType[];
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: unknown;
  accepted_audiences: string[];
}


@doc("管理者が SsfStream の event_types を更新する JSON リクエスト。省略フィールドは現状維持。")
model SsfStreamUpdateRequest {
  event_types?: CaepEventType[];
}


model RevocationEpochResponse {
  @doc("前進済みの epoch。未失効の Agent には存在しない。")
  epoch?: utcDateTime;
}


@doc("Agent ごとの revocation epoch を保持する 1 対 1 の付随レコード。epoch 以前に issued_at を持つ access token / session は無効とみなす (fail-closed)。KillAgent・DisableAgent・UnbindAgentCredential・所有者 (owner_user_id) の Disable/Delete・受理済み inbound SecurityEvent のいずれかで前進する。行が存在しない Agent は未失効を意味する。")
model AgentRevocationEpoch {
  agent_id: string;
  tenant_id: string;
  epoch: utcDateTime;
  reason: RevocationReason;
  advanced_at: utcDateTime;
  @doc("epoch を前進させた domain event / inbound SET の jti など追跡用の参照。")
  source_event_id?: string;
}


@doc("tenant が登録する SSF stream の共通メタデータ。direction ごとに SsfTransmitterConfig または SsfReceiverConfig が 1 対 1 で付随する。")
model SsfStream {
  id: unknown;
  tenant_id: string;
  direction: SsfStreamDirection;
  @doc("この stream が送受信対象とするイベント種別の部分集合。")
  event_types: CaepEventType[];
  status: SsfStreamStatus = SsfStreamStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("direction=Transmit の SsfStream に 1 対 1 で付随する配送先設定。")
model SsfTransmitterConfig {
  stream_id: unknown;
  @doc("外部 receiver の push 受理 endpoint。https 必須。")
  delivery_endpoint: url;
  @doc("発行する SET の aud claim。")
  audience: string;
  @doc("receiver への push 時に Authorization header で提示する任意の bearer secret。")
  delivery_authorization?: string;
  max_delivery_attempts: int64 = 8;
}


@doc("direction=Receive の SsfStream に 1 対 1 で付随する、外部 transmitter の信頼設定。WorkloadTrustBundle と同型の登録済み issuer 検証を再利用する。jwks_uri と jwks (inline) は少なくとも一方が必須。")
model SsfReceiverConfig {
  stream_id: unknown;
  @doc("外部 transmitter の SET iss claim と一致しなければならない発行者 URL。")
  trusted_issuer: url;
  jwks_uri?: url;
  jwks?: unknown;
  accepted_audiences: string[];
}


@doc("outbound SET の配送状態。RevocationEpochAdvanced から projector が生成し、at-least-once で配送・再試行する。")
model SecurityEventDelivery {
  id: unknown;
  tenant_id: string;
  stream_id: unknown;
  set_jti: string;
  set: SecurityEventToken;
  status: SecurityEventDeliveryStatus = SecurityEventDeliveryStatus.pending;
  attempt_count: int64 = 0;
  next_attempt_at?: utcDateTime;
  last_error?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("inbound SET の受理記録。set_jti は stream 内で一意とし、重複 (replay) を検知して拒否する。")
model ReceivedSecurityEvent {
  id: unknown;
  tenant_id: string;
  stream_id: unknown;
  set_jti: string;
  event_type: CaepEventType;
  subject: SsfSubject;
  verification_result: SecurityEventVerificationResult;
  received_at: utcDateTime;
  @doc("検証を通過し、ローカル revocation epoch へ反映した時刻。")
  reflected_at?: utcDateTime;
}


@doc("Agent の revocation epoch が前進した (LocalRevocation の確定)。transmitter projector がこのイベントを起点に CaepEvent/SecurityEventDelivery を生成する。")
model RevocationEpochAdvanced {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
  epoch: utcDateTime;
}


@doc("kill-switch / 所有者オフボードを起点に Agent のアクセスが即時失効したことを表す、監査・ガバナンス向けの要約イベント。")
model AgentAccessRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  agentId: string;
  reason: RevocationReason;
}


@doc("管理者が SsfStream を新規登録した。")
model SsfStreamRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  direction: SsfStreamDirection;
}


@doc("管理者が SsfStream のメタデータを更新した。")
model SsfStreamUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
}


@doc("管理者が SsfStream を無効化した。以後配送/受理を行わない。")
model SsfStreamDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
}


@doc("管理者が無効化済みの SsfStream を再有効化した。")
model SsfStreamEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
}


@doc("管理者が SsfStream を削除した。付随する Transmitter/ReceiverConfig は cascade で削除する。")
model SsfStreamDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
}


@doc("SecurityEventDelivery が配送に成功した。")
model SecurityEventTransmitted {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  setJti: string;
}


@doc("SecurityEventDelivery が配送に失敗し再試行をスケジュールした。")
model SecurityEventDeliveryFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  setJti: string;
  attemptCount: int64;
}


@doc("failed 状態の SecurityEventDelivery を再試行のため pending へ戻した。")
model SecurityEventDeliveryRetried {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  setJti: string;
  attemptCount: int64;
}


@doc("SecurityEventDelivery が max_delivery_attempts を使い切り dead_letter へ遷移した。")
model SecurityEventDeliveryDeadLettered {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  setJti: string;
}


@doc("inbound SET を検証し、ローカル revocation として反映した。")
model SecurityEventReceived {
  occurredAt: utcDateTime;
  tenantId: string;
  streamId: unknown;
  setJti: string;
  eventType: CaepEventType;
}


@doc("inbound SET の検証が fail-closed で失敗し反映しなかった。")
model SecurityEventRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("issuer 解決前に拒否した場合は stream を特定できないことがある。")
  streamId?: unknown;
  verificationResult: SecurityEventVerificationResult;
}

}
