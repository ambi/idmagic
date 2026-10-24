---
status: cancelled
---

# LDAP / Active Directory 閉域コネクタでユーザーをプロビジョニングする

一回で実装しきれる規模に収めるため、この項目を中止し、独立して受け入れられる五つの項目に分けた。
Connector の登録と認証（[[wi-46712-enroll-a-directory-connector-and-authenticate-its-calls]]）、User の取り込み（[[wi-23043-ingest-directory-users-from-connector-batches]]）、Connector の実行環境（[[wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]]）、管理画面（[[wi-37317-manage-directory-connectors-in-the-admin-ui]]）、グループの取り込み（[[wi-29505-ingest-directory-groups-and-direct-memberships]]）である。
Connector の認証は、エッジに証明書の転送を求める mTLS をやめ、Connector の鍵対で署名した短命の JWT にした。
