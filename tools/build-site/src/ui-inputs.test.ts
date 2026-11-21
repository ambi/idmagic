import { afterAll, describe, expect, it } from 'bun:test'
import { existsSync } from 'node:fs'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { formControls, reachableFiles, validateInputs } from './ui-inputs.ts'
import type { UiCatalog } from './ui-pages.ts'

const root = await mkdtemp(join(tmpdir(), 'ui-inputs-'))
afterAll(() => rm(root, { recursive: true, force: true }))

async function write(path: string, source: string) {
  await mkdir(dirname(join(root, path)), { recursive: true })
  await writeFile(join(root, path), source)
}

await write(
  'frontend/src/routes/edit.tsx',
  `import { EditPage } from '../features/edit/EditPage'
   import { Shell } from '../components/Shell'
   export const Route = createFileRoute('/edit')({ component: EditPage })`,
)
await write(
  'frontend/src/routes/view.tsx',
  `import { Dialogs } from '../features/edit/Dialogs'
   export const Route = createFileRoute('/view')({ component: Dialogs })`,
)
await write(
  'frontend/src/features/edit/EditPage.tsx',
  `import { Dialogs } from './Dialogs'
   import labels from './EditPage.i18n'
   export function EditPage() {
     return <form>
       <input name="display_name" />
       <Select value={v} onValueChange={set} />
       <Input id="dom-only" />
     </form>
   }`,
)
await write('frontend/src/features/edit/EditPage.i18n.ts', 'export default {}')
await write(
  'frontend/src/features/edit/Dialogs.tsx',
  'export function Dialogs() { return <input name="confirm_name" /> }',
)
await write(
  'frontend/src/components/Shell.tsx',
  'export function Shell() { return <input name="global_search" /> }',
)

const field = (name: string) => ({
  id: name,
  name,
  label: name,
  control: 'text',
  initial: '空',
  condition: '常時',
  type: 'string',
})
const page = (id: string, route: string, names: string[]) => ({
  id,
  route,
  path: route,
  parameters: [],
  forms: [
    { id: 'form', name: 'form', condition: '常時', commit: '保存', fields: names.map(field) },
  ],
})
const catalogOf = (...pages: ReturnType<typeof page>[]) => ({ pages }) as unknown as UiCatalog
const exists = (path: string) => existsSync(join(root, path))

describe('実装の入力との照合', () => {
  it('経路から機能スライスの import をたどり、共通部品と辞書は含めない', () => {
    const closure = reachableFiles(['frontend/src/routes/edit.tsx'], root, exists)
    expect(closure.get('frontend/src/routes/edit.tsx')).toEqual([
      'frontend/src/features/edit/Dialogs.tsx',
      'frontend/src/features/edit/EditPage.tsx',
      'frontend/src/routes/edit.tsx',
    ])
  })

  it('静的な name だけを部品の識別子にし、DOM の id は使わない', () => {
    expect(formControls(['frontend/src/features/edit/EditPage.tsx'], root)).toEqual([
      { file: 'frontend/src/features/edit/EditPage.tsx', line: 5, name: 'display_name' },
      { file: 'frontend/src/features/edit/EditPage.tsx', line: 6, name: undefined },
      { file: 'frontend/src/features/edit/EditPage.tsx', line: 7, name: undefined },
    ])
  })

  const closure = reachableFiles(
    ['frontend/src/routes/edit.tsx', 'frontend/src/routes/view.tsx'],
    root,
    exists,
  )
  const reach = new Map([
    ['/edit', closure.get('frontend/src/routes/edit.tsx')!],
    ['/view', closure.get('frontend/src/routes/view.tsx')!],
  ])
  const controls = formControls([...new Set([...closure.values()].flat())], root)

  it('どのページも宣言していない実装の入力を拒否する', () => {
    const result = validateInputs(
      catalogOf(page('edit', '/edit', ['confirm_name']), page('view', '/view', [])),
      reach,
      controls,
    )
    expect(result.findings).toEqual([expect.stringContaining('display_name')])
  })

  it('共有する部品の入力は、到達するページのどれかが宣言していれば受け入れる', () => {
    const result = validateInputs(
      catalogOf(page('edit', '/edit', ['display_name', 'confirm_name']), page('view', '/view', [])),
      reach,
      controls,
    )
    expect(result.findings).toEqual([])
    expect(result.unidentified.get('edit')).toEqual({
      'frontend/src/features/edit/EditPage.tsx': 2,
    })
    expect(result.unidentified.has('view')).toBe(false)
  })
})
