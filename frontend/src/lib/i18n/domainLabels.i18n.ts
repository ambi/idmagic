import { defineDictionary } from './dictionary'

// domainLabels.i18n は internal 値(enum等)から表示名を得る、複数 feature 共有の変換
// helper (attributeGroupTitle / requiredActionLabel) 用の辞書。値の変換ロジックは
// lib/utils.ts / types.ts に残し、ここは文言の正だけを持つ (責務分離)。
export const domainLabelsDictionary = defineDictionary(
  {
    attributeGroupProfile: 'OIDC 標準クレーム',
    attributeGroupOrganization: '組織情報',
    attributeGroupCustom: 'カスタム属性',
    requiredActionUpdatePassword: 'パスワードの変更',
    requiredActionVerifyEmail: 'メールアドレスの確認',
    requiredActionConfigureTotp: '二要素認証の設定',
    requiredActionUpdateProfile: 'プロフィールの更新',
    requiredActionTermsAndConditions: '利用規約への同意',
    requiredActionOther: 'その他の必須対応',
  },
  {
    attributeGroupProfile: 'OIDC standard claims',
    attributeGroupOrganization: 'Organization info',
    attributeGroupCustom: 'Custom attributes',
    requiredActionUpdatePassword: 'Change password',
    requiredActionVerifyEmail: 'Verify email address',
    requiredActionConfigureTotp: 'Set up two-factor authentication',
    requiredActionUpdateProfile: 'Update profile',
    requiredActionTermsAndConditions: 'Accept terms and conditions',
    requiredActionOther: 'Other required action',
  },
)

export type DomainLabelsDictionary = (typeof domainLabelsDictionary)['ja']
