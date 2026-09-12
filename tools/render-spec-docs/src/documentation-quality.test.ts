import { expect, it } from 'bun:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const repositoryRoot = resolve(import.meta.dir, '../../..')

it('ER 図が PostgreSQL の全テーブルを網羅する', () => {
  const schema = readFileSync(resolve(repositoryRoot, 'infra/schema/postgres.sql'), 'utf8')
  const databaseDesign = readFileSync(
    resolve(repositoryRoot, 'docs/design/data/database.md'),
    'utf8',
  )
  const tables = [...schema.matchAll(/^CREATE (?:UNLOGGED )?TABLE ([a-z0-9_]+) \(/gm)].map(
    (match) => match[1],
  )
  const missing = tables.filter(
    (table) => !new RegExp(`^\\s+${table} \\{`, 'm').test(databaseDesign),
  )

  expect(missing).toEqual([])
})

it('人向けの品質説明に「被覆」を使わない', () => {
  const paths = ['DOCUMENTATION_GUIDE.md', 'docs/requirements/quality.md']
  const occurrences = paths.flatMap((path) =>
    readFileSync(resolve(repositoryRoot, path), 'utf8')
      .split('\n')
      .filter((line) => line.includes('被覆'))
      .map((line) => `${path}: ${line}`),
  )

  expect(occurrences).toEqual([])
})

it('主要な開発文書に英語見出しを残さない', () => {
  const paths = [
    'SPECIFICATION_FORMAT.md',
    'WORK_ITEM_FORMAT.md',
    'docs/development/specification-first-workflow.md',
    'docs/development/release.md',
  ]
  const englishHeadings = paths.flatMap((path) =>
    readFileSync(resolve(repositoryRoot, path), 'utf8')
      .split('\n')
      .filter((line) => /^#{1,6} (?:\d+\. )?[A-Za-z][A-Za-z -]+$/.test(line))
      .map((line) => `${path}: ${line}`),
  )

  expect(englishHeadings).toEqual([])
})
