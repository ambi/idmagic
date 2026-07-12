import { defineDictionary } from '../../lib/i18n'

export const adminAuthorizationDetailTypesDictionary = defineDictionary(
  {
    schemaInvalidError: 'スキーマ JSON が不正です。',
    updatedNotice: '{type} を更新しました。',
    registeredNotice: '{type} を登録しました。',
    saveFailedError: '保存に失敗しました。',
    deletedNotice: '{type} を削除しました。',
    deleteFailedError: '削除に失敗しました。',
    pageTitle: '認可詳細の種類',
    pageDescription:
      'エージェントに渡す細粒度の認可詳細 (RFC 9396 authorization_details) の種類を定義します。受理するのはここに登録した種類のみです。',
    registerType: '種類を登録',
    typeIdLabel: '種類 ID (type)',
    descriptionLabel: '説明',
    displayTemplateLabel: '同意画面の表示テンプレート',
    displayTemplatePlaceholder:
      '口座 {creditorAccount} に対して {actions} を、最大 {instructedAmount} まで',
    schemaLabel: '検証スキーマ (JSON)',
    schemaHelp:
      '各フィールドの semantics は set (集合包含) / at_most (上限) / enum / exact のいずれか。要求はここで束縛した範囲に限定され、同意・委譲・交換でこの半順序を超えられません。',
    stateLabel: '状態',
    update: '更新',
    register: '登録',
    cancel: 'キャンセル',
    emptyNotice: 'まだ認可詳細の種類が登録されていません。',
    edit: '編集',
    footerLinkLabel: 'アプリケーション',
    footerText: 'が要求した詳細は、ここで定義した検証ルールで fail-closed に検査されます。',
  },
  {
    schemaInvalidError: 'The schema JSON is invalid.',
    updatedNotice: '{type} has been updated.',
    registeredNotice: '{type} has been registered.',
    saveFailedError: 'Failed to save.',
    deletedNotice: '{type} has been deleted.',
    deleteFailedError: 'Failed to delete.',
    pageTitle: 'Authorization detail types',
    pageDescription:
      'Define fine-grained authorization detail types (RFC 9396 authorization_details) passed to agents. Only types registered here are accepted.',
    registerType: 'Register type',
    typeIdLabel: 'Type ID (type)',
    descriptionLabel: 'Description',
    displayTemplateLabel: 'Consent screen display template',
    displayTemplatePlaceholder:
      'Up to {instructedAmount} of {actions} for account {creditorAccount}',
    schemaLabel: 'Validation schema (JSON)',
    schemaHelp:
      "Each field's semantics is one of set (subset), at_most (upper bound), enum, or exact. Requests are bound to the range defined here and cannot exceed this partial order through consent, delegation, or exchange.",
    stateLabel: 'State',
    update: 'Update',
    register: 'Register',
    cancel: 'Cancel',
    emptyNotice: 'No authorization detail types have been registered yet.',
    edit: 'Edit',
    footerLinkLabel: 'Applications',
    footerText:
      "'s requested details are checked fail-closed against the validation rules defined here.",
  },
)
