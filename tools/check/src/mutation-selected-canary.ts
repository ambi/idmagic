import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

type Mutant = {
  id: string
  diff: string
  mutator: { mutatorName: string }
}

type Report = {
  killed: Mutant[] | null
  escaped: Mutant[] | null
  skipped: Mutant[] | null
}

const root = resolve(import.meta.dir, '../../..')
const fixture = 'tools/check/testdata/mutation-mapping'
const temporary = await mkdtemp(join(tmpdir(), 'idmagic-selected-canary-'))
const log = join(temporary, 'unrelated.log')

async function run(
  name: string,
  tests: string,
  { target = 'target', id = '', brokenBaseline = false } = {},
): Promise<Report | null> {
  const report = join(temporary, `${name}.json`)
  const started = performance.now()
  const child = Bun.spawn(
    ['mise', 'run', 'test-go-mutation-selected', '--', `${fixture}/${target}`, tests, report, id],
    {
      cwd: root,
      env: {
        ...process.env,
        MUTATION_MAPPING_LOG: log,
        MUTATION_MAPPING_FAIL_BASELINE: brokenBaseline ? '1' : '0',
      },
      stdout: 'pipe',
      stderr: 'pipe',
    },
  )
  const [stdout, stderr, code] = await Promise.all([
    new Response(child.stdout).text(),
    new Response(child.stderr).text(),
    child.exited,
  ])
  await Bun.write(join(temporary, `${name}.log`), stdout + stderr)
  if (brokenBaseline) {
    assert.notEqual(code, 0, 'a failing selected caller baseline must stop the run')
    assert.equal(
      await Bun.file(report).exists(),
      false,
      'a failed baseline must not produce verdicts',
    )
    assert.match(stdout + stderr, /baseline failure requested/)
    return null
  }
  assert.equal(code, 0, `${name} failed:\n${stdout}\n${stderr}`)
  console.log(`mutation selection canary: ${name} ${(performance.now() - started).toFixed(0)}ms`)
  return (await Bun.file(report).json()) as Report
}

try {
  const full = await run('full-scope', './...')
  assert.match(await Bun.file(log).text(), /TestMain/, 'the full scope must run unrelated setup')
  await rm(log)
  const local = await run('local', './target')
  const boundary = local?.escaped?.find(
    (mutant) => mutant.mutator.mutatorName === 'expression/comparison',
  )
  assert.ok(boundary?.id, 'local tests must miss the boundary mutant and expose its stable ID')

  const selected = await run('caller', './caller')
  assert.ok(
    selected?.killed?.some((mutant) => mutant.id === boundary.id),
    'the selected caller must kill the same boundary mutant',
  )
  assert.deepEqual(selected?.escaped ?? [], [], 'the caller must detect both comparison faults')
  assert.deepEqual(
    selected?.killed?.map((mutant) => mutant.id).sort(),
    full?.killed?.map((mutant) => mutant.id).sort(),
    'selecting the caller must retain the full-scope verdicts by stable ID',
  )

  const rerun = await run('single', './caller', { id: boundary.id })
  assert.deepEqual(
    rerun?.killed?.map((mutant) => mutant.id),
    [boundary.id],
  )
  assert.deepEqual(rerun?.escaped ?? [], [])

  const buildFailure = await run('build-failure', './buildfailure', { target: 'buildfailure' })
  assert.ok(
    buildFailure?.skipped?.some((mutant) => mutant.diff.includes('1 / 2')),
    'an unbuildable arithmetic mutant must be skipped, never counted as killed',
  )
  await run('broken-baseline', './caller', { brokenBaseline: true })
  assert.equal(await Bun.file(log).exists(), false, 'unselected tests must never execute')
  console.log('ok  selected caller, stable ID, build failure, baseline, and unselected tests')
} finally {
  await rm(temporary, { recursive: true, force: true })
}
