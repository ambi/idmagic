export type OpenAPIOperation = {
  operationId?: string
  deprecated?: boolean
  'x-api-token-scopes'?: string[]
}

export type OpenAPIDocument = {
  paths?: Record<string, Record<string, OpenAPIOperation>>
}

export type ContractOperation = {
  name: string
  method: string
  path: string
  deprecated: boolean
  apiTokenScopes: string[]
}

const HTTP_METHODS = new Set(['delete', 'get', 'head', 'options', 'patch', 'post', 'put'])

export function collectOperations(document: OpenAPIDocument): ContractOperation[] {
  const operations: ContractOperation[] = []
  const seen = new Map<string, { method: string; path: string }>()
  for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!operation.operationId || !HTTP_METHODS.has(method)) continue
      const current = { method: method.toUpperCase(), path }
      const previous = seen.get(operation.operationId)
      if (previous) {
        throw new Error(
          `duplicate operationId ${operation.operationId}: ${previous.method} ${previous.path} and ${current.method} ${current.path}`,
        )
      }
      seen.set(operation.operationId, current)
      operations.push({
        name: operation.operationId,
        method: current.method,
        path,
        deprecated: operation.deprecated === true,
        apiTokenScopes: operation['x-api-token-scopes'] ?? [],
      })
    }
  }
  return operations.sort((left, right) => left.name.localeCompare(right.name))
}

const quote = (value: string): string => JSON.stringify(value)

// ApiTokenScopes は operation ごとの x-api-token-scopes 拡張をそのまま運ぶ。空 (nil) は
// 「API アクセストークンからの到達可否が宣言されていない」を意味し、実行時はフェイルクローズに拒否する。
const scopeLiteral = (scopes: string[]): string =>
  scopes.length === 0 ? 'nil' : `[]string{${scopes.map(quote).join(', ')}}`

export function renderContract(operations: ContractOperation[]): string {
  const entries = operations
    .map(
      ({ name, method, path, deprecated, apiTokenScopes }) =>
        `\t${quote(name)}: {Method: ${quote(method)}, Path: ${quote(path)}, Deprecated: ${deprecated}, ApiTokenScopes: ${scopeLiteral(apiTokenScopes)}},`,
    )
    .join('\n')
  return `// Code generated from spec/main.tsp by mise run spec-render; DO NOT EDIT.\n\npackage spec\n\nvar generatedOperations = map[string]Operation{\n${entries}\n}\n`
}
