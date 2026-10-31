import { defineDictionary } from '../../lib/i18n'

// runStatus* と itemError* の鍵は spec/modules/sourcing/models.tsp の IngestionRunStatus と
// DirectoryItemError に対応する。仕様が値を増やしたらここも増やす。
export const adminDirectoryConnectorsDictionary = defineDictionary(
  {
    pageTitle: 'Directory Connector',
    pageDescription:
      '閉域の Active Directory と LDAP から User を取り込む Connector を登録し、同期の状態を確かめます。',
    createConnector: 'Connector を作成',
    emptyNotice: 'まだ Connector がありません。',
    backToList: '一覧に戻る',
    detail: '詳細',
    cancel: 'キャンセル',
    create: '作成',
    copyAria: 'コピー',

    displayNameLabel: '表示名',
    displayNameHelp: '取り込み元が分かる名前を付けます。200 文字までです。',
    saveFailedError: 'Connector を作成できませんでした。',

    statusAwaitingEnrollment: '登録待ち',
    statusActive: '有効',
    statusDisabled: '無効',

    headerName: '名前',
    headerStatus: '状態',
    headerLastSeen: '最終接続',
    headerCreatedAt: '作成時刻',

    summaryHeading: '状態',
    createdAtLabel: '作成時刻',
    lastSeenLabel: '最終接続時刻',
    enrolledAtLabel: '登録時刻',
    disabledAtLabel: '無効化時刻',
    enrollmentExpiresAtLabel: '登録トークンの有効期限',
    keyIdLabel: '鍵の ID',
    connectorIdLabel: 'Connector の ID',
    enrollmentExpiredNotice:
      '登録トークンの期限が切れました。登録トークンを再発行すると登録を続けられます。',

    tokenLabel: '登録トークン',
    tokenNotice:
      'この登録トークンは再表示できません。発行から 1 時間で期限が切れます。Connector の登録に使ってください。失くしたときは再発行できます。',
    toDetail: 'Connector の詳細へ',

    reissue: '登録トークンを再発行',
    reissueConfirmMessage:
      '再発行すると登録済みの鍵を破棄します。新しいトークンで登録し直すまで、この Connector の同期は止まります。',
    reissueConfirm: '再発行する',
    reissueFailedError: '登録トークンを再発行できませんでした。',
    disable: '無効化',
    disableConfirmMessage:
      '無効化は取り消せません。以後の登録と同期を拒否します。置き換えるときは新しい Connector を作成します。',
    disableConfirm: '無効化する',
    disableFailedError: 'Connector を無効化できませんでした。',

    runsHeading: '取り込みの実行',
    runsDescription: '開始時刻の新しい順に最大 50 件を表示します。',
    runsEmptyNotice: 'まだ取り込みの実行がありません。',
    headerMode: '種類',
    headerRunStatus: '状態',
    headerStartedAt: '開始',
    headerCompletedAt: '完了',
    headerItems: '項目',
    headerCreated: '作成',
    headerUpdated: '変更',
    headerUnchanged: '変更なし',
    headerDeleted: '削除',
    headerFailed: '失敗',
    headerDisabledUsers: '無効化',

    modeFull: '完全な再同期',
    modeIncremental: '差分',
    runStatusRunning: '実行中',
    runStatusAwaitingApproval: '承認待ち',
    runStatusApplying: '適用中',
    runStatusDiscarded: '破棄',
    runStatusCompleted: '完了',
    runStatusSuperseded: '打ち切り',
    holdSummary: '無効化の候補 {candidates} 件（有効な User {active} 件のうち）',

    showFailures: '失敗を表示',
    hideFailures: '失敗を閉じる',
    failuresLoading: '読み込んでいます。',
    failuresLoadError: '失敗の一覧を読み込めませんでした。',
    failuresHidden: 'ほかに {count} 件の失敗があります。',
    headerExternalId: '外部の不変 ID',
    headerError: '原因',
    headerBatchId: 'バッチの ID',

    itemErrorUsernameConflict: 'ユーザー名が既存の User と衝突しています',
    itemErrorEmailConflict: 'メールアドレスが既存の User と衝突しています',
    itemErrorAttributeConflict: '一意の属性の値が既存の User と衝突しています',
    itemErrorInvalidAttribute: '属性が属性スキーマに違反しています',
    itemErrorQuotaExceeded: 'User の上限を超えています',
    itemErrorUserDeleted: '対応する User は削除されています',
  },
  {
    pageTitle: 'Directory Connector',
    pageDescription:
      'Register connectors that ingest users from Active Directory and LDAP on private networks, and check how they sync.',
    createConnector: 'Create connector',
    emptyNotice: 'No connectors yet.',
    backToList: 'Back to list',
    detail: 'Details',
    cancel: 'Cancel',
    create: 'Create',
    copyAria: 'Copy',

    displayNameLabel: 'Display name',
    displayNameHelp: 'A name that identifies the source, up to 200 characters.',
    saveFailedError: 'Could not create the connector.',

    statusAwaitingEnrollment: 'Awaiting enrollment',
    statusActive: 'Active',
    statusDisabled: 'Disabled',

    headerName: 'Name',
    headerStatus: 'Status',
    headerLastSeen: 'Last connected',
    headerCreatedAt: 'Created',

    summaryHeading: 'Status',
    createdAtLabel: 'Created',
    lastSeenLabel: 'Last connected',
    enrolledAtLabel: 'Enrolled',
    disabledAtLabel: 'Disabled at',
    enrollmentExpiresAtLabel: 'Enrollment token expires',
    keyIdLabel: 'Key ID',
    connectorIdLabel: 'Connector ID',
    enrollmentExpiredNotice:
      'The enrollment token has expired. Reissue the enrollment token to continue enrollment.',

    tokenLabel: 'Enrollment token',
    tokenNotice:
      'This enrollment token cannot be shown again. It expires 1 hour after issue. Use it to enroll the connector. If you lose it, you can reissue it.',
    toDetail: 'Go to connector details',

    reissue: 'Reissue enrollment token',
    reissueConfirmMessage:
      'Reissuing discards the enrolled key. This connector stops syncing until it enrolls again with the new token.',
    reissueConfirm: 'Reissue',
    reissueFailedError: 'Could not reissue the enrollment token.',
    disable: 'Disable',
    disableConfirmMessage:
      'Disabling cannot be undone. Later enrollment and sync calls are refused. To replace this connector, create a new one.',
    disableConfirm: 'Disable connector',
    disableFailedError: 'Could not disable the connector.',

    runsHeading: 'Ingestion runs',
    runsDescription: 'Up to 50 runs, newest first.',
    runsEmptyNotice: 'No ingestion runs yet.',
    headerMode: 'Mode',
    headerRunStatus: 'Status',
    headerStartedAt: 'Started',
    headerCompletedAt: 'Completed',
    headerItems: 'Items',
    headerCreated: 'Created',
    headerUpdated: 'Updated',
    headerUnchanged: 'Unchanged',
    headerDeleted: 'Deleted',
    headerFailed: 'Failed',
    headerDisabledUsers: 'Disabled',

    modeFull: 'Full resync',
    modeIncremental: 'Incremental',
    runStatusRunning: 'Running',
    runStatusAwaitingApproval: 'Awaiting approval',
    runStatusApplying: 'Applying',
    runStatusDiscarded: 'Discarded',
    runStatusCompleted: 'Completed',
    runStatusSuperseded: 'Superseded',
    holdSummary: '{candidates} users to disable, out of {active} active users',

    showFailures: 'Show failures',
    hideFailures: 'Hide failures',
    failuresLoading: 'Loading.',
    failuresLoadError: 'Could not load the failures.',
    failuresHidden: '{count} more failures are not shown.',
    headerExternalId: 'External ID',
    headerError: 'Cause',
    headerBatchId: 'Batch ID',

    itemErrorUsernameConflict: 'The username conflicts with an existing user',
    itemErrorEmailConflict: 'The email address conflicts with an existing user',
    itemErrorAttributeConflict: 'A unique attribute value conflicts with an existing user',
    itemErrorInvalidAttribute: 'An attribute violates the attribute schema',
    itemErrorQuotaExceeded: 'The user quota is exceeded',
    itemErrorUserDeleted: 'The corresponding user has been deleted',
  },
)
