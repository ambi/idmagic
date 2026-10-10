import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Audit")
namespace IdMagic.Audit {

namespace Operations {
model ListAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminAuditEventsError403Body;
}

@doc("管理者が、自身のテナントの監査の DomainEvent を、新しい順に双方向のキーセットページングで読む。SOC とコンプライアンスの業務のためのもので、時間の範囲（`after` / `before`）、ページの大きさ（`limit`）、続きの取得（`cursor`。Link レスポンスヘッダーの `rel=\"prev\"` と `rel=\"next\"` のリンクからたどる）、`type`、`user_id`、`username`（操作した主体）による絞り込みを提供する。認証のイベント（ログインの成功と失敗、MFA、セッション、集約）も同じストアにあるので、`category`（authentication / success / fail / aggregated / user / group / client / consent / token / tenant / key）でセキュリティまたは運用の調査のための検索を絞り込める。さらに、`q` は生のテキストに部分一致し、`filter` はレジストリのフィールドと演算子の許可リストに対する構造化された式を受け付ける。`filter` に使えるのは AuditEventSearchAttribute のレジストリが許可するフィールドだけで、任意の SQL や JSONPath は拒否する。`user_id` と `filter` の `actor.id` は操作した主体を指し、`username` は検索時に操作した主体のログイン名をユーザー ID へ解決する。ハッシュ化はせず、該当するユーザーが存在しなければ何にも一致しない。`filter` の `actor.username` は認証の失敗に限った平文の一致で、存在しなかったアカウント名も追跡できるようにする。`filter` の `target.id` は別の概念で、管理操作の対象のユーザーを指す（AuditActorVsTarget）。範囲は経路が決める呼び出し元自身のテナントであり、検索は tenant_id の入力も範囲を広げる入力もとらないので、閉じ込めは構造的である。ここでは制御面の資格情報でもこれは解除されず、テナントを横断する検索は ListSystemAuditEvents だけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListAdminAuditEvents")
@route("/api/admin/v1/audit-events")
@get
op ListAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ListAdminAuditEventsSuccess_200 | ListAdminAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminAuditEventsError403;

model ExportAdminAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ExportAdminAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportAdminAuditEventsError403Body;
}

@doc("管理者が、自身のテナントの監査イベントを、認証のイベントも含めて、時間の範囲と絞り込み条件でエクスポートする。SIEM へのストリーミングは別の work item が扱い、このインターフェースは一回のダウンロードを返す。絞り込み条件は ListAdminAuditEvents と同じ AuditEventQuery（q / filter / category / type / 時間の範囲）で、`filter` に使えるのはレジストリの許可リストにあるフィールドだけである。範囲は ListAdminAuditEvents とまったく同じで、制御面の資格情報であっても、経路が決める呼び出し元自身のテナントである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ExportAdminAuditEvents")
@route("/api/admin/v1/audit-events/export")
@get
op ExportAdminAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ExportAdminAuditEventsSuccess_200 | ExportAdminAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ExportAdminAuditEventsError403;

model GetAdminAuditEventSearchOptionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuditEventSearchOptionsResponse;
}

union GetAdminAuditEventSearchOptionsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventSearchOptionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventSearchOptionsError403Body;
}

@doc("管理者が、監査の検索フォームの `event.type` と `outcome` のフィールドを選択肢の一覧にするための値を読む。値は AuditSearchRegistry や auditEventCategoryTypes と同じ Go の唯一の情報源から得るので、UI に直接書いた許可リストがサーバーの受け付けるものと食い違うことはない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetAdminAuditEventSearchOptions")
@route("/api/admin/v1/audit-events/search-options")
@get
op GetAdminAuditEventSearchOptions(

): GetAdminAuditEventSearchOptionsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminAuditEventSearchOptionsError403;

model GetAdminAuditEventSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventResponse;
}

union GetAdminAuditEventError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminAuditEventError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError403Body;
}

union GetAdminAuditEventError404Body {
  IdMagic.Contract.AuditEventNotFoundError,
}

model GetAdminAuditEventError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminAuditEventError404Body;
}

@doc("管理者が、自身のテナントの監査イベントを id で一つ読む。他のテナントのイベントは存在しないものとして扱うので、id だけでは呼び出し元が見られないテナントについて何も明かさず、制御面の資格情報でもそれは変わらない。他のテナントに届く読み取りは GetSystemAuditEvent だけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetAdminAuditEvent")
@route("/api/admin/v1/audit-events/{id}")
@get
op GetAdminAuditEvent(
  @path id: string
): GetAdminAuditEventSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminAuditEventError403 | GetAdminAuditEventError404;

model ListSystemAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ListSystemAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListSystemAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListSystemAuditEventsError400Body;
}

union ListSystemAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSystemAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSystemAuditEventsError403Body;
}

@doc("制御面の運用者が、すべてのテナントの監査の DomainEvent を、ListAdminAuditEvents と同じ双方向のキーセットページング、時間の範囲、ページの大きさ、絞り込みの文法で、新しい順に検索する。範囲は入力ではなく経路が決めるすべてのテナントなので、テナント向けの経路にパラメーターを足してテナント横断の検索に至ることはない。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならない。そうした呼び出し元に自身の `admin` ロールは不要で、三つの条件のどれかを満たさない呼び出し元は、イベントを読む前に AccessDeniedError で拒否する。範囲はカーソルが結び付く対象の一部なので、ListAdminAuditEvents が発行したカーソルは、ここでは続きとして扱わず拒否する。検索フォームの選択肢の一覧は GetAdminAuditEventSearchOptions から得る。これはどのテナントの記録でもなく語彙を返す（REQ-AUDIT-007）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListSystemAuditEvents")
@route("/api/admin/v1/system/audit-events")
@get
op ListSystemAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ListSystemAuditEventsSuccess_200 | ListSystemAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListSystemAuditEventsError403;

model ExportSystemAuditEventsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventListResponse;
}

union ExportSystemAuditEventsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ExportSystemAuditEventsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ExportSystemAuditEventsError400Body;
}

union ExportSystemAuditEventsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ExportSystemAuditEventsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ExportSystemAuditEventsError403Body;
}

@doc("制御面の運用者が、ListSystemAuditEvents と同じ時間の範囲と絞り込み条件で、すべてのテナントの監査イベントをエクスポートする。検索で見つけたものが、そのままダウンロードの内容になる。SIEM へのストリーミングは別の work item が扱い、このインターフェースは一回のダウンロードを返す。呼び出し元は、ListSystemAuditEvents が定める制御面の三つの条件を満たさなければならない（REQ-AUDIT-007）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ExportSystemAuditEvents")
@route("/api/admin/v1/system/audit-events/export")
@get
op ExportSystemAuditEvents(
  @query query: IdMagic.Contract.AuditEventQuery
): ExportSystemAuditEventsSuccess_200 | ExportSystemAuditEventsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ExportSystemAuditEventsError403;

model GetSystemAuditEventSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminAuditEventResponse;
}

union GetSystemAuditEventError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetSystemAuditEventError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSystemAuditEventError403Body;
}

union GetSystemAuditEventError404Body {
  IdMagic.Contract.AuditEventNotFoundError,
}

model GetSystemAuditEventError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetSystemAuditEventError404Body;
}

@doc("制御面の運用者が、任意のテナントの監査イベントを id で一つ読む。呼び出し元は、ListSystemAuditEvents が定める制御面の三つの条件を満たさなければならない。一つでも満たさない呼び出し元には、イベントが存在するかを伝えず、AccessDeniedError で拒否する（REQ-AUDIT-007）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("GetSystemAuditEvent")
@route("/api/admin/v1/system/audit-events/{id}")
@get
op GetSystemAuditEvent(
  @path id: string
): GetSystemAuditEventSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetSystemAuditEventError403 | GetSystemAuditEventError404;
}

}
