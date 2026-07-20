# Tenant Quotas Runbook

## 概要

IdMagic では、テナントごとのリソース消費量（ユーザー、グループ、エージェント、アプリケーションなど）に対して制限（クォータ）を設け、意図しない大量作成や悪意のある使用による影響を抑え、システムの可用性を保ちます (ADR-134)。

クォータに達した場合、操作は直ちに失敗し、テナント管理者およびシステム管理者にメトリクスとログを通じて状況が報告されます。

## クォータ超過時の対応 (Quota Exceeded)

テナント管理者がクォータ超過に遭遇した場合、あるいは監視アラート (`quota_exceeded_total` metric) で異常な拒否率を検知した場合、以下の対応を行います。

1. **状況の確認**
   - System Admin として `System Console > Tenants` にアクセスし、該当テナントの現在のリソース消費量 (Usage) とクォータ (Quota) を確認します。
   - `idmagic_quota_exceeded_total` メトリクス (Prometheus) および `TenantQuotaExceeded` 構造化ログ (zog/slog) で、どのリソース (`resource` 属性: `users`, `groups` など) が超過しているかを特定します。

2. **原因の特定**
   - 正当な利用の増加によるものか、ループスクリプトのバグや不正アクセスによる急増かをヒアリングおよび監査イベントログ (Audit Events) から判断します。

3. **対応策: クォータ上限の引き上げ**
   - 正当な理由がある場合は、`System Console > Tenants` 画面から対象テナントの設定を開き、該当リソースのクォータ上限を必要な数だけ引き上げます。
   - 更新は即座に反映され、再度作成操作が可能になります。

4. **対応策: リソースの削除/整理**
   - クォータ上限の引き上げが適切でない場合、テナント管理者に不要なリソース (休眠ユーザー、未使用のアプリなど) の整理を促します。
   - 削除操作によって Usage カウンタが低下し、再び作成が可能になります。

## Usage カウンタの再計算 (Reconciliation)

バックアップリストア時や、DBの直接操作、極めて稀な競合などで Quota Usage カウンタと実際のレコード数にズレが生じた場合、差分を補正する Reconciliation ジョブを実行する必要があります。

1. (現行バージョンでは、サーバー起動時または定期的なバックグラウンドジョブで Usage カウンタと実際のCOUNTを比較・同期する仕組みが実行されます)
2. 手動で強制的に同期する場合は、指定の admin API もしくは CLI コマンド (`idmagic-cli system quota reconcile`) を使用します。(将来実装予定)

## メトリクスとログ

* **Metrics**
  * `quota_exceeded_total` (Counter): クォータ制限により作成系操作が拒否された回数。
    * Labels: `resource` (例: `users`, `groups`)
* **Structured Logs**
  * `TenantQuotaExceeded`: 操作拒否時に出力されるログイベント。
    * Fields: `tenant_id`, `resource`, `current_usage`, `limit`
