import type {
  Access,
  Authorization,
  Interface,
  SclBundle,
  SclDocument,
} from '../../scl-to-html/src/types.ts'

export interface AuthorizationAction {
  name: string
  memberOf: string
  policies: string[]
  resourceId: string
  appliesTo: {
    principalTypes: string[]
    resourceTypes: string[]
  }
}

export interface AuthorizationActionGroup {
  name: string
  policies: string[]
  actions: AuthorizationAction[]
}

export interface AuthorizationMetadata {
  groups: AuthorizationActionGroup[]
}

const documents = (bundle: SclBundle | SclDocument): SclDocument[] =>
  'contexts' in bundle
    ? [bundle.root, ...bundle.contexts.map((context) => context.document)]
    : [bundle]

const protectedAccess = (access: Access | undefined): Exclude<Access, string> | undefined =>
  access && typeof access === 'object' ? access : undefined

const groupName = (policies: string[]): string => `policy:${policies.join('+')}`

/**
 * Derive a deterministic AuthZEN/Cedar-oriented action inventory from SCL.
 * Interface names are actions; identical policy sets share one action group.
 */
export function buildAuthorizationMetadata(bundle: SclBundle | SclDocument): AuthorizationMetadata {
  const interfaces = new Map<string, Interface>()
  const authorizations: Authorization[] = []
  for (const document of documents(bundle)) {
    for (const [name, operation] of Object.entries(document.interfaces ?? {})) {
      interfaces.set(name, operation)
    }
    if (document.authorization) authorizations.push(document.authorization)
  }

  const policyPrincipalTypes = new Map<string, Set<string>>()
  for (const authorization of authorizations) {
    for (const [policyName, policy] of Object.entries(authorization.policies ?? {})) {
      const principal = authorization.principals?.[policy.principal]
      if (!principal) continue
      const types = policyPrincipalTypes.get(policyName) ?? new Set<string>()
      types.add(principal.type)
      policyPrincipalTypes.set(policyName, types)
    }
  }

  const groups = new Map<string, AuthorizationActionGroup>()
  for (const [name, operation] of [...interfaces].sort(([a], [b]) => a.localeCompare(b))) {
    const access = protectedAccess(operation.access)
    if (!access) continue
    const policies = [...new Set(access.policies)].sort()
    const memberOf = groupName(policies)
    const principalTypes = [
      ...new Set(policies.flatMap((policy) => [...(policyPrincipalTypes.get(policy) ?? [])])),
    ].sort()
    const action: AuthorizationAction = {
      name,
      memberOf,
      policies,
      resourceId: access.resource.id,
      appliesTo: {
        principalTypes,
        resourceTypes: [access.resource.type],
      },
    }
    const group = groups.get(memberOf) ?? { name: memberOf, policies, actions: [] }
    group.actions.push(action)
    groups.set(memberOf, group)
  }

  return {
    groups: [...groups.values()]
      .map((group) => ({
        ...group,
        actions: group.actions.sort((a, b) => a.name.localeCompare(b.name)),
      }))
      .sort((a, b) => a.name.localeCompare(b.name)),
  }
}
