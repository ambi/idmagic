import { defineDictionary } from '../../lib/i18n'

export const agentAccessDictionary = defineDictionary(
  {
    title: 'エージェントのアクセス権',
    add: 'アクセス権を追加',
    description: '対象とエージェントを選んで、直接付与するアクセス権を管理します。',
    connection: '接続先',
    permission: '権限',
    resource: '対象 ID',
    agent: 'エージェント',
    choose: '選択してください',
    resourceHelp:
      '接続先が文書などに付けた ID を入力します。外部の対象一覧は取得していません。候補は既存の付与から表示します。',
    scopeHelp: 'この権限の評価規則に登録されたスコープ',
    shared:
      'この付与は同じ対象型と権限を使うほかの接続先にも適用されます。接続先だけに限定した付与ではありません。',
    limits:
      'ここに表示するのは直接の付与です。実行にはクライアントのスコープ、承認、利用者やエージェントの状態などの条件も適用されます。',
    noPermission:
      'この接続先にはエージェントへ直接付与できる権限がありません。開発者向け設定でモデルと評価規則を確認してください。',
    developer: '開発者向け設定',
    model: 'モデルを定義',
    tuples: '関係データを管理',
    check: '判定を調べる',
    grant: 'アクセス権を付与',
    granting: '付与中…',
    cancel: '戻る',
    empty: '条件に合う直接の付与はありません。',
    revoke: '取り消す',
    revokeTitle: 'アクセス権を取り消す',
    revokeResult:
      '選択した直接の付与だけを削除します。同じ内容を追加し直すこともできます。別の関係から得られる権限は残る場合があります。',
    granted: 'アクセス権を追加しました。',
    revoked: '直接の付与を取り消しました。',
    error: '操作に失敗しました。入力を確認して再試行してください。',
    filter: '対象またはエージェントで検索',
    results: '直接の付与の一覧',
    truncated:
      '取得上限に達しました。ほかの付与が存在する可能性があります。開発者向けの関係データ画面で条件を絞って確認してください。',
    agentLimit:
      'エージェントの候補は最大 1000 件です。見つからない場合はエージェント詳細からアクセス権を開いてください。',
    preview: '追加する内容',
    approval:
      '操作への承認を要求する規則が含まれています。適用される承認条件を、この付与で省略することはできません。',
    conditions: 'この規則には追加条件があります。詳細は開発者向け設定で確認してください。',
  },
  {
    title: 'Agent access',
    add: 'Add access',
    description: 'Choose a resource and an agent to manage direct access assignments.',
    connection: 'Connection',
    permission: 'Permission',
    resource: 'Resource ID',
    agent: 'Agent',
    choose: 'Choose an option',
    resourceHelp:
      'Enter the ID assigned to the document or other resource by the connection. External resources are not discovered. Suggestions come from existing assignments.',
    scopeHelp: 'Scopes registered in evaluation rules for this permission',
    shared:
      'This assignment also applies to other connections using the same resource type and permission. It is not isolated to this connection.',
    limits:
      'These are direct assignments. Execution also depends on client scopes, approvals, user and agent status, and other conditions.',
    noPermission:
      'This connection has no permissions that can be granted directly to agents. Review the model and evaluation rules in developer settings.',
    developer: 'Developer settings',
    model: 'Define the model',
    tuples: 'Manage relation data',
    check: 'Inspect a decision',
    grant: 'Grant access',
    granting: 'Granting…',
    cancel: 'Back',
    empty: 'No matching direct assignments.',
    revoke: 'Revoke',
    revokeTitle: 'Revoke access',
    revokeResult:
      'Only the selected direct assignment will be deleted. You can add the same assignment again. Access derived from other relations may remain.',
    granted: 'Access assignment added.',
    revoked: 'Direct assignment revoked.',
    error: 'The operation failed. Check the input and try again.',
    filter: 'Search by resource or agent',
    results: 'Direct assignments',
    truncated:
      'The retrieval limit was reached. More assignments may exist. Narrow the search in the developer relation data page.',
    agentLimit:
      'Up to 1000 agents are suggested. If an agent is missing, open access from its detail page.',
    preview: 'Assignment to add',
    approval:
      'These rules include operation approval requirements. Granting access does not bypass applicable approval conditions.',
    conditions: 'This rule has additional conditions. Inspect them in developer settings.',
  },
)
