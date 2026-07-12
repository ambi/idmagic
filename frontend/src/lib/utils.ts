import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import type { DomainLabelsDictionary } from './i18n/domainLabels.i18n'
import type { UserAttributeDef } from '../types'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// attributeLabel は属性を「日本語表示名 (key)」で表す。label が無ければ key のみ。
export function attributeLabel(def: Pick<UserAttributeDef, 'key' | 'label'>): string {
  return def.label ? `${def.label} (${def.key})` : def.key
}

const organizationAttributeKeys = new Set([
  'organization',
  'organization_name',
  'department',
  'division',
  'title',
  'job_title',
  'employee_number',
  'employee_id',
  'cost_center',
  'manager',
])

export type AttributeGroupKey = 'profile' | 'organization' | 'custom'

export function attributeGroupKey(
  def: Pick<UserAttributeDef, 'key' | 'oidc_scope'>,
): AttributeGroupKey {
  if (def.oidc_scope) return 'profile'
  return organizationAttributeKeys.has(def.key) ? 'organization' : 'custom'
}

// t は呼び出し側 (component) が useDictionary(domainLabelsDictionary) で取得したものを渡す。
// key → 表示名の変換ロジックはここに残し、文言そのものは辞書側の責務にする。
export function attributeGroupTitle(key: AttributeGroupKey, t: DomainLabelsDictionary): string {
  switch (key) {
    case 'profile':
      return t.attributeGroupProfile
    case 'organization':
      return t.attributeGroupOrganization
    case 'custom':
      return t.attributeGroupCustom
  }
}
