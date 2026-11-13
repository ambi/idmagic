import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { isPortalApiCallTarget, unmarkedPortalApiFetches } from './portal-api-calls.ts'
import type { CheckOutcome } from './runner.ts'

const MESSAGES = {
  unmarked:
    'call the admin or account API through the shared request path so the bearer token is attached, or state why the call must use the session cookie with // session-cookie: <reason>',
  'missing-reason': '// session-cookie: needs a reason',
} as const

export async function checkPortalApiCalls(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const files = (await snapshot.files('frontend/src', ['node_modules'])).filter(
    isPortalApiCallTarget,
  )
  const lines: string[] = []
  let exceptions = 0
  for (const path of files) {
    const source = await snapshot.read(path)
    exceptions += source.match(/^\s*\/\/\s*session-cookie:/gm)?.length ?? 0
    for (const call of unmarkedPortalApiFetches(source)) {
      lines.push(`fail  ${path}:${call.line}: ${MESSAGES[call.problem]}`)
    }
  }
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0
        ? lines
        : [
            `ok  ${files.length} frontend file(s) call portal APIs through the shared request path, ${exceptions} session-cookie exception(s)`,
          ],
  }
}
