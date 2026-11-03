import { defineDictionary } from '../../lib/i18n'

// agentCreateFlowDictionary は Agent の登録で認証と接続先を同時に準備する文言 (REQ-IDMANAGEMENT-105)。
export const agentCreateFlowDictionary = defineDictionary(
  {
    originYou: 'あなたが決める',
    originDefault: 'デフォルト値。変更できます',
    authHeading: '認証',
    authHelp:
      'Agent が本物であることを、秘密鍵の署名で証明します。IdMagic へ渡すのは公開鍵だけです。登録と同時に認証用クライアントを作成して紐付けます。',
    authKey: '公開鍵を貼る',
    authJwksUri: 'JWKS の URL を指定する',
    authLater: '後で設定する',
    publicKeyLabel: '公開鍵（PEM、JWK、JWKS）',
    jwksUriLabel: 'JWKS の URL',
    jwksUriHelp:
      'Agent が公開鍵を公開している https の URL です。鍵を更新しても、管理画面の操作は要りません。',
    connectionsHeading: '利用する接続先と操作',
    connectionsHelp:
      'この Agent に使わせる操作を選びます。選ばなくても登録でき、後で「利用する接続先」から変えられます。',
    connectionsEmpty: '公開中のツールがある接続先はまだありません。',
    summary: '「{name}」を登録し、{auth}。{operations}',
    summaryAuthKey: '公開鍵で認証できるようにします',
    summaryAuthJwksUri: 'JWKS の URL の鍵で認証できるようにします',
    summaryAuthLater: '認証はまだ設定しません',
    summaryOperations: '{operations} を許可します。',
    summaryNoOperations: '操作はまだ許可しません。',
    stageAgent: 'Agent は登録済みです。',
    stageClient: '認証用クライアントは作成済みです（{clientId}）。',
    stageBound: '認証用クライアントは紐付け済みです。',
    stageRetry: '再試行すると、未完了の段階だけを続けます。作り直しはしません。',
    jwksUriInvalid: 'https の URL を入力してください。',
    resultHeading: '登録しました',
    resultIntro:
      'Agent の配置先へ次の値を渡します。トークンの要求ではスコープを省略できます。IdMagic が割り当てた操作から決めます。',
    resultClientId: 'クライアント ID',
    resultTokenEndpoint: 'トークンエンドポイント',
    resultResources: '接続先のリソース',
    resultConfig: '配置先の設定の例',
    resultNoClient:
      '認証はまだ設定していません。Agent の詳細の「クレデンシャルを管理」から追加します。',
    resultOpenAgent: 'Agent の詳細を開く',
    copy: 'コピー',
  },
  {
    originYou: 'You decide',
    originDefault: 'Default value; you can change it',
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
    connectionsHeading: 'Connections and operations',
    connectionsHelp:
      'Choose the operations this agent may use. You can register without any and change them later in Resource connections.',
    connectionsEmpty: 'No connection publishes tools yet.',
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
    jwksUriInvalid: 'Enter an https URL.',
    resultHeading: 'Registered',
    resultIntro:
      'Give these values to where the agent runs. Token requests can omit the scope; IdMagic decides it from the assigned operations.',
    resultClientId: 'Client ID',
    resultTokenEndpoint: 'Token endpoint',
    resultResources: 'Connection resources',
    resultConfig: 'Example deployment settings',
    resultNoClient:
      'Authentication is not set up yet. Add it from Manage credentials on the agent.',
    resultOpenAgent: 'Open agent details',
    copy: 'Copy',
  },
)
