namespace IdMagic.Contract {

@doc("Tenancy モジュールが所有する属性の公開範囲を、公開契約として写したもの。クレームの発行のフェイルクローズの下限が無条件に除外するのは `private` だけである。")
enum AttrVisibility {
  Private: "private",
  SelfReadable: "self_readable",
  AdminReadable: "admin_readable",
  ClaimExposed: "claim_exposed",
}


@doc("Tenancy モジュールが所有する属性の定義を、公開契約として写したもの。クレームの発行で属性の公開範囲を判断するほか、テナントの user_attribute_schema のリクエストとレスポンスの両方のボディにそのまま現れる。")
model UserAttributeDef {
  @doc("属性の機械名。英字で始まる snake_case で、組み込みの一覧とテナントのカスタム属性を通じて一意である。")
  @minLength(1)
  key: string;
  @doc("管理コンソールとセルフサービスのフォームに表示するラベル。省略するとキーそのものを表示する。")
  @maxLength(100)
  label?: string;
  @doc("属性値の型。型が `multi_valued` と矛盾する定義に対して値を検証すると失敗する。")
  type: AttributeType;
  @doc("属性が複数の値を保持するかどうか。true にできるのは `type` が `string_array` のときだけである。")
  multi_valued: boolean = false;
  @doc("User の作成時と更新時に値が必須かどうか。")
  required: boolean = false;
  @doc("ユーザーが自身のセルフサービスのプロフィールからこの属性を編集できるかどうか。")
  editable_by_user: boolean = false;
  @doc("属性をクレームとして発行するときのクレーム名。省略するとキーそのものをクレーム名にする。")
  @minLength(1)
  @maxLength(100)
  claim_name?: string;
  @doc("クレームとしての公開を可能にする OIDC のスコープ。省略すると `custom_attribute` のスコープで公開できる。")
  @minLength(1)
  @maxLength(64)
  oidc_scope?: string;
  @doc("属性をどこまで開示してよいか。デフォルトはなく、値のない定義は拒否し、列挙値の外の値も拒否する。クレームに含められるのは `claim_exposed` だけである。")
  visibility: AttrVisibility;
  @doc("値が個人情報を含むかどうか。省略すると false なので、個人情報として扱うべき属性は明示的に true にしなければならない。")
  pii: boolean = false;
}


@doc("クレームの解決をフェイルクローズで拒否した。必須の取得元を解決できなかった、公開範囲が `private` の属性を規則または NameID の取得元に指定した、または予約済みのクレームの種類（iss、sub、aud、exp、iat、nbf、jti、azp、nonce、at_hash、c_hash、acr、amr、sid）を `claim_type` に指定した。クライアントに届くのは OAuth 2.0 のトークンと UserInfo のエンドポイントを通じてだけなので、Problem Details ではなく OAuthError のボディで返す。")
model ClaimReleaseDeniedError is OAuthError;


@doc("一つの出力クレームの宣言的な対応付け。取得元から、出力するクレームの種類（URI）を組み立てる。")
model ClaimMappingRule {
  @doc("出力するクレームの種類の URI。`http://schemas.xmlsoap.org/claims/UPN` など。")
  @minLength(1)
  claim_type: string;
  @doc("クレームの値の取得元。")
  source: ClaimMappingSource;
  @doc("`source` が `user_attribute` のときに読む属性の名前。")
  source_key?: string;
  @doc("`source` が `fixed` のときに出力する固定値。")
  fixed_value?: string;
  @doc("true のとき、取得元を解決できなければ発行を拒否する（フェイルクローズ）。")
  required: boolean = false;
}


@doc("クレームの値の取得元。`user_attribute` は ID の主体の属性を読み、`fixed` は静的な値を出力し、`nameid` は NameIdConfiguration が解決した NameID をクレームとしても運ぶ。")
enum ClaimMappingSource {
  user_attribute: "user_attribute",
  fixed: "fixed",
  nameid: "nameid",
}


@doc("発行するアサーションの NameID（サブジェクトの識別子）の形式と、その値の取得元。")
model NameIdConfiguration {
  @doc("NameID の形式の URI。`urn:oasis:names:tc:SAML:2.0:nameid-format:persistent` など。")
  @minLength(1)
  format: string;
  @doc("NameID の値を与える ID の属性。")
  @minLength(1)
  source_attribute: string;
}


@doc("クレームの発行エンジンの出力。一つのクレームの種類とその値からなる。")
model IssuedClaim {
  @doc("クレームの種類の URI。")
  @minLength(1)
  claim_type: string;
  @doc("クレームの値。複数値の属性からは複数の値が得られる。")
  values: string[];
}

}
