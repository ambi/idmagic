/**
 * The join between what a scenario step says and what the contract declares.
 *
 * Kept apart from the script so the ranking can be read and tested without a
 * repository around it. The IO — which document declares the rule, which
 * TypeSpec files belong to the context, which tests name a sibling id — stays
 * in main.ts.
 */

export type ContractOperation = {
  name: string
  method: string
  path: string
  scopes: string[]
}

/** Per operation, the error types each status answers with. */
export type DeclaredErrors = ReadonlyMap<string, ReadonlyMap<string, readonly string[]>>

export type Join = {
  errorTypes: ReadonlySet<string>
  paths: ReadonlySet<string>
  scopes: ReadonlySet<string>
}

export type Candidate = {
  operation: ContractOperation
  matched: string[]
}

/**
 * What the steps of an example give the contract to join on.
 *
 * Three things, because one is not enough on its own. An error type alone does
 * not narrow: `AccessDeniedError` is answered by most of a context's admin
 * operations, so a refusal example naming only the type ranks thirty
 * operations equally. A normal-path example names no error type at all, and
 * would have nothing. Endpoints and granular scopes are the other two facts
 * scenarios state literally and the contract also carries.
 */
export function joinableFacts(stepTexts: readonly string[]): Join {
  const text = stepTexts.join('\n')
  return {
    errorTypes: new Set([...text.matchAll(/\b(\w+Error)\b/g)].map((match) => match[1] ?? '')),
    paths: new Set([...text.matchAll(/`(\/[A-Za-z0-9_./-]*)`/g)].map((match) => match[1] ?? '')),
    scopes: new Set(
      [...text.matchAll(/\b([a-z][a-z0-9-]*(?::[a-z0-9-]+)+)\b/g)].map((match) => match[1] ?? ''),
    ),
  }
}

/**
 * The context's operations, most closely joined first.
 *
 * Every operation is returned, not just the matches. An example that joins on
 * nothing still needs somewhere to start, and the caller says in its heading
 * which case it is printing.
 */
export function rankCandidates(
  operations: readonly ContractOperation[],
  errors: DeclaredErrors,
  join: Join,
): Candidate[] {
  return operations
    .map((operation) => {
      const matched: string[] = []
      for (const [status, types] of errors.get(operation.name) ?? []) {
        for (const type of types) if (join.errorTypes.has(type)) matched.push(`${status} ${type}`)
      }
      if (join.paths.has(operation.path)) matched.push(`path ${operation.path}`)
      for (const scope of operation.scopes) {
        if (join.scopes.has(scope)) matched.push(`scope ${scope}`)
      }
      return { operation, matched }
    })
    .sort(
      (left, right) =>
        right.matched.length - left.matched.length ||
        left.operation.name.localeCompare(right.operation.name),
    )
}

/** The rule an id belongs to. An example id carries its rule in its first three fields. */
export function ruleOf(id: string): string {
  return id.startsWith('EX-') ? `REQ-${id.slice(3).replace(/-\d+$/, '')}` : id
}

/** The operations the generated Go contract declares, with method, path and granular scopes. */
export function parseGeneratedContract(source: string): ContractOperation[] {
  const operations: ContractOperation[] = []
  for (const row of source.matchAll(
    /"(\w+)":\s*\{Method:\s*"(\w+)",\s*Path:\s*"([^"]*)".*?ApiTokenScopes:\s*(?:nil|\[\]string\{([^}]*)\})/g,
  )) {
    const [, name = '', method = '', path = '', scopeList] = row
    operations.push({
      name,
      method,
      path,
      scopes: [...(scopeList ?? '').matchAll(/"([^"]+)"/g)].map((match) => match[1] ?? ''),
    })
  }
  return operations
}

/** Per operation and status, the error types TypeSpec declares as that status' body. */
export function parseDeclaredErrors(typespec: string): DeclaredErrors {
  const errors = new Map<string, Map<string, string[]>>()
  for (const union of typespec.matchAll(/union\s+(\w+)Error(\d{3})Body\s*\{([^}]*)\}/g)) {
    const [, operation = '', status = '', body = ''] = union
    const types = [...body.matchAll(/\b(\w+Error)\b/g)].map((match) => match[1] ?? '')
    errors.set(operation, (errors.get(operation) ?? new Map()).set(status, types))
  }
  return errors
}

/** The operation names TypeSpec declares in the given sources. */
export function parseDeclaredOperations(typespec: string): Set<string> {
  return new Set([...typespec.matchAll(/^op\s+(\w+)\s*\(/gm)].map((match) => match[1] ?? ''))
}
