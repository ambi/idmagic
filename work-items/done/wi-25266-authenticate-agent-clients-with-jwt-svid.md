---
status: completed
---

# JWT-SVID をクライアントアサーションとして受け付け、Agent のクライアントを長期の鍵なしで認証する

- 認証方式 `spiffe_jwt` を足し、`client_assertion_type` が `jwt-spiffe` の要求で、Agent に束縛したクライアントを JWT-SVID だけで認証できるようにした（REQ-OAUTH2-089、REQ-WORKLOADIDENTITY-017）。標準の表に draft-ietf-oauth-spiffe-client-auth-03 の節（SPIFFECA03-*）を足し、X.509-SVID、WIT-SVID、CIMD の `spiffe_id` は `excluded` とした。
- AIMS00-AGENT-CLIENT-AUTH、REQ-OAUTH2-055、REQ-IDMANAGEMENT-074 は、Agent のクライアントに `spiffe_jwt` も許す形に改めた。FAPI 2.0 の許可集合には含めない。
- 信頼設定は `sub` のトラストドメインの `spiffe` の信頼設定だけから選び、`iss` では選ばない（ドラフトの 8.3 節）。`aud` は信頼設定の audience ではなく、テナントの発行者識別子だけを求める。`iat` のない JWT-SVID は最大 TTL を確かめられないため拒否するので、SPIFFECA03-JWT-SVID は `partial` とした。
- SPIFFE ID とクライアントの対応付けは Agent の関連付けだけで登録する（案 A）。要求のクライアントが関連付けの Agent に束縛されているかは WorkloadIdentity が確かめ、拒否の理由 `client_not_bound` を監査に残す。OAuth2 は登録した認証方式と `jti` の再生（`jti` があるときだけ）を確かめ、成功で `WorkloadClientAuthenticated` を発行する。`WorkloadAttestationRejected` には用途（`token_exchange` か `client_authentication`）を足した。
- メタデータの三つの認証方式の一覧に `spiffe_jwt` を載せた。管理画面の Agent の資格情報の画面から、鍵を持たない `spiffe_jwt` のクライアントを作成して紐付けられる。
- 実際の SPIRE によるこの経路の確認は wi-76418 の対象範囲へ移した。
