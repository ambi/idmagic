namespace IdMagicUi;

model TokenVaultAccountPages {
  @page(#{...accountContext, id: "account-connections", name: "本人の外部サービス接続", route: "/account/connections", path: "/account/connections", entry: "本人のポータル、上流サービス認可の戻り先", purpose: "外部サービスとの接続と再認可、解除を管理する", content: #["コネクタ名と状態", "接続・再認可の必要性", "許可済みスコープ、最終利用日時", "上流サービスの認可結果"]}) @category(Category.connections)
  connections: PageData<{}, {
    @transition(#{...navigation, event: "接続・再接続", condition: "コネクタが有効で接続が必要", result: "上流サービスの認可へ移り、このページへ戻る", toState: "external"}, Destinations.idp) @uses(IdMagic.TokenVault.Operations.AuthorizeMyTokenVaultConnection) connect: unknown;
    @transition(#{...update, event: "接続を解除", condition: "解除成功", result: "接続一覧を再取得して解除を通知する"}, TokenVaultAccountPages.connections) @uses(IdMagic.TokenVault.Operations.UnlinkMyTokenVaultConnection) unlink: unknown;
  }, {
    @field(#{label: "上流サービスの認可結果", control: "URL", initial: "result", condition: "認可から戻った場合", mapping: "linked、invalid_state、denied、exchange_failed を通知に変換する"}) result: string;
  }>;
}
