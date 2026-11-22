namespace IdMagicUi;

model AuditPages {
  @page(#{...adminContext, id: "admin-audit-events", name: "監査イベント", entry: "管理メニュー", purpose: "テナントの操作と関係者を検索して確認する", content: #["日時、種類、結果、主体と対象による検索。条件は URL に保持", "イベントの詳細と関係者からの再検索", "結果のページ送り", "出力ジョブの開始、再取得、停止とダウンロード"]}) @category(Category.audit)
  admin: PageData<{
    @transition(#{...update, event: "検索・再取得・ページ送り", result: "条件に合う監査イベントを表示する"}, AuditPages.admin) @uses(IdMagic.Audit.Operations.ListAdminAuditEvents) search: unknown;
    @transition(#{...update, event: "詳細または関係者を選択", result: "イベント詳細またはその関係者を含む検索結果を表示する"}, AuditPages.admin) detail: unknown;
    @transition(#{...update, event: "出力を開始・再取得・キャンセル", result: "検索条件に対応する出力ジョブを更新する"}, AuditPages.admin) @uses(IdMagic.Audit.Operations.StartAdminAuditExport, IdMagic.Audit.Operations.ListAdminAuditExports, IdMagic.Audit.Operations.CancelAdminAuditExport) exportCsv: unknown;
    @transition(#{event: "出力を取得", condition: "出力が完了し期限内", result: "監査イベントのファイルを取得する", toState: "external"}, Destinations.download) @uses(IdMagic.Audit.Operations.DownloadAdminAuditExport) download: unknown;
  }, ListStates>;
  @page(#{...systemContext, id: "system-audit-events", name: "システム監査イベント", entry: "システム管理メニュー", purpose: "テナントを横断した操作と結果を調べる", content: #["テナント、日時、種類と関係者による検索", "イベントの詳細、ページ送りと関係者からの再検索", "出力ジョブの開始、再取得、停止とダウンロード", "必要な認証保証への再認証"]}) @category(Category.audit)
  system: PageData<{
    @transition(#{...update, event: "検索・再取得・ページ送り", result: "条件に合うシステム監査イベントを表示する"}, AuditPages.system) @uses(IdMagic.Audit.Operations.ListSystemAuditEvents) search: unknown;
    @transition(#{...update, event: "詳細または関係者を選択", result: "詳細または関係者で絞り込んだ結果を表示する"}, AuditPages.system) detail: unknown;
    @transition(#{...update, event: "出力を開始・再取得・キャンセル", result: "出力ジョブを更新する"}, AuditPages.system) @uses(IdMagic.Audit.Operations.StartSystemAuditExport, IdMagic.Audit.Operations.ListSystemAuditExports, IdMagic.Audit.Operations.CancelSystemAuditExport) exportCsv: unknown;
    @transition(#{event: "出力を取得", condition: "出力が完了し期限内", result: "監査ファイルを取得する", toState: "external"}, Destinations.download) @uses(IdMagic.Audit.Operations.DownloadSystemAuditExport) download: unknown;
  }, ListStates>;
}
