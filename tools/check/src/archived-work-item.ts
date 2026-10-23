/**
 * `work-items/done/` の記録は、状態と題名と要約だけに縮めてある。縮めた記録は形だけを検査する。
 * 過去の記録が持つ `archived_from` は互換性のため受理する。
 */

const FIELDS = new Set(['id', 'title', 'status', 'archived_from'])

export function isArchived(path: string): boolean {
  return path.startsWith('work-items/done/')
}

/** 縮めた記録の形の所見。`id` と `title` はファイル名と見出しから解析器が補う。 */
export function verifyArchivedShape(data: Record<string, unknown>): string[] {
  const findings: string[] = []
  if (data.status !== 'completed' && data.status !== 'cancelled')
    findings.push('archived record must be completed or cancelled')
  if (
    data.archived_from !== undefined &&
    (typeof data.archived_from !== 'string' || !/^[0-9a-f]{7,40}$/.test(data.archived_from))
  )
    findings.push('archived_from must be a commit id')
  if (typeof data.title !== 'string' || data.title === '')
    findings.push('archived record must keep its title')
  const extra = Object.keys(data).filter((key) => !FIELDS.has(key))
  if (extra.length > 0)
    findings.push(`archived record keeps only status and archived_from (found ${extra.join(', ')})`)
  return findings
}

/**
 * `archived_from` のコミットに記録の全文があるか。一回の `git cat-file` で確かめる。
 * Git の外や履歴のないクローンでは確かめられないので、空の結果を返す。
 */
export function missingArchivedRevisions(
  root: string,
  records: readonly { path: string; revision: string }[],
): string[] {
  if (records.length === 0) return []
  const result = Bun.spawnSync(['git', 'cat-file', '--batch-check'], {
    cwd: root,
    stdin: new TextEncoder().encode(
      records.map((record) => `${record.revision}:${record.path}\n`).join(''),
    ),
  })
  if (result.exitCode !== 0) return []
  const answers = result.stdout.toString().trimEnd().split('\n')
  return records.flatMap((record, index) =>
    answers[index]?.endsWith(' missing')
      ? [`${record.path}: archived_from ${record.revision} does not hold this record`]
      : [],
  )
}
