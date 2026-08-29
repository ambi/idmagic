#!/usr/bin/env bun

import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import {
  collectConsumedEventFields,
  collectDeclaredEventFields,
  diffEventFieldVocabulary,
} from './event-contract.ts'

const root = resolve(import.meta.dir, '../../..')

/** 公開項目の語彙を宣言する正本。System が配信点を所有するので System が持つ。 */
const DECLARATION = 'spec/contexts/system/models.tsp'

/**
 * Context をまたいでイベントの payload を項目名で読む箇所。ここに挙がっていない
 * 読み取りは、その Context の内部にとどまるものとして契約の対象にしない。
 */
const CONSUMERS = [
  'backend/audit/usecases/audit_search_extractor.go',
  'backend/authentication/securitynotification/domain/catalog.go',
  'backend/authentication/securitynotification/usecases/dispatch.go',
]

const declared = collectDeclaredEventFields(await readFile(resolve(root, DECLARATION), 'utf8'))

const sources = new Map<string, string>()
for (const path of CONSUMERS) sources.set(path, await readFile(resolve(root, path), 'utf8'))
const consumed = collectConsumedEventFields(sources)

const { missing, undeclared } = diffEventFieldVocabulary(declared, consumed)

for (const field of missing) {
  console.error(
    `fail  ${DECLARATION}: DomainEventPayload does not declare the consumed field ${field}`,
  )
}
for (const field of undeclared) {
  console.error(
    `fail  ${DECLARATION}: DomainEventPayload declares ${field}, which no consumer reads`,
  )
}
if (missing.length + undeclared.length > 0) process.exit(1)

console.log(
  `ok  ${declared.size} published event payload field(s), ${CONSUMERS.length} consumer(s)`,
)
