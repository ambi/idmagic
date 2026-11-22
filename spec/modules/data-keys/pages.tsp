namespace IdMagicUi;

model DataKeyPages {
  @page(#{...systemContext, id: "system-data-key-health", name: "暗号化鍵の稼働状態", entry: "システム管理メニュー", purpose: "全テナントの保存データの暗号化鍵の状態を確認する", content: #["テナント別の鍵の状態と更新状況", "再取得と失敗の表示。変更操作はこのページに置かない"]}) @category(Category.keys)
  health: PageData<{
    @transition(#{...update, event: "再取得", result: "暗号化鍵の状態を更新する"}, DataKeyPages.health) @uses(IdMagic.DataKeys.Operations.ListTenantDataKeyHealth) refresh: unknown;
  }, ReadStates>;
}
