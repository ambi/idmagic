namespace IdMagicUi;

model LifecycleWorkflowPages {
  @page(#{...adminContext, id: "admin-lifecycle-workflows", name: "ライフサイクルワークフロー", entry: "管理メニュー", purpose: "状態変化に応じた自動処理と実行結果を管理する", content: #["ワークフローの一覧と選択した定義", "有効化、無効化と削除確認", "試行ダイアログと実行履歴", "失敗した実行の再試行"]}) @category(Category.lifecycle)
  list: PageData<{
    @transition(#{event: "作成"}, LifecycleWorkflowPages.create) create: unknown;
    @transition(#{event: "編集"}, LifecycleWorkflowPages.edit) edit: unknown;
    @transition(#{...update, event: "再取得または実行履歴を選択", result: "定義と実行結果を表示する"}, LifecycleWorkflowPages.list) @uses(IdMagic.IdGovernance.Operations.ListLifecycleWorkflows, IdMagic.IdGovernance.Operations.ListLifecycleWorkflowRuns) refresh: unknown;
    @transition(#{...update, event: "有効化または無効化・削除を確認", result: "対象の状態と一覧を更新する"}, LifecycleWorkflowPages.list) @uses(IdMagic.IdGovernance.Operations.EnableLifecycleWorkflow, IdMagic.IdGovernance.Operations.DisableLifecycleWorkflow, IdMagic.IdGovernance.Operations.DeleteLifecycleWorkflow) status: unknown;
    @transition(#{...update, event: "試行または失敗した実行を再試行", result: "試行結果または再実行の状態を表示する"}, LifecycleWorkflowPages.list) @uses(IdMagic.IdGovernance.Operations.DryRunLifecycleWorkflow, IdMagic.IdGovernance.Operations.RetryLifecycleWorkflowRun) run: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-lifecycle-workflow-create", name: "ワークフローを作成", entry: "ワークフロー一覧", purpose: "開始条件と順序付きの処理を定義する", content: #["対象と開始条件", "アクションの追加、順序変更と削除", "作成とキャンセル"]}) @category(Category.lifecycle)
  create: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "作成したワークフローの編集へ進む"}, LifecycleWorkflowPages.edit) @uses(IdMagic.IdGovernance.Operations.CreateLifecycleWorkflow) save: unknown;
    @transition(#{event: "キャンセル"}, LifecycleWorkflowPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-lifecycle-workflow-edit", name: "ワークフローを編集", entry: "一覧または作成後", purpose: "リビジョンを確認して処理の定義を変更する", content: #["現在の定義とリビジョン", "対象と順序付きアクションの編集", "保存後も編集に留まり、競合と失敗を表示する"]}) @category(Category.lifecycle)
  edit: PageData<{
    @transition(#{...update, event: "保存", condition: "リビジョンが一致し更新が成功", result: "更新した定義とリビジョンを表示する"}, LifecycleWorkflowPages.edit) @uses(IdMagic.IdGovernance.Operations.UpdateLifecycleWorkflow) save: unknown;
    @transition(#{event: "キャンセルまたは戻る"}, LifecycleWorkflowPages.list) cancel: unknown;
  }>;
}

model AccessReviewPages {
  @page(#{...adminContext, id: "admin-access-reviews", name: "アクセスレビュー計画", entry: "管理メニュー", purpose: "定期レビューの計画と次回開始を確認する", content: #["対象の種類、有効状態、次回開始と開始失敗", "作成と計画詳細への入口"]}) @category(Category.lifecycle)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "計画の一覧を更新する"}, AccessReviewPages.list) @uses(IdMagic.IdGovernance.Operations.ListAccessReviewPlans) refresh: unknown;
    @transition(#{event: "作成"}, AccessReviewPages.create) create: unknown;
    @transition(#{event: "詳細"}, AccessReviewPages.detail) detail: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-access-review-create", name: "レビュー計画を作成", entry: "レビュー計画の一覧", purpose: "対象、レビュー担当と繰り返しを指定する", content: #["Agent、アプリ割り当て、グループ所属の選択", "対象とレビュー担当、代替担当", "開始日時、期間、繰り返し、未回答時の判断"]}) @category(Category.lifecycle)
  create: PageData<{
    @transition(#{event: "作成", condition: "保存が成功", result: "作成した計画の詳細へ進む"}, AccessReviewPages.detail) @uses(IdMagic.IdGovernance.Operations.CreateAccessReviewPlan) save: unknown;
    @transition(#{event: "キャンセル"}, AccessReviewPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-access-review-edit", name: "レビュー計画を編集", entry: "計画詳細", purpose: "今後のレビューの条件を変更する", content: #["既存の対象、担当と繰り返し設定", "次回以降に用いる条件と保存"]}) @category(Category.lifecycle)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "現在値を取得した詳細へ戻る"}, AccessReviewPages.detail) @uses(IdMagic.IdGovernance.Operations.UpdateAccessReviewPlan) save: unknown;
    @transition(#{event: "キャンセル"}, AccessReviewPages.detail) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-access-review-detail", name: "レビュー計画の詳細と結果", entry: "計画一覧または保存後", purpose: "計画の稼働状態と各レビューの判断・適用結果を確認する", content: #["対象、担当、期間とデフォルトの判断", "次回開始、開始失敗と通知失敗", "回答、デフォルトの判断の適用、取消と適用エラー", "計画の無効化を確認するダイアログ"]}) @category(Category.lifecycle)
  detail: PageData<{
    @transition(#{event: "編集"}, AccessReviewPages.edit) edit: unknown;
    @transition(#{...update, event: "有効化または無効化を確認", result: "計画の有効状態を更新する"}, AccessReviewPages.detail) @uses(IdMagic.IdGovernance.Operations.UpdateAccessReviewPlan) toggle: unknown;
    @transition(#{...update, event: "再取得", result: "計画とレビュー結果を更新する"}, AccessReviewPages.detail) @uses(IdMagic.IdGovernance.Operations.ListAccessReviewPlans, IdMagic.IdGovernance.Operations.ListAccessReviews) refresh: unknown;
    @transition(#{event: "戻る"}, AccessReviewPages.list) back: unknown;
  }>;
  @page(#{...accountContext, id: "account-access-reviews", name: "本人のアクセスレビュー", entry: "本人のポータル", purpose: "担当するアクセスの継続または取消を判断する", content: #["対象、期限、未回答時の判断と現在の適用結果", "未回答かつ期限前のレビューへの回答", "取消の確認ダイアログ", "アプリ割り当てへの注意"]}) @category(Category.lifecycle)
  mine: PageData<{
    @transition(#{...update, event: "継続または取消を確認", condition: "レビューが未回答かつ期限前", result: "判断と反映結果を表示する"}, AccessReviewPages.mine) @uses(IdMagic.IdGovernance.Operations.DecideAccessReview) decide: unknown;
    @transition(#{...update, event: "再取得", result: "担当レビューを更新する"}, AccessReviewPages.mine) @uses(IdMagic.IdGovernance.Operations.ListMyAccessReviews) refresh: unknown;
  }, ListStates>;
}
