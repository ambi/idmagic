import { describe, expect, it } from 'bun:test'
import {
  type PrimaryUseCaseEnvironment,
  verifyPrimaryUseCaseEvidence,
} from './primary-use-case-evidence.ts'

const requirement = 'REQ-DEMO-001'
const unitPath = 'backend/demo/usecases/demo_test.go'
const e2ePath = 'backend/demo/e2e_test.go'
const files: Record<string, string> = {
  [unitPath]: `//spec:covers ${requirement}: 保存した結果を読み戻す。\nfunc TestDemoRule_REQ_DEMO_001(t *testing.T) {}`,
  [e2ePath]: `//spec:covers ${requirement}: 正式な入口から結果を読み戻す。\nfunc TestE2E_Demo_REQ_DEMO_001(t *testing.T) {}`,
}
const environment: PrimaryUseCaseEnvironment = {
  read: (path) => files[path],
  requiredTasks: new Set(['test-go-race']),
  executed: () => true,
}

const plan = {
  id: 'demo-success',
  requirement,
  observable_result: '保存された結果を読み戻せる。',
  boundary: 'acceptance',
  test: { path: unitPath, name: 'TestDemoRule_REQ_DEMO_001', task: 'test-go-race' },
  fault_model: '保存を行わない。',
  setup: {
    state: '保存対象がある。',
    entry_points: '公開された保存操作を使う。',
    fixtures: '外部の保存先はメモリの実装に置き換える。',
  },
}

const applicable = {
  status: 'in_progress',
  change_kind: 'feature',
  affected_spec: [{ path: 'docs/modules/demo/features/run.md', requirement }],
}

describe('verifyPrimaryUseCaseEvidence', () => {
  it('準備方法を記した計画を受理し、完了時にテストの参照と実行を確認する', () => {
    const record = { ...applicable, primary_use_cases: [plan] }
    expect(verifyPrimaryUseCaseEvidence(record, environment)).toEqual([])
    const completed = { ...record, status: 'completed' }
    expect(verifyPrimaryUseCaseEvidence(completed, environment)).toEqual([])
  })

  it('requires a plan for feature, bugfix, and standards work after implementation starts', () => {
    for (const record of [
      applicable,
      { ...applicable, change_kind: 'bugfix' },
      {
        ...applicable,
        change_kind: 'tooling',
        affected_spec: [{ path: 'docs/modules/demo/standards.md', requirement: 'RFC-DEMO' }],
      },
    ]) {
      expect(verifyPrimaryUseCaseEvidence(record, environment)).toContain(
        'primary_use_cases is required for feature, bugfix, and standards work',
      )
    }
  })

  it('applies the contract when tooling work declares primary use cases', () => {
    const tooling = {
      ...applicable,
      change_kind: 'tooling',
      primary_use_cases: [{ ...plan, boundary: 'e2e' }],
    }
    expect(verifyPrimaryUseCaseEvidence(tooling, environment)).toContain(
      'primary_use_cases demo-success E2E requires a narrower-boundary reason',
    )
  })

  it('does not impose the plan on pending work', () => {
    expect(verifyPrimaryUseCaseEvidence({ ...applicable, status: 'pending' }, environment)).toEqual(
      [],
    )
  })

  it('accepts a complete in-progress plan before the planned tests exist', () => {
    const noFiles = { ...environment, read: () => undefined }
    expect(
      verifyPrimaryUseCaseEvidence({ ...applicable, primary_use_cases: [plan] }, noFiles),
    ).toEqual([])
  })

  it('requires the plan requirement to be one of affected_spec', () => {
    const findings = verifyPrimaryUseCaseEvidence(
      {
        ...applicable,
        primary_use_cases: [{ ...plan, requirement: 'REQ-DEMO-002' }],
      },
      environment,
    )
    expect(findings).toContain(
      'primary_use_cases demo-success requirement is not declared in affected_spec: REQ-DEMO-002',
    )
  })

  it('requires a reason why a narrower boundary cannot detect an E2E fault', () => {
    const e2e = { ...plan, boundary: 'e2e' }
    expect(
      verifyPrimaryUseCaseEvidence({ ...applicable, primary_use_cases: [e2e] }, environment),
    ).toContain('primary_use_cases demo-success E2E requires a narrower-boundary reason')
    expect(
      verifyPrimaryUseCaseEvidence(
        { ...applicable, primary_use_cases: [{ ...e2e, reason: '配線は E2E でしか通らない。' }] },
        environment,
      ),
    ).toEqual([])
  })

  it('checks completed test existence, identifier, requirement, and required-task reachability', () => {
    const brokenEnvironment: PrimaryUseCaseEnvironment = {
      read: (path) => (path === unitPath ? 'func TestSomethingElse(t *testing.T) {}' : undefined),
      requiredTasks: new Set(),
    }
    const wired = {
      ...plan,
      id: 'demo-wired',
      boundary: 'e2e',
      reason: '配線は E2E でしか通らない。',
      test: { path: e2ePath, name: 'TestE2E_Demo_REQ_DEMO_001', task: 'test-go-race' },
    }
    const findings = verifyPrimaryUseCaseEvidence(
      {
        ...applicable,
        status: 'completed',
        primary_use_cases: [plan, wired],
      },
      brokenEnvironment,
    )
    expect(findings).toContain(
      `primary_use_cases demo-success acceptance test name not found in ${unitPath}`,
    )
    expect(findings).toContain(
      `primary_use_cases demo-success requirement ${requirement} not found in ${unitPath}`,
    )
    expect(findings).toContain(
      `primary_use_cases demo-wired e2e test path does not exist: ${e2ePath}`,
    )
    expect(findings).toContain(
      'primary_use_cases demo-success acceptance test task is not required by verify or CI: test-go-race',
    )
  })

  it('detects the pre-fix evidence shapes from wi-439, wi-440, and wi-441 without feature-specific rules', () => {
    const preFixRecords = [
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          {
            path: 'docs/modules/provisioning/standards.md',
            requirement: 'RFC7643-OUT-CORE-RESOURCES',
          },
        ],
      },
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          {
            path: 'docs/modules/provisioning/standards.md',
            requirement: 'RFC7644-OUT-AUTHENTICATION',
          },
        ],
      },
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          {
            path: 'docs/modules/provisioning/features/task.md',
            requirement: 'REQ-PROVISIONING-013',
          },
        ],
      },
    ]
    for (const record of preFixRecords) {
      expect(verifyPrimaryUseCaseEvidence(record, environment)).toEqual([
        'primary_use_cases is required for feature, bugfix, and standards work',
      ])
    }
  })

  it('コメント内のテスト名や別のテストの規範引用を、選んだ Go テストの証拠に数えない', () => {
    const selected = 'TestSelected'
    const source = `// ${selected}\n//spec:covers ${requirement}: 別の処理を検査する。\nfunc TestOther(t *testing.T) {}\nfunc ${selected}(t *testing.T) {}`
    const findings = verifyPrimaryUseCaseEvidence(
      {
        ...applicable,
        status: 'completed',
        primary_use_cases: [{ ...plan, test: { ...plan.test, name: selected } }],
      },
      { ...environment, read: () => source },
    )
    expect(findings).toContain(
      `primary_use_cases demo-success acceptance test ${selected} does not cite ${requirement} in ${unitPath}`,
    )
  })

  it('準備の不足と、完了時の未実行を成功にしない', () => {
    const record = { ...applicable, primary_use_cases: [{ ...plan, setup: undefined }] }
    expect(verifyPrimaryUseCaseEvidence(record, environment)).toHaveLength(3)
    expect(
      verifyPrimaryUseCaseEvidence(
        { ...applicable, status: 'completed', primary_use_cases: [plan] },
        { ...environment, executed: () => false },
      ),
    ).toContain(
      `primary_use_cases demo-success acceptance test has no current passed execution: ${unitPath} ${plan.test.name}`,
    )
  })
})
