---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# 下流 SaaS への outbound provisioning (SCIM client / lifecycle management) を実装する

idmagic を SCIM 2.0 outbound client にする独立 `Provisioning` bounded
context を新設した (ADR-128)。Domain (`ProvisioningConnection`/
`AttributeMappingRule`/`DeprovisionPolicy`/`RemoteResourceLink`/
`ProvisioningDelivery` とその状態機械) から、same-Tx-adjacent capture →
Jobs 経由の非同期 dispatch → protocol-agnostic core が呼ぶ SCIM feature
slice (`backend/provisioning/scim`) による配送、admin API (12 interface)
と Application 詳細ページの「プロビジョニング」サブルート + テナント集約
ビューまで、SCL → ADR → Domain → Use Cases → Adapters → Infrastructure の
全層を実装した。IdManagement (User create/update/disable/delete) と
Application (assignment add/remove) の実 usecase から outbound
provisioning への配線を、Jobs キューを含む実コンポーネントを通した
end-to-end テストで検証し、その過程で `OnDelete=deactivate` (既定ポリシー)
が削除済みユーザーの属性解決失敗によりサイレントに no-op していた実バグを
発見・修正した。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
