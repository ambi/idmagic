// Drift detection for the admin API's API-access-token scope declarations.
//
// The generated OpenAPI document is the single source of truth: every admin
// operation carries `x-api-token-scopes`, listing the ApiTokenScope values that
// let an API access token reach it, or the `interactive_session` sentinel when
// no token may. This module reports the two ways that source drifts from the
// scope vocabulary it is written against — an operation that declares nothing or
// declares something the vocabulary does not contain, and a vocabulary entry no
// operation ever requires.

export const ADMIN_PATH_PREFIX = '/api/admin/v1/'
export const INTERACTIVE_SESSION_SCOPE = 'interactive_session'
export const SCOPE_EXTENSION = 'x-api-token-scopes'

const HTTP_METHODS = new Set(['delete', 'get', 'head', 'options', 'patch', 'post', 'put'])

export type AdminOperation = { name: string; path: string; method: string; scopes: string[] }

// A path item also carries entries that are not operations (`parameters`,
// `summary`, `$ref`), so its values stay unknown until a method name selects one.
export type OpenApiDocument = { paths?: Record<string, Record<string, unknown>> }

export function collectAdminOperations(document: OpenApiDocument): AdminOperation[] {
  const operations: AdminOperation[] = []
  for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
    if (!path.startsWith(ADMIN_PATH_PREFIX)) continue
    for (const [method, value] of Object.entries(pathItem)) {
      if (!HTTP_METHODS.has(method) || typeof value !== 'object' || value === null) continue
      const operation = value as { operationId?: string; [key: string]: unknown }
      if (!operation.operationId) continue
      const declared = operation[SCOPE_EXTENSION]
      operations.push({
        name: operation.operationId,
        path,
        method: method.toUpperCase(),
        scopes: Array.isArray(declared) ? declared.map(String) : [],
      })
    }
  }
  return operations.sort((left, right) => left.name.localeCompare(right.name))
}

// parseApiTokenScopes reads the ApiTokenScope enum values out of the TypeSpec
// source. The enum is the vocabulary an operation may draw from, so reading it
// here keeps the check honest even when the enum changes.
export function parseApiTokenScopes(source: string): string[] {
  const body = source.match(/enum\s+ApiTokenScope\s*\{([\s\S]*?)\n\}/)?.[1]
  if (!body) throw new Error('spec/contexts/api-tokens/models.tsp must declare enum ApiTokenScope')
  return [...body.matchAll(/:\s*"([^"]+)"/g)].map((match) => match[1] ?? '')
}

// Account and SCIM scopes are enforced by their own APIs, not by an admin route,
// so they are never expected to appear in an admin operation's declaration.
export function isAdminScope(scope: string): boolean {
  return !scope.startsWith('account:') && !scope.startsWith('scim:')
}

export function verifyAdminScopes(operations: AdminOperation[], vocabulary: string[]): string[] {
  const findings: string[] = []
  const allowed = new Set([...vocabulary, INTERACTIVE_SESSION_SCOPE])
  const required = new Set<string>()
  for (const operation of operations) {
    if (operation.scopes.length === 0) {
      findings.push(
        `${operation.name} (${operation.method} ${operation.path}) declares no ${SCOPE_EXTENSION}; an admin operation must name the scopes that reach it, or "${INTERACTIVE_SESSION_SCOPE}"`,
      )
      continue
    }
    for (const scope of operation.scopes) {
      if (!allowed.has(scope)) {
        findings.push(`${operation.name} requires "${scope}", which ApiTokenScope does not define`)
      }
      required.add(scope)
    }
    if (operation.scopes.includes(INTERACTIVE_SESSION_SCOPE) && operation.scopes.length > 1) {
      findings.push(
        `${operation.name} mixes "${INTERACTIVE_SESSION_SCOPE}" with a granular scope; an operation is either reachable by some token or by none`,
      )
    }
  }
  for (const scope of vocabulary) {
    if (isAdminScope(scope) && !required.has(scope)) {
      findings.push(`ApiTokenScope defines "${scope}", but no admin operation requires it`)
    }
  }
  return findings
}
