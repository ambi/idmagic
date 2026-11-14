---
status: completed
---

# どの経路で Job を取り消しても、`active_jobs` の使用量を減らす

REQ-JOBS-013 と REQ-IDMANAGEMENT-041 を変え、受理した取り消しは `active_jobs` の使用量を一つ減らし、拒否した取り消しは使用量を変えないとした。
REQ-IDMANAGEMENT-041 では、データエクスポートの取り消しも `JobCanceled` を発行するようにした。`JobLifecycle` の遷移表が取り消しを `JobCanceled` とするのに、この経路だけが発行していなかったためで、利用者の確認を経て決めた。
取り消しの遷移、減算、`JobCanceled` の発行を Jobs の公開操作 `ports.Canceler`（`jobs.NewCanceler`、`Module.Canceler`）に集め、管理者、監査エクスポート、データエクスポートの取り消しはすべてそこを通る。Audit の個別の減算と発行は消した。
`JobRepository.Cancel` を直接呼ぶのは、加算に失敗した Job を補償する `Enqueue` だけである。
取り消しの確定後に減算が失敗した場合は、ランナーと同じく記録して続ける。ずれた使用量の補正は対象外とした。
