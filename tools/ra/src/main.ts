#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { TOOLS_DIR } from './workspace.ts'

const command = process.argv[2]
const rest = process.argv.slice(3)

function usage(): string {
  return [
    'Usage: ra <command>',
    '',
    'Commands:',
    '  init    Create the standard TypeSpec/requirements/work-item layout',
    '  check   Validate discovered RA records',
    '  verify  Alias for check',
    '',
  ].join('\n')
}

async function initWorkspace(): Promise<void> {
  const root = resolve(process.cwd())
  await mkdir(resolve(root, 'spec/contexts'), { recursive: true })
  await mkdir(resolve(root, 'work-items/done'), { recursive: true })
  const specPath = resolve(root, 'spec/main.tsp')
  if (!existsSync(specPath)) await writeFile(specPath, 'namespace Contract;\n', 'utf8')
  const requirementsPath = resolve(root, 'spec/requirements.md')
  if (!existsSync(requirementsPath)) {
    await writeFile(requirementsPath, '# Repository Requirements\n', 'utf8')
  }
  console.log('created standard RA layout')
}

async function runCheck(args: string[]): Promise<never> {
  const proc = Bun.spawn(['bun', 'run', 'ra/src/check-workspace.ts', ...args], {
    cwd: TOOLS_DIR,
    env: { ...process.env, RA_WORKSPACE_ROOT: process.cwd() },
    stdout: 'inherit',
    stderr: 'inherit',
  })
  process.exit(await proc.exited)
}

if (command === undefined || command === '--help' || command === '-h') {
  process.stdout.write(usage())
  process.exit(0)
}
if (command === 'check' || command === 'verify') await runCheck(rest)
if (command === 'init') {
  await initWorkspace()
  process.exit(0)
}
console.error(`ra: unknown command ${command}`)
process.stdout.write(usage())
process.exit(2)
