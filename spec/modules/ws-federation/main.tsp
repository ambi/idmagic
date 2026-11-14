import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("WS-Federation")
namespace IdMagic.WsFederation {

namespace Operations {
model WsFederationSignInSuccess_200 {
  @statusCode statusCode: 200;
}

model WsFederationSignInSuccess_303 {
  @statusCode statusCode: 303;
  @header wwwAuthenticate?: string;
}

model WsFederationSignInError400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model WsFederationNegotiateChallenge {
  @statusCode statusCode: 401;
  @header wwwAuthenticate?: string;
  @header contentType: "text/plain";
  @body body: string;
}

model WsFederationPolicyForbidden {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.AccessDeniedError;
}

model WsFederationRateLimited {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @header("Retry-After") retryAfter: int32;
  @body body: IdMagic.Contract.RateLimitedError;
}

model WsFederationQuotaExceeded {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.QuotaExceededError;
}

model WsFederationSignInError403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

@doc("WS-Federation のパッシブリクエスターのプロファイルの唯一の HTTP エンドポイント。`wa` でサインインとサインアウトを振り分ける。サインインは `wtrealm` から Relying Party を解決し、`wreply` を許可した集合に限るので、オープンリダイレクトにはならない。未認証の呼び出し元はサインインへ送る。認証済みの呼び出し元には、Relying Party のクレームのポリシーが組み立てたクレームを、署名した SAML アサーション（デフォルトは Entra が求める SAML 1.1、RP が設定していれば SAML 2.0）に収め、RSTR で包んで自動 POST する。認証が `wfresh` の示す最大の経過時間より古いときは呼び出し元を再認証へ送り、`wauth` が求める認証方式に従う。サインアウトの意味は WsFederationSignOut が定める。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsFederationSignIn")
@route("/wsfed")
@get
op WsFederationSignIn(
  @query wa: string,
  @query wtrealm: string,
  @query wreply?: string,
  @query wctx?: string,
  @query wfresh?: string,
  @query wauth?: string,
  @query whr?: string
): WsFederationSignInSuccess_200 | WsFederationSignInSuccess_303 | WsFederationSignInError400 | WsFederationSignInError403 | WsFederationNegotiateChallenge | WsFederationPolicyForbidden | WsFederationRateLimited | WsFederationQuotaExceeded | IdMagic.Contract.AuthenticationRequiredResponse;

model WsTrustIssueSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/soap+xml; charset=utf-8";
  @body body: string;
}

model WsTrustIssueError400 {
  @statusCode statusCode: 400;
  @header contentType: "text/plain";
  @body body: string;
}

model WsTrustIssueError403 {
  @statusCode statusCode: 403;
  @header contentType: "text/plain";
  @body body: string;
}

model WsTrustIssueError401 {
  @statusCode statusCode: 401;
  @header contentType: "text/plain";
  @body body: string;
}

@doc("WS-Trust 1.3 のアクティブ STS の Issue エンドポイント。UsernameToken を検証し、`To` を現在のレルムのアクティブ STS のエンドポイントに、`Action` と `RequestType` を Issue に、`KeyType` を Bearer に限り、`AppliesTo` を登録済みの WsFedRelyingParty へ解決する。RP のクレームのポリシーが組み立てたクレームを署名した SAML アサーションに収め、RSTR として返す。未登録の `AppliesTo`、期限切れの Timestamp、形式の誤った `To`、`Action`、`RequestType`、`KeyType`、`TokenType` は、いずれもフェイルクローズで拒否する。")
@TypeSpec.OpenAPI.operationId("WsTrustIssue")
@route("/trust/usernamemixed")
@post
op WsTrustIssue(
  @header contentType: "application/soap+xml",
  @body body: string
): WsTrustIssueSuccess_200 | WsTrustIssueError400 | IdMagic.Contract.AuthenticationRequiredResponse | WsTrustIssueError401 | WsTrustIssueError403;

model WsTrustMetadataExchangeSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("WS-MetadataExchange の文書。`mex:Metadata` の XML で返す。")
  @body body: string;
}

@doc("WS-Trust の MEX エンドポイント。アクティブ STS の usernamemixed のエンドポイントを、TLS と UsernameToken を求める WS-SecurityPolicy の方針、その方針を参照する SOAP 1.2 の WS-Trust 1.3 の Issue（soapAction は `http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue`）の束縛、束縛を参照する WS-Addressing のエンドポイント参照を持つポートとして WSDL で示し、いずれもレルムの発行者から導く。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("WsTrustMetadataExchange")
@route("/trust/mex")
@get
op WsTrustMetadataExchange(

): WsTrustMetadataExchangeSuccess_200;

model PublishWsFederationMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @header contentType: "application/xml; charset=utf-8";
  @doc("AD FS 互換の `federationmetadata.xml` の文書。`EntityDescriptor` の XML で返す。")
  @body body: string;
}

@doc("WS-Federation と WS-Trust のメタデータの公開。レルムの entityID、パッシブのエンドポイント、アクティブ STS のエンドポイント、MEX のエンドポイント、テナントの XmlFederationSigning の証明書を、AD FS 互換の `federationmetadata.xml` として公開する。有効な証明書に加えて、ローテーションの猶予の内にある verifying の証明書も載せる。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PublishWsFederationMetadata")
@route("/federationmetadata/2007-06/federationmetadata.xml")
@get
op PublishWsFederationMetadata(

): PublishWsFederationMetadataSuccess_200;

model ConfigureEntraFederationHttpRequest {
  @doc("Entra で確認済みの DNS ドメイン。")
  domain: string;
  @doc("IssuerUri。空のままにすると、ドメインから安定して導く。")
  issuer_uri?: string;
  @doc("ImmutableID の元になる、変わらないユーザーの属性。Directory Connector が AD から取り込む値は組み込みの属性 `on_premises_immutable_id` にある。")
  source_anchor_attribute: string;
  @doc("Entra が `wreply` として使う URL。空のままにすると、Microsoft Online のデフォルトの URL を使う。")
  reply_url?: string;
  @doc("このドメインを持つ Entra のテナントの ID（GUID）。省くと、ドメインの組はテナント ID を持たない。")
  entra_tenant_id?: string;
}

model ConfigureEntraFederationHttpResponse {
  profile: IdMagic.Contract.EntraFederationProfile;
  relying_party: IdMagic.Contract.WsFedRelyingParty;
  @doc("Microsoft Graph PowerShell のパラメーター名をキーとする。IssuerUri、PassiveSignInUri、ActiveSignInUri、MetadataExchangeUri、SignOutUri、SigningCertificate、PreferredAuthenticationProtocol、FederatedIdpMfaBehavior を含む。SigningCertificate は有効な XML 署名証明書の公開部分の base64 とし、準備済みの XML 鍵があるときはその証明書の公開部分の base64 を NextSigningCertificate として追加する。")
  powershell: Record<string>;
  @doc("値を埋めた作成と更新のコマンド。Microsoft Graph に接続した PowerShell で管理者が実行する。更新は同じドメインの既存の構成から ID を取得する。")
  commands: {
    create: string;
    update: string;
  };
  @doc("この定型設定が提供しない機能のコード。画面は選択した言語の辞書で表示する。")
  known_limitations: {
    code: "hybrid_join_not_supported";
  }[];
  @doc("取り込み元の管理下の User のうち、UPN がどの構成済みのドメインにも属さない削除されていない User。該当する User がいなければ省く。構成は拒否しない。")
  unfederated_source_users?: {
    @doc("該当する User の数。")
    @minValue(1)
    user_count: int32;
    @doc("該当する User のうち、ユーザー名の順で先頭の 5 人までのユーザー名。")
    @minItems(1)
    @maxItems(5)
    sample_usernames: string[];
  };
}

model ConfigureEntraFederationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ConfigureEntraFederationHttpResponse;
}

model ConfigureEntraFederationSuccess_201 {
  @statusCode statusCode: 201;
  @body body: ConfigureEntraFederationHttpResponse;
}

union ConfigureEntraFederationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfigureEntraFederationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError400Body;
}

union ConfigureEntraFederationError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ConfigureEntraFederationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError403Body;
}

union ConfigureEntraFederationError422Body {
  IdMagic.Contract.SourceAnchorNotUniqueError,
  IdMagic.Contract.SourceAnchorEditableError,
  IdMagic.Contract.SourceAnchorValueError,
}

model ConfigureEntraFederationError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ConfigureEntraFederationError422Body;
}

@doc("Microsoft Entra のドメインフェデレーションの定型設定を構成する。確認済みのドメイン、IssuerUri、sourceAnchor の属性を受け取り、属性が unique で本人が編集できないこと、対象の利用者の sourceAnchor の値、IssuerUri の重複を確かめる。テナントに一つの Entra の RP にドメインと IssuerUri を追加または置き換え、Microsoft Graph PowerShell の設定値と作成と更新のコマンドを返す。SigningCertificate は署名器の有効な証明書の公開部分の base64 とする。制限はコードで返す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("ConfigureEntraFederation")
@route("/api/admin/v1/wsfed/entra-federation")
@post
op ConfigureEntraFederation(
  @body body: ConfigureEntraFederationHttpRequest
): ConfigureEntraFederationSuccess_201 | ConfigureEntraFederationSuccess_200 | ConfigureEntraFederationError400 | IdMagic.Contract.AuthenticationRequiredResponse | ConfigureEntraFederationError403 | ConfigureEntraFederationError422;

union EntraFederationAdminError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model EntraFederationAdminError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EntraFederationAdminError403Body;
}

model EntraFederationDomainError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.EntraFederationDomainNotFoundError;
}

model EntraFederationDomainSummary {
  ...IdMagic.Contract.EntraDomainFederation;
  verification: IdMagic.Contract.EntraFederationVerification;
}

model ListEntraFederationDomainsHttpResponse {
  @doc("構成済みのドメイン。ドメインの順に並べる。Entra の構成がなければ空である。")
  domains: EntraFederationDomainSummary[];
}

model ListEntraFederationDomainsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListEntraFederationDomainsHttpResponse;
}

@doc("Entra のドメインフェデレーションで構成したドメインを、確認の状態とともに一覧する。RP と観測記録を変えない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:read"])
@TypeSpec.OpenAPI.operationId("ListEntraFederationDomains")
@route("/api/admin/v1/wsfed/entra-federation/domains")
@get
op ListEntraFederationDomains(

): ListEntraFederationDomainsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | EntraFederationAdminError403;

model EntraFederationDomainHttpResponse {
  domain: IdMagic.Contract.EntraDomainFederation;
  profile: IdMagic.Contract.EntraFederationProfile;
  @doc("Entra が `wreply` として使う URL。")
  reply_url: string;
  @doc("構成の応答と同じ、Microsoft Graph PowerShell のパラメーター名をキーとする期待値。")
  powershell: Record<string>;
  @doc("構成の応答と同じ、値を埋めた作成と更新のコマンド。")
  commands: {
    create: string;
    update: string;
  };
  @doc("Graph PowerShell の手順。`connect` は `Domain.Read.All` と `Domain-InternalFederation.ReadWrite.All` で、`connect_read_only` は `Domain.Read.All` と `Domain-InternalFederation.Read.All` で接続し、ドメインの組が Entra のテナント ID を持てばそのテナントに限る。`backup` はフェデレーションの構成だけを書き出し、`compare` は期待値との差分を表示し、`apply` は既存の構成があれば更新し、なければ作成し、`read_back` は観測文書を書き出す。")
  procedure: {
    connect: string;
    connect_read_only: string;
    backup: string;
    compare: string;
    apply: string;
    read_back: string;
  };
  certificates: {
    signing: IdMagic.Contract.EntraCertificateSummary;
    next?: IdMagic.Contract.EntraCertificateSummary;
  };
  verification: IdMagic.Contract.EntraFederationVerification;
  @doc("この定型設定が提供しない機能のコード。")
  known_limitations: {
    code: "hybrid_join_not_supported";
  }[];
  @doc("構成の応答と同じく、取り込み元の管理下の User のうち、UPN がどの構成済みのドメインにも属さない削除されていない User。該当する User がいなければ省く。")
  unfederated_source_users?: {
    @minValue(1)
    user_count: int32;
    @minItems(1)
    @maxItems(5)
    sample_usernames: string[];
  };
}

model GetEntraFederationDomainSuccess_200 {
  @statusCode statusCode: 200;
  @body body: EntraFederationDomainHttpResponse;
}

@doc("Entra のドメインフェデレーションで構成した一つのドメインの、保存した設定、Graph の期待値、証明書、コマンド、PowerShell の手順、確認の状態を返す。RP と観測記録を変えないので、何度でも再取得できる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:read"])
@TypeSpec.OpenAPI.operationId("GetEntraFederationDomain")
@route("/api/admin/v1/wsfed/entra-federation/domains/{domain_name}")
@get
op GetEntraFederationDomain(
  @path domain_name: string
): GetEntraFederationDomainSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | EntraFederationAdminError403 | EntraFederationDomainError404;

model RecordEntraFederationObservationHttpResponse {
  verification: IdMagic.Contract.EntraFederationVerification;
}

model RecordEntraFederationObservationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RecordEntraFederationObservationHttpResponse;
}

union RecordEntraFederationObservationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RecordEntraFederationObservationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RecordEntraFederationObservationError400Body;
}

model RecordEntraFederationObservationError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.EntraObservationOutdatedError;
}

@doc("Graph PowerShell の読み戻しの手順が書き出した観測文書を取り込み、現在の期待値と比べ、テナントとドメインごとに最新の一件の観測記録として保存する。RP を変えない。本文は 64 KiB までで、未知の項目、形式の違い、必須の項目の欠落、異なるドメイン、5 件以上の構成、確認時刻より 5 分を超えて後の取得時刻を拒否する。保存済みの記録より前の取得時刻は 409 で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("RecordEntraFederationObservation")
@route("/api/admin/v1/wsfed/entra-federation/domains/{domain_name}/observations")
@post
op RecordEntraFederationObservation(
  @path domain_name: string,
  @body body: IdMagic.Contract.EntraFederationObservation
): RecordEntraFederationObservationSuccess_200 | RecordEntraFederationObservationError400 | IdMagic.Contract.AuthenticationRequiredResponse | EntraFederationAdminError403 | EntraFederationDomainError404 | RecordEntraFederationObservationError409;

model RemoveEntraFederationDomainSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.RemoveEntraFederationDomainResponse;
}

union RemoveEntraFederationDomainError422Body {
  IdMagic.Contract.EntraFederationStillActiveError,
  IdMagic.Contract.EntraRollbackUnconfirmedError,
}

model RemoveEntraFederationDomainError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RemoveEntraFederationDomainError422Body;
}

@doc("Entra のドメインフェデレーションから一つのドメインを解除する。構成済みのドメインなら、Microsoft のレルム探索にドメインの送り先を問い合わせる。探索が `Federated` で受動のエンドポイントへ送ると答えたら EntraFederationStillActiveError で拒否する。`Managed` か IdMagic を指さない `Federated` なら解除する。探索が失敗するか、無効か、ほかの値を答えたら、取り込み済みの観測記録が `stale` でなく切り戻しを示すときだけ解除し、そうでなければ EntraRollbackUnconfirmedError で拒否する。解除は、行を固定して読み直した RP からそのドメインの組だけを外して保存し、ほかのドメインの組、sourceAnchor の属性、エンドポイント、署名鍵は変えない。最後のドメインなら RP を削除する。解除したドメインの観測記録を消し、EntraFederationDomainRemoved を発行する。構成していないドメインは探索を呼ばずに 404 で拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("RemoveEntraFederationDomain")
@route("/api/admin/v1/wsfed/entra-federation/domains/{domain_name}")
@delete
op RemoveEntraFederationDomain(
  @path domain_name: string
): RemoveEntraFederationDomainSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | EntraFederationAdminError403 | EntraFederationDomainError404 | RemoveEntraFederationDomainError422;

model RegisterWsFedRelyingPartySuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.WsFedRelyingPartyRequest;
}

model RegisterWsFedRelyingPartySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WsFedRelyingParty;
}

union RegisterWsFedRelyingPartyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterWsFedRelyingPartyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError400Body;
}

union RegisterWsFedRelyingPartyError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError403Body;
}

union RegisterWsFedRelyingPartyError409Body {
  IdMagic.Contract.EntraRelyingPartyError,
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model RegisterWsFedRelyingPartyError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RegisterWsFedRelyingPartyError409Body;
}

@doc("管理者が、WS-Federation の Relying Party の信頼を登録または更新する。`wtrealm`、`reply_urls`、`audience`、`token_type`、`claim_policy` を契約として保存し、`reply_urls` は空でない閉じた集合である。同じ `wtrealm` の RP が Entra の定型設定を持てば EntraRelyingPartyError（HTTP 409）で、`application_id` が null でなければ ApplicationOwnedProtocolError（HTTP 409）で拒否する。この操作は AdminFederationTrustsManage が守る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("RegisterWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@post
op RegisterWsFedRelyingParty(
  @body body: IdMagic.Contract.WsFedRelyingPartyRequest
): RegisterWsFedRelyingPartySuccess_201 | RegisterWsFedRelyingPartySuccess_200 | RegisterWsFedRelyingPartyError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterWsFedRelyingPartyError403 | RegisterWsFedRelyingPartyError409;

model ListWsFedRelyingPartiesHttpResponse {
  relying_parties: IdMagic.Contract.WsFedRelyingParty[];
}

model ListWsFedRelyingPartiesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListWsFedRelyingPartiesHttpResponse;
}

union ListWsFedRelyingPartiesError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListWsFedRelyingPartiesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWsFedRelyingPartiesError403Body;
}

@doc("管理者が、自身のテナントの WS-Federation の Relying Party の信頼を一覧する。この操作は AdminFederationTrustsManage が守る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:read"])
@TypeSpec.OpenAPI.operationId("ListWsFedRelyingParties")
@route("/api/admin/v1/wsfed/relying-parties")
@get
op ListWsFedRelyingParties(

): ListWsFedRelyingPartiesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListWsFedRelyingPartiesError403;

model DeleteWsFedRelyingPartySuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteWsFedRelyingPartyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteWsFedRelyingPartyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError400Body;
}

union DeleteWsFedRelyingPartyError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteWsFedRelyingPartyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError403Body;
}

union DeleteWsFedRelyingPartyError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
  IdMagic.Contract.EntraRelyingPartyError,
}

model DeleteWsFedRelyingPartyError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteWsFedRelyingPartyError409Body;
}

@doc("管理者が、自身のテナントの、カタログの外にある WS-Federation の Relying Party の信頼を `wtrealm` で削除する。`application_id` が null でない RP は ApplicationOwnedProtocolError（HTTP 409）で拒否する。Entra のドメインフェデレーションの定型設定を持つ RP は、すべてのドメインのサインインを止めるので EntraRelyingPartyError（HTTP 409）で拒否し、RemoveEntraFederationDomain でドメインごとに解除させる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["wsfed:write"])
@TypeSpec.OpenAPI.operationId("DeleteWsFedRelyingParty")
@route("/api/admin/v1/wsfed/relying-parties")
@delete
op DeleteWsFedRelyingParty(
  @query wtrealm: string
): DeleteWsFedRelyingPartySuccess_204 | DeleteWsFedRelyingPartyError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteWsFedRelyingPartyError403 | DeleteWsFedRelyingPartyError409;
}

}
