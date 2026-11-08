import { defineDictionary } from '../../lib/i18n'

// agentCreateFlowDictionary はエージェントの登録で認証と MCP リソースサーバーを同時に準備する文言 (REQ-IDMANAGEMENT-105)。
export const agentCreateFlowDictionary = defineDictionary(
  {
    authHeading: '認証',
    authHelp:
      'エージェントが本物であることを、秘密鍵の署名で証明します。IdMagic へ渡すのは公開鍵だけです。登録と同時に認証用クライアントを作成して紐付けます。',
    authKey: '公開鍵を貼る',
    authJwksUri: 'JWKS の URL を指定する',
    authLater: '後で設定する',
    publicKeyLabel: '公開鍵（PEM、JWK、JWKS）',
    jwksUriLabel: 'JWKS の URL',
    jwksUriHelp:
      'エージェントが公開鍵を公開している https の URL です。鍵を更新しても、管理画面の操作は要りません。',
    connectionsHeading: '利用する MCP リソースサーバーと操作',
    connectionsHelp:
      'このエージェントに使わせる操作を選びます。選ばなくても登録でき、後で「利用する MCP リソースサーバー」から変えられます。',
    connectionsEmpty: '公開中のツールがある MCP リソースサーバーはまだありません。',
    targetScopedNote:
      '対象は登録後に「利用する MCP リソースサーバー」で選びます。選ぶまで、どの対象も許可しません。',
    summary: '「{name}」を登録し、{auth}。{operations}',
    summaryAuthKey: '公開鍵で認証できるようにします',
    summaryAuthJwksUri: 'JWKS の URL の鍵で認証できるようにします',
    summaryAuthLater: '認証はまだ設定しません',
    summaryOperations: '{operations} を許可します。',
    summaryNoOperations: '操作はまだ許可しません。',
    stageAgent: 'エージェントは登録済みです。',
    stageClient: '認証用クライアントは作成済みです（{clientId}）。',
    stageBound: '認証用クライアントは紐付け済みです。',
    stageRetry: '再試行すると、未完了の段階だけを続けます。作り直しはしません。',
    resultHeading: '登録しました',
    resultIntro:
      'エージェントの配置先へ次の値を渡します。トークンの要求ではスコープを省略できます。IdMagic が割り当てた操作から決めます。',
    resultClientId: 'クライアント ID',
    resultTokenEndpoint: 'トークンエンドポイント',
    resultResources: 'MCP リソースサーバーのリソース',
    resultConfig: '配置先の設定の例',
    resultNoClient:
      '認証はまだ設定していません。エージェントの詳細の「クレデンシャルを管理」から追加します。',
    resultOpenAgent: 'エージェントの詳細を開く',
    copy: 'コピー',
  },
  {
    authHeading: 'Authentication',
    authHelp:
      'The agent proves itself by signing with its private key. Only the public key is given to IdMagic. An authentication client is created and bound when you register.',
    authKey: 'Paste a public key',
    authJwksUri: 'Use a JWKS URL',
    authLater: 'Set up later',
    publicKeyLabel: 'Public key (PEM, JWK, or JWKS)',
    jwksUriLabel: 'JWKS URL',
    jwksUriHelp:
      'The https URL where the agent publishes its public keys. Rotating keys needs no change in the console.',
    connectionsHeading: 'MCP resource servers and operations',
    connectionsHelp:
      'Choose the operations this agent may use. You can register without any and change them later in MCP resource servers.',
    connectionsEmpty: 'No MCP resource server publishes tools yet.',
    targetScopedNote:
      'Choose its targets in MCP resource servers after registering. No target is allowed until then.',
    summary: 'Register "{name}", {auth}. {operations}',
    summaryAuthKey: 'and let it authenticate with a public key',
    summaryAuthJwksUri: 'and let it authenticate with keys from a JWKS URL',
    summaryAuthLater: 'without authentication for now',
    summaryOperations: 'Allow {operations}.',
    summaryNoOperations: 'No operations are allowed yet.',
    stageAgent: 'The agent is registered.',
    stageClient: 'The authentication client was created ({clientId}).',
    stageBound: 'The authentication client is bound.',
    stageRetry: 'Retrying continues only the unfinished steps. Nothing is created again.',
    resultHeading: 'Registered',
    resultIntro:
      'Give these values to where the agent runs. Token requests can omit the scope; IdMagic decides it from the assigned operations.',
    resultClientId: 'Client ID',
    resultTokenEndpoint: 'Token endpoint',
    resultResources: 'MCP resource server resources',
    resultConfig: 'Example deployment settings',
    resultNoClient:
      'Authentication is not set up yet. Add it from Manage credentials on the agent.',
    resultOpenAgent: 'Open agent details',
    copy: 'Copy',
  },
)
