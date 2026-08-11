import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.Audit {

namespace Operations {
model ListAdminAuditEventsHttpResponse {
  response: IdMagic.Contract.AdminAuditEventListResponse;
}

model ListAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminAuditEventsHttpResponse;
}

union ListAdminAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError400Body;
}

union ListAdminAuditEventsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError403Body;
}

@doc("管理者が所属テナントの監査用 DomainEvent を時系列降順の双方向 keyset pagination で取得する。\nSOC / コンプライアンス用途を想定し、期間絞り込み (after / before) と件数上限 (limit)、継続取得\n(cursor、Link response header の rel=\"prev\" / rel=\"next\" 経由) と type / user_id / username\n(操作者) による絞り込みを提供する。認証イベント (ログイン成功 / 失敗 / MFA / セッション / 集約) も\n同一ストアに属し、category (authentication / success / fail / aggregated / user / group / client /\nconsent / token / tenant / key) で絞り込んでセキュリティ調査や運用調査に使える。さらに q (raw text\n部分一致) と filter (registry allowlist の field/operator による構造化式) で汎用検索できる。filter の\nfield は AuditEventSearchAttribute registry で許可されたもののみで、任意 SQL / JSONPath は受け付けない。\nuser_id と filter の actor.id は操作者を指し、username は操作者のログイン名を検索時に user_id へ\n解決する (hash 化せず、該当ユーザーが無ければ 0 件)。filter の actor.username は実在しないアカウント名も\n追跡する認証失敗イベント専用の平文一致検索。filter の target.id は管理操作の対象ユーザーを指す別概念\n(AuditActorVsTarget)。system_admin であれば default tenant 経路から全テナント横断、admin は所属テナント内に\n閉じる (クエリが tenant_id 入力を取らないため所属テナントに構造的に閉じ、system_admin だけが専用経路から\n横断できる)。")
@TypeSpec.OpenAPI.operationId("ListAdminAuditEvents")
@route("/api/admin/v1/audit_events")
@get
op ListAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ListAdminAuditEventsSuccess_200 | ListAdminAuditEventsError400 | ListAdminAuditEventsError403;

model ExportAdminAuditEventsHttpResponse {
  response: IdMagic.Contract.AdminAuditEventListResponse;
}

model ExportAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ExportAdminAuditEventsHttpResponse;
}

union ExportAdminAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ExportAdminAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError400Body;
}

union ExportAdminAuditEventsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ExportAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError403Body;
}

@doc("管理者が監査イベント (認証イベントを含む) を期間 / フィルタ指定でエクスポートする。\nSIEM への streaming は別 WI で、本 interface は 1 回分のダウンロードを返すのみ。\nフィルタは ListAdminAuditEvents と同じ AuditEventQuery (q / filter / category / type / 期間)。\nfilter の field は registry allowlist のみ。")
@TypeSpec.OpenAPI.operationId("ExportAdminAuditEvents")
@route("/api/admin/v1/audit_events/export")
@get
op ExportAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ExportAdminAuditEventsSuccess_200 | ExportAdminAuditEventsError400 | ExportAdminAuditEventsError403;

model GetAdminAuditEventSearchOptionsHttpResponse {
  response: IdMagic.Contract.AuditEventSearchOptionsResponse;
}

model GetAdminAuditEventSearchOptionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminAuditEventSearchOptionsHttpResponse;
}

union GetAdminAuditEventSearchOptionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventSearchOptionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventSearchOptionsError403Body;
}

@doc("管理者が監査イベント検索フォームの event.type / outcome を選択式にするための選択肢一覧を取得する。\n値は AuditSearchRegistry / auditEventCategoryTypes と同じ Go 側の単一の正から導出し、UI 側で\nハードコードした allowlist との drift を防ぐ。")
@TypeSpec.OpenAPI.operationId("GetAdminAuditEventSearchOptions")
@route("/api/admin/v1/audit_events/search_options")
@get
op GetAdminAuditEventSearchOptions(

): GetAdminAuditEventSearchOptionsSuccess_200 | GetAdminAuditEventSearchOptionsError403;

model GetAdminAuditEventHttpResponse {
  event: IdMagic.Contract.AdminAuditEventResponse;
}

model GetAdminAuditEventSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminAuditEventHttpResponse;
}

union GetAdminAuditEventError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminAuditEventError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError400Body;
}

union GetAdminAuditEventError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError403Body;
}

@doc("管理者が監査イベント 1 件を ID で取得する。別テナントのイベントは未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAdminAuditEvent")
@route("/api/admin/v1/audit_events/{id}")
@get
op GetAdminAuditEvent(
  @path id: string
): GetAdminAuditEventSuccess_200 | GetAdminAuditEventError400 | GetAdminAuditEventError403;
}

}
