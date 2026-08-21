#!/usr/bin/env bun

import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'

type OpenAPIOperation = {
  operationId?: string
  deprecated?: boolean
  'x-api-token-scopes'?: string[]
}

type OpenAPIDocument = {
  paths?: Record<string, Record<string, OpenAPIOperation>>
}

const root = resolve(import.meta.dir, '../../..')
const input = await discoverGeneratedOpenApi(root)
const output = resolve(root, 'backend/shared/spec/operations_gen.go')
const check = process.argv.includes('--check')

const document = JSON.parse(await readFile(input, 'utf8')) as OpenAPIDocument
const operations: Array<{
  name: string
  method: string
  path: string
  deprecated: boolean
  apiTokenScopes: string[]
}> = []
const seen = new Set<string>()
for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (
      !operation.operationId ||
      seen.has(operation.operationId) ||
      !['delete', 'get', 'head', 'options', 'patch', 'post', 'put'].includes(method)
    ) {
      continue
    }
    seen.add(operation.operationId)
    operations.push({
      name: operation.operationId,
      method: method.toUpperCase(),
      path,
      deprecated: operation.deprecated === true,
      apiTokenScopes: operation['x-api-token-scopes'] ?? [],
    })
  }
}
operations.sort((left, right) => left.name.localeCompare(right.name))

const quote = (value: string): string => JSON.stringify(value)
// ApiTokenScopes は operation ごとの x-api-token-scopes 拡張をそのまま運ぶ。空 (nil) は
// 「API アクセストークンからの到達可否が宣言されていない」を意味し、実行時はフェイルクローズに拒否する。
const scopeLiteral = (scopes: string[]): string =>
  scopes.length === 0 ? 'nil' : `[]string{${scopes.map(quote).join(', ')}}`
const entries = operations
  .map(
    ({ name, method, path, deprecated, apiTokenScopes }) =>
      `\t${quote(name)}: {Method: ${quote(method)}, Path: ${quote(path)}, Deprecated: ${deprecated}, ApiTokenScopes: ${scopeLiteral(apiTokenScopes)}},`,
  )
  .join('\n')
const generated = `// Code generated from spec/main.tsp by mise run spec-render; DO NOT EDIT.\n\npackage spec\n\nvar generatedOperations = map[string]Operation{\n${entries}\n}\n`

if (check) {
  const current = await readFile(output, 'utf8').catch(() => '')
  if (current !== generated) {
    console.error('backend/shared/spec/operations_gen.go is stale; run mise run spec-render')
    process.exit(1)
  }
} else {
  await writeFile(output, generated, 'utf8')
}
