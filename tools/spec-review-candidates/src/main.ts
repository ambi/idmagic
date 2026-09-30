#!/usr/bin/env bun

/**
 * 既存コードから仕様漏れの候補を報告する。
 *
 *   spec-review-candidates <package-directory>
 *
 * 仕様 ID を引くテストだけを実行して Go の文レベル被覆を取り、そのテスト群が実行しない
 * 本番コードの位置を並べる。報告は既存コードを書き起こすときの着手点であり、ゲートではない。
 * 計算は candidates.ts にあり、このファイルは Go の起動とファイルの読み取りだけを担う。
 */

import { mkdtemp, rm, stat } from 'node:fs/promises'
import { availableParallelism, tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { isProductionCode, productionCodeExclusions } from '../../check/src/production-code.ts'
import { readWorkingTree } from '../../check/src/spec-diff.ts'
import { validateDocument } from '../../check/src/specification-doc.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  type CoverBlock,
  type PackageImports,
  formatReport,
  goTestArguments,
  packagesReaching,
  parseCoverProfile,
  specCitingTests,
  unexecutedRanges,
} from './candidates.ts'

const ROOT = resolve(import.meta.dir, '../../..')
const EXCLUDED_DIRECTORIES = ['node_modules', 'vendor', 'dist', 'build', 'generated']

function fail(message: string): never {
  console.error(message)
  process.exit(2)
}

function directory(path: string): string {
  return path.slice(0, Math.max(0, path.lastIndexOf('/')))
}

async function declaredNormativeIds(): Promise<string[]> {
  const ids: string[] = []
  for (const [path, source] of await readWorkingTree(ROOT)) {
    if (!path.endsWith('.md')) continue
    const document = validateDocument(path, source)
    for (const declared of [
      ...document.scenarioIds,
      ...document.exampleIds,
      ...document.standardIds,
    ]) {
      ids.push(declared.id)
    }
  }
  return ids
}

async function goPackageImports(): Promise<PackageImports[]> {
  const format =
    '{{.ImportPath}}\t{{join .Imports ","}}\t{{join .TestImports ","}},{{join .XTestImports ","}}'
  const proc = Bun.spawn(['go', 'list', '-f', format, './backend/...'], {
    cwd: ROOT,
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  if (code !== 0) fail(`go list failed: ${stderr.trim()}`)
  return stdout
    .split('\n')
    .filter((line) => line.length > 0)
    .map((line) => {
      const [importPath = '', imports = '', testImports = ''] = line.split('\t')
      const list = (value: string) => value.split(',').filter((one) => one.length > 0)
      return { importPath, imports: list(imports), testImports: list(testImports) }
    })
}

/** 一つのパッケージで、名指したテストだけを対象の被覆つきで実行する。 */
async function runCitingTests(
  packageDirectory: string,
  tests: readonly string[],
  target: string,
  profile: string,
): Promise<{ blocks: CoverBlock[]; ok: boolean }> {
  const proc = Bun.spawn(['go', ...goTestArguments(packageDirectory, tests, target, profile)], {
    cwd: ROOT,
    stdout: 'ignore',
    stderr: 'ignore',
  })
  const code = await proc.exited
  const file = Bun.file(profile)
  return {
    blocks: (await file.exists()) ? parseCoverProfile(await file.text()) : [],
    ok: code === 0,
  }
}

const argument = process.argv[2]
if (!argument) fail('usage: mise run spec-review-candidates -- <package-directory>')
const target = argument.replace(/^\.\//, '').replace(/\/+$/, '')
const targetIsDirectory = await stat(resolve(ROOT, target))
  .then((entry) => entry.isDirectory())
  .catch(() => false)
if (!target.startsWith('backend/') || !targetIsDirectory) {
  fail(
    `${argument} is not a package directory under backend/; name the directory, not an import path`,
  )
}

const started = performance.now()
const snapshot = createWorkspaceSnapshot(ROOT)
const modulePath = (await snapshot.read('go.mod')).match(/^module\s+(\S+)/m)?.[1]
if (!modulePath) fail('go.mod declares no module path')

const declared = await declaredNormativeIds()
const sources = new Map<string, string>()
for (const path of await snapshot.files('backend', EXCLUDED_DIRECTORIES)) {
  if (path.endsWith('.go')) sources.set(path, await snapshot.read(path))
}
const citing = new Map<string, string[]>()
for (const [path, source] of sources) {
  if (!path.endsWith('_test.go')) continue
  const names = specCitingTests(source, declared)
  if (names.length === 0) continue
  citing.set(directory(path), [...(citing.get(directory(path)) ?? []), ...names])
}

const reaching = packagesReaching(`${modulePath}/${target}`, await goPackageImports()).map(
  (importPath) => importPath.slice(modulePath.length + 1),
)
const pending = reaching.filter((packageDirectory) => citing.has(packageDirectory))
const tests = new Map(
  pending.map((packageDirectory) => [packageDirectory, citing.get(packageDirectory)!]),
)

const profiles = await mkdtemp(join(tmpdir(), 'spec-review-candidates-'))
const blocks: CoverBlock[] = []
const failedPackages: string[] = []
try {
  const queue = [...pending.entries()]
  const worker = async (): Promise<void> => {
    for (let next = queue.shift(); next !== undefined; next = queue.shift()) {
      const [index, packageDirectory] = next
      const result = await runCitingTests(
        packageDirectory,
        tests.get(packageDirectory)!,
        target,
        join(profiles, `${index}.out`),
      )
      blocks.push(...result.blocks)
      if (!result.ok) failedPackages.push(packageDirectory)
    }
  }
  await Promise.all(Array.from({ length: Math.min(4, availableParallelism()) }, worker))
} finally {
  await rm(profiles, { recursive: true, force: true })
}

const exclusions = productionCodeExclusions(modulePath, sources)
const ranges = new Map(
  [...unexecutedRanges(blocks)]
    .map(([file, fileRanges]) => [file.slice(modulePath.length + 1), fileRanges] as const)
    .filter(([file]) => isProductionCode(file, exclusions)),
)
process.stdout.write(formatReport({ target, tests, failedPackages: failedPackages.sort(), ranges }))
console.error(`elapsed: ${((performance.now() - started) / 1000).toFixed(1)}s`)
