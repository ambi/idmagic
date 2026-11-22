namespace IdMagicUi;

model JobPages {
  @page(#{...adminContext, id: "admin-jobs", name: "テナントのジョブ", entry: "管理メニュー", purpose: "非同期処理の進捗と失敗を調べ、停止を要求する", content: #["種類、状態と期間による絞り込み", "追加取得と選択したジョブの詳細", "進捗、失敗とキャンセルの確認"]}) @category(Category.jobs)
  admin: PageData<{
    @transition(#{...update, event: "検索・再取得・追加取得", result: "対象ジョブと進捗を表示する"}, JobPages.admin) @uses(IdMagic.Jobs.Operations.ListJobs) search: unknown;
    @transition(#{...update, event: "ジョブを選択", result: "選択したジョブの内容を表示する"}, JobPages.admin) detail: unknown;
    @transition(#{...update, event: "キャンセルを確認", condition: "停止可能なジョブ", result: "停止を要求し状態を更新する"}, JobPages.admin) @uses(IdMagic.Jobs.Operations.CancelJob) cancel: unknown;
  }, ListStates>;
  @page(#{...systemContext, id: "system-jobs", name: "システムのジョブ", entry: "システム管理メニュー", purpose: "テナントを横断する非同期処理を調べる", content: #["テナント、種類、状態と期間の絞り込み", "追加取得とジョブの詳細", "キャンセルの確認と必要な再認証"]}) @category(Category.jobs)
  system: PageData<{
    @transition(#{...update, event: "検索・再取得・追加取得", result: "対象ジョブを更新する"}, JobPages.system) @uses(IdMagic.Jobs.Operations.ListSystemJobs) search: unknown;
    @transition(#{...update, event: "ジョブを選択", result: "選択したジョブの内容を表示する"}, JobPages.system) detail: unknown;
    @transition(#{...update, event: "キャンセルを確認", condition: "停止可能なジョブ", result: "停止を要求し状態を更新する"}, JobPages.system) @uses(IdMagic.Jobs.Operations.CancelSystemJob) cancel: unknown;
  }, ListStates>;
}
