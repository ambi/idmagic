import { appendFile } from 'node:fs/promises'
import { evaluateCIResults, planCIRevision } from './ci-plan.ts'

async function summary(lines: string[]) {
  if (process.env.GITHUB_STEP_SUMMARY) {
    await appendFile(process.env.GITHUB_STEP_SUMMARY, `${lines.join('\n')}\n`)
  }
}

const [command, base, head] = process.argv.slice(2)
if (command === 'plan' && process.argv.length <= 5) {
  const plan = planCIRevision(process.cwd(), base, head)
  const json = JSON.stringify(plan)
  console.log(json)
  if (process.env.GITHUB_OUTPUT) {
    const flags = Object.entries(plan.groups).map(
      ([group, decision]) => `${group}=${decision.selected}`,
    )
    await appendFile(process.env.GITHUB_OUTPUT, [`plan=${json}`, ...flags, ''].join('\n'))
  }
  await summary([
    '## CI verification plan',
    '',
    ...Object.entries(plan.groups).map(
      ([group, decision]) =>
        `- ${group}: ${decision.selected ? 'selected' : 'not selected'}; ${decision.reasons.join('; ')}`,
    ),
  ])
} else if (command === 'results' && process.argv.length === 3) {
  let needs: unknown
  try {
    needs = JSON.parse(process.env.CI_NEEDS ?? '')
  } catch {
    needs = null
  }
  const result = evaluateCIResults(needs)
  console.log(result.lines.join('\n'))
  await summary(['## CI verification results', '', ...result.lines.map((line) => `- ${line}`)])
  if (!result.success) process.exitCode = 1
} else {
  console.error('usage: ci-main.ts plan [base head] | results (CI_NEEDS JSON required)')
  process.exitCode = 2
}
