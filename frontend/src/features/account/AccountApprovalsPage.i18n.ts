import { defineDictionary } from '../../lib/i18n'

export const accountApprovalsDictionary = defineDictionary(
  {
    title: '承認リクエスト',
    description:
      'エージェントやアプリケーションがあなたに代わって行おうとしている操作を確認できます。内容を確認して承認または拒否してください。',
    empty: '保留中の承認リクエストはありません。',
    requestedBy: 'リクエスト元',
    agent: 'エージェント',
    scopes: '要求された権限',
    details: '操作の詳細',
    bindingMessage: '確認メッセージ',
    expires: '有効期限: {date}',
    approve: '承認する',
    deny: '拒否する',
    deciding: '処理中…',
    approved: '「{name}」のリクエストを承認しました。',
    denied: '「{name}」のリクエストを拒否しました。',
    failed: '承認リクエストを処理できませんでした。',
  },
  {
    title: 'Approval requests',
    description:
      'Review actions that agents or applications want to perform on your behalf, then approve or deny each request.',
    empty: 'There are no pending approval requests.',
    requestedBy: 'Requested by',
    agent: 'Agent',
    scopes: 'Requested permissions',
    details: 'Action details',
    bindingMessage: 'Verification message',
    expires: 'Expires: {date}',
    approve: 'Approve',
    deny: 'Deny',
    deciding: 'Processing…',
    approved: 'Approved the request from “{name}”.',
    denied: 'Denied the request from “{name}”.',
    failed: 'Could not process the approval request.',
  },
)
