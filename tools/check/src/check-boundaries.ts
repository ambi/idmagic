#!/usr/bin/env bun

import { readdir, readFile } from 'node:fs/promises'
import { extname, relative, resolve } from 'node:path'

const root = resolve(import.meta.dir, '../../..')
const excluded = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])
const goModule = (await readFile(resolve(root, 'go.mod'), 'utf8')).match(/^module\s+(\S+)$/m)?.[1]
if (!goModule) throw new Error('go.mod must declare a module path')
const backendImportPrefix = `${goModule}/backend/`

async function walk(dir: string, result: string[] = []): Promise<string[]> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && excluded.has(entry.name)) continue
    const path = resolve(dir, entry.name)
    if (entry.isDirectory()) await walk(path, result)
    else if (entry.isFile()) result.push(path)
  }
  return result
}

const files = await walk(root)
let failed = false
for (const file of files) {
  const rel = relative(root, file)
  if (rel === 'architecture.yaml' || rel.endsWith('/architecture.yaml')) {
    console.error(
      `${rel}: exhaustive architecture ledgers are retired; enforce only forbidden imports`,
    )
    failed = true
    continue
  }
  const extension = extname(file)
  if (extension === '.go' && rel.startsWith('backend/') && !rel.endsWith('_test.go')) {
    const source = await readFile(file, 'utf8')
    // Startup configuration is read in one place. `backend/cmd/internal/bootstrap`
    // owns env access and validates it, so an invalid value fails startup instead
    // of silently falling back; `*_env` adapters resolve runtime locators rather
    // than startup configuration and keep their own env access.
    const ownsEnvAccess =
      rel.startsWith('backend/cmd/internal/bootstrap/') || /\/[a-z0-9]+_env\//.test(rel)
    if (!ownsEnvAccess) {
      const envReads = [...source.matchAll(/os\.(?:Getenv|LookupEnv|Environ)\b/g)].filter(
        (match) =>
          !source
            .slice(Math.max(0, (match.index ?? 0) - 30), match.index)
            .includes('bootstrap.NewConfigLoader('),
      )
      if (envReads.length > 0) {
        console.error(
          `${rel}: startup configuration must be read through bootstrap's ConfigLoader, not os.Getenv`,
        )
        failed = true
      }
    }
    const imports = [...source.matchAll(/"([^"\n]+)"/g)]
      .map((match) => match[1] ?? '')
      .filter((imported) => imported.startsWith(backendImportPrefix))
    const isDomain = rel.split('/').includes('domain')
    const isUseCase = rel.split('/').includes('usecases')
    for (const imported of imports) {
      const outward = /\/(?:handlers?_[^/]+|db_[^/]+|delivery|cmd)(?:\/|$)/.test(imported)
      const useCase = /\/usecases(?:\/|$)/.test(imported)
      if ((isDomain && (outward || useCase)) || (isUseCase && outward)) {
        console.error(`${rel}: forbidden outward dependency ${imported}`)
        failed = true
      }
    }
  }
  if (['.ts', '.tsx'].includes(extension) && rel.startsWith('frontend/src/')) {
    const source = await readFile(file, 'utf8')
    if (/from\s+['"](?:\.\.\/)+(?:backend|tools)\//.test(source)) {
      console.error(`${rel}: frontend source must not import backend or repository tooling`)
      failed = true
    }
  }
}

if (failed) process.exit(1)
console.log('ok  forbidden dependency boundaries')
