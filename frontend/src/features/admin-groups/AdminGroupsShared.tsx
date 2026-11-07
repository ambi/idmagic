import { formSchema, optionalList, optionalText, requiredText, text, when } from '../../lib/form'
import { LENGTH } from '../../lib/lengthLimits'

export function parseRoles(value: string) {
  return [
    ...new Set(
      value
        .split(',')
        .map((role) => role.trim())
        .filter(Boolean),
    ),
  ]
}

// 上限は TypeSpec の GroupCreateRequest、GroupUpdateRequest、DynamicGroupRuleRequest に従う。
export function createGroupSchema(dynamic: boolean) {
  return formSchema({
    name: requiredText(LENGTH.name),
    description: optionalText(LENGTH.description),
    email: optionalText(LENGTH.email),
    expression: when(dynamic, requiredText(LENGTH.expression)),
    roles: optionalList(parseRoles),
  })
}

export const editGroupSchema = formSchema({
  name: requiredText(LENGTH.name),
  description: text(LENGTH.description),
  email: text(LENGTH.email),
  roles: optionalList(parseRoles),
})
