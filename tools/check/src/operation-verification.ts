export type OperationVerification = {
  operationId: string
  state: 'verified' | 'unverified' | 'failed'
  reasons: string[]
}

export function operationVerification(
  ids: readonly string[],
  findings: readonly { operationId: string; message: string }[],
  unresolved: readonly { operationId: string; reason: string }[],
): OperationVerification[] {
  return [...new Set(ids)].map((operationId) => {
    const failures = findings.filter((entry) => entry.operationId === operationId)
    const missing = unresolved.filter((entry) => entry.operationId === operationId)
    return {
      operationId,
      state: failures.length > 0 ? 'failed' : missing.length > 0 ? 'unverified' : 'verified',
      reasons: [
        ...new Set([
          ...failures.map((entry) => entry.message),
          ...missing.map((entry) => entry.reason),
        ]),
      ],
    }
  })
}

export function operationIds(document: {
  paths?: Record<string, Record<string, { operationId?: string }>>
}): string[] {
  const methods = new Set(['get', 'post', 'put', 'patch', 'delete', 'head', 'options'])
  return Object.values(document.paths ?? {}).flatMap((item) =>
    Object.entries(item).flatMap(([method, operation]) =>
      methods.has(method) && operation.operationId ? [operation.operationId] : [],
    ),
  )
}

export function verificationSummary(
  name: string,
  operations: readonly OperationVerification[],
): string {
  const count = (state: OperationVerification['state']) =>
    operations.filter((entry) => entry.state === state).length
  return `${name}: verified=${count('verified')}, unverified=${count('unverified')}, failed=${count('failed')} (operation(s)); analysis gaps are not passes`
}
