import { defineDictionary } from '../../lib/i18n'

export const credentialFlowDictionary = defineDictionary(
  {
    choose: '認証用クライアントの追加方法',
    create: '新しく作成する',
    existing: '既存のクライアントを使う',
    help: 'どちらか一つを選びます。初めて接続する場合は公開鍵から新しく作成します。すでに登録した認証用クライアントがある場合だけ、既存を選びます。',
    idHelp:
      'ここに表示する ID は認証用クライアントの client_id です。エージェント ID や秘密鍵ではありません。',
    existingHelp:
      '同じテナントに登録済みの、公開鍵、クライアント証明書、SPIFFE の JWT-SVID のいずれかで認証するクライアントの client_id を指定します。',
    result: '作成した認証用クライアント',
    bound: 'エージェントに追加済みです。下の一覧でも確認できます。',
    pending:
      '作成は完了しましたが、エージェントへの追加は未完了です。作成済みの ID で「紐付け」を再試行してください。新しく作り直す必要はありません。',
    list: '追加済みの認証用クライアント',
  },
  {
    choose: 'How to add an authentication client',
    create: 'Create a new client',
    existing: 'Use an existing client',
    help: 'Choose one method. Create a client from a public key for a new connection. Use an existing client only if it is already registered.',
    idHelp:
      'The ID shown here is the authentication client_id. It is not the agent ID or a private key.',
    existingHelp:
      'Enter the client_id of a client registered in this tenant that authenticates with a public key, a client certificate, or a SPIFFE JWT-SVID.',
    result: 'Created authentication client',
    bound: 'Added to this agent. It is also shown in the list below.',
    pending:
      'Created, but not yet added to the agent. Retry Bind with this ID. You do not need to create another client.',
    list: 'Authentication clients added to this agent',
  },
)
