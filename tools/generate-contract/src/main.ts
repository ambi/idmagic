#!/usr/bin/env bun

import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import { collectOperations, renderContract, type OpenAPIDocument } from './contract.ts'

const root = resolve(import.meta.dir, '../../..')
const valueAfter = (flag: string): string | undefined => {
  const index = process.argv.indexOf(flag)
  return index === -1 ? undefined : process.argv[index + 1]
}
const input = valueAfter('--input') ?? (await discoverGeneratedOpenApi(root))
const output = valueAfter('--output') ?? resolve(root, 'backend/shared/spec/operations_gen.go')
const check = process.argv.includes('--check')

const document = JSON.parse(await readFile(input, 'utf8')) as OpenAPIDocument
const generated = renderContract(collectOperations(document))

if (check) {
  const current = await readFile(output, 'utf8').catch(() => '')
  if (current !== generated) {
    console.error('backend/shared/spec/operations_gen.go is stale; run mise run spec-render')
    process.exit(1)
  }
} else {
  await writeFile(output, generated, 'utf8')
}
