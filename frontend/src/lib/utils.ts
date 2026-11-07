import { type ClassNameValue, twMerge } from 'tailwind-merge'
import type { DomainLabelsDictionary } from './i18n/domainLabels.i18n'
import type { UserAttributeDef } from '../types'
import type { Locale } from './i18n'
import { builtinAttributesDictionary } from './i18n/builtinAttributes.i18n'

export function cn(...inputs: ClassNameValue[]) {
  return twMerge(...inputs)
}

// Base UI の className は状態から文字列を返す関数も受け取るが、cn() は関数を空文字として捨てる。
// cn() で既定のクラスと合成する部品は、この型で className を文字列に限る。
export type StringClassNameProps<P> = Omit<P, 'className'> & { className?: string }

export function attributeDisplayName(
  def: Pick<UserAttributeDef, 'key' | 'label'>,
  locale: Locale,
): string {
  const labels = builtinAttributesDictionary[locale]
  // 組み込みキーは予約済みであり、カスタム属性には使えない。
  return Object.hasOwn(labels, def.key)
    ? labels[def.key as keyof typeof labels]
    : def.label || def.key
}

export function attributeLabel(
  def: Pick<UserAttributeDef, 'key' | 'label'>,
  locale: Locale = 'en',
): string {
  const name = attributeDisplayName(def, locale)
  return name === def.key ? def.key : `${name} (${def.key})`
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
  'manager_sub',
  'hire_date',
  'termination_date',
  'employment_type',
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
