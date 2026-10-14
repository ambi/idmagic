---
status: completed
archived_from: 29f3395e3e16f7aba6be8421f1bfc65efb1ae0f1
---

# テナント単位のリソースクォータを定義して強制する

T004 (users / groups / agents / applications / oauth2_clients / active_sessions /
consents / active_jobs の 8 resource すべてで `CheckQuotaAndIncrement` / `DecrementQuota` を
作成系・削除/失効/終端系 usecase に実配線し、resource ごとに test-first で検証) と T008
(yaml-check / verify-go / verify-ui / 手動 HTTP 検証) を完了した。quota 超過は
`{"error":"quota_exceeded", ...}` + HTTP 422 + `QuotaExceeded` 監査イベント +
`quota_exceeded_total` メトリクスで一貫して返るよう `domain.QuotaExceededError` に
`IsQuotaExceeded/GetResource/GetTenantID` を実装し `support_http.ErrorHandler` と整合させた。
ADR-134 の既定値 (`DefaultTenantQuota`) を `tenancy/domain` の単一の情報源にまとめ、db_memory /
db_postgres で重複していたリテラルを解消した。手動検証の過程で、composition root
(`server_http/routes.go`) に `tenancyhttp.Deps.QuotaRepo` が一度も配線されておらず system admin
の quota 更新 API が silent no-op だったこと、および Application / OAuth2Client admin 作成の
error mapper が `QuotaExceededError` を 400 に握り潰していたことを発見し、修正した (T004.10)。
開示: Backfill / Reconciliation job (T005 が前提とする usage counter 乖離補正の安全弁) は
調査した限り実装が見当たらず、本 wi の reopen 対象 (T004/T008) 外のため未着手のまま。soft
quota (`audit_events_retained`, `export_artifacts_bytes`) の非同期警告は ADR-134 の定義通り
スコープ外。tenant admin が他 tenant の quota/usage を見えないことは既存のテナント境界解決ロジック
に依存しており今回変更・再検証していない。

全文は `git show 29f3395e3e16f7aba6be8421f1bfc65efb1ae0f1:<このファイルのパス>` で読む。
