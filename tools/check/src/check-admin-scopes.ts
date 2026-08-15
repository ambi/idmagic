#!/usr/bin/env bun

import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { discoverGeneratedOpenApi } from '../../workspace/src/workspace.ts'
import {
  collectAdminOperations,
  type OpenApiDocument,
  parseApiTokenScopes,
  verifyAdminScopes,
} from './admin-scopes.ts'

const root = resolve(import.meta.dir, '../../..')
const document = JSON.parse(
  await readFile(await discoverGeneratedOpenApi(root), 'utf8'),
) as OpenApiDocument
const vocabulary = parseApiTokenScopes(
  await readFile(resolve(root, 'spec/contexts/api-tokens/models.tsp'), 'utf8'),
)

const operations = collectAdminOperations(document)
const findings = verifyAdminScopes(operations, vocabulary)
for (const finding of findings) console.error(`admin API scope declaration: ${finding}`)
if (findings.length > 0) process.exit(1)
console.log(`ok  admin API scope declarations (${operations.length} operation(s))`)
