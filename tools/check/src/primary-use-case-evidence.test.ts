import { describe, expect, it } from 'bun:test'
import {
  type PrimaryUseCaseEnvironment,
  verifyPrimaryUseCaseEvidence,
} from './primary-use-case-evidence.ts'

const requirement = 'REQ-DEMO-001'
const unitPath = 'backend/demo/usecases/demo_test.go'
const e2ePath = 'backend/demo/e2e_test.go'
const files: Record<string, string> = {
  [unitPath]: `func TestDemoRule_REQ_DEMO_001(t *testing.T) { /* ${requirement} */ }`,
  [e2ePath]: `func TestE2E_Demo_REQ_DEMO_001(t *testing.T) { /* ${requirement} */ }`,
}
const environment: PrimaryUseCaseEnvironment = {
  read: (path) => files[path],
  requiredTasks: new Set(['test-go-race']),
}

const plan = {
  id: 'demo-success',
  requirement,
  observable_result: 'The caller observes the completed demo effect.',
  unit_test: { path: unitPath, name: 'TestDemoRule_REQ_DEMO_001', task: 'test-go-race' },
  e2e_test: { path: e2ePath, name: 'TestE2E_Demo_REQ_DEMO_001', task: 'test-go-race' },
  unit_fault_model: 'The use case skips the effect.',
  e2e_fault_model: 'The configured route is disconnected.',
}

const evidence = {
  id: plan.id,
  unit_red: 'The unit test failed because the effect was absent.',
  e2e_red: 'The E2E test failed because no final effect was observed.',
  unit_fault_injection: 'Skipping the effect made the unit test fail.',
  e2e_fault_injection: 'Disconnecting the route made the E2E test fail.',
}

const applicable = {
  status: 'in_progress',
  evidence_policy: 'risk-based-v3',
  change_kind: 'feature',
  affected_spec: [{ path: 'docs/contexts/demo/scenarios.md', requirement }],
}

describe('verifyPrimaryUseCaseEvidence', () => {
  it('requires a plan for feature, bugfix, and standards work after implementation starts', () => {
    for (const record of [
      applicable,
      { ...applicable, change_kind: 'bugfix' },
      {
        ...applicable,
        change_kind: 'tooling',
        affected_spec: [{ path: 'docs/contexts/demo/standards.md', requirement: 'RFC-DEMO' }],
      },
    ]) {
      expect(verifyPrimaryUseCaseEvidence(record, environment)).toContain(
        'primary_use_cases is required for feature, bugfix, and standards work',
      )
    }
  })

  it('does not impose the plan on pending work or completed legacy evidence', () => {
    expect(verifyPrimaryUseCaseEvidence({ ...applicable, status: 'pending' }, environment)).toEqual(
      [],
    )
    expect(
      verifyPrimaryUseCaseEvidence(
        { ...applicable, status: 'completed', evidence_policy: 'risk-based-v2' },
        environment,
      ),
    ).toEqual([])
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

  it('requires distinct Unit and E2E references', () => {
    const findings = verifyPrimaryUseCaseEvidence(
      {
        ...applicable,
        primary_use_cases: [{ ...plan, e2e_test: plan.unit_test }],
      },
      environment,
    )
    expect(findings).toContain(
      'primary_use_cases demo-success uses the same test for Unit and E2E evidence',
    )
  })

  it('checks completed test existence, identifier, requirement, and required-task reachability', () => {
    const brokenEnvironment: PrimaryUseCaseEnvironment = {
      read: (path) => (path === unitPath ? 'func TestSomethingElse(t *testing.T) {}' : undefined),
      requiredTasks: new Set(),
    }
    const findings = verifyPrimaryUseCaseEvidence(
      {
        ...applicable,
        status: 'completed',
        primary_use_cases: [plan],
        completion: { primary_use_case_evidence: [evidence] },
      },
      brokenEnvironment,
    )
    expect(findings).toContain(
      `primary_use_cases demo-success Unit test name not found in ${unitPath}`,
    )
    expect(findings).toContain(
      `primary_use_cases demo-success requirement ${requirement} not found in ${unitPath}`,
    )
    expect(findings).toContain(
      `primary_use_cases demo-success E2E test path does not exist: ${e2ePath}`,
    )
    expect(findings).toContain(
      'primary_use_cases demo-success Unit test task is not required by verify or CI: test-go-race',
    )
  })

  it('requires one complete evidence result for every planned use case', () => {
    const base = { ...applicable, status: 'completed', primary_use_cases: [plan] }
    expect(verifyPrimaryUseCaseEvidence(base, environment)).toContain(
      'completion.primary_use_case_evidence is required for applicable risk-based-v3 work',
    )
    const missingResults = {
      unit_red: 'Unit RED',
      e2e_red: 'E2E RED',
      unit_fault_injection: 'Unit fault-injection',
      e2e_fault_injection: 'E2E fault-injection',
    } as const
    for (const [field, label] of Object.entries(missingResults)) {
      expect(
        verifyPrimaryUseCaseEvidence(
          {
            ...base,
            completion: {
              primary_use_case_evidence: [{ ...evidence, [field]: '' }],
            },
          },
          environment,
        ),
      ).toContain(`primary use case demo-success has no ${label} result`)
    }
    expect(
      verifyPrimaryUseCaseEvidence(
        { ...base, completion: { primary_use_case_evidence: [evidence] } },
        environment,
      ),
    ).toEqual([])
  })

  it('keeps alternate Acceptance and Unit RED evidence for non-applicable v3 work', () => {
    const tooling = {
      status: 'completed',
      evidence_policy: 'risk-based-v3',
      change_kind: 'tooling',
      affected_spec: [],
      completion: {},
    }
    expect(verifyPrimaryUseCaseEvidence(tooling, environment)).toEqual([
      'completion.acceptance_red_evidence is required for non-applicable risk-based-v3 work',
      'completion.unit_red_evidence is required for non-applicable risk-based-v3 work',
    ])
  })

  it('detects the pre-fix evidence shapes from wi-439, wi-440, and wi-441 without feature-specific rules', () => {
    const preFixRecords = [
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          {
            path: 'docs/contexts/provisioning/standards.md',
            requirement: 'RFC7643-OUT-CORE-RESOURCES',
          },
        ],
      },
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          {
            path: 'docs/contexts/provisioning/standards.md',
            requirement: 'RFC7644-OUT-AUTHENTICATION',
          },
        ],
      },
      {
        ...applicable,
        change_kind: 'bugfix',
        affected_spec: [
          { path: 'docs/contexts/provisioning/scenarios.md', requirement: 'REQ-PROVISIONING-013' },
        ],
      },
    ]
    for (const record of preFixRecords) {
      expect(verifyPrimaryUseCaseEvidence(record, environment)).toEqual([
        'primary_use_cases is required for feature, bugfix, and standards work',
      ])
    }
  })
})
