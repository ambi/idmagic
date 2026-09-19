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

/**
 * 「正準文書」と「正準 Markdown」は文書体系を指すときにしか現れないので、語形だけで
 * 判定できる。「正本」は所有者を指す一般名としても現れるため、文書体系そのものを
 * 主題とする文書と、リポジトリの道具立てに限って禁じる。
 */
const PRIMARY_SOURCE_VOCABULARY_ROOTS = [
  'DOCUMENTATION_GUIDE.md',
  'SPECIFICATION_FORMAT.md',
  'WORK_ITEM_FORMAT.md',
  'docs/README.md',
  'docs/domain/glossary.md',
  'docs/domain/structure.md',
  'docs/development',
  'docs/verification',
  'frontend/src',
  'tools',
]

/** 生成物と依存を除いて、人が書いた原稿だけを読む。 */
const SKIPPED_DIRECTORIES = new Set(['node_modules', 'dist', 'generated'])

/**
 * version の表記を確かめる範囲。`docs/domain/` の規範シナリオと `backend/` は、
 * 仕様先行の手順を通す別の作業項目が扱う。
 */
const VERSION_VOCABULARY_ROOTS = [
  'AGENTS.md',
  'CONTRIBUTING.md',
  'SECURITY.md',
  'SPECIFICATION_FORMAT.md',
  'WORK_ITEM_FORMAT.md',
  'docs/architecture',
  'docs/design',
  'docs/development',
  'docs/operations',
  'docs/requirements',
  'docs/runbooks',
  'docs/verification',
  'docs/README.md',
  'docs/domain/glossary.md',
  'docs/design/product-overview.md',
  'docs/domain/standards.md',
  'docs/domain/structure.md',
  'infra/schema/README.md',
  'frontend/src',
  'tools',
]

/** 禁じる語を書かずに規則を述べられないので、この規則を宣言するファイル自身は除く。 */
const RULE_DECLARATION = 'tools/render-docs/src/documentation-quality.test.ts'

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

it('文書体系を指す語に「正本」と「正準文書」を使わない', () => {
  const paths = PRIMARY_SOURCE_VOCABULARY_ROOTS.flatMap(prose)

  expect(citations(paths, /正本|正準文書|正準 ?Markdown|正準 ?scope/)).toEqual([])
})

/**
 * 定着したカタカナ語を一般的な日本語へ言い換えると、読み手はそれを技術用語として
 * 認識できない。`DOCUMENTATION_GUIDE.md` が version の表記を定めているので、規則を
 * 述べる同文書だけを除いて語形で確かめる。`docs/domain/` と `backend/` は
 * 別の作業項目が扱う。
 */
it('version を「版」と書かない', () => {
  const paths = VERSION_VOCABULARY_ROOTS.flatMap(prose)

  // revision の訳語である「第 N 版」は version の表記ではない。
  expect(citations(paths, /(?<!\{revision\})版/)).toEqual([])
})
