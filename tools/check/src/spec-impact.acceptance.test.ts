import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { TOOLS_DIR } from '../../workspace/src/workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

const MODULE = 'example.test/product'
const SCENARIOS = 'docs/domain/demo/scenarios.feature.md'
const TYPESPEC = 'spec/contexts/demo/main.tsp'
const SERVICE = 'backend/demo/service.go'
const SERVICE_TEST = 'backend/demo/service_test.go'
const ITEM = 'work-items/wi-10001-demo.md'
const CONCRETE_REASON = 'keeps the returned task, the stored state, and the emitted events'

const scenario = (result: string): string =>
  [
    '# Feature: Demo Scenarios',
    '',
    '## Rule: REQ-DEMO-001 Demo succeeds',
    '',
    '### Example: EX-DEMO-001-01 valid request',
    '',
    '- When the user submits a request',
    `- Then ${result}`,
    '',
  ].join('\n')

const service = (body: string): string =>
  `package demo\n\nfunc Start() string { return "${body}" }\n`

const workItem = (frontmatter: string[]): string =>
  ['---', ...frontmatter, '---', '', '# Demo change', ''].join('\n')

const affected = (status: string, entry: string): string =>
  workItem([`status: ${status}`, 'change_kind: bugfix', 'affected_spec:', `  - ${entry}`])

const notAffected = (status: string, reason: string): string =>
  workItem([
    `status: ${status}`,
    'change_kind: refactor',
    `spec_impact: { kind: none, reason: "${reason}" }`,
  ])

type Repository = {
  root: string
  git(...args: string[]): Promise<string>
  write(path: string, content: string): Promise<void>
  /** 作業ツリーのすべての変更をコミットし、そのコミットを返す。 */
  commit(message: string): Promise<string>
  check(base: string): Promise<{ code: number; output: string }>
}

async function run(cwd: string, command: string[], env: Record<string, string> = {}) {
  const proc = Bun.spawn(command, {
    cwd,
    env: { ...process.env, ...env },
    stdout: 'pipe',
    stderr: 'pipe',
  })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { code, output: `${stdout}${stderr}` }
}

/** 仕様、本番コード、work item の置き場所を備えた最小のリポジトリ。 */
async function repository(): Promise<Repository & { base: string }> {
  const root = await mkdtemp(join(tmpdir(), 'check-spec-impact-test-'))
  cleanup.push(root)
  const git = async (...args: string[]): Promise<string> => {
    const result = await run(root, [
      'git',
      '-c',
      'user.name=test',
      '-c',
      'user.email=test@example.test',
      '-c',
      'commit.gpgsign=false',
      ...args,
    ])
    if (result.code !== 0) throw new Error(`git ${args.join(' ')} failed: ${result.output}`)
    return result.output.trim()
  }
  const write = async (path: string, content: string): Promise<void> => {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), content)
  }
  const commit = async (message: string): Promise<string> => {
    await git('add', '-A')
    await git('commit', '--quiet', '--message', message)
    return git('rev-parse', 'HEAD')
  }
  await git('init', '--quiet', '--initial-branch', 'main')
  await write('go.mod', `module ${MODULE}\n`)
  await write(
    'backend/cmd/server/main.go',
    `package main\n\nimport "${MODULE}/backend/demo"\n\nfunc main() { demo.Start() }\n`,
  )
  await write(SERVICE, service('started'))
  await write(SCENARIOS, scenario('the request succeeds'))
  await write(TYPESPEC, 'model Task {\n  id: string;\n}\n')
  await write('work-items/README.txt', 'records\n')
  const base = await commit('chore: initial')
  return {
    root,
    base,
    git,
    write,
    commit,
    check: (revision) =>
      run(
        TOOLS_DIR,
        [
          'bun',
          'run',
          resolve(TOOLS_DIR, 'check/src/runner.ts'),
          'spec-impact',
          '--base-revision',
          revision,
        ],
        { SPEC_WORKSPACE_ROOT: root },
      ),
  }
}

describe('spec-impact: 本番コードを変更するコミットの宣言', () => {
  it('仕様影響を宣言しないコミットを含む範囲を拒否する', async () => {
    const repo = await repository()
    await repo.write(SERVICE, service('running'))
    await repo.commit('fix(demo): report running')

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      `fix(demo): report running: changes production code (${SERVICE}) without declaring its specification impact`,
    )
    expect(result.code).toBe(1)
  })

  it('テストだけを変更するコミットには宣言を求めない', async () => {
    const repo = await repository()
    await repo.write(SERVICE_TEST, 'package demo\n')
    await repo.commit('test(demo): add a case')

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('維持する振る舞いを名指すトレーラーのあるコミットを通す', async () => {
    const repo = await repository()
    await repo.write(SERVICE, `${service('started')}\nfunc helper() {}\n`)
    await repo.commit(`refactor(demo): extract helper\n\nSpec-Impact: none — ${CONCRETE_REASON}`)

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('理由が定型句だけのトレーラーを拒否する', async () => {
    const repo = await repository()
    await repo.write(SERVICE, `${service('started')}\nfunc helper() {}\n`)
    await repo.commit('refactor(demo): extract helper\n\nSpec-Impact: none — refactoring only')

    const result = await repo.check(repo.base)
    expect(result.output).toContain('Spec-Impact reason names nothing that stays the same')
    expect(result.code).toBe(1)
  })

  it('Spec-Impact: none のコミットが規範シナリオを変更していれば拒否する', async () => {
    const repo = await repository()
    await repo.write(SERVICE, service('running'))
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.commit(`refactor(demo): rename\n\nSpec-Impact: none — ${CONCRETE_REASON}`)

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      'declares Spec-Impact: none, but changes the normative specification: REQ-DEMO-001',
    )
    expect(result.code).toBe(1)
  })

  it('同じコミットで変更した work item の宣言を、そのコミットの宣言として読む', async () => {
    const repo = await repository()
    await repo.write(SERVICE, `${service('started')}\nfunc helper() {}\n`)
    await repo.write(ITEM, notAffected('completed', CONCRETE_REASON))
    await repo.commit('refactor(demo): extract helper')

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('checkpoint コミットは、件名が名指す work item の宣言を使う', async () => {
    const repo = await repository()
    await repo.write(ITEM, notAffected('in_progress', CONCRETE_REASON))
    await repo.commit('checkpoint(wi-10001): T001 record started')
    await repo.write(SERVICE, `${service('started')}\nfunc helper() {}\n`)
    await repo.commit('checkpoint(wi-10001): T002 helper extracted')

    expect((await repo.check(repo.base)).code).toBe(0)

    await repo.write(SERVICE, `${service('started')}\nfunc other() {}\n`)
    await repo.commit('checkpoint(wi-99999): T001 names no record')
    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      'checkpoint(wi-99999): T001 names no record: changes production code',
    )
    expect(result.code).toBe(1)
  })
})

describe('spec-impact: 宣言と仕様差分の整合', () => {
  it('impact: modifies の規範要素を変更していなければ拒否する', async () => {
    const repo = await repository()
    await repo.write(SERVICE, service('running'))
    await repo.write(
      ITEM,
      affected('in_progress', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      `${ITEM}: affected_spec ${SCENARIOS} REQ-DEMO-001 is impact: modifies, but the work range does not change it`,
    )
    expect(result.code).toBe(1)
  })

  it('impact: modifies の規範要素を作業ツリーで変更していれば通す', async () => {
    const repo = await repository()
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.write(
      ITEM,
      affected('in_progress', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('TypeSpec のモデルの項目を変えた変更を、シンボルの modifies として受け入れる', async () => {
    const repo = await repository()
    await repo.write(TYPESPEC, 'model Task {\n  id: string;\n  name: string;\n}\n')
    await repo.write(ITEM, affected('in_progress', `{ path: ${TYPESPEC}, symbol: Demo.Task }`))

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('impact: conforms の規範要素を引くテストが作業範囲になければ拒否する', async () => {
    const repo = await repository()
    await repo.write(SERVICE, service('running'))
    await repo.write(
      ITEM,
      affected(
        'in_progress',
        `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001, impact: conforms }`,
      ),
    )

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      'REQ-DEMO-001 is impact: conforms, but no test added or changed in the work range names it',
    )
    expect(result.code).toBe(1)
  })

  it('impact: conforms の規範要素を引くテストを同じ作業で追加していれば通す', async () => {
    const repo = await repository()
    await repo.write(SERVICE, service('running'))
    await repo.write(
      SERVICE_TEST,
      'package demo\n\n//spec:covers EX-DEMO-001-01: 返す値\nfunc TestStart(t *testing.T) {}\n',
    )
    await repo.write(
      ITEM,
      affected(
        'in_progress',
        `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001, impact: conforms }`,
      ),
    )

    const result = await repo.check(repo.base)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('spec_impact: none の work item と規範仕様の変更が同じ範囲にあれば拒否する', async () => {
    const repo = await repository()
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.write(ITEM, notAffected('in_progress', CONCRETE_REASON))

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      `${ITEM}: spec_impact is none, but the work range changes the normative specification: REQ-DEMO-001`,
    )
    expect(result.code).toBe(1)
  })

  it('差分に現れた規範要素をどの work item も挙げていなければ拒否する', async () => {
    const repo = await repository()
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.commit('docs(demo): reword the scenario')

    const result = await repo.check(repo.base)
    expect(result.output).toContain(
      'REQ-DEMO-001: the specification changes here, but no work item changed in this range lists it',
    )
    expect(result.code).toBe(1)
  })

  it('着手後、基準より前に入れた仕様の変更を作業範囲に含める', async () => {
    const repo = await repository()
    await repo.write(ITEM, affected('pending', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`))
    await repo.commit('docs(work-items): file the demo change')
    await repo.write(
      ITEM,
      affected('in_progress', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    const specified = await repo.commit('docs(demo): specify the accepted result')
    await repo.write(
      ITEM,
      affected('completed', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )
    await repo.write(SERVICE, service('accepted'))
    await repo.commit('feat(demo): accept the request')

    const result = await repo.check(specified)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('分岐した後に基準の側で入った仕様の変更を、このブランチの差分に数えない', async () => {
    const repo = await repository()
    await repo.git('switch', '--quiet', '--create', 'work')
    await repo.write(SERVICE_TEST, 'package demo\n')
    await repo.commit('test(demo): add a case')
    await repo.git('switch', '--quiet', 'main')
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.write(
      ITEM,
      affected('completed', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )
    await repo.commit('feat(demo): accept the request')
    await repo.git('switch', '--quiet', 'work')

    const result = await repo.check('main')
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })

  it('長く続く none の記録は、同じ範囲で別の work item が挙げた仕様の変更を矛盾にしない', async () => {
    const repo = await repository()
    await repo.write(ITEM, notAffected('in_progress', CONCRETE_REASON))
    await repo.commit('docs(work-items): start the long refactoring')
    await repo.write(SCENARIOS, scenario('the request is accepted'))
    await repo.write(
      'work-items/done/wi-10002-other.md',
      affected('completed', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )
    const other = await repo.commit('feat(demo): accept the request')
    await repo.write(ITEM, `${notAffected('in_progress', CONCRETE_REASON)}\nProgress.\n`)

    const result = await repo.check(other)
    expect(result.output).toContain('ok  spec impact (1 work item(s)')
    expect(result.code).toBe(0)
  })

  it('基準の時点で完了していた記録は、書き換えても再検査しない', async () => {
    const repo = await repository()
    await repo.write(
      'work-items/done/wi-10001-demo.md',
      affected('completed', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`),
    )
    const completed = await repo.commit('docs(work-items): an old record')
    await repo.write(
      'work-items/done/wi-10001-demo.md',
      `${affected('completed', `{ path: ${SCENARIOS}, requirement: REQ-DEMO-001 }`)}\nA corrected link.\n`,
    )

    const result = await repo.check(completed)
    expect(result.output).toContain('ok  spec impact')
    expect(result.code).toBe(0)
  })
})
