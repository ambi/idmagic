namespace IdMagicUi;

model TokenVaultAdminPages {
  @page(#{...adminContext, id: "admin-token-vault", name: "外部サービスの接続定義", entry: "管理メニュー", purpose: "外部サービスの認可と呼び出しの接続定義を管理する", content: #["接続名、状態、トークン URL と規則数", "再取得、新規登録、有効化と停止・削除の確認"]}) @category(Category.connections)
  list: PageData<{
    @transition(#{event: "登録"}, TokenVaultAdminPages.create) create: unknown;
    @transition(#{...update, event: "再取得", result: "接続定義を更新する"}, TokenVaultAdminPages.list) @uses(IdMagic.TokenVault.Operations.ListTokenVaultConnectors) refresh: unknown;
    @transition(#{...update, event: "有効化または停止・削除を確認", result: "定義の状態と一覧を更新する"}, TokenVaultAdminPages.list) @uses(IdMagic.TokenVault.Operations.UpdateTokenVaultConnector, IdMagic.TokenVault.Operations.DeleteTokenVaultConnector) status: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-token-vault-create", name: "外部サービスの接続定義を登録", entry: "接続定義の一覧", purpose: "OAuth の接続先と利用を許可する要求を登録する", content: #["認可とトークンの接続先、クライアント認証", "接続を使えるアプリケーション", "要求の規則と権限"]}) @category(Category.connections)
  create: PageData<{
    @transition(#{event: "登録", condition: "作成が成功", result: "接続定義の一覧へ戻る"}, TokenVaultAdminPages.list) @uses(IdMagic.TokenVault.Operations.CreateTokenVaultConnector) save: unknown;
    @transition(#{event: "キャンセル"}, TokenVaultAdminPages.list) cancel: unknown;
  }>;
}

model TokenVaultAccountPages {
  @page(#{...accountContext, id: "account-connections", name: "本人の外部サービス接続", entry: "本人のポータル、上流サービス認可の戻り先", purpose: "外部サービスとの接続と再認可、解除を管理する", content: #["コネクタ名と状態", "接続・再認可の必要性", "許可済みスコープ、最終利用日時", "上流サービスの認可結果"]}) @category(Category.connections)
  connections: PageData<{
    @transition(#{...navigation, event: "接続・再接続", condition: "コネクタが有効で接続が必要", result: "上流サービスの認可へ移り、このページへ戻る", toState: "external"}, Destinations.idp) @uses(IdMagic.TokenVault.Operations.AuthorizeMyTokenVaultConnection) connect: unknown;
    @transition(#{...update, event: "接続を解除", condition: "解除成功", result: "接続一覧を再取得して解除を通知する"}, TokenVaultAccountPages.connections) @uses(IdMagic.TokenVault.Operations.UnlinkMyTokenVaultConnection) unlink: unknown;
  }>;
}
