---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Application を上位 aggregate として実装し、protocol binding・割当・プロビジョニングを束ねる

OAuth2 client / WS-Fed RP / SAML SP / service client を単一の Application aggregate に
束ね、protocol を意識させない一覧・CRUD・割当・利用者ポータルを実装した。ADR-066 を起票。
- 集約: spec に Application / ApplicationKind (federated / weblink / service) /
ApplicationStatus / ProtocolBinding (oidc / saml / wsfed) / ApplicationAssignment /
AssignmentSubject (user / group) / AssignmentVisibility (visible / hidden) と
ライフサイクルイベント (Created/Updated/Deleted, BindingAttached/Detached,
Assigned/Unassigned) を定義。domain に weblink=launch_url 必須・binding 不可、
federated=binding 検証の不変条件を実装。
- 永続化: Postgres / memory adapter (applications / protocol_bindings /
application_assignments、tenant scope、binding は既存 client / RP / SP を opaque key で参照)。
- fail-closed 割当ゲート: ApplicationAccessAllowed を /authorize と WS-Fed sign-in に
適用し (invariant AssignmentGatesProtocol)、未割当 subject はフェデレーションを
完了できない。ListMyApplications は visible 割当のみ返す (AssignmentGatesPortalVisibility)。
- HTTP / UI: /admin/applications の CRUD・binding・割当・icon と
/api/account/applications を実装。admin ナビを「アプリケーション」に一本化し、
利用者ポータルに icon 付きアプリ一覧 (SSO 起動 / 外部リンク) を追加。
- SCIM provisioning binding (ProvisioningBinding / ProvisioningDirection) は実装しない。
out_of_scope のとおり inbound / outbound とも [[wi-31-scim2-provisioning]] /
[[wi-45-outbound-scim-provisioning]] が所有し、Application↔connector の関係は
ProvisioningConnector / ProvisioningAssignment を設計する wi-45 時点で確定する。
- 後続価値は別 WI に分割: ポータル並び替え (wi-70)、サインオンポリシー (wi-71)、
テンプレートカタログ (wi-72)、クレーム上書き (wi-73)、icon 画像アップロード (wi-74)、
advanced 設定の畳み込みと低レベル画面撤去 (wi-76, ADR-066, 完了済)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
