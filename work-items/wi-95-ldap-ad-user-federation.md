---
depends_on: [wi-126-async-job-runner]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# LDAP / Active Directory 閉域コネクタでユーザーをプロビジョニングする

## Motivation
エンタープライズ利用では、既存の LDAP / Active Directory (AD) をユーザーの
source of truth として使いたい要求が強い。しかし AD / LDAP は通常インターネットから
到達不能な閉域ネットワークにあり、SaaS 型 IdM が LDAP endpoint へ直接接続する前提は
現実的でも安全でもない。

本 WI は AD と同じプライベートネットワークに配置する **Directory Connector** を導入する。
Connector は最小権限の AD サービスアカウントで LDAP(S) を読み取り、IdMagic の
connector API へ mTLS で**アウトバウンド接続だけ**を行い、ユーザー・最小限のグループ・
属性差分を IdMagic に片方向プロビジョニングする。IdMagic は AD の到達先・bind DN・
bind password を保管せず、公開されたインターネットから Connector / AD への受信接続も
行わない。

これは protocol federation ([[wi-30-inbound-federation-and-identity-broker]]) が
OIDC/SAML の上流 IdP を扱うのとは異なり、**閉域 directory からの identity
provisioning** を扱う。認証用の LDAP bind 委譲はパスワード配送と可用性の設計を別途
必要とするため、本 WI には含めない。

## Scope
- **decision**:
  - 新規 ADR: Connector を DC 上ではなく AD と同じ閉域ネットワーク内の専用ホストに
    配置し、IdMagic への outbound-only + mTLS 接続とする境界を記録する。Connector
    enrollment、証明書の発行・失効・ローテーション、テナントへの所属、および最小権限の
    AD サービスアカウントの責務を定める。
  - 初期同期を AD → IdMagic の read-only・片方向プロビジョニングに限定し、外部不変 ID
    による冪等な upsert、削除・無効化の扱い、属性・グループ mapping、差分同期 cursor と
    再同期の規則を定める。IdMagic から AD への write-back、IdMagic による LDAP 直接接続、
    password import / LDAP bind 認証委譲を採らない理由も記録する。
- **scl**:
  - glossary: Directory Connector、directory source、external immutable ID、同期 cursor、
    enrollment を追加し、LDAP federation / protocol federation / SCIM provisioning との
    用語上の境界を明確にする。
  - §3.2 models: DirectoryConnector、DirectorySource、ConnectorCredential、
    DirectorySyncCursor、外部 identity mapping を追加する。
  - §3.3 interfaces: 管理者による source / Connector の設定・enrollment・無効化・状態照会、
    および Connector 認証済みの差分同期・再同期・結果報告 API を追加する。
  - §3.4 states/events: ConnectorEnrolled / ConnectorDisabled / DirectorySyncStarted /
    DirectorySyncCompleted / DirectorySyncFailed / DirectoryUserProvisioned /
    DirectoryUserDeprovisioned を追加する。
  - §3.5 invariants: Connector は所属 tenant 以外を更新できないこと、外部 immutable ID を
    tenant + source 内で一意とすること、同一差分の再送は冪等であること、import 属性は
    tenant の属性 schema に整合すること、IdMagic が AD 接続資格情報・ユーザーパスワードを
    保持しないことを明示する。
  - scenarios: 初回同期、差分の再送、削除／無効化、期限切れ・失効済み証明書の拒否、属性
    schema 不整合の部分失敗、Connector 停止からの cursor 再開を受け入れ例として追加する。
  - permissions: source / Connector 管理は tenant admin に限定し、同期 API は enrollment 済み
    Connector の mTLS identity と tenant 所属の双方で認可する。
  - objectives: IdMagic から Connector / AD への inbound 接続を要求しないこと、同期 payload
    と監査ログにパスワードを含めないこと、失敗した差分を安全に再試行できることを定める。
- **architecture**:
  - Connector runtime とその IdMagic 側受信境界を追加するため、bounded context の所有関係、
    新規 process / directory 規約を `ARCHITECTURE.md` に同期する。
- **go**:
  - Directory Connector の runtime（LDAP(S) read、AD 実装向けの差分取得、mapping、local
    cursor、再試行）と、IdMagic 側の enrollment / mTLS 認証 / 同期受理 use case を追加する。
  - 外部 immutable ID に基づく User / Group の冪等 provisioning と、無効化・削除・属性
    schema 検証を実装する。Connector の AD 接続資格情報は Connector の実行環境だけに置き、
    IdMagic の DB には保存しない。
- **http**:
  - admin の directory source / Connector 設定、enrollment token 発行、状態・同期履歴照会、
    無効化 API と、Connector 専用の mTLS 同期 API を追加する。
- **ui**:
  - Admin に directory source / Connector の登録手順、enrollment 情報、接続状態、最終同期、
    エラー・再同期操作を提供する画面を追加する。
- **documentation**:
  - README にネットワーク配置、Connector のインストール・enrollment、最小権限の AD
    サービスアカウント、証明書ローテーション、同期範囲・制約、障害時の運用を追記する。

## Out of Scope
- write-back / 双方向同期。
- IdMagic から AD / LDAP への直接ネットワーク接続（自己ホスト環境向けの任意モードを含む）。
- password import、Connector 経由の LDAP bind 認証委譲、Kerberos / SPNEGO SSO
  ([[wi-65-kerberos-spnego-inbound-silent-sso]])。
- ネストを含むグループの完全同期（初期は明示した最小限のグループ・属性に留める）。
- outbound provisioning (SCIM は [[wi-31-scim2-provisioning]] / [[wi-45-outbound-scim-provisioning]])。

## Plan
- SaaS APIから閉域LDAPへ直接接続させず、outbound-poll型の `idmagic-directory-connector` processを新規deploy unitとし、idmagicとの通信は短命のconnector credential/mTLSで行う。このprocess追加をADRとARCHITECTUREに記録する。
- server側はtenant-scoped DirectoryConnection、attribute/group mapping、sync cursor、external object linkを所有する。connectorへ暗号化済みbind secretを返さず、閉域側secret storeから参照するbootstrap方式を採る。
- initial full sync→paged delta sync→disable/delete reconciliationを分離し、`(connection, external objectGUID/entryUUID)`を不変linkにする。rename/DN変更を新規userと誤認しない。
- inbound SCIMのIdentity Management use case/soft-delete policyを再利用し、LDAP固有属性やgroup nestingはconnector adapterでcanonical commandへ変換する。conflictは自動上書きせずquarantineにする。
- password bindによるinteractive federationはprovisioning syncと別trust boundaryなので初期scopeを明確化し、含める場合もraw passwordをserver/event/logへ残さない。

## Tasks
- [ ] T001 [ADR/Architecture] connector deployment/trust/secret/bootstrap、sync保証、interactive auth範囲を決定しARCHITECTUREを同期する。
- [ ] T002 [SCL] DirectoryConnection、cursor/link/quarantine lifecycle、connector/admin interfaces/events/invariants/scenariosを追加して再生成する。
- [ ] T003 [Server Domain] connection/mapping/link/sync batch use caseとmemory/PostgreSQL repositoryを実装し、Identity Management commandsへ接続する。
- [ ] T004 [Connector] LDAPS/StartTLS、paged search、AD DirSyncまたはmodifyTimestamp cursor、objectGUID/entryUUID、nested group解決を実装する。
- [ ] T005 [Transport/Security] connector registration、credential rotation、mTLS/short-lived token、batch upload/ack/resumeを実装する。
- [ ] T006 [Admin UI] setup bundle、mapping、connection test、sync status/error/quarantine/retryを追加する。
- [ ] T007 [Verify] Samba/OpenLDAP fixtureでfull/delta/rename/delete/group nesting、TLS failure、cursor replay、conflict、secret非露出、tenant越境を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動（テスト用 LDAP / AD と別ネットワークの Connector）: tenant admin が source と
  enrollment token を作成 → Connector が mTLS で enrollment → 初回同期でユーザー属性を
  provisioning → 属性変更・無効化を差分同期 → 同一差分の再送が冪等であることを確認する。
- Connector の証明書失効、tenant 不一致、属性 schema 不整合を拒否し、AD 接続資格情報・
  ユーザーパスワードが IdMagic の DB・API payload・監査ログに存在しないことを確認する。

## Risk Notes
外部システム連携では、Connector の配布・更新、AD 実装ごとの差分取得、証明書の失効・
ローテーション、ネットワーク断時の cursor 整合、PII を含む同期データがリスクとなる。
Connector は AD の近傍に配置し outbound-only とし、mTLS・tenant binding・外部 immutable
ID の冪等性・再送可能な差分処理で扱う。テスト用 LDAP（例: コンテナ）を用いた統合テストで
初回同期、差分、再送、無効化、証明書拒否を検証する。初期は read-only の片方向同期に絞り、
認証委譲を混在させない。
