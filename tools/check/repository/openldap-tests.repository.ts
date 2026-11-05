import { expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { parseMiseTasks, taskClosure } from '../src/verification-tasks.ts'

const root = resolve(import.meta.dir, '../../..')
const tasks = parseMiseTasks(await Bun.file(resolve(root, 'mise.toml')).text())

function testFiles(pkg: string, tags?: string): string[] {
  const child = Bun.spawnSync(
    [
      'go',
      'list',
      '-f',
      '{{join .TestGoFiles " "}} {{join .XTestGoFiles " "}}',
      ...(tags ? [`-tags=${tags}`] : []),
      pkg,
    ],
    { cwd: root },
  )
  expect(child.exitCode, child.stderr.toString()).toBe(0)
  return child.stdout.toString().trim().split(' ')
}

it('通常の Go 試験から OpenLDAP の結合試験と E2E を除き、タグで明示的に選べる', () => {
  for (const [pkg, file] of [
    ['./backend/directoryconnector', 'connector_e2e_test.go'],
    ['./backend/directoryconnector/internal/ldap_goldap', 'directory_integration_test.go'],
    ['./backend/directoryconnector/internal/testing_openldap', 'server_test.go'],
  ] as const) {
    expect(testFiles(pkg)).not.toContain(file)
    expect(testFiles(pkg, 'openldap')).toContain(file)
  }
  expect(testFiles('./backend/directoryconnector/internal/ldap_goldap')).toContain(
    'directory_test.go',
  )
})

it('通常検証では Docker を要求せず、全体検証と CI が専用試験とタグ付き lint を実行する', async () => {
  const normal = taskClosure(tasks, 'verify')
  const full = taskClosure(tasks, 'verify-full')
  const workflow = Bun.YAML.parse(
    await Bun.file(resolve(root, '.github/workflows/idmagic-ci.yaml')).text(),
  ) as { jobs: Record<string, { steps: { run?: string; if?: string }[] }> }
  for (const task of ['test-go-openldap-integration', 'test-go-openldap-e2e', 'lint-go-openldap']) {
    expect(normal.has(task)).toBe(false)
    expect(full.has(task)).toBe(true)
    const job = task.startsWith('lint') ? 'go-lint' : 'go-test'
    const step = workflow.jobs[job]!.steps.find((step) => step.run === `mise run ${task}`)
    expect(step).toBeDefined()
    expect(step?.if).toBeUndefined()
    expect(tasks[task]?.run).toContain('-tags=openldap')
  }
})
