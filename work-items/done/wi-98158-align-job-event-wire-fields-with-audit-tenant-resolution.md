---
status: completed
---

# Jobs のイベントの送信表現を、監査のテナント解決と一致させる

`spec-diff` は規範仕様の差分なしを示し、実装を既存の TypeSpec（`JobEnqueued` から `JobCanceled` までの六種類）と REQ-AUDIT-001 に合わせた。
Jobs のイベントは `jobId`、`tenantId` などの TypeSpec の項目名で直列化され、監査が `tenantId` からテナントを解決するので、テナントに限った監査の検索に現れ、別テナントの検索には現れない。
Audit に旧表現を読ませる案は、所有者以外へ Jobs 固有の知識を広げるので採らなかった。
公開状態が `unpublished` なので、保存済みの旧表現（`TenantID` と `JobID`）の記録は移行せず、保持期間の掃引に任せた。
