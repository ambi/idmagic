namespace IdMagicUi;

model WorkloadIdentityPages {
  @page(#{...adminContext, id: "admin-workload-identity", name: "ワークロードの信頼バンドル", entry: "管理メニュー", purpose: "外部ワークロードの認証元を管理する", content: #["発行者、方式と状態", "再取得、登録と詳細"]}) @category(Category.workload)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "信頼バンドルを更新する"}, WorkloadIdentityPages.list) @uses(IdMagic.WorkloadIdentity.Operations.ListWorkloadTrustBundles) refresh: unknown;
    @transition(#{event: "登録"}, WorkloadIdentityPages.create) create: unknown;
    @transition(#{event: "詳細"}, WorkloadIdentityPages.detail) detail: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-workload-trust-bundle-create", name: "信頼バンドルを登録", entry: "信頼バンドルの一覧", purpose: "外部発行者と検証に使う信頼情報を登録する", content: #["OIDC と SPIFFE の方式、発行者と公開鍵", "発行者メタデータの取得候補"]}) @category(Category.workload)
  create: PageData<{
    @transition(#{...update, event: "メタデータを取得", result: "発行者の設定候補を表示する"}, WorkloadIdentityPages.create) @uses(IdMagic.WorkloadIdentity.Operations.DiscoverWorkloadIssuerMetadata) discover: unknown;
    @transition(#{event: "登録", condition: "作成が成功", result: "一覧へ戻る"}, WorkloadIdentityPages.list) @uses(IdMagic.WorkloadIdentity.Operations.RegisterWorkloadTrustBundle) save: unknown;
    @transition(#{event: "キャンセル"}, WorkloadIdentityPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-workload-trust-bundle-detail", name: "信頼バンドルの詳細", entry: "信頼バンドル一覧", purpose: "発行者の信頼状態と Agent への関連付けを管理する", content: #["信頼情報と JWKS の更新結果", "有効化、無効化と削除の確認", "関連付けの追加・編集・有効化と解除", "対象パターンのプレビューと過大な一致の確認"]}) @category(Category.workload)
  detail: PageData<{
    @transition(#{event: "編集"}, WorkloadIdentityPages.edit) edit: unknown;
    @transition(#{...update, event: "公開鍵を再取得", result: "JWKS の更新結果を表示する"}, WorkloadIdentityPages.detail) @uses(IdMagic.WorkloadIdentity.Operations.RefreshWorkloadTrustBundleJWKS) refresh: unknown;
    @transition(#{...update, event: "有効化または無効化を確認", result: "信頼バンドルの状態を更新する"}, WorkloadIdentityPages.detail) @uses(IdMagic.WorkloadIdentity.Operations.EnableWorkloadTrustBundle, IdMagic.WorkloadIdentity.Operations.DisableWorkloadTrustBundle) status: unknown;
    @transition(#{event: "削除を確認", condition: "削除が成功", result: "一覧へ戻る"}, WorkloadIdentityPages.list) @uses(IdMagic.WorkloadIdentity.Operations.DeleteWorkloadTrustBundle) delete: unknown;
    @transition(#{...update, event: "関連付けを試行・追加・有効化または無効化・削除を確認", result: "対象パターンの結果または関連付けの一覧を更新する"}, WorkloadIdentityPages.detail) @uses(IdMagic.WorkloadIdentity.Operations.PreviewAgentWorkloadBindings, IdMagic.WorkloadIdentity.Operations.CreateAgentWorkloadBinding, IdMagic.WorkloadIdentity.Operations.EnableAgentWorkloadBinding, IdMagic.WorkloadIdentity.Operations.DisableAgentWorkloadBinding, IdMagic.WorkloadIdentity.Operations.DeleteAgentWorkloadBinding) bindings: unknown;
    @transition(#{event: "関連付けを編集"}, WorkloadIdentityPages.binding) binding: unknown;
    @transition(#{event: "戻る"}, WorkloadIdentityPages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-workload-trust-bundle-edit", name: "信頼バンドルを編集", entry: "信頼バンドル詳細", purpose: "発行者の信頼情報を更新する", content: #["現在の発行者と検証設定", "保存とキャンセル"]}) @category(Category.workload)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "詳細へ戻る"}, WorkloadIdentityPages.detail) @uses(IdMagic.WorkloadIdentity.Operations.UpdateWorkloadTrustBundle) save: unknown;
    @transition(#{event: "キャンセル"}, WorkloadIdentityPages.detail) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-workload-binding-edit", name: "ワークロードの関連付けを編集", entry: "信頼バンドル詳細", purpose: "主体のパターンと関連付ける Agent を変更する", content: #["一致する対象のプレビュー", "広いパターンの確認", "保存とキャンセル"]}) @category(Category.workload)
  binding: PageData<{
    @transition(#{...update, event: "パターンを試す", result: "対象の一致と警告を表示する"}, WorkloadIdentityPages.binding) @uses(IdMagic.WorkloadIdentity.Operations.PreviewAgentWorkloadBindings) preview: unknown;
    @transition(#{event: "保存", condition: "確認と更新が成功", result: "信頼バンドルの詳細へ戻る"}, WorkloadIdentityPages.detail) @uses(IdMagic.WorkloadIdentity.Operations.UpdateAgentWorkloadBinding) save: unknown;
    @transition(#{event: "キャンセル"}, WorkloadIdentityPages.detail) cancel: unknown;
  }>;
}
