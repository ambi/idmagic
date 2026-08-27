#!/usr/bin/env bun

import { readdir, readFile, stat } from 'node:fs/promises'
import type { Stats } from 'node:fs'
import { posix, resolve } from 'node:path'
import { verifyMarkdownLinks } from './markdown-links.ts'

const root = resolve(import.meta.dir, '../../..')
const documents = new Map<string, string>()
const existingPaths = new Set<string>()

function excluded(path: string): boolean {
  const segments = path.split('/')
  return (
    segments.includes('.git') ||
    segments.includes('node_modules') ||
    path === 'spec/generated' ||
    path.startsWith('spec/generated/')
  )
}

async function collect(directory = ''): Promise<void> {
  const entries = await readdir(resolve(root, directory), { withFileTypes: true })
  entries.sort((left, right) => left.name.localeCompare(right.name))

  for (const entry of entries) {
    const path = directory === '' ? entry.name : posix.join(directory, entry.name)
    if (excluded(path)) continue
    const absolutePath = resolve(root, path)
    if (entry.isSymbolicLink()) {
      let target: Stats
      try {
        target = await stat(absolutePath)
      } catch {
        continue
      }
      if (!target.isFile()) continue
      existingPaths.add(path)
      if (path.endsWith('.md')) documents.set(path, await readFile(absolutePath, 'utf8'))
      continue
    }
    existingPaths.add(path)
    if (entry.isDirectory()) {
      await collect(path)
      continue
    }
    if (entry.isFile() && path.endsWith('.md'))
      documents.set(path, await readFile(absolutePath, 'utf8'))
  }
}

await collect()
const findings = verifyMarkdownLinks(documents, existingPaths)
for (const finding of findings) {
  console.error(`${finding.file}:${finding.line}: ${finding.message} (${finding.target})`)
}

if (findings.length > 0) process.exit(1)
console.log(`ok  Markdown links (${documents.size} document(s))`)
