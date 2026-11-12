import { defineDictionary } from '../../lib/i18n'

export const delegatedAuthenticationDictionary = defineDictionary(
  {
    title: '委譲認証',
    description:
      '取り込み元のパスワードで認証する設定と接続の試験を管理します。操作はすぐに反映されます。',
    back: 'Connector の詳細へ戻る',
    enabled: '有効',
    disabled: '無効',
    status: '委譲認証の状態',
    capable: '暗号化能力を登録済み',
    unavailable: 'Connector の常駐を起動して暗号化能力を登録してください。',
    enable: '委譲認証を有効にする',
    disable: '委譲認証を無効にする',
    working: '処理中…',
    settingFailed:
      '委譲認証の設定を変更できませんでした。Connector の状態とローカルパスワードの有無を確認してください。',
    confirm: '設定を変更する',
    confirmDescription:
      'この取り込み元のユーザーのパスワード認証に影響します。無効にしてもローカルパスワードは作成されません。この設定は後から変更できます。',
    testTitle: '接続の試験',
    testDescription:
      'この Connector から取り込んだユーザーの ID と AD のパスワードを入力します。試験はログインセッションを発行しません。',
    userID: 'ユーザーの ID',
    password: 'AD のパスワード',
    test: '照合を試す',
    testing: '照合中…',
    testFailed: '試験を実行できませんでした。権限、入力と試行の制限を確認してください。',
    accepted: 'パスワードの照合に成功しました。',
    invalid_credentials: '資格情報が拒否されました。',
    account_locked: 'AD のアカウントがロックされています。',
    account_disabled: 'AD のアカウントが無効です。',
    account_expired: 'AD のアカウントの期限が切れています。',
    password_expired: 'AD のパスワードの期限が切れています。',
    password_change_required: 'AD でパスワードを変更してください。',
    connector_unavailable:
      'Connector から期限内に応答がありません。常駐プロセスと IdMagic への接続を確認してください。',
    directory_unavailable:
      'ディレクトリへ接続できません。DC の接続、証明書とサービスアカウントを確認してください。',
  },
  {
    title: 'Delegated authentication',
    description:
      'Manage authentication with source directory passwords and test connectivity. Actions take effect immediately.',
    back: 'Back to connector details',
    enabled: 'Enabled',
    disabled: 'Disabled',
    status: 'Delegated authentication status',
    capable: 'Encryption capability registered',
    unavailable: 'Start the connector daemon to register its encryption capability.',
    enable: 'Enable delegated authentication',
    disable: 'Disable delegated authentication',
    working: 'Working…',
    settingFailed:
      'Could not change delegated authentication. Check the connector status and whether any local passwords remain.',
    confirm: 'Change setting',
    confirmDescription:
      'This affects password authentication for users from this source. Disabling does not create local passwords. You can change this setting again later.',
    testTitle: 'Connection test',
    testDescription:
      'Enter the ID of a user imported by this connector and their AD password. This test does not issue a login session.',
    userID: 'User ID',
    password: 'AD password',
    test: 'Test credentials',
    testing: 'Testing…',
    testFailed: 'Could not run the test. Check permissions, input, and attempt limits.',
    accepted: 'Password verification succeeded.',
    invalid_credentials: 'The credentials were refused.',
    account_locked: 'The AD account is locked.',
    account_disabled: 'The AD account is disabled.',
    account_expired: 'The AD account has expired.',
    password_expired: 'The AD password has expired.',
    password_change_required: 'Change the password in AD.',
    connector_unavailable:
      'The connector did not respond before the deadline. Check its daemon and connection to IdMagic.',
    directory_unavailable:
      'The directory is unavailable. Check the DC connection, certificate, and service account.',
  },
)
