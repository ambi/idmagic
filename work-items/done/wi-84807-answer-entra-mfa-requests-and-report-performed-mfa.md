---
status: completed
---

# Entra の MFA の要求に応え、実施した MFA を Entra に伝える

REQ-WSFEDERATION-002 に三文を加え、standards.md に ENTRA-MFA-AUTHENTICATION-METHOD、ADFS-MFA-WAUTH、ENTRA-GRAPH-MFA-BEHAVIOR を加えた。
Entra の定型設定を持つ RP へ発行するとき、記憶済みの端末を除く第二要素で認証したセッションなら、Assertion の認証方式（SAML 1.1 の `AuthenticationMethod`、SAML 2.0 の `AuthnContextClassRef`）を `http://schemas.microsoft.com/claims/multipleauthn` にする。
Entra が `wauth` に同じ URI を指定し、セッションが第二要素で認証していなければ、セッションを第二要素の照合の待ちにし、元の要求を `return_to` に持つ第二要素の画面へ 303 で送る。第二要素を持たない利用者には 400 で拒否する。
着手時の設計は MFA を `authnmethodsreferences` のクレームで伝えるとしていたが、Entra の一次資料は認証方式の位置を求めるので改めた。
第二要素の手順は OAuth2 の `acr_values` と同じ Authentication の操作を、WsFederation が定める最小のポートで呼ぶ。
利用者の確認により、案内する `FederatedIdpMfaBehavior` は `acceptIfMfaDoneByFederatedIdp` のまま、`tdev` は MFA に数えず、第二要素のない利用者を登録へ誘導しない。
Entra 以外の RP の `wauth` の扱い、`wiaormultiauthn`、`insidecorporatenetwork` は対象外で、実テナントでの `enforceMfaByFederatedIdp` の確認は wi-79 に残る。
