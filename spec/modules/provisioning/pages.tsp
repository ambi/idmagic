namespace IdMagicUi;

model ApplicationProvisioningPages {
  @page(#{...adminContext, id: "admin-application-provisioning", name: "アプリケーションのプロビジョニング", entry: "アプリケーション詳細", purpose: "接続先の同期設定、状態と処理タスクを管理する", content: #["未接続時の登録", "接続設定、資格情報と属性対応", "接続テスト、停止状態からの再開", "個別実行と全件再同期", "状態によるタスク一覧、詳細と再試行", "接続解除の確認"]}) @category(Category.provisioning)
  detail: PageData<{
    @transition(#{...update, event: "接続または設定を保存", result: "接続状態と設定を更新する"}, ApplicationProvisioningPages.detail) @uses(IdMagic.Provisioning.Operations.RegisterProvisioningConnection, IdMagic.Provisioning.Operations.UpdateProvisioningConnection) save: unknown;
    @transition(#{...update, event: "接続をテストまたは再開", result: "テスト結果または接続の状態を表示する"}, ApplicationProvisioningPages.detail) @uses(IdMagic.Provisioning.Operations.TestProvisioningConnection, IdMagic.Provisioning.Operations.ResumeProvisioningConnection) test: unknown;
    @transition(#{...update, event: "個別実行または全件再同期", result: "実行結果と開始したジョブを表示する"}, ApplicationProvisioningPages.detail) @uses(IdMagic.Provisioning.Operations.ProvisionOnDemand, IdMagic.Provisioning.Operations.StartFullResync) run: unknown;
    @transition(#{...update, event: "タスクを取得または再試行", result: "対象タスクの詳細と状態を更新する"}, ApplicationProvisioningPages.detail) @uses(IdMagic.Provisioning.Operations.ListProvisioningTasks, IdMagic.Provisioning.Operations.GetProvisioningTask, IdMagic.Provisioning.Operations.RetryProvisioningTask) tasks: unknown;
    @transition(#{...update, event: "接続解除を確認", result: "接続情報を削除して未接続の表示に戻す"}, ApplicationProvisioningPages.detail) @uses(IdMagic.Provisioning.Operations.DeleteProvisioningConnection) disconnect: unknown;
    @transition(#{event: "戻る"}, AdminApplicationPages.detail) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-provisioning", name: "プロビジョニングの概要", entry: "管理メニュー", purpose: "テナント内の接続状態を一覧で確認する", content: #["接続先、状態とエラー", "対象アプリケーションの同期管理への入口"]}) @category(Category.provisioning)
  overview: PageData<{
    @transition(#{event: "接続先を開く", data: "アプリケーション ID"}, ApplicationProvisioningPages.detail) detail: unknown;
  }, ReadStates>;
}
