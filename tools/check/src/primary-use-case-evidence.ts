export type PrimaryUseCaseEnvironment = {
  /** リポジトリ相対パスの本文。存在しない場合は undefined。 */
  read: (path: string) => string | undefined
  /** 標準の verify または CI から到達する mise タスク。 */
  requiredTasks: ReadonlySet<string>
  /** 完了の変更が触れたリポジトリ相対パス。Git の外の workspace では undefined。 */
  changedPaths: ReadonlySet<string> | undefined
}

type RecordValue = Record<string, unknown>

function object(value: unknown): RecordValue | undefined {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as RecordValue)
    : undefined
}

function objects(value: unknown): RecordValue[] {
  return Array.isArray(value) ? value.map(object).filter((item) => item !== undefined) : []
}

function text(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

function applicable(record: RecordValue): boolean {
  if (record.change_kind === 'feature' || record.change_kind === 'bugfix') return true
  if (objects(record.primary_use_cases).length > 0) return true
  return objects(record.affected_spec).some((reference) =>
    text(reference.path)?.endsWith('/standards.md'),
  )
}

const TEST_PATH = /(?:_test\.go|\.(?:test|spec)\.[cm]?[jt]sx?)$|(?:^|\/)(?:testdata|tests)\//

/**
 * 完了の変更が振る舞いを保つか。仕様影響のない Markdown だけの変更と、テストを変えない
 * refactor が該当する。作業項目の自己申告ではなく差分で判定するので、文書変更と名乗っても
 * コードを変えた変更は該当しない。refactor でテストを変えた変更は、振る舞いも変えている。
 */
function preservesBehavior(
  record: RecordValue,
  changedPaths: ReadonlySet<string> | undefined,
): boolean {
  if (changedPaths === undefined || object(record.spec_impact) === undefined) return false
  const paths = [...changedPaths]
  if (paths.every((path) => path.endsWith('.md'))) return true
  return record.change_kind === 'refactor' && !paths.some((path) => TEST_PATH.test(path))
}

function affectedRequirements(record: RecordValue): Set<string> {
  return new Set(
    objects(record.affected_spec)
      .map((reference) => text(reference.requirement))
      .filter((value) => value !== undefined),
  )
}

function verifyTestReference(
  useCaseId: string,
  role: string,
  value: unknown,
  requirement: string,
  environment: PrimaryUseCaseEnvironment,
): string[] {
  const reference = object(value)
  if (!reference) return [`primary_use_cases ${useCaseId} has no ${role} test reference`]
  const path = text(reference.path)
  const name = text(reference.name)
  const task = text(reference.task)
  const findings: string[] = []

  if (!path) findings.push(`primary_use_cases ${useCaseId} has no ${role} test path`)
  if (!name) findings.push(`primary_use_cases ${useCaseId} has no ${role} test name`)
  if (!task) findings.push(`primary_use_cases ${useCaseId} has no ${role} test task`)
  if (task && !environment.requiredTasks.has(task)) {
    findings.push(
      `primary_use_cases ${useCaseId} ${role} test task is not required by verify or CI: ${task}`,
    )
  }

  if (!path) return findings
  const source = environment.read(path)
  if (source === undefined) {
    findings.push(`primary_use_cases ${useCaseId} ${role} test path does not exist: ${path}`)
    return findings
  }
  if (name && !source.includes(name)) {
    findings.push(`primary_use_cases ${useCaseId} ${role} test name not found in ${path}`)
  }
  if (!source.includes(requirement)) {
    findings.push(`primary_use_cases ${useCaseId} requirement ${requirement} not found in ${path}`)
  }
  return findings
}

function verifyCompletionEvidence(
  plans: RecordValue[],
  completion: RecordValue | undefined,
): string[] {
  const rawEvidence = completion?.primary_use_case_evidence
  const evidence = objects(rawEvidence)
  if (!Array.isArray(rawEvidence) || evidence.length === 0) {
    return ['completion.primary_use_case_evidence is required for applicable work']
  }

  const findings: string[] = []
  const byId = new Map<string, RecordValue>()
  for (const result of evidence) {
    const id = text(result.id)
    if (!id) {
      findings.push('primary use case completion evidence has no id')
      continue
    }
    if (byId.has(id)) findings.push(`duplicate primary use case completion evidence id: ${id}`)
    byId.set(id, result)
  }

  const requiredResults: Array<[keyof RecordValue, string]> = [
    ['red', 'RED'],
    ['fault_injection', 'fault-injection'],
  ]
  const planIds = new Set<string>()
  for (const plan of plans) {
    const id = text(plan.id)
    if (!id) continue
    planIds.add(id)
    const result = byId.get(id)
    if (!result) {
      findings.push(`primary use case ${id} has no completion evidence`)
      continue
    }
    for (const [key, label] of requiredResults) {
      if (!text(result[key])) findings.push(`primary use case ${id} has no ${label} result`)
    }
  }
  for (const id of byId.keys()) {
    if (!planIds.has(id)) findings.push(`primary use case completion evidence has no plan: ${id}`)
  }
  return findings
}

/**
 * スキーマ検証済みの work item に、主要ユースケースの証拠の契約を適用する。
 * ファイルシステムとタスク探索は環境へ追い出し、この関数自身は判断だけを行う。
 */
export function verifyPrimaryUseCaseEvidence(
  record: RecordValue,
  environment: PrimaryUseCaseEnvironment,
): string[] {
  if (record.status !== 'in_progress' && record.status !== 'completed') return []

  const completion = object(record.completion)
  if (!applicable(record)) {
    if (record.status !== 'completed') return []
    // 既存の検査が変更前から成功する改善に、記録のための失敗を作らせない。
    if (preservesBehavior(record, environment.changedPaths)) return []
    const findings: string[] = []
    if (!object(completion?.acceptance_red_evidence)) {
      findings.push('completion.acceptance_red_evidence is required for non-applicable work')
    }
    if (!object(completion?.unit_red_evidence)) {
      findings.push('completion.unit_red_evidence is required for non-applicable work')
    }
    return findings
  }

  const rawPlans = record.primary_use_cases
  const plans = objects(rawPlans)
  if (!Array.isArray(rawPlans) || plans.length === 0) {
    return ['primary_use_cases is required for feature, bugfix, and standards work']
  }

  const findings: string[] = []
  const requirements = affectedRequirements(record)
  const seen = new Set<string>()
  for (const plan of plans) {
    const id = text(plan.id)
    if (!id) {
      findings.push('primary_use_cases entry has no id')
      continue
    }
    if (seen.has(id)) findings.push(`duplicate primary_use_cases id: ${id}`)
    seen.add(id)

    const requirement = text(plan.requirement)
    if (!requirement) {
      findings.push(`primary_use_cases ${id} has no requirement`)
      continue
    }
    if (!requirements.has(requirement)) {
      findings.push(
        `primary_use_cases ${id} requirement is not declared in affected_spec: ${requirement}`,
      )
    }
    if (plan.boundary === 'e2e' && !text(plan.reason)?.trim())
      findings.push(`primary_use_cases ${id} E2E requires a narrower-boundary reason`)
    if (record.status === 'completed')
      findings.push(
        ...verifyTestReference(id, String(plan.boundary), plan.test, requirement, environment),
      )
  }

  if (record.status === 'completed') {
    findings.push(...verifyCompletionEvidence(plans, completion))
  }
  return findings
}
