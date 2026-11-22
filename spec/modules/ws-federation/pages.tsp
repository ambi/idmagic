namespace IdMagicUi;

model EntraFederationPages {
  @page(#{...adminContext, id: "admin-entra-federation", name: "Entra フェデレーション", entry: "管理メニュー", purpose: "ドメインの信頼状態と署名鍵の適用状況を確認する", content: #["ドメインごとの設定と観測結果", "署名鍵の更新への案内", "登録と再取得"]}) @category(Category.federation)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "ドメインと信頼先の情報を更新する"}, EntraFederationPages.list) @uses(IdMagic.WsFederation.Operations.ListEntraFederationDomains) refresh: unknown;
    @transition(#{event: "追加"}, EntraFederationPages.create) create: unknown;
    @transition(#{event: "ドメインを開く"}, EntraFederationPages.detail) detail: unknown;
    @transition(#{event: "署名鍵を管理"}, XmlFederationKeyPages.keys) keys: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-entra-federation-add", name: "Entra ドメインを追加", entry: "フェデレーション一覧", purpose: "ドメインの信頼を登録する", content: #["ドメインと信頼先の設定", "Immutable ID と移行の案内"]}) @category(Category.federation)
  create: PageData<{
    @transition(#{event: "登録", condition: "保存が成功", result: "登録したドメインへ進む"}, EntraFederationPages.detail) @uses(IdMagic.WsFederation.Operations.ConfigureEntraFederation) save: unknown;
    @transition(#{event: "キャンセル"}, EntraFederationPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-entra-federation-domain", name: "Entra ドメインの詳細", entry: "フェデレーション一覧", purpose: "ドメインの設定と外部で観測した信頼情報を照合する", content: #["設定と署名鍵の適用状況", "コマンドとメタデータのコピー", "観測 JSON の貼り付け・ファイル読込", "登録解除の確認と移行手順への案内"]}) @category(Category.federation)
  detail: PageData<{
    @transition(#{...update, event: "観測結果を記録", result: "信頼情報の照合結果を更新する"}, EntraFederationPages.detail) @uses(IdMagic.WsFederation.Operations.RecordEntraFederationObservation) record: unknown;
    @transition(#{event: "編集"}, EntraFederationPages.edit) edit: unknown;
    @transition(#{event: "登録解除を確認", condition: "解除が成功", result: "一覧へ戻る"}, EntraFederationPages.list) @uses(IdMagic.WsFederation.Operations.RemoveEntraFederationDomain) remove: unknown;
    @transition(#{event: "署名鍵を管理"}, XmlFederationKeyPages.keys) keys: unknown;
    @transition(#{event: "戻る"}, EntraFederationPages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-entra-federation-edit", name: "Entra ドメインを編集", entry: "ドメイン詳細", purpose: "ドメインの信頼設定を更新する", content: #["現在の設定と変更", "保存とキャンセル"]}) @category(Category.federation)
  edit: PageData<{
    @transition(#{event: "保存", condition: "保存が成功", result: "ドメイン詳細へ戻る"}, EntraFederationPages.detail) @uses(IdMagic.WsFederation.Operations.ConfigureEntraFederation) save: unknown;
    @transition(#{event: "キャンセル"}, EntraFederationPages.detail) cancel: unknown;
  }>;
}
