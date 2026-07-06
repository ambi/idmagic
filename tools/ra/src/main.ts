#!/usr/bin/env bun
import { existsSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import { basename, resolve } from 'node:path'
import { TOOLS_DIR } from './workspace.ts'

const command = process.argv[2]
const rest = process.argv.slice(3)

function usage(): string {
  return [
    'Usage: ra <command>',
    '',
    'Commands:',
    '  init                                    Create the standard RA layout',
    '  yaml-check [--work-items] [--scl] [--ids]  Validate discovered RA records',
    '  verify                                  Alias for yaml-check',
    '  render                                  Regenerate discovered SCL artifacts',
    '',
  ].join('\n')
}

async function initWorkspace(): Promise<void> {
  const root = resolve(process.cwd())
  await mkdir(resolve(root, 'spec/contexts'), { recursive: true })
  await mkdir(resolve(root, 'decisions'), { recursive: true })
  await mkdir(resolve(root, 'work-items/done'), { recursive: true })
  const sclPath = resolve(root, 'spec/scl.yaml')
  if (!existsSync(sclPath)) {
    const system = basename(root)
      .replace(/[^a-zA-Z0-9_-]+/g, '-')
      .toLowerCase()
    await writeFile(sclPath, `system: ${system}\nspec_version: "2.0"\n`, 'utf8')
  }
  console.log('created standard RA layout')
}

async function runScript(args: string[]): Promise<never> {
  const proc = Bun.spawn(['bun', 'run', ...args], {
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

if (command === 'yaml-check') {
  await runScript(['ra/src/yaml-check-workspace.ts', ...rest])
}

if (command === 'init') {
  await initWorkspace()
  process.exit(0)
}

if (command === 'verify') {
  await runScript(['ra/src/yaml-check-workspace.ts'])
}

if (command === 'render') {
  await runScript(['ra/src/render-workspace.ts'])
}

console.error(`ra: unknown command ${command}`)
process.stdout.write(usage())
process.exit(2)
