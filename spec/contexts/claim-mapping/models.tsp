namespace IdMagic.Contract {

@doc("Tenancy context が publish する属性開示範囲の published language stub。Private だけが claim 発行の fail-closed floor で常に除外される。")
enum AttrVisibility {
  Private: "Private",
  SelfReadable: "SelfReadable",
  AdminReadable: "AdminReadable",
  ClaimExposed: "ClaimExposed",
}


@doc("Tenancy context が publish する属性定義の published language stub。claim 発行時の属性可視性判定に使うほか、\ntenant の user_attribute_schema の要求本体と応答本体にそのまま現れる。")
model UserAttributeDef {
  @doc("属性の機械名。先頭が英字の snake_case で、builtin カタログと tenant custom 属性を通じて一意。")
  @minLength(1)
  key: string;
  @doc("管理画面と self-service フォームに出す表示名。省略時は key をそのまま表示に使う。")
  @maxLength(100)
  label?: string;
  @doc("属性値の型。multi_valued と食い違う定義に対する属性値の検証は失敗する。")
  type: AttributeType;
  @doc("値を複数持つか。type=StringArray のときだけ true にできる。")
  multi_valued: boolean = false;
  @doc("User の作成と更新で値を必須とするか。")
  required: boolean = false;
  @doc("利用者自身が self-service プロフィールからこの属性を編集できるか。")
  editable_by_user: boolean = false;
  @doc("claim へ出すときの claim 名。省略時は key をそのまま claim 名に使う。")
  @minLength(1)
  @maxLength(100)
  claim_name?: string;
  @doc("claim 露出を解禁する OIDC scope。省略時は custom_attribute scope が解禁の条件になる。")
  @minLength(1)
  @maxLength(64)
  oidc_scope?: string;
  @doc("属性の開示範囲。既定値は無く、値の無い定義も enum 外の値も拒否する。ClaimExposed だけが claim に出せる。")
  visibility: AttrVisibility;
  @doc("値が個人情報を含むか。省略時は false になるので、PII として扱わせたい属性には明示的に true を送る。")
  pii: boolean = false;
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
