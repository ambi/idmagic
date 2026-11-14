import { expect, it } from 'bun:test'
import { readFileSync, readdirSync, statSync } from 'node:fs'
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
  const paths = ['docs/formats/documentation-guide.md', 'docs/requirements/quality.md']
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
    'docs/formats/specification-format.md',
    'docs/formats/work-item-format.md',
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

/**
 * 「正準文書」と「正準 Markdown」は文書体系を指すときにしか現れないので、語形だけで
 * 判定できる。「正本」は所有者を指す一般名としても現れるため、語形では判定せず規則に置かない。
 */
const PRIMARY_SOURCE_VOCABULARY_ROOTS = [
  'docs/formats/documentation-guide.md',
  'docs/formats/specification-format.md',
  'docs/formats/work-item-format.md',
  'docs/README.md',
  'docs/modules/glossary.md',
  'docs/design/application/backend.md',
  'docs/development',
  'docs/design/verification',
  'frontend/src',
  'tools',
]

/** 生成物と依存を除いて、人が書いた原稿だけを読む。 */
const SKIPPED_DIRECTORIES = new Set(['node_modules', 'dist', 'generated'])

/** 禁じる語を書かずに規則を述べられないので、この規則を宣言するファイル自身は除く。 */
const RULE_DECLARATION = 'tools/check/repository/documentation.repository.ts'

function prose(target: string): string[] {
  const absolute = resolve(repositoryRoot, target)
  if (!statSync(absolute).isDirectory()) return [target]
  return readdirSync(absolute, { withFileTypes: true }).flatMap((entry) => {
    if (entry.isDirectory())
      return SKIPPED_DIRECTORIES.has(entry.name) ? [] : prose(`${target}/${entry.name}`)
    const path = `${target}/${entry.name}`
    return /\.(md|ts|tsx|json)$/.test(entry.name) && path !== RULE_DECLARATION ? [path] : []
  })
}

function citations(paths: string[], pattern: RegExp): string[] {
  return paths.flatMap((path) =>
    readFileSync(resolve(repositoryRoot, path), 'utf8')
      .split('\n')
      .flatMap((line, index) =>
        pattern.test(line) ? [`${path}:${index + 1}: ${line.trim()}`] : [],
      ),
  )
}

it('文書体系を指す語に「正準文書」を使わない', () => {
  const paths = PRIMARY_SOURCE_VOCABULARY_ROOTS.flatMap(prose)

  expect(citations(paths, /正準文書|正準 ?Markdown|正準 ?scope/)).toEqual([])
})
