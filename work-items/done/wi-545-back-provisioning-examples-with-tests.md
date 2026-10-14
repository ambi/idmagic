---
status: completed
archived_from: c68e5cd94eb832e3e2196827de45d1eccc181ad6
---

# Provisioning が宣言する具体例 27 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change` を返した。
Provisioning が台帳に残していた 27 件の具体例を実装と照合し、14 件を既存テストまたは追加した Provisioning API スコープと On-Demand Provision のテストへ結び付けて台帳から外した。
残る 13 件は猶予期間、Full Resync 完了、ライフサイクルイベント、テナント境界の公開 HTTP 観測が欠けるため、[[wi-96960-defer-and-cancel-user-deprovisioning-after-grace-period]]、[[wi-22987-track-full-resync-completion]]、[[wi-85060-publish-provisioning-lifecycle-events]]、[[wi-37560-return-access-denied-for-cross-tenant-provisioning-api-tokens]]、[[wi-93622-acceptance-test-provisioning-tenant-isolation]] へ切り出し、台帳へ `blocked_by` と `finding` を残した。
総合検証を妨げていた [[wi-595-declare-uniqueness-conflict-responses]] の完了記録の相対リンクも、`work-items/done/` から解決するパスへ修正した。

全文は `git show c68e5cd94eb832e3e2196827de45d1eccc181ad6:<このファイルのパス>` で読む。
