import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("SAML")
namespace IdMagic.Saml {

namespace Operations {
model SamlSingleSignOn1Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn1Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleSignOn1Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model SamlSingleSignOn1Error403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleSignOn1Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleSignOn1Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleSignOn1Error404Body;
}

@doc("SAML 2.0 の Web Browser SSO のサインイン。SP 起点のリクエストは HTTP-Redirect または HTTP-POST のバインディングで AuthnRequest として届き、Version=2.0、IssueInstant の受理範囲、Issuer、Destination、HTTP-POST のレスポンスの ProtocolBinding、ACS の URL と index の排他と対応可否、NameIDPolicy の形式をすべて検査してから、はじめてサービスプロバイダーを解決する。経路の IdP プロファイル、SP が結び付くプロファイル、Destination が一致しないときは、リクエストをフェイルクローズで拒否する。署名を求める SP では、Redirect バインディングの署名か XML 署名のどちらかを検証する。対応していないリクエストや自己矛盾するリクエストはフェイルクローズで拒否し、SAML のプロトコルエラーを HTTP-POST で返すのは、返信先として検証済みの ACS があるときだけである。ForceAuthn=true は、最近再認証していない限り呼び出し元をサインインへ送る。IsPassive=true のときは、既存のセッションが使えなければ、サインインのページへ移らずに NoPassive のレスポンスを返す。IdP 起点のリクエストは、entityID のクエリで対象の SP を指定する。未認証の呼び出し元はサインインへ送り、リクエストはその往復の後も保たれる。認証後はアプリケーションの割り当てのゲートをフェイルクローズで適用し、割り当てのないサブジェクトには何も発行しない。SP のクレームのポリシーが生成する NameID と属性を bearer の SAML アサーションに収め、割り当てたプロファイルの XML 署名の資格情報で、SP の設定に従ってアサーションとレスポンスに署名し、SAMLResponse を HTTP-POST で ACS へ自動 POST する。テナント、SP、AuthnRequest の ID は、アサーションを発行する直前に、TTL 付きのリプレイのストアへ原子的に一度だけ記録する。InResponseTo、AudienceRestriction、Recipient、Destination は互いに整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOnDefault")
@route("/saml/sso")
@get
op SamlSingleSignOn1(
  @query saml_request?: string,
  @query entity_id?: string,
  @query relay_state?: string,
  @query profile_id: string = "default"
): SamlSingleSignOn1Success_200 | SamlSingleSignOn1Success_303 | SamlSingleSignOn1Error400 | SamlSingleSignOn1Error403 | SamlSingleSignOn1Error404;

model SamlSingleSignOn2Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn2Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleSignOn2Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model SamlSingleSignOn2Error403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleSignOn2Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleSignOn2Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleSignOn2Error404Body;
}

@doc("SAML 2.0 の Web Browser SSO のサインイン。SP 起点のリクエストは HTTP-Redirect または HTTP-POST のバインディングで AuthnRequest として届き、Version=2.0、IssueInstant の受理範囲、Issuer、Destination、HTTP-POST のレスポンスの ProtocolBinding、ACS の URL と index の排他と対応可否、NameIDPolicy の形式をすべて検査してから、はじめてサービスプロバイダーを解決する。経路の IdP プロファイル、SP が結び付くプロファイル、Destination が一致しないときは、リクエストをフェイルクローズで拒否する。署名を求める SP では、Redirect バインディングの署名か XML 署名のどちらかを検証する。対応していないリクエストや自己矛盾するリクエストはフェイルクローズで拒否し、SAML のプロトコルエラーを HTTP-POST で返すのは、返信先として検証済みの ACS があるときだけである。ForceAuthn=true は、最近再認証していない限り呼び出し元をサインインへ送る。IsPassive=true のときは、既存のセッションが使えなければ、サインインのページへ移らずに NoPassive のレスポンスを返す。IdP 起点のリクエストは、entityID のクエリで対象の SP を指定する。未認証の呼び出し元はサインインへ送り、リクエストはその往復の後も保たれる。認証後はアプリケーションの割り当てのゲートをフェイルクローズで適用し、割り当てのないサブジェクトには何も発行しない。SP のクレームのポリシーが生成する NameID と属性を bearer の SAML アサーションに収め、割り当てたプロファイルの XML 署名の資格情報で、SP の設定に従ってアサーションとレスポンスに署名し、SAMLResponse を HTTP-POST で ACS へ自動 POST する。テナント、SP、AuthnRequest の ID は、アサーションを発行する直前に、TTL 付きのリプレイのストアへ原子的に一度だけ記録する。InResponseTo、AudienceRestriction、Recipient、Destination は互いに整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleSignOn")
@route("/saml/idp/{profile_id}/sso")
@get
op SamlSingleSignOn2(
  @query saml_request?: string,
  @query entity_id?: string,
  @query relay_state?: string,
  @path profile_id: string
): SamlSingleSignOn2Success_200 | SamlSingleSignOn2Success_303 | SamlSingleSignOn2Error400 | SamlSingleSignOn2Error403 | SamlSingleSignOn2Error404;

model SamlSingleSignOn3HttpRequest {
  @doc("SP 起点の AuthnRequest。バインディングに応じて deflate+base64 または base64 で表す。IdP 起点のリクエストでは空である。")
  saml_request?: string;
  @doc("IdP 起点のリクエストで、対象の SP を指す entityID。")
  entity_id?: string;
  @doc("SP が往復させる不透明なコンテキスト。SAMLResponse で変わらずに戻る。")
  relay_state?: string;
}

model SamlSingleSignOn3Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn3Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleSignOn3Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model SamlSingleSignOn3Error403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleSignOn3Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleSignOn3Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleSignOn3Error404Body;
}

@doc("SAML 2.0 の Web Browser SSO のサインイン。SP 起点のリクエストは HTTP-Redirect または HTTP-POST のバインディングで AuthnRequest として届き、Version=2.0、IssueInstant の受理範囲、Issuer、Destination、HTTP-POST のレスポンスの ProtocolBinding、ACS の URL と index の排他と対応可否、NameIDPolicy の形式をすべて検査してから、はじめてサービスプロバイダーを解決する。経路の IdP プロファイル、SP が結び付くプロファイル、Destination が一致しないときは、リクエストをフェイルクローズで拒否する。署名を求める SP では、Redirect バインディングの署名か XML 署名のどちらかを検証する。対応していないリクエストや自己矛盾するリクエストはフェイルクローズで拒否し、SAML のプロトコルエラーを HTTP-POST で返すのは、返信先として検証済みの ACS があるときだけである。ForceAuthn=true は、最近再認証していない限り呼び出し元をサインインへ送る。IsPassive=true のときは、既存のセッションが使えなければ、サインインのページへ移らずに NoPassive のレスポンスを返す。IdP 起点のリクエストは、entityID のクエリで対象の SP を指定する。未認証の呼び出し元はサインインへ送り、リクエストはその往復の後も保たれる。認証後はアプリケーションの割り当てのゲートをフェイルクローズで適用し、割り当てのないサブジェクトには何も発行しない。SP のクレームのポリシーが生成する NameID と属性を bearer の SAML アサーションに収め、割り当てたプロファイルの XML 署名の資格情報で、SP の設定に従ってアサーションとレスポンスに署名し、SAMLResponse を HTTP-POST で ACS へ自動 POST する。テナント、SP、AuthnRequest の ID は、アサーションを発行する直前に、TTL 付きのリプレイのストアへ原子的に一度だけ記録する。InResponseTo、AudienceRestriction、Recipient、Destination は互いに整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PostSamlSingleSignOn")
@route("/saml/idp/{profile_id}/sso")
@post
op SamlSingleSignOn3(
  @path profile_id: string,
  @body body: SamlSingleSignOn3HttpRequest
): SamlSingleSignOn3Success_200 | SamlSingleSignOn3Success_303 | SamlSingleSignOn3Error400 | SamlSingleSignOn3Error403 | SamlSingleSignOn3Error404;

model SamlSingleSignOn4HttpRequest {
  @doc("SP 起点の AuthnRequest。バインディングに応じて deflate+base64 または base64 で表す。IdP 起点のリクエストでは空である。")
  saml_request?: string;
  @doc("IdP 起点のリクエストで、対象の SP を指す entityID。")
  entity_id?: string;
  @doc("SP が往復させる不透明なコンテキスト。SAMLResponse で変わらずに戻る。")
  relay_state?: string;
  @doc("リクエストの経路が選んだ IdP プロファイルの id。")
  profile_id: string = "default";
}

model SamlSingleSignOn4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleSignOn4Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleSignOn4Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model SamlSingleSignOn4Error403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleSignOn4Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleSignOn4Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleSignOn4Error404Body;
}

@doc("SAML 2.0 の Web Browser SSO のサインイン。SP 起点のリクエストは HTTP-Redirect または HTTP-POST のバインディングで AuthnRequest として届き、Version=2.0、IssueInstant の受理範囲、Issuer、Destination、HTTP-POST のレスポンスの ProtocolBinding、ACS の URL と index の排他と対応可否、NameIDPolicy の形式をすべて検査してから、はじめてサービスプロバイダーを解決する。経路の IdP プロファイル、SP が結び付くプロファイル、Destination が一致しないときは、リクエストをフェイルクローズで拒否する。署名を求める SP では、Redirect バインディングの署名か XML 署名のどちらかを検証する。対応していないリクエストや自己矛盾するリクエストはフェイルクローズで拒否し、SAML のプロトコルエラーを HTTP-POST で返すのは、返信先として検証済みの ACS があるときだけである。ForceAuthn=true は、最近再認証していない限り呼び出し元をサインインへ送る。IsPassive=true のときは、既存のセッションが使えなければ、サインインのページへ移らずに NoPassive のレスポンスを返す。IdP 起点のリクエストは、entityID のクエリで対象の SP を指定する。未認証の呼び出し元はサインインへ送り、リクエストはその往復の後も保たれる。認証後はアプリケーションの割り当てのゲートをフェイルクローズで適用し、割り当てのないサブジェクトには何も発行しない。SP のクレームのポリシーが生成する NameID と属性を bearer の SAML アサーションに収め、割り当てたプロファイルの XML 署名の資格情報で、SP の設定に従ってアサーションとレスポンスに署名し、SAMLResponse を HTTP-POST で ACS へ自動 POST する。テナント、SP、AuthnRequest の ID は、アサーションを発行する直前に、TTL 付きのリプレイのストアへ原子的に一度だけ記録する。InResponseTo、AudienceRestriction、Recipient、Destination は互いに整合させる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PostSamlSingleSignOnDefault")
@route("/saml/sso")
@post
op SamlSingleSignOn4(
  @body body: SamlSingleSignOn4HttpRequest
): SamlSingleSignOn4Success_200 | SamlSingleSignOn4Success_303 | SamlSingleSignOn4Error400 | SamlSingleSignOn4Error403 | SamlSingleSignOn4Error404;

model SamlSingleLogout1Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout1Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleLogout1Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleLogout1Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleLogout1Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleLogout1Error404Body;
}

@doc("SAML の Single Logout。LogoutRequest は HTTP-Redirect または HTTP-POST のバインディングで届き、Issuer から SP を解決し、経路の IdP プロファイルを SP が結び付くプロファイルと照合し、署名を求める SP では署名を検証する。Destination は選んだプロファイルの SLO のエンドポイントでなければならず、返信は登録済みの SingleLogoutService にだけ送る。ローカルのセッションを破棄し、割り当てたプロファイルの鍵で署名した LogoutResponse を、登録済みの SLO の URL へ返す。entityID または sp を指定したローカルのログアウトも受け付けるが、解決できない宛先や登録されていない宛先へリダイレクトすることはないので、オープンリダイレクトにはならない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogoutDefault")
@route("/saml/slo")
@get
op SamlSingleLogout1(
  @query entity_id?: string,
  @query relay_state?: string,
  @query profile_id: string = "default"
): SamlSingleLogout1Success_200 | SamlSingleLogout1Success_303 | SamlSingleLogout1Error400 | SamlSingleLogout1Error404;

model SamlSingleLogout2HttpRequest {
  @doc("ログアウトを求めた SP の entityID。")
  entity_id?: string;
  relay_state?: string;
  @doc("リクエストの経路が選んだ IdP プロファイルの id。")
  profile_id: string = "default";
}

model SamlSingleLogout2Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout2Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleLogout2Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleLogout2Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleLogout2Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleLogout2Error404Body;
}

@doc("SAML の Single Logout。LogoutRequest は HTTP-Redirect または HTTP-POST のバインディングで届き、Issuer から SP を解決し、経路の IdP プロファイルを SP が結び付くプロファイルと照合し、署名を求める SP では署名を検証する。Destination は選んだプロファイルの SLO のエンドポイントでなければならず、返信は登録済みの SingleLogoutService にだけ送る。ローカルのセッションを破棄し、割り当てたプロファイルの鍵で署名した LogoutResponse を、登録済みの SLO の URL へ返す。entityID または sp を指定したローカルのログアウトも受け付けるが、解決できない宛先や登録されていない宛先へリダイレクトすることはないので、オープンリダイレクトにはならない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PostSamlSingleLogoutDefault")
@route("/saml/slo")
@post
op SamlSingleLogout2(
  @body body: SamlSingleLogout2HttpRequest
): SamlSingleLogout2Success_200 | SamlSingleLogout2Success_303 | SamlSingleLogout2Error400 | SamlSingleLogout2Error404;

model SamlSingleLogout3Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout3Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleLogout3Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleLogout3Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleLogout3Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleLogout3Error404Body;
}

@doc("SAML の Single Logout。LogoutRequest は HTTP-Redirect または HTTP-POST のバインディングで届き、Issuer から SP を解決し、経路の IdP プロファイルを SP が結び付くプロファイルと照合し、署名を求める SP では署名を検証する。Destination は選んだプロファイルの SLO のエンドポイントでなければならず、返信は登録済みの SingleLogoutService にだけ送る。ローカルのセッションを破棄し、割り当てたプロファイルの鍵で署名した LogoutResponse を、登録済みの SLO の URL へ返す。entityID または sp を指定したローカルのログアウトも受け付けるが、解決できない宛先や登録されていない宛先へリダイレクトすることはないので、オープンリダイレクトにはならない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SamlSingleLogout")
@route("/saml/idp/{profile_id}/slo")
@get
op SamlSingleLogout3(
  @query entity_id?: string,
  @query relay_state?: string,
  @path profile_id: string
): SamlSingleLogout3Success_200 | SamlSingleLogout3Success_303 | SamlSingleLogout3Error400 | SamlSingleLogout3Error404;

model SamlSingleLogout4HttpRequest {
  @doc("ログアウトを求めた SP の entityID。")
  entity_id?: string;
  relay_state?: string;
}

model SamlSingleLogout4Success_200 {
  @statusCode statusCode: 200;
}

model SamlSingleLogout4Success_303 {
  @statusCode statusCode: 303;
}

model SamlSingleLogout4Error400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

union SamlSingleLogout4Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model SamlSingleLogout4Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: SamlSingleLogout4Error404Body;
}

@doc("SAML の Single Logout。LogoutRequest は HTTP-Redirect または HTTP-POST のバインディングで届き、Issuer から SP を解決し、経路の IdP プロファイルを SP が結び付くプロファイルと照合し、署名を求める SP では署名を検証する。Destination は選んだプロファイルの SLO のエンドポイントでなければならず、返信は登録済みの SingleLogoutService にだけ送る。ローカルのセッションを破棄し、割り当てたプロファイルの鍵で署名した LogoutResponse を、登録済みの SLO の URL へ返す。entityID または sp を指定したローカルのログアウトも受け付けるが、解決できない宛先や登録されていない宛先へリダイレクトすることはないので、オープンリダイレクトにはならない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PostSamlSingleLogout")
@route("/saml/idp/{profile_id}/slo")
@post
op SamlSingleLogout4(
  @path profile_id: string,
  @body body: SamlSingleLogout4HttpRequest
): SamlSingleLogout4Success_200 | SamlSingleLogout4Success_303 | SamlSingleLogout4Error400 | SamlSingleLogout4Error404;

model PublishSamlMetadata1Success_200 {
  @statusCode statusCode: 200;
}

union PublishSamlMetadata1Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model PublishSamlMetadata1Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: PublishSamlMetadata1Error404Body;
}

@doc("SAML 2.0 の IdP のメタデータを公開する。デフォルトの経路はレルムのデフォルトのプロファイルを、プロファイルの経路は指定したプロファイルを公開する。その entityID、SSO と SLO のエンドポイント、対応する NameID の形式、プロファイルのスコープの XmlFederationSigning の証明書を、<EntityDescriptor><IDPSSODescriptor> として返す。有効な証明書に加えて、ローテーションの猶予の内にある verifying の証明書も載せる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishDefaultSamlMetadata")
@route("/saml/metadata")
@get
op PublishSamlMetadata1(

): PublishSamlMetadata1Success_200 | PublishSamlMetadata1Error404;

model PublishSamlMetadata2Success_200 {
  @statusCode statusCode: 200;
}

union PublishSamlMetadata2Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model PublishSamlMetadata2Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: PublishSamlMetadata2Error404Body;
}

@doc("SAML 2.0 の IdP のメタデータを公開する。デフォルトの経路はレルムのデフォルトのプロファイルを、プロファイルの経路は指定したプロファイルを公開する。その entityID、SSO と SLO のエンドポイント、対応する NameID の形式、プロファイルのスコープの XmlFederationSigning の証明書を、<EntityDescriptor><IDPSSODescriptor> として返す。有効な証明書に加えて、ローテーションの猶予の内にある verifying の証明書も載せる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishSamlMetadata")
@route("/saml/idp/{profile_id}/metadata")
@get
op PublishSamlMetadata2(
  @path profile_id: string
): PublishSamlMetadata2Success_200 | PublishSamlMetadata2Error404;

model DownloadSamlSigningCertificate1Success_200 {
  @statusCode statusCode: 200;
}

union DownloadSamlSigningCertificate1Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model DownloadSamlSigningCertificate1Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DownloadSamlSigningCertificate1Error404Body;
}

@doc("SAML の SP で信頼を手作業で設定するために、リクエストのテナントの有効な XmlFederationSigning の X.509 証明書を PEM で公開する。ローテーションの重複期間内の verifying の証明書を含む、信頼の基準となる文書は引き続き PublishSamlMetadata であり、このエンドポイントは最初に信頼を設定するための有効な証明書だけを返す。秘密鍵は返さない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadDefaultSamlSigningCertificate")
@route("/saml/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate1(

): DownloadSamlSigningCertificate1Success_200 | DownloadSamlSigningCertificate1Error404;

model DownloadSamlSigningCertificate2Success_200 {
  @statusCode statusCode: 200;
}

union DownloadSamlSigningCertificate2Error404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model DownloadSamlSigningCertificate2Error404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DownloadSamlSigningCertificate2Error404Body;
}

@doc("SAML の SP で信頼を手作業で設定するために、リクエストのテナントの有効な XmlFederationSigning の X.509 証明書を PEM で公開する。ローテーションの重複期間内の verifying の証明書を含む、信頼の基準となる文書は引き続き PublishSamlMetadata であり、このエンドポイントは最初に信頼を設定するための有効な証明書だけを返す。秘密鍵は返さない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DownloadSamlSigningCertificate")
@route("/saml/idp/{profile_id}/signing-certificate.pem")
@get
op DownloadSamlSigningCertificate2(
  @path profile_id: string
): DownloadSamlSigningCertificate2Success_200 | DownloadSamlSigningCertificate2Error404;

model CreateSamlIdentityProviderProfileSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
}

union CreateSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateSamlIdentityProviderProfileError400Body;
}

union CreateSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateSamlIdentityProviderProfileError403Body;
}

@doc("テナントの管理者が、追加の共有または専用の SAML IdP プロファイルを作成する。デフォルトのプロファイルはこの方法では作成できない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("CreateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles")
@post
op CreateSamlIdentityProviderProfile(
  @body body: IdMagic.Contract.SamlIdentityProviderProfileRequest
): CreateSamlIdentityProviderProfileSuccess_201 | CreateSamlIdentityProviderProfileError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateSamlIdentityProviderProfileError403;

model ListSamlIdentityProviderProfilesHttpResponse {
  profiles: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse[];
}

model ListSamlIdentityProviderProfilesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSamlIdentityProviderProfilesHttpResponse;
}

union ListSamlIdentityProviderProfilesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSamlIdentityProviderProfilesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSamlIdentityProviderProfilesError403Body;
}

@doc("テナントの SAML IdP プロファイルを、デフォルトのプロファイルを先頭に、残りを名前の昇順で一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:read"])
@TypeSpec.OpenAPI.operationId("ListSamlIdentityProviderProfiles")
@route("/api/admin/v1/saml/idp-profiles")
@get
op ListSamlIdentityProviderProfiles(

): ListSamlIdentityProviderProfilesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListSamlIdentityProviderProfilesError403;

model UpdateSamlIdentityProviderProfileSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SamlIdentityProviderProfileAdminResponse;
}

union UpdateSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError400Body;
}

union UpdateSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError403Body;
}

union UpdateSamlIdentityProviderProfileError409Body {
  IdMagic.Contract.DefaultIdPProfileError,
}

model UpdateSamlIdentityProviderProfileError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError409Body;
}

union UpdateSamlIdentityProviderProfileError404Body {
  IdMagic.Contract.SamlIdentityProviderProfileNotFoundError,
}

model UpdateSamlIdentityProviderProfileError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateSamlIdentityProviderProfileError404Body;
}

@doc("テナントの管理者が、追加の IdP プロファイルの名前またはモードを変更する。デフォルトのプロファイルは変更できず、複数の SP がプロファイルを参照している間は dedicated への変更を拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("UpdateSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@put
op UpdateSamlIdentityProviderProfile(
  @path profile_id: string,
  @body body: IdMagic.Contract.SamlIdentityProviderProfileRequest
): UpdateSamlIdentityProviderProfileSuccess_200 | UpdateSamlIdentityProviderProfileError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateSamlIdentityProviderProfileError403 | UpdateSamlIdentityProviderProfileError404 | UpdateSamlIdentityProviderProfileError409;

model DeleteSamlIdentityProviderProfileSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteSamlIdentityProviderProfileError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteSamlIdentityProviderProfileError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError400Body;
}

union DeleteSamlIdentityProviderProfileError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteSamlIdentityProviderProfileError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError403Body;
}

union DeleteSamlIdentityProviderProfileError409Body {
  IdMagic.Contract.DefaultIdPProfileError,
  IdMagic.Contract.IdPProfileInUseError,
}

model DeleteSamlIdentityProviderProfileError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlIdentityProviderProfileError409Body;
}

@doc("何からも参照されていない追加の IdP プロファイルを削除する。デフォルトのプロファイルと、SP がまだ参照しているプロファイルは、どちらも拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("DeleteSamlIdentityProviderProfile")
@route("/api/admin/v1/saml/idp-profiles/{profile_id}")
@delete
op DeleteSamlIdentityProviderProfile(
  @path profile_id: string
): DeleteSamlIdentityProviderProfileSuccess_204 | DeleteSamlIdentityProviderProfileError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteSamlIdentityProviderProfileError403 | DeleteSamlIdentityProviderProfileError409;

model RegisterSamlServiceProviderSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SamlServiceProvider;
}

model RegisterSamlServiceProviderSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.SamlServiceProvider;
}

union RegisterSamlServiceProviderError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterSamlServiceProviderError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterSamlServiceProviderError400Body;
}

union RegisterSamlServiceProviderError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterSamlServiceProviderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterSamlServiceProviderError403Body;
}

@doc("SAML のサービスプロバイダーを登録または更新（upsert）する。entityID、許可する ACS の URL、NameID の形式、オーディエンス、署名のポリシー、クレームのポリシーを受け取り、テナントの境界の中に保存する。entityID と閉じた ACS の URL の集合はどちらも必須である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("RegisterSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@post
op RegisterSamlServiceProvider(
  @body body: IdMagic.Contract.SamlServiceProviderRequest
): RegisterSamlServiceProviderSuccess_200 | RegisterSamlServiceProviderSuccess_201 | RegisterSamlServiceProviderError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterSamlServiceProviderError403;

model ListSamlServiceProvidersHttpResponse {
  service_providers: IdMagic.Contract.SamlServiceProvider[];
}

model ListSamlServiceProvidersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSamlServiceProvidersHttpResponse;
}

union ListSamlServiceProvidersError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSamlServiceProvidersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSamlServiceProvidersError403Body;
}

@doc("テナントの SAML のサービスプロバイダーを entityID の昇順で一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:read"])
@TypeSpec.OpenAPI.operationId("ListSamlServiceProviders")
@route("/api/admin/v1/saml/service-providers")
@get
op ListSamlServiceProviders(

): ListSamlServiceProvidersSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListSamlServiceProvidersError403;

model DeleteSamlServiceProviderSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteSamlServiceProviderError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteSamlServiceProviderError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError400Body;
}

union DeleteSamlServiceProviderError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteSamlServiceProviderError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError403Body;
}

union DeleteSamlServiceProviderError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model DeleteSamlServiceProviderError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteSamlServiceProviderError409Body;
}

@doc("entityID が一致する、カタログの外の SAML のサービスプロバイダーを冪等に削除する。`application_id` が null でない SP は ApplicationOwnedProtocolError（HTTP 409）で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["saml:write"])
@TypeSpec.OpenAPI.operationId("DeleteSamlServiceProvider")
@route("/api/admin/v1/saml/service-providers")
@delete
op DeleteSamlServiceProvider(
  @query entity_id: string
): DeleteSamlServiceProviderSuccess_204 | DeleteSamlServiceProviderError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteSamlServiceProviderError403 | DeleteSamlServiceProviderError409;
}

}
