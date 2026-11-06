---
status: completed
---

# SPIFFE の信頼設定では、SPIFFE Bundle Endpoint の形式で鍵を読む

`spiffe` の信頼設定では、取得した JWKS の形式で鍵を選ぶようにした（REQ-WORKLOADIDENTITY-003 に表を追加）。`use` が `x509-svid`、`jwt-svid`、`wit-svid` の鍵を含む文書は SPIFFE バンドルとして `jwt-svid` の鍵だけを使い、含まない文書は SPIRE の OIDC Discovery Provider の形として `sig` と `use` のない鍵を使う。`x509-svid` の鍵はどちらでも使わない。
REQ-WORKLOADIDENTITY-012 を追加し、`jwks_uri` の鍵の再利用を 5 分以内にした。SPIFFE バンドルの `spiffe_refresh_hint` が短ければ 30 秒を下限としてその秒数まで縮める。`jwks_uri` は `https_web` の方式で取得し、`https_spiffe` には対応しない。
REQ-WORKLOADIDENTITY-008 の再取得の鍵の数は、署名の検証に使う鍵の数になった。TypeSpec は `WorkloadTrustBundle` の `jwks_uri`、`jwks` と `WorkloadTrustBundleRefreshResponse.key_count` の説明だけを変え、形は変えていない。管理画面は `spiffe` の信頼設定で、鍵の数を JWT-SVID の検証に使う鍵の数として通知する。
判断は Keycloak、Vault Enterprise、Entra ID、Google Cloud IAM、Okta、PingFederate との比較とともに `decisions.md` に残した。Keycloak と違い、SPIFFE バンドルの `use` のない鍵は仕様どおり使わない。ヒントを 5 分より長く使わないのは、`kid` の不一致で取得し直す仕組みを持たないためである。
`keys` の配列のない取得結果は取得の失敗とし、直近に取得できた鍵を空の集合で上書きしない。
残るリスク：実際の SPIRE の Bundle Endpoint と OIDC Discovery Provider での確認はしていない（wi-76418 が扱う）。SPIFFE Federation が求める `https_spiffe` への対応は欠けており、要求が生じたら見直す。
