namespace IdMagicUi;

model OAuthPages {
  @page(#{...browserContext, id: "consent", name: "アプリケーションへの同意", route: "/consent", path: "/consent", entry: "認可トランザクション", purpose: "開示する情報と認可詳細を確認して許可または拒否する", content: #["要求元アプリケーション", "スコープごとの説明", "認可詳細", "許可と拒否"]}) @category(Category.consent)
  consent: PageData<{
    @form(#{name: "同意の判断", condition: "トランザクションを表示できる場合", commit: "許可または拒否"}) decision: {
      @field(#{label: "同意の判断", control: "許可・拒否ボタン", initial: "未選択", condition: "送信中以外", mapping: "BrowserConsentRequest.action"}) @bind(IdMagic.Contract.BrowserConsentRequest.action) action: "allow" | "deny";
    };
  }, {
    @transition(#{...update, event: "許可・拒否", condition: "判断の記録成功", result: "認可トランザクションの結果を呼び出し元へ返す", toState: "external"}, Destinations.client) @uses(IdMagic.OAuth2.Operations.SubmitBrowserConsent) decide: unknown;
    @transition(#{...update, event: "送信", condition: "拒否または通信失敗", result: "判断が完了していないことを表示し再試行できる", toState: "error"}, OAuthPages.consent) failure: unknown;
  }>;
  @page(#{...browserContext, id: "device", name: "デバイスの認可", route: "/device", path: "/device", entry: "デバイスが提示する確認用 URL", purpose: "デバイスのコードを指定して認可を許可または拒否する", content: #["ユーザーコードの入力", "許可と拒否", "認証切れとコードの問題"]}) @category(Category.consent)
  device: PageData<{
    @form(#{name: "デバイスの認可", condition: "常時", commit: "許可または拒否"}) device: {
      @field(#{label: "ユーザーコード", control: "text", initial: "表示 API の user_code。なければ空", condition: "送信中以外", mapping: "BrowserDeviceRequest.user_code"}) @bind(IdMagic.Contract.BrowserDeviceRequest.user_code) code: string;
      @field(#{label: "認可の判断", control: "許可・拒否ボタン", initial: "未選択", condition: "コードを入力した場合", mapping: "BrowserDeviceRequest.action"}) @bind(IdMagic.Contract.BrowserDeviceRequest.action) action: string;
    };
  }, {
    @transition(#{...navigation, event: "許可・拒否", condition: "判断の記録成功", result: "承認または拒否の完了を表示する"}, FrontendPages.status) @uses(IdMagic.OAuth2.Operations.SubmitBrowserDevice) decide: unknown;
    @transition(#{...navigation, event: "送信", condition: "authentication_required", result: "認証が必要なことを案内する"}, FrontendPages.status) expired: unknown;
    @transition(#{...update, event: "送信", condition: "コードまたは通信の問題", result: "コードを保持して問題を表示する", toState: "error"}, OAuthPages.device) failure: unknown;
  }, {
    @field(#{label: "事前入力するコード", control: "URL", initial: "user_code", condition: "URL に含まれる場合", mapping: "表示 API とフォームの初期値"}) code: string;
  }>;
  @page(#{...accountContext, id: "account-applications", name: "本人の同意と Agent への委任", route: "/account/applications", path: "/account/applications", entry: "本人のポータル", purpose: "アプリケーションへの同意を取り消し、Agent の委任を管理する", content: #["同意したアプリケーションとスコープ", "Agent ごとの委任・期限・利用履歴", "委任の追加"]}) @category(Category.consent)
  applications: PageData<{
    @form(#{name: "Agent への委任", condition: "委任の追加欄", commit: "委任"}) delegation: {
      @field(#{...entered, label: "委任先 Agent", control: "select", mapping: "委任可能 Agent 一覧の agent_id"}) agentId: string;
      @field(#{...entered, label: "委任するスコープ", control: "text", mapping: "空白で区切って scopes 配列へ変換する"}) scopes: string;
    };
  }, {
    @transition(#{...update, event: "同意を取り消す", condition: "取り消し成功", result: "アプリケーションを同意一覧から除く"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.RevokeMyConsent) revokeConsent: unknown;
    @transition(#{...update, event: "Agent に委任", condition: "付与成功", result: "委任を一覧へ反映し入力を消す"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.GrantMyAgentDelegation) grant: unknown;
    @transition(#{...update, event: "委任を取り消す", condition: "取り消し成功", result: "委任を一覧から除く"}, OAuthPages.applications) @uses(IdMagic.OAuth2.Operations.RevokeMyAgentDelegation) revokeDelegation: unknown;
    @transition(#{...update, event: "更新", condition: "API の拒否または通信失敗", result: "問題を表示して未確定の操作を残す", toState: "error"}, OAuthPages.applications) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-approvals", name: "本人宛ての承認依頼", route: "/account/approvals", path: "/account/approvals", entry: "本人のポータル", purpose: "Agent の実行や認可の依頼を確認して承認または拒否する", content: #["要求元、対象、要求内容と期限", "承認と拒否", "判断の完了通知"]}) @category(Category.consent)
  approvals: PageData<{
    @form(#{name: "承認の判断", condition: "未決の依頼ごと", commit: "承認または拒否"}) decision: {
      @field(#{label: "承認の判断", control: "承認・拒否ボタン", initial: "未選択", condition: "対象の依頼を処理していない場合", mapping: "AccountApprovalDecisionRequest の decision", repeated: true}) decision: "approve" | "deny";
    };
  }, {
    @transition(#{...update, event: "承認・拒否", condition: "再認証と判断の記録成功", result: "対象を一覧から除き、判断を通知する"}, OAuthPages.approvals) @uses(IdMagic.OAuth2.Operations.DecideMyApprovalRequest) decide: unknown;
    @transition(#{...update, event: "判断", condition: "再認証の中止、期限切れまたは API の拒否", result: "判断を確定せず理由を表示する", toState: "error"}, OAuthPages.approvals) failure: unknown;
  }>;
}
