namespace IdMagicUi;

model AdminConsentPages {
  @page(#{...adminContext, id: "admin-consents", name: "同意と Agent 委任の管理", entry: "管理メニュー", purpose: "利用者の同意を確認して取り消し、Agent への委任を確認する", content: #["ユーザー、アプリケーション、スコープと期限", "検索と再取得", "同意の取消確認", "Agent 委任の一覧"]}) @category(Category.consent)
  list: PageData<{
    @transition(#{...update, event: "検索または再取得", result: "対象の同意と委任を表示する"}, AdminConsentPages.list) @uses(IdMagic.OAuth2.Operations.ListAdminConsents, IdMagic.OAuth2.Operations.ListAdminAgentDelegations) refresh: unknown;
    @transition(#{...update, event: "取消を確認", result: "対象の同意を取り消し一覧を更新する"}, AdminConsentPages.list) @uses(IdMagic.OAuth2.Operations.RevokeAdminConsent) revoke: unknown;
  }, ListStates>;
}

model CrossAppAccessPages {
  @page(#{...adminContext, id: "admin-cross-app-access", name: "アプリ間アクセスの規則", entry: "管理メニュー", purpose: "アプリ間の発行と受け入れの規則を管理する", content: #["発行規則と受け入れ規則の一覧", "再取得、作成と削除の確認"]}) @category(Category.authorization)
  list: PageData<{
    @transition(#{...update, event: "再取得または削除を確認", result: "発行と受け入れの規則を更新する"}, CrossAppAccessPages.list) @uses(IdMagic.OAuth2.Operations.ListAdminCrossAppAccessIssuePolicies, IdMagic.OAuth2.Operations.ListAdminCrossAppAccessRedeemPolicies, IdMagic.OAuth2.Operations.DeleteAdminCrossAppAccessIssuePolicy, IdMagic.OAuth2.Operations.DeleteAdminCrossAppAccessRedeemPolicy) refresh: unknown;
    @transition(#{event: "発行規則を追加"}, CrossAppAccessPages.issue) issue: unknown;
    @transition(#{event: "受け入れ規則を追加"}, CrossAppAccessPages.redeem) redeem: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-cross-app-access-new-issue", name: "アプリ間の発行規則を追加", entry: "アプリ間アクセスの一覧", purpose: "対象アプリと発行できる範囲を登録する", content: #["要求元、対象と発行の条件", "作成とキャンセル"]}) @category(Category.authorization)
  issue: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "規則の一覧へ戻る"}, CrossAppAccessPages.list) @uses(IdMagic.OAuth2.Operations.CreateAdminCrossAppAccessIssuePolicy) save: unknown;
    @transition(#{event: "キャンセル"}, CrossAppAccessPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-cross-app-access-new-redeem", name: "アプリ間の受け入れ規則を追加", entry: "アプリ間アクセスの一覧", purpose: "受け入れる発行者と利用範囲を登録する", content: #["発行者、対象と権限の制約", "作成とキャンセル"]}) @category(Category.authorization)
  redeem: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "規則の一覧へ戻る"}, CrossAppAccessPages.list) @uses(IdMagic.OAuth2.Operations.CreateAdminCrossAppAccessRedeemPolicy) save: unknown;
    @transition(#{event: "キャンセル"}, CrossAppAccessPages.list) cancel: unknown;
  }>;
}

model McpResourceServerPages {
  @page(#{...adminContext, id: "admin-mcp-resource-servers", name: "MCP リソースサーバー", entry: "管理メニュー", purpose: "MCP の接続先、認証とアクセス規則を管理する", content: #["接続先と有効状態", "選択した対象の詳細、編集と削除確認", "Agent ごとのアクセス概要"]}) @category(Category.agents)
  list: PageData<{
    @transition(#{...update, event: "再取得または削除を確認", result: "対象サーバーの一覧を更新する"}, McpResourceServerPages.list) @uses(IdMagic.OAuth2.Operations.ListAdminMcpResourceServers, IdMagic.OAuth2.Operations.DeleteAdminMcpResourceServer) refresh: unknown;
    @transition(#{event: "作成"}, McpResourceServerPages.create) create: unknown;
    @transition(#{event: "編集"}, McpResourceServerPages.edit) edit: unknown;
    @transition(#{event: "Agent のアクセス"}, McpResourceServerPages.access) access: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-mcp-resource-server-create", name: "MCP リソースサーバーを登録", entry: "リソースサーバー一覧", purpose: "接続先と認証・アクセス規則を登録する", content: #["既存アプリケーションの利用または新規作成", "接続 URL、リソース識別子と認証", "ツールとアクセスの規則", "登録結果と初期資格情報の表示", "作成途中の失敗時は作成済みの対象を保持"]}) @category(Category.agents)
  create: PageData<{
    @transition(#{...update, event: "登録", result: "必要なアプリケーションと接続先を登録し結果を表示する", toState: "success"}, McpResourceServerPages.create) @uses(IdMagic.Application.Operations.CreateAdminApplication, IdMagic.Application.Operations.UpdateApplicationOidcConfig, IdMagic.OAuth2.Operations.CreateAdminMcpResourceServer) save: unknown;
    @transition(#{event: "ツールを設定", from: "success"}, McpResourceServerPages.edit) edit: unknown;
    @transition(#{event: "キャンセルまたは一覧へ"}, McpResourceServerPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-mcp-resource-server-edit", name: "MCP リソースサーバーを編集", entry: "リソースサーバー一覧", purpose: "接続先とツール・アクセス規則を更新する", content: #["現在の接続設定と規則", "ツールの取得とカタログ", "保存とキャンセル"]}) @category(Category.agents)
  edit: PageData<{
    @transition(#{...update, event: "ツールを取得", result: "発見したツールを編集候補として表示する"}, McpResourceServerPages.edit) @uses(IdMagic.OAuth2.Operations.DiscoverAdminMcpResourceServerTools) discover: unknown;
    @transition(#{event: "保存", condition: "更新が成功", result: "一覧へ戻る"}, McpResourceServerPages.list) @uses(IdMagic.OAuth2.Operations.UpdateAdminMcpResourceServer) save: unknown;
    @transition(#{event: "キャンセル"}, McpResourceServerPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-mcp-agent-access", name: "MCP サーバーの Agent アクセス", entry: "リソースサーバー一覧", purpose: "接続可能な Agent とアクセス設定を確認する", content: #["対象サーバーと Agent のアクセス概要", "Agent のリソース割り当ての編集へのリンク"]}) @category(Category.agents)
  access: PageData<{
    @transition(#{event: "Agent のアクセスを編集", data: "Agent ID と対象サーバー"}, AdminAgentPages.editResources) edit: unknown;
    @transition(#{event: "戻る"}, McpResourceServerPages.list) back: unknown;
  }, ReadStates>;
}

model AgentGuardrailPages {
  @page(#{...adminContext, id: "admin-agent-guardrail-edit", name: "Agent のガードレールを編集", entry: "Agent 詳細", purpose: "Agent 固有の制約を保存し、継承するデフォルト値と区別する", content: #["デフォルトのポリシーと個別の上書き", "許可する操作、制約と上限", "変更の保存と個別設定の削除確認"]}) @category(Category.agents)
  edit: PageData<{
    @transition(#{event: "保存", condition: "保存が成功", result: "Agent の詳細へ戻る"}, AdminAgentPages.detail) @uses(IdMagic.OAuth2.Operations.PutAdminAgentGuardrailPolicy) save: unknown;
    @transition(#{event: "個別設定の削除を確認", condition: "削除が成功", result: "デフォルト値を使う詳細へ戻る"}, AdminAgentPages.detail) @uses(IdMagic.OAuth2.Operations.DeleteAdminAgentGuardrailPolicy) delete: unknown;
    @transition(#{event: "キャンセル"}, AdminAgentPages.detail) cancel: unknown;
  }>;
}

model AdminRolePages {
  @page(#{...adminContext, id: "admin-roles", name: "ロール一覧", entry: "管理メニュー", purpose: "定義済みのロールと権限、付与人数を確認する", content: #["ロールの一覧と選択した詳細", "許可する操作と付与されたユーザー", "再取得と詳細への入口"]}) @category(Category.authorization)
  list: PageData<{
    @transition(#{...update, event: "再取得または選択", result: "ロールと付与されたユーザーを表示する"}, AdminRolePages.list) @uses(IdMagic.OAuth2.Operations.ListAdminRolePolicies) refresh: unknown;
    @transition(#{event: "詳細"}, AdminRolePages.detail) detail: unknown;
    @transition(#{event: "付与ユーザーを開く"}, AdminUserPages.users) users: unknown;
  }, ReadStates>;
  @page(#{...adminContext, id: "admin-role-detail", name: "ロール詳細", entry: "ロール一覧", purpose: "一つのロールの権限と付与先を確認する", content: #["操作と対象範囲", "付与人数とユーザー名"]}) @category(Category.authorization)
  detail: PageData<{
    @transition(#{event: "付与ユーザーを開く", data: "ロール名の検索条件"}, AdminUserPages.users) users: unknown;
    @transition(#{event: "戻る"}, AdminRolePages.list) back: unknown;
  }, ReadStates>;
}

model AuthorizationDetailTypePages {
  @page(#{...adminContext, id: "admin-authz-detail-types", name: "認可詳細の型", entry: "管理メニュー", purpose: "認可詳細のスキーマと利用するアプリを管理する", content: #["型、説明とスキーマ", "再取得、編集と削除確認", "アプリケーション管理への案内"]}) @category(Category.authorization)
  list: PageData<{
    @transition(#{event: "作成"}, AuthorizationDetailTypePages.create) create: unknown;
    @transition(#{event: "編集"}, AuthorizationDetailTypePages.edit) edit: unknown;
    @transition(#{...update, event: "再取得または削除を確認", result: "型の一覧を更新する"}, AuthorizationDetailTypePages.list) @uses(IdMagic.OAuth2.Operations.ListAuthorizationDetailTypes, IdMagic.OAuth2.Operations.DeleteAuthorizationDetailType) refresh: unknown;
    @transition(#{event: "アプリケーションへ"}, AdminApplicationPages.list) applications: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-authz-detail-type-create", name: "認可詳細の型を作成", entry: "認可詳細の型一覧", purpose: "型の識別子と JSON Schema を登録する", content: #["型、説明と JSON Schema", "作成とキャンセル"]}) @category(Category.authorization)
  create: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "型の一覧へ戻る"}, AuthorizationDetailTypePages.list) @uses(IdMagic.OAuth2.Operations.CreateAuthorizationDetailType) save: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationDetailTypePages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-authz-detail-type-edit", name: "認可詳細の型を編集", entry: "認可詳細の型一覧", purpose: "既存の型の説明とスキーマを変更する", content: #["現在の型、説明と JSON Schema", "保存とキャンセル"]}) @category(Category.authorization)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "型の一覧へ戻る"}, AuthorizationDetailTypePages.list) @uses(IdMagic.OAuth2.Operations.UpdateAuthorizationDetailType) save: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationDetailTypePages.list) cancel: unknown;
  }>;
}

model OAuthPages {
  // 経路は kind を認可トランザクションから決めるため、route を明示する。
  @page(#{...browserContext, id: "consent", name: "アプリケーションへの同意", route: "/consent", entry: "認可トランザクション", purpose: "開示する情報と認可詳細を確認して許可または拒否する", content: #["要求元アプリケーション", "スコープごとの説明", "認可詳細", "許可と拒否"]}) @category(Category.consent)
  consent: PageData<{
    @transition(#{...update, event: "許可・拒否", condition: "判断の記録成功", result: "認可トランザクションの結果を呼び出し元へ返す", toState: "external"}, Destinations.client) @uses(IdMagic.OAuth2.Operations.SubmitBrowserConsent) decide: unknown;
    @transition(#{...update, event: "送信", condition: "拒否または通信失敗", result: "判断が完了していないことを表示し再試行できる", toState: "error"}, OAuthPages.consent) failure: unknown;
  }>;
  @page(#{...browserContext, id: "device", name: "デバイスの認可", entry: "デバイスが提示する確認用 URL", purpose: "デバイスのコードを指定して認可を許可または拒否する", content: #["ユーザーコードの入力。URL の user_code があれば事前に入力する", "許可と拒否", "認証切れとコードの問題"]}) @category(Category.consent)
  device: PageData<{
    @transition(#{...navigation, event: "許可・拒否", condition: "判断の記録成功", result: "承認または拒否の完了を表示する"}, FrontendPages.status) @uses(IdMagic.OAuth2.Operations.SubmitBrowserDevice) decide: unknown;
    @transition(#{...navigation, event: "送信", condition: "authentication_required", result: "認証が必要なことを案内する"}, FrontendPages.status) expired: unknown;
    @transition(#{...update, event: "送信", condition: "コードまたは通信の問題", result: "コードを保持して問題を表示する", toState: "error"}, OAuthPages.device) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-applications", name: "本人の同意と Agent への委任", entry: "本人のポータル", purpose: "アプリケーションへの同意を取り消し、Agent の委任を管理する", content: #["同意したアプリケーションとスコープ", "Agent ごとの委任・期限・利用履歴", "委任の追加"]}) @category(Category.consent)
  applications: PageData<{
    @transition(#{...update, event: "同意を取り消す", condition: "取り消し成功", result: "アプリケーションを同意一覧から除く"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.RevokeMyConsent) revokeConsent: unknown;
    @transition(#{...update, event: "Agent に委任", condition: "付与成功", result: "委任を一覧へ反映し入力を消す"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.GrantMyAgentDelegation) grant: unknown;
    @transition(#{...update, event: "委任を取り消す", condition: "取り消し成功", result: "委任を一覧から除く"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.RevokeMyAgentDelegation) revokeDelegation: unknown;
    @transition(#{...update, event: "更新", condition: "API の拒否または通信失敗", result: "問題を表示して未確定の操作を残す", toState: "error"}, OAuthPages.applications) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-approvals", name: "本人宛ての承認依頼", entry: "本人のポータル", purpose: "Agent の実行や認可の依頼を確認して承認または拒否する", content: #["要求元、対象、要求内容と期限", "承認と拒否", "判断の完了通知"]}) @category(Category.consent)
  approvals: PageData<{
    @transition(#{...update, event: "承認・拒否", condition: "再認証と判断の記録成功", result: "対象を一覧から除き、判断を通知する"}, OAuthPages.approvals) @uses(IdMagic.OAuth2.Operations.DecideMyApprovalRequest) decide: unknown;
    @transition(#{...update, event: "判断", condition: "再認証の中止、期限切れまたは API の拒否", result: "判断を確定せず理由を表示する", toState: "error"}, OAuthPages.approvals) failure: unknown;
  }>;
}
