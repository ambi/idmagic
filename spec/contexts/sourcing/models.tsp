namespace IdMagic.Contract {

@doc("RFC 7644 §3.12 に従う SCIM protocol error response。")
model ScimProtocolError {

}


@doc("SCIM 2.0 の ID と IdP 内部の User (sub) を紐付けるマッピング。")
model ScimUserRef {
  tenant_id: string;
  scim_id: string;
  user_id: string;
}


@doc("SCIM 2.0 の ID と IdP 内部の Group (id) を紐付けるマッピング。")
model ScimGroupRef {
  tenant_id: string;
  scim_id: string;
  group_id: string;
}

}
