import { defineDictionary } from '../../lib/i18n'

export const groupRoleDelegationDictionary = defineDictionary(
  {
    title: '自動所属による業務ロールの委任',
    explanation:
      '承認すると、このグループへの自動所属によって下記の業務ロールが付与されます。属性や所属を書き込めるすべての上流連携を信頼する操作です。特定の連携元だけには限定されません。',
    adminNotice: 'admin と system_admin は自動所属から付与されません。',
    pending: '未承認です。自動所属による業務ロールは付与されません。',
    approved: '現在のロールとルールへの委任が承認されています。',
    stale: '設定が変わったため、承認は無効です。再承認するまで業務ロールは付与されません。',
    noRoles: '業務ロールが設定されていません。',
    saveFirst: '変更を先に保存してください。',
    approve: '業務ロールの委任を承認',
    revoke: '委任を取り消す',
    failed: '業務ロールの委任を更新できませんでした。',
    savedRule: '承認の対象となる保存済みルール',
  },
  {
    title: 'Delegate business roles through automatic membership',
    explanation:
      'Approval allows automatic membership in this group to grant the business roles below. This trusts every upstream integration that can write the attributes or membership, without limiting trust to a specific source.',
    adminNotice: 'Automatic membership never grants admin or system_admin.',
    pending: 'Not approved. Automatic membership does not grant business roles.',
    approved: 'Delegation is approved for the current roles and rule.',
    stale: 'Settings have changed. Business roles require approval again.',
    noRoles: 'No business roles are configured.',
    saveFirst: 'Save changes first.',
    approve: 'Approve business role delegation',
    revoke: 'Revoke delegation',
    failed: 'Could not update business role delegation.',
    savedRule: 'Saved rule to approve',
  },
)
