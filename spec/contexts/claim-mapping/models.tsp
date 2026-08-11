namespace IdMagic.Contract {

@doc("Tenancy context が publish する属性開示範囲の published language stub。Private だけが claim 発行の fail-closed floor で常に除外される。")
enum AttrVisibility {
  Private: "Private",
  SelfReadable: "SelfReadable",
  AdminReadable: "AdminReadable",
  ClaimExposed: "ClaimExposed",
}


@doc("Tenancy context が publish する属性定義の published language stub。claim 発行時の属性可視性判定にのみ使う。")
model UserAttributeDef {
  @minLength(1)
  key: string;
  visibility: AttrVisibility = AttrVisibility.Private;
}


@doc("claim 解決が fail-closed で拒否された。required source が解決できない、\nvisibility=Private の属性を rule / NameID の source に指定している、または reserved claim type\n(iss/sub/aud/exp/iat/nbf/jti/azp/nonce/at_hash/c_hash/acr/amr/sid) を claim_type に指定している\nのいずれか。")
model ClaimReleaseDeniedError {

}


@doc("1 つの出力 claim の宣言的 mapping。出力 claim 型 (URI) を source から組み立てる。")
model ClaimMappingRule {
  @doc("出力する claim 型 URI (例: http://schemas.xmlsoap.org/claims/UPN)。")
  @minLength(1)
  claim_type: string;
  @doc("claim 値の供給元。")
  source: ClaimMappingSource;
  @doc("source=user_attribute のとき参照する属性名。")
  source_key?: string;
  @doc("source=fixed のとき出力する固定値。")
  fixed_value?: string;
  @doc("true のとき source が解決できなければ発行を拒否する (fail-closed)。")
  required: boolean = false;
}


@doc("claim 値の供給元。user_attribute は identity principal の属性、fixed は静的な固定値、 nameid は NameIdConfiguration が解決した NameID 値を claim にも反映する場合に使う。")
enum ClaimMappingSource {
  user_attribute: "user_attribute",
  fixed: "fixed",
  nameid: "nameid",
}


@doc("発行 assertion の NameID (subject 識別子) の format と供給元。")
model NameIdConfiguration {
  @doc("NameID format URI (例: urn:oasis:names:tc:SAML:2.0:nameid-format:persistent)。")
  @minLength(1)
  format: string;
  @doc("NameID 値を供給する identity 属性名。")
  @minLength(1)
  source_attribute: string;
}


@doc("claim 発行エンジンの出力。1 つの claim 型と、その値群。")
model IssuedClaim {
  @doc("claim 型 URI。")
  @minLength(1)
  claim_type: string;
  @doc("claim の値群 (多値属性は複数値になる)。")
  values: string[];
}

}
