namespace IdMagicUi;

model AdminAgentPages {
  @page(#{...adminContext, id: "admin-agents", name: "Agent 一覧", entry: "管理メニュー", purpose: "Agent を選び、状態と責任者を確認する", content: #["一覧、ページ送りと再取得", "選択対象の詳細と削除の確認", "追加と棚卸しへの入口"]}) @category(Category.agents)
  list: PageData<{
    @transition(#{...update, event: "選択または再取得", result: "一覧と選択した Agent を表示する"}, AdminAgentPages.list) @uses(IdMagic.IdManagement.Operations.ListAgents) refresh: unknown;
    @transition(#{event: "作成"}, AdminAgentPages.create) create: unknown;
    @transition(#{event: "詳細"}, AdminAgentPages.detail) detail: unknown;
    @transition(#{event: "編集", condition: "緊急停止していない"}, AdminAgentPages.edit) edit: unknown;
    @transition(#{event: "アクセス概要"}, AdminAgentPages.access) access: unknown;
    @transition(#{event: "リソース"}, AdminAgentPages.resources) resources: unknown;
    @transition(#{event: "資格情報"}, AdminAgentPages.credentials) credentials: unknown;
    @transition(#{event: "棚卸し"}, AdminAgentPages.inventory) inventory: unknown;
    @transition(#{...update, event: "削除を確認", result: "対象を削除し一覧を更新する"}, AdminAgentPages.list) @uses(IdMagic.IdManagement.Operations.DeleteAgent) delete: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-agent-inventory", name: "Agent の棚卸し", entry: "Agent 一覧", purpose: "責任者、資格情報、利用状態の問題を見つける", content: #["検出事項と休眠日数による絞り込み", "責任者、最終利用と問題の一覧", "対象 Agent の詳細", "観測された Agent の候補、確認・除外・再開"]}) @category(Category.agents)
  inventory: PageData<{
    @transition(#{...update, event: "絞り込みまたは再取得", result: "条件に合う棚卸し結果を表示する"}, AdminAgentPages.inventory) @uses(IdMagic.IdManagement.Operations.ListAgentInventory) filter: unknown;
    @transition(#{...update, event: "観測を絞り込み・確認・除外・再開", result: "観測の判断と対応する Agent を更新する"}, AdminAgentPages.inventory) @uses(IdMagic.IdManagement.Operations.ListAgentObservations, IdMagic.IdManagement.Operations.ConfirmAgentObservation, IdMagic.IdManagement.Operations.ExcludeAgentObservation, IdMagic.IdManagement.Operations.ReopenAgentObservation) observations: unknown;
    @transition(#{event: "詳細"}, AdminAgentPages.detail) detail: unknown;
    @transition(#{event: "戻る"}, AdminAgentPages.list) back: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-agent-create", name: "Agent を登録", entry: "Agent 一覧", purpose: "責任者と初期の資格情報・アクセスを指定する", content: #["基本情報と責任者", "資格情報は後で設定、公開鍵、既存の資格情報から選択", "初期アクセス", "段階ごとの進捗と失敗した段階からの再試行"]}) @category(Category.agents)
  create: PageData<{
    @transition(#{...update, event: "登録", result: "Agent、資格情報と初期アクセスを順に登録し、結果を表示する", toState: "success"}, AdminAgentPages.create) @uses(IdMagic.IdManagement.Operations.RegisterAgent, IdMagic.OAuth2.Operations.CreateAdminOAuth2Client, IdMagic.IdManagement.Operations.BindAgentCredential, IdMagic.Authorization.Operations.WriteRelationTuples) save: unknown;
    @transition(#{...update, event: "失敗した登録を再試行", condition: "途中の段階が失敗", result: "作成済みの対象を保ち、残りの設定を再試行する"}, AdminAgentPages.create) retry: unknown;
    @transition(#{event: "作成した Agent を開く", from: "success"}, AdminAgentPages.detail) detail: unknown;
    @transition(#{event: "キャンセルまたは戻る"}, AdminAgentPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-agent-detail", name: "Agent 詳細", entry: "一覧または棚卸し", purpose: "Agent の責任者、状態と利用を管理する", content: #["基本情報、責任者と資格情報の概要", "無効化、有効化、緊急停止と削除の確認", "アクセス、リソース、資格情報、ガードレールの入口"]}) @category(Category.agents)
  detail: PageData<{
    @transition(#{event: "編集"}, AdminAgentPages.edit) edit: unknown;
    @transition(#{...update, event: "有効化または無効化・緊急停止を確認", result: "状態を変更して詳細を再取得する"}, AdminAgentPages.detail) @uses(IdMagic.IdManagement.Operations.EnableAgent, IdMagic.IdManagement.Operations.DisableAgent, IdMagic.IdManagement.Operations.KillAgent) status: unknown;
    @transition(#{event: "削除を確認", condition: "削除が成功", result: "一覧へ戻る"}, AdminAgentPages.list) @uses(IdMagic.IdManagement.Operations.DeleteAgent) delete: unknown;
    @transition(#{event: "アクセス概要"}, AdminAgentPages.access) access: unknown;
    @transition(#{event: "リソース"}, AdminAgentPages.resources) resources: unknown;
    @transition(#{event: "資格情報"}, AdminAgentPages.credentials) credentials: unknown;
    @transition(#{event: "ガードレールを編集"}, AgentGuardrailPages.edit) guardrail: unknown;
    @transition(#{event: "戻る"}, AdminAgentPages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-agent-edit", name: "Agent を編集", entry: "Agent 詳細", purpose: "表示情報と責任者を更新する", content: #["基本情報と責任者の変更", "保存と未保存の変更の破棄"]}) @category(Category.agents)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "詳細へ戻る"}, AdminAgentPages.detail) @uses(IdMagic.IdManagement.Operations.UpdateAgent) save: unknown;
    @transition(#{event: "キャンセル"}, AdminAgentPages.detail) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-agent-credentials", name: "Agent の資格情報", entry: "Agent 詳細", purpose: "認証に使う資格情報を作成または関連付ける", content: #["既存の資格情報と新規の公開鍵・SPIFFE の選択", "信頼バンドルと公開鍵の案内", "作成後の関連付けの部分失敗", "解除の確認と関連するリソースサーバー"]}) @category(Category.agents)
  credentials: PageData<{
    @transition(#{...update, event: "作成または関連付け", result: "資格情報を登録し Agent へ関連付けて再取得する"}, AdminAgentPages.credentials) @uses(IdMagic.OAuth2.Operations.CreateAdminOAuth2Client, IdMagic.IdManagement.Operations.BindAgentCredential) bind: unknown;
    @transition(#{...update, event: "解除を確認", result: "関連付けを解除して資格情報を再取得する"}, AdminAgentPages.credentials) @uses(IdMagic.IdManagement.Operations.UnbindAgentCredential) unbind: unknown;
    @transition(#{event: "信頼バンドルを管理"}, WorkloadIdentityPages.list) trustBundles: unknown;
    @transition(#{event: "リソースサーバーを登録"}, McpResourceServerPages.create) resourceServer: unknown;
    @transition(#{event: "戻る"}, AdminAgentPages.detail) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-agent-resources", name: "Agent のリソース", entry: "Agent 詳細", purpose: "リソースへの割り当てを確認する", content: #["認可モデルに応じたリソースと関係", "アクセス管理、資格情報とリソースサーバーへの案内"]}) @category(Category.agents)
  resources: PageData<{
    @transition(#{event: "編集"}, AdminAgentPages.editResources) edit: unknown;
    @transition(#{event: "アクセス管理"}, AuthorizationPages.access) access: unknown;
    @transition(#{event: "資格情報"}, AdminAgentPages.credentials) credentials: unknown;
    @transition(#{event: "リソースサーバー"}, McpResourceServerPages.list) resourceServers: unknown;
    @transition(#{event: "戻る"}, AdminAgentPages.detail) back: unknown;
  }, ReadStates>;
  @page(#{...adminContext, id: "admin-agent-resources-edit", name: "Agent のリソースを編集", entry: "Agent のリソース", purpose: "対象リソースと関係の割り当てを変更する", content: #["現在の割り当てと差分", "認可モデルがない場合の案内", "保存とキャンセル"]}) @category(Category.agents)
  editResources: PageData<{
    @transition(#{event: "保存", condition: "書き込みが成功", result: "リソースの表示へ戻る"}, AdminAgentPages.resources) @uses(IdMagic.Authorization.Operations.WriteRelationTuples) save: unknown;
    @transition(#{event: "キャンセル"}, AdminAgentPages.resources) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-agent-access-summary", marker: "admin-agent-access", route: "/admin/agents_/$agentId/access", name: "Agent のアクセス概要", entry: "Agent 詳細", purpose: "Agent に関連する認証、委任、リソースへの権限を確認する", content: #["資格情報、本人からの委任と認可詳細", "リソースサーバーの設定と各リソースへのアクセスの説明"]}) @category(Category.agents)
  access: PageData<{
    @transition(#{event: "資格情報"}, AdminAgentPages.credentials) credentials: unknown;
    @transition(#{event: "対象リソースサーバーを編集"}, McpResourceServerPages.edit) server: unknown;
    @transition(#{event: "委任したユーザーを開く", data: "ユーザー ID"}, AdminUserPages.detail) user: unknown;
    @transition(#{event: "戻る"}, AdminAgentPages.detail) back: unknown;
  }, ReadStates>;
}

const userAdminContext = #{area: Area.admin, actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"]};

/** IdManagement が主担当となるユーザー管理のページ。 */
model AdminUserPages {
  @page(#{...userAdminContext, id: "admin-user-export", name: "ユーザーの CSV エクスポート", purpose: "出力ジョブを開始し、完了した CSV を取得する", content: #["出力列の選択", "ジョブの状態と進捗", "再取得、キャンセルと期限内のダウンロード"]}) @category(Category.users)
  exportCsv: PageData<{
    @transition(#{...update, event: "出力を開始", result: "出力ジョブを一覧へ追加する"}, AdminUserPages.exportCsv) @uses(IdMagic.IdManagement.Operations.StartUserCsvExport) start: unknown;
    @transition(#{...update, event: "再取得", result: "ジョブの状態と進捗を更新する"}, AdminUserPages.exportCsv) @uses(IdMagic.IdManagement.Operations.ListUserExports) refresh: unknown;
    @transition(#{...update, event: "キャンセル", condition: "停止可能なジョブ", result: "ジョブの停止を要求する"}, AdminUserPages.exportCsv) @uses(IdMagic.IdManagement.Operations.CancelUserExport) cancel: unknown;
    @transition(#{event: "ダウンロード", condition: "完了し保持期限内", result: "CSV を取得する", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.DownloadUserExportFile) download: unknown;
    @transition(#{event: "一覧へ戻る"}, AdminUserPages.users) back: unknown;
  }, ListStates>;
  @page(#{...userAdminContext, id: "admin-users", name: "ユーザー一覧", purpose: "ユーザーを検索し、対象の確認と管理の入口を示す", content: #["検索語と状態による絞り込み。条件は URL に保持する", "ユーザー名、表示名、メールアドレス、状態とロール", "総件数、ページ送りと再取得", "選択したユーザーの概要、必須アクションと所属グループ", "無効化、削除の予約、完全削除と復元の確認", "作成と CSV の入口"]})
  @category(Category.users)
  users: PageData<{
    @transition(#{event: "検索または絞り込み", result: "条件に合う一覧を表示する", data: "query、status、cursor", mode: "data"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.ListAdminUsers) search: unknown;
    @transition(#{...update, event: "ユーザーを選択", result: "選択したユーザーの概要を表示する"}, AdminUserPages.users) select: unknown;
    @transition(#{...update, event: "無効化を確認または有効化", result: "状態を変更して一覧を再取得する"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.DisableAdminUser, IdMagic.IdManagement.Operations.EnableAdminUser) status: unknown;
    @transition(#{...update, event: "削除・完全削除を確認または復元", result: "削除の予約、完全削除または復元を行い一覧を再取得する"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.DeleteAdminUser, IdMagic.IdManagement.Operations.RestoreAdminUser) lifecycle: unknown;
    @transition(#{event: "編集", data: "ユーザー ID"}, AdminUserPages.edit) edit: unknown;
    @transition(#{event: "所属グループを開く", data: "グループ ID"}, AdminGroupPages.list) group: unknown;
    @transition(#{event: "詳細", data: "ユーザー ID"}, AdminUserPages.detail) detail: unknown;
    @transition(#{event: "ユーザーを追加"}, AdminUserPages.create) create: unknown;
    @transition(#{event: "CSV インポート"}, AdminUserPages.importCsv) importCsv: unknown;
    @transition(#{event: "CSV エクスポート"}, AdminUserPages.exportCsv) exportCsv: unknown;
  }, ListStates>;

  @page(#{...userAdminContext, id: "admin-user-detail", name: "ユーザー詳細", purpose: "現在のプロフィール、権限、所属と認証の状態を確認する", content: #["プロフィール", "ロールとグループ", "認証情報とセッションの一覧", "必須アクション", "無効化、削除の予約、完全削除と復元の確認。削除はユーザー名の入力で確認する", "MFA 登録の一時免除の発行と取消", "解除する認証情報を選ぶリセットのダイアログ"]})
  @category(Category.users)
  detail: PageData<{
    @transition(#{event: "編集", data: "ユーザー ID"}, AdminUserPages.edit) edit: unknown;
    @transition(#{event: "所属グループを開く", data: "グループ ID"}, AdminGroupPages.list) group: unknown;
    @transition(#{...update, event: "無効化を確認または有効化", result: "状態を変更して詳細を再取得する"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.DisableAdminUser, IdMagic.IdManagement.Operations.EnableAdminUser) status: unknown;
    @transition(#{...update, event: "削除を確認または復元", condition: "確認が成立し API が受理", result: "削除の予約または復元を反映して詳細を再取得する", data: "ユーザー ID"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.DeleteAdminUser, IdMagic.IdManagement.Operations.RestoreAdminUser) delete: unknown;
    @transition(#{event: "完全削除を確認", condition: "削除予約中で、確認が成立し API が受理", result: "一覧へ移る", data: "ユーザー ID"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.DeleteAdminUser) purge: unknown;
    @transition(#{...update, event: "MFA 登録の免除を発行または取消", result: "免除の状態を更新する"}, AdminUserPages.detail) @uses(IdMagic.Authentication.Operations.IssueMfaEnrollmentBypass, IdMagic.Authentication.Operations.RevokeMfaEnrollmentBypass) bypass: unknown;
    @transition(#{...update, event: "認証情報のリセットを確認", result: "選んだ認証情報を解除し結果を表示する"}, AdminUserPages.detail) @uses(IdMagic.Authentication.Operations.ResetUserAuthenticators) resetAuthenticators: unknown;
    @transition(#{event: "一覧へ戻る"}, AdminUserPages.users) back: unknown;
    @transition(#{event: "管理操作", condition: "API が再認証を要求", result: "再認証後に元の操作を一度だけ再試行する", toState: "dialog", data: "元の操作と再認証の要求"}, Destinations.stepUp) stepUp: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-create", name: "ユーザーを追加", purpose: "初期資格情報とプロフィールを指定してユーザーを作成する", content: #["ユーザー名と表示名", "メールと確認済みの指定", "初期パスワード", "初期ロール"]})
  @category(Category.users)
  create: PageData<{
    @transition(#{event: "作成", condition: "作成に成功", result: "作成したユーザーの詳細を取得する", data: "作成したユーザー ID"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.CreateAdminUser) create: unknown;
    @transition(#{event: "キャンセル"}, AdminUserPages.users) cancel: unknown;
    @transition(#{event: "作成", condition: "作成を拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error", mode: "data"}, AdminUserPages.create) failure: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-edit", name: "ユーザーを編集", purpose: "プロフィール、ロールとカスタム属性を一回の保存で変更する", content: #["プロフィールの編集", "ロールの変更内容の確認", "属性定義に応じたカスタム属性", "所属グループの追加と除外", "必須アクションの設定と解除", "セッションの個別失効とすべての失効"]})
  @category(Category.users)
  edit: PageData<{
    @transition(#{...update, event: "グループへ追加または除外", result: "所属グループを再取得する"}, AdminUserPages.edit) @uses(IdMagic.IdManagement.Operations.AddGroupMember, IdMagic.IdManagement.Operations.RemoveGroupMember) groups: unknown;
    @transition(#{event: "所属グループを開く", data: "グループ ID"}, AdminGroupPages.list) group: unknown;
    @transition(#{...update, event: "必須アクションを切り替える", result: "ユーザーを再取得して切り替えを通知する"}, AdminUserPages.edit) @uses(IdMagic.IdManagement.Operations.SetUserRequiredAction, IdMagic.IdManagement.Operations.ClearUserRequiredAction) requiredAction: unknown;
    @transition(#{...update, event: "セッションを失効またはすべて失効を確認", result: "セッション一覧を更新する"}, AdminUserPages.edit) @uses(IdMagic.Authentication.Operations.RevokeSession, IdMagic.Authentication.Operations.RevokeUserSessions) sessions: unknown;
    @transition(#{event: "保存", condition: "変更内容を確認し保存に成功", result: "更新後の詳細を再取得して一度通知する", data: "ユーザー ID"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.UpdateAdminUser) save: unknown;
    @transition(#{event: "キャンセル", result: "未保存の入力を破棄して詳細へ戻る", data: "ユーザー ID"}, AdminUserPages.detail) cancel: unknown;
    @transition(#{event: "保存", condition: "更新の拒否または競合", result: "入力を保持し問題を操作の近くに残す", toState: "error", mode: "data"}, AdminUserPages.edit) failure: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-import", name: "ユーザーの CSV インポート", purpose: "CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する", content: #["テンプレートのダウンロード", "ファイル選択", "プレビュー", "適用の確認ダイアログ", "行と列ごとの結果と問題"]})
  @category(Category.users)
  importCsv: PageData<{
    @transition(#{event: "プレビュー", condition: "プレビューのジョブが完了", result: "適用前の内容と行ごとの問題を表示する", toState: "preview", data: "CSV 内容", mode: "data"}, AdminUserPages.importCsv) preview: unknown;
    @transition(#{event: "取り込みを適用", from: "preview", condition: "プレビューが完了", result: "適用前の確認ダイアログを表示する", toState: "confirm", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv) confirm: unknown;
    @transition(#{event: "キャンセル", from: "confirm", condition: "適用を取りやめる", result: "確認を閉じてプレビューへ戻る", toState: "preview", mode: "data"}, AdminUserPages.importCsv) cancelApply: unknown;
    @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "確認が成立し適用が完了", result: "処理件数と失敗した行を表示する", toState: "success", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv) apply: unknown;
    @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "一部の行の処理が失敗", result: "成功と失敗を分けて表示する", toState: "partial", mode: "data"}, AdminUserPages.importCsv) partial: unknown;
    @transition(#{event: "一覧へ戻る"}, AdminUserPages.users) back: unknown;
  }, ImportStates>;
}

model AdminGroupPages {
  @page(#{...adminContext, id: "admin-groups", name: "グループ一覧", entry: "管理メニュー", purpose: "グループを選び、ロールと所属人数を確認する", content: #["ページ送りと再取得", "選択したグループの詳細と削除の確認", "作成、CSV 取り込みと出力への入口"]}) @category(Category.groups)
  list: PageData<{
    @transition(#{...update, event: "選択または再取得", result: "選択対象の詳細と一覧を表示する"}, AdminGroupPages.list) @uses(IdMagic.IdManagement.Operations.ListGroups) refresh: unknown;
    @transition(#{event: "追加"}, AdminGroupPages.create) create: unknown;
    @transition(#{event: "詳細"}, AdminGroupPages.detail) detail: unknown;
    @transition(#{event: "編集"}, AdminGroupPages.edit) edit: unknown;
    @transition(#{event: "メンバーを出力する"}, AdminGroupPages.exportMembers) exportMembers: unknown;
    @transition(#{event: "メンバーのユーザーを開く", data: "ユーザー名"}, AdminUserPages.users) member: unknown;
    @transition(#{...update, event: "削除を確認", result: "対象を削除して一覧を更新する"}, AdminGroupPages.list) @uses(IdMagic.IdManagement.Operations.DeleteGroup) delete: unknown;
    @transition(#{event: "CSV 取り込み"}, AdminGroupPages.importCsv) importCsv: unknown;
    @transition(#{event: "CSV 出力"}, AdminGroupPages.exportCsv) exportCsv: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-group-detail", name: "グループ詳細", entry: "グループへの直接アクセス", purpose: "グループの属性とメンバーを確認する", content: #["ロール、属性と管理元", "メンバーの一覧", "メンバーの CSV 出力の入口", "削除の確認"]}) @category(Category.groups)
  detail: PageData<{
    @transition(#{event: "編集"}, AdminGroupPages.edit) edit: unknown;
    @transition(#{event: "メンバーを出力する"}, AdminGroupPages.exportMembers) exportMembers: unknown;
    @transition(#{event: "メンバーのユーザーを開く", data: "ユーザー名"}, AdminUserPages.users) member: unknown;
    @transition(#{event: "削除を確認", condition: "確認後に削除が成功", result: "一覧へ戻る"}, AdminGroupPages.list) @uses(IdMagic.IdManagement.Operations.DeleteGroup) delete: unknown;
    @transition(#{event: "戻る"}, AdminGroupPages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-group-create", name: "グループを作成", entry: "グループ一覧", purpose: "グループの基本情報とカスタム属性を登録する", content: #["名前、説明、メールとロール", "定義に応じたグループ属性"]}) @category(Category.groups)
  create: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "作成したグループの詳細へ進む"}, AdminGroupPages.detail) @uses(IdMagic.IdManagement.Operations.CreateGroup) save: unknown;
    @transition(#{event: "キャンセル"}, AdminGroupPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-group-edit", name: "グループを編集", entry: "一覧または詳細", purpose: "属性、ロール、所属と動的ルールを管理する", content: #["基本情報と属性の保存", "SCIM 管理元の注意", "メンバーの追加と除外", "メンバーの CSV 取り込みと出力の入口", "動的ルールのビルダーと CEL 編集、プレビュー、有効化", "ロール委任の申請と取消"]}) @category(Category.groups)
  edit: PageData<{
    @transition(#{event: "保存", condition: "変更があり保存が成功", result: "詳細へ戻る"}, AdminGroupPages.detail) @uses(IdMagic.IdManagement.Operations.UpdateGroup) save: unknown;
    @transition(#{...update, event: "ルールを保存・試行・有効化または無効化", result: "ルールまたは試行結果を表示する"}, AdminGroupPages.edit) @uses(IdMagic.IdManagement.Operations.UpdateDynamicGroupRule, IdMagic.IdManagement.Operations.PreviewDynamicGroupRule, IdMagic.IdManagement.Operations.EnableDynamicGroupRule, IdMagic.IdManagement.Operations.DisableDynamicGroupRule) rule: unknown;
    @transition(#{...update, event: "委任を申請または取消", result: "委任の状態を更新する"}, AdminGroupPages.edit) @uses(IdMagic.IdManagement.Operations.SetGroupRoleDelegation) delegation: unknown;
    @transition(#{...update, event: "メンバーを追加または除外", result: "メンバーを再取得する"}, AdminGroupPages.edit) @uses(IdMagic.IdManagement.Operations.AddGroupMember, IdMagic.IdManagement.Operations.RemoveGroupMember) members: unknown;
    @transition(#{event: "メンバーを取り込む", condition: "静的なメンバーで外部の管理元がない"}, AdminGroupPages.importMembers) importMembers: unknown;
    @transition(#{event: "メンバーを出力する"}, AdminGroupPages.exportMembers) exportMembers: unknown;
    @transition(#{event: "メンバーのユーザーを開く", data: "ユーザー名"}, AdminUserPages.users) member: unknown;
    @transition(#{event: "キャンセルまたは戻る"}, AdminGroupPages.detail) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-group-import", name: "グループの CSV インポート", entry: "グループ一覧", purpose: "CSV の内容を確認してグループを取り込む", content: #["テンプレート、ファイル選択とプレビュー", "適用確認と行ごとの結果", "エラーのページ送りと部分失敗"]}) @category(Category.groups)
  importCsv: PageData<{
    @transition(#{...update, event: "プレビュー", result: "完了まで状態を取得して行ごとの問題を表示する", toState: "preview"}, AdminGroupPages.importCsv) @uses(IdMagic.IdManagement.Operations.ImportAdminGroups, IdMagic.IdManagement.Operations.GetAdminGroupImport) preview: unknown;
    @transition(#{...update, event: "適用を確認", from: "preview", result: "確認ダイアログを表示する", toState: "confirm"}, AdminGroupPages.importCsv) confirm: unknown;
    @transition(#{...update, event: "キャンセル", from: "confirm", result: "プレビューへ戻る", toState: "preview"}, AdminGroupPages.importCsv) cancel: unknown;
    @transition(#{...update, event: "確認して適用", from: "confirm", result: "適用結果と失敗した行を表示する", toState: "success"}, AdminGroupPages.importCsv) @uses(IdMagic.IdManagement.Operations.ApplyAdminGroupImport, IdMagic.IdManagement.Operations.GetAdminGroupImport) apply: unknown;
    @transition(#{event: "戻る"}, AdminGroupPages.list) back: unknown;
  }, ImportStates>;
  @page(#{...adminContext, id: "admin-group-member-import", name: "グループメンバーの CSV インポート", entry: "グループ詳細", purpose: "指定グループの所属を CSV で追加または除外する", content: #["所属の有無を指定する CSV とテンプレート", "プレビュー、確認、行ごとの適用結果と部分失敗"]}) @category(Category.groups)
  importMembers: PageData<{
    @transition(#{...update, event: "プレビュー", result: "行ごとの所属変更を表示する", toState: "preview"}, AdminGroupPages.importMembers) @uses(IdMagic.IdManagement.Operations.ImportAdminGroupMembers, IdMagic.IdManagement.Operations.GetAdminGroupMemberImport) preview: unknown;
    @transition(#{...update, event: "適用を確認", from: "preview", result: "確認ダイアログを表示する", toState: "confirm"}, AdminGroupPages.importMembers) confirm: unknown;
    @transition(#{...update, event: "キャンセル", from: "confirm", result: "プレビューへ戻る", toState: "preview"}, AdminGroupPages.importMembers) cancel: unknown;
    @transition(#{...update, event: "確認して適用", from: "confirm", result: "所属変更の結果と失敗した行を表示する", toState: "success"}, AdminGroupPages.importMembers) @uses(IdMagic.IdManagement.Operations.ApplyAdminGroupMemberImport, IdMagic.IdManagement.Operations.GetAdminGroupMemberImport) apply: unknown;
    @transition(#{event: "戻る"}, AdminGroupPages.detail) back: unknown;
  }, ImportStates>;
  @page(#{...adminContext, id: "admin-group-export", name: "グループの CSV エクスポート", entry: "グループ一覧", purpose: "グループ情報を CSV で取得する", content: #["出力列の選択", "ジョブの状態、キャンセルと期限内の取得"]}) @category(Category.groups)
  exportCsv: PageData<{
    @transition(#{...update, event: "出力を開始または再取得", result: "ジョブの状態と進捗を表示する"}, AdminGroupPages.exportCsv) @uses(IdMagic.IdManagement.Operations.StartGroupCsvExport, IdMagic.IdManagement.Operations.ListGroupExports) start: unknown;
    @transition(#{...update, event: "キャンセル", result: "停止可能なジョブの停止を要求する"}, AdminGroupPages.exportCsv) @uses(IdMagic.IdManagement.Operations.CancelGroupExport) cancel: unknown;
    @transition(#{event: "ダウンロード", condition: "完了し保持期限内", result: "CSV を取得する", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.DownloadGroupExportFile) download: unknown;
    @transition(#{event: "戻る"}, AdminGroupPages.list) back: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-group-members-export", name: "グループメンバーの CSV エクスポート", entry: "グループ詳細", purpose: "指定グループの所属を CSV で取得する", content: #["出力列の選択", "ジョブの状態、キャンセルと期限内の取得"]}) @category(Category.groups)
  exportMembers: PageData<{
    @transition(#{...update, event: "出力を開始または再取得", result: "対象グループの出力ジョブを表示する"}, AdminGroupPages.exportMembers) @uses(IdMagic.IdManagement.Operations.StartGroupMemberCsvExport, IdMagic.IdManagement.Operations.ListGroupMemberExports) start: unknown;
    @transition(#{...update, event: "キャンセル", result: "停止可能なジョブの停止を要求する"}, AdminGroupPages.exportMembers) @uses(IdMagic.IdManagement.Operations.CancelGroupMemberExport) cancel: unknown;
    @transition(#{event: "ダウンロード", condition: "完了し保持期限内", result: "CSV を取得する", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.DownloadGroupMemberExportFile) download: unknown;
    @transition(#{event: "戻る"}, AdminGroupPages.detail) back: unknown;
  }, ListStates>;
}

@flow(#{id: "manage-users", name: "ユーザーを作成して編集する", goal: "ユーザーを作成し、プロフィールとロールを確認して変更する。", notes: #["既存のユーザーは一覧から詳細へ進み、同じ編集の手順を使う。", "入力不備、権限不足、保存の失敗は各ページの操作表に示す。"]})
model ManageUsersFlow {
  @view(#{state: "ready", visible: #["ユーザー名・状態・ロール", "検索と絞り込み"]}, AdminUserPages.users)
  list: unknown;
  @view(#{state: "ready", visible: #["プロフィール", "初期パスワード・ロール"]}, AdminUserPages.create)
  create: unknown;
  @view(#{state: "ready", visible: #["プロフィール・ロール", "所属・認証状態"]}, AdminUserPages.detail)
  detail: unknown;
  @view(#{state: "ready", visible: #["プロフィール・ロール", "変更内容"]}, AdminUserPages.edit)
  edit: unknown;
  @flowLink(#{trigger: "追加", action: "create"}, ManageUsersFlow.list, ManageUsersFlow.create)
  startCreate: unknown;
  @flowLink(#{trigger: "作成", action: "create"}, ManageUsersFlow.create, ManageUsersFlow.detail)
  created: unknown;
  @flowLink(#{trigger: "ユーザーを選択", action: "detail"}, ManageUsersFlow.list, ManageUsersFlow.detail)
  selectExisting: unknown;
  @flowLink(#{trigger: "編集", action: "edit"}, ManageUsersFlow.detail, ManageUsersFlow.edit)
  startEdit: unknown;
  @flowLink(#{trigger: "保存", action: "save"}, ManageUsersFlow.edit, ManageUsersFlow.detail)
  saved: unknown;
}

@flow(#{id: "import-users", name: "CSV を確認して取り込む", goal: "取り込み前に内容を確認し、確認後にだけ変更を適用する。", notes: #["四つの表示は同じ CSV インポートページ内で切り替わる。", "結果では成功と失敗の件数、失敗した行と項目を確認できる。"]})
model ImportUsersFlow {
  @view(#{name: "ファイルの選択", state: "ready", visible: #["CSV ファイル", "テンプレートの取得"]}, AdminUserPages.importCsv)
  upload: unknown;
  @view(#{name: "内容のプレビュー", state: "preview", visible: #["適用前の変更内容", "行ごとの問題"]}, AdminUserPages.importCsv)
  preview: unknown;
  @view(#{name: "適用の確認", state: "confirm", visible: #["対象・影響の確認", "確認ダイアログ"]}, AdminUserPages.importCsv)
  confirmation: unknown;
  @view(#{name: "取り込み結果", state: "success", visible: #["成功と失敗の件数", "失敗した行と項目"]}, AdminUserPages.importCsv)
  result: unknown;
  @flowLink(#{trigger: "プレビュー", action: "preview"}, ImportUsersFlow.upload, ImportUsersFlow.preview)
  previewFile: unknown;
  @flowLink(#{trigger: "適用", action: "confirm"}, ImportUsersFlow.preview, ImportUsersFlow.confirmation)
  openConfirmation: unknown;
  @flowLink(#{trigger: "確認して実行", action: "apply"}, ImportUsersFlow.confirmation, ImportUsersFlow.result)
  apply: unknown;
  @flowLink(#{trigger: "キャンセル", alternate: true, action: "cancelApply"}, ImportUsersFlow.confirmation, ImportUsersFlow.preview)
  cancel: unknown;
}

model AccountProfilePages {
  @page(#{...accountContext, id: "account-profile", name: "本人のプロフィール", entry: "本人のポータル、編集完了", purpose: "本人の基本情報と公開された属性を確認する", content: #["ユーザー名、表示名、氏名、メール", "属性グループごとの登録内容", "編集の入口と更新通知"]}) @category(Category.profile)
  profile: PageData<{
    @transition(#{...navigation, event: "プロフィールを編集", result: "編集可能な項目を表示する"}, AccountProfilePages.edit) edit: unknown;
    @transition(#{...navigation, event: "メールアドレスを変更", result: "確認の依頼へ進む"}, AccountProfilePages.emails) email: unknown;
  }>;
  @page(#{...accountContext, id: "account-profile-edit", name: "本人のプロフィールを編集", entry: "本人のプロフィール", purpose: "本人に編集が許可された基本情報と属性を更新する", content: #["基本情報の入力", "editable_attributes の定義に応じた入力", "保存とキャンセル"]}) @category(Category.profile)
  edit: PageData<{
    @transition(#{...navigation, event: "保存", condition: "更新成功", result: "プロフィールへ戻り success 通知を表示する"}, AccountProfilePages.profile) @uses(IdMagic.IdManagement.Operations.UpdateUserProfile) save: unknown;
    @transition(#{...navigation, event: "キャンセル・戻る", result: "変更を送信せずプロフィールへ戻る"}, AccountProfilePages.profile) cancel: unknown;
    @transition(#{...update, event: "保存", condition: "更新拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error"}, AccountProfilePages.edit) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-emails", name: "本人のメールアドレス", entry: "本人のポータル", purpose: "現在の確認状態を確認し、新しいアドレスの確認を依頼する", content: #["現在のメールと確認状態", "新しいメールアドレス。確認リンクの完了前は現在のアドレスを変えない", "確認通知の送信結果"]}) @category(Category.profile)
  emails: PageData<{
    @transition(#{...update, event: "確認メールを送信", condition: "依頼の受け付け成功", result: "送信結果を表示し、確認リンクをメールへ届ける", toState: "external"}, Destinations.email) @uses(IdMagic.IdManagement.Operations.RequestEmailChange) send: unknown;
    @transition(#{...update, event: "送信", condition: "入力の拒否または送信失敗", result: "入力の近くに問題を表示する", toState: "error"}, AccountProfilePages.emails) failure: unknown;
  }>;
  @page(#{...browserContext, id: "email-verify", name: "メールアドレスの確認", entry: "確認メールのリンク", purpose: "メールに含まれるトークンで変更を確認する", content: #["確認の実行ボタン", "確認成功、無効・期限切れなどの結果"]}) @category(Category.profile)
  verify: PageData<{
    @transition(#{...update, event: "メールアドレスを確認", condition: "トークンが有効で確認成功", result: "確認完了を表示する", toState: "success"}, AccountProfilePages.verify) confirm: unknown;
    @transition(#{...update, event: "確認", condition: "トークンがない、期限切れ、使用済みまたは拒否", result: "確認できない理由を表示する", toState: "error"}, AccountProfilePages.verify) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-data", name: "本人のデータ", entry: "本人のポータル", purpose: "本人のデータを JSON として取得する", content: #["取得するデータの説明", "ダウンロードの入口", "取得失敗の表示"]}) @category(Category.profile)
  data: PageData<{
    @transition(#{...update, event: "データをダウンロード", condition: "取得成功", result: "ブラウザーへ本人のデータを渡す", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.ExportAccountData) download: unknown;
  }>;
}
