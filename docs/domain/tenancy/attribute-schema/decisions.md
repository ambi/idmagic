# 属性スキーマの設計判断

- `TenantGroupAttributeSchema` を `TenantUserAttributeSchema` と統合せず、別の Aggregate として持つ。テナント単位の属性スキーマはどのプリンシパルのものであれ `Tenancy` に置くが、`Group` には照合先となる組込みカタログが存在しないためである。
