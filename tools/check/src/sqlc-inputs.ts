import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

/** sqlc の生成入力を宣言する設定。クエリのファイルの集合はこの設定だけが決める。 */
export const SQLC_CONFIG_PATH = 'sqlc.yaml'

export type SqlcQueryInput = {
  schema: string
  packagePath: string
  files: string[]
}

/** `sqlc.yaml` の各入力を、クエリのファイルと、それを置いたパッケージにする。 */
export async function sqlcQueryInputs(snapshot: WorkspaceSnapshot): Promise<SqlcQueryInput[]> {
  const config = Bun.YAML.parse(await snapshot.read(SQLC_CONFIG_PATH)) as {
    sql?: { schema?: unknown; queries?: unknown }[]
  }
  const inputs: SqlcQueryInput[] = []
  for (const entry of config.sql ?? []) {
    if (typeof entry.schema !== 'string' || typeof entry.queries !== 'string') {
      throw new Error(
        `${SQLC_CONFIG_PATH}: every sql entry needs one schema path and one queries path`,
      )
    }
    const queries = entry.queries.replace(/\/+$/, '')
    const files = queries.endsWith('.sql')
      ? [queries]
      : (await snapshot.files(queries)).filter((path) => path.endsWith('.sql'))
    inputs.push({
      schema: entry.schema,
      packagePath: queries.endsWith('.sql') ? queries.split('/').slice(0, -1).join('/') : queries,
      files,
    })
  }
  return inputs
}
