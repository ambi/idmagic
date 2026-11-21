import { dirname, join, normalize } from 'node:path'
import type { UiCatalog } from './ui-pages.ts'

/**
 * 実装の一つのフォーム部品。name は静的な name 属性の値。
 * id は DOM の識別子として項目名と別に命名されているため、照合に使わない。
 */
export type UiControl = { file: string; line: number; name?: string }

const controlTags =
  '^(input|select|textarea|Input|Select|Textarea|Checkbox|Switch|Field|RadioGroup|NativeSelect)$'
const controlRule = `
id: control
language: tsx
rule:
  any: [{ kind: jsx_opening_element }, { kind: jsx_self_closing_element }]
  has: { field: name, regex: ${controlTags} }
---
id: identifier
language: tsx
rule:
  kind: string_fragment
  inside:
    kind: string
    inside:
      kind: jsx_attribute
      has: { kind: property_identifier, regex: ^name$ }
      inside:
        any: [{ kind: jsx_opening_element }, { kind: jsx_self_closing_element }]
        has: { field: name, regex: ${controlTags} }
`

type Match = {
  ruleId: string
  file: string
  text: string
  range: { byteOffset: { start: number; end: number }; start: { line: number } }
}

const importRule = `
id: import
language: tsx
rule:
  kind: string_fragment
  inside:
    kind: string
    inside:
      any: [{ kind: import_statement }, { kind: export_statement }]
`

/** 静的な import をたどり、ページを構成する経路と機能スライスのファイルを集める。 */
export function reachableFiles(
  entries: readonly string[],
  cwd: string,
  exists: (path: string) => boolean,
): Map<string, string[]> {
  const imports = new Map<string, string[]>()
  let pending = [...new Set(entries)]
  while (pending.length) {
    const result = Bun.spawnSync(
      ['ast-grep', 'scan', '--json=compact', '--inline-rules', importRule, ...pending],
      { cwd },
    )
    if (result.exitCode !== 0) throw new Error(result.stderr.toString())
    for (const file of pending) imports.set(file, [])
    for (const match of JSON.parse(result.stdout.toString() || '[]') as Match[]) {
      if (!match.text.startsWith('.')) continue
      const base = normalize(join(dirname(match.file), match.text))
      // 共通部品はページの入力を持たないため、経路と機能スライスだけをたどる。
      if (!/^frontend\/src\/(routes|features)\//.test(base)) continue
      const target = [base, `${base}.tsx`, `${base}.ts`, `${base}/index.tsx`].find(
        (candidate) => /\.tsx?$/.test(candidate) && exists(candidate),
      )
      if (target && !/\.(test|i18n)\.tsx?$/.test(target)) imports.get(match.file)!.push(target)
    }
    pending = [...new Set([...imports.values()].flat())].filter((file) => !imports.has(file))
  }
  const closure = new Map<string, string[]>()
  for (const entry of entries) {
    const seen = new Set<string>()
    const stack = [entry]
    while (stack.length) {
      const file = stack.pop()!
      if (seen.has(file)) continue
      seen.add(file)
      stack.push(...(imports.get(file) ?? []))
    }
    closure.set(entry, [...seen].sort())
  }
  return closure
}

/** ファイルごとのフォーム部品。識別子を式で決める部品は name を持たない。 */
export function formControls(files: readonly string[], cwd: string): UiControl[] {
  if (!files.length) return []
  const result = Bun.spawnSync(
    ['ast-grep', 'scan', '--json=compact', '--inline-rules', controlRule, ...files],
    { cwd },
  )
  if (result.exitCode !== 0) throw new Error(result.stderr.toString())
  const matches: Match[] = JSON.parse(result.stdout.toString() || '[]')
  const identifiers = matches.filter((match) => match.ruleId === 'identifier')
  return matches
    .filter((match) => match.ruleId === 'control')
    .map((control) => {
      const inside = identifiers.filter(
        (one) =>
          one.file === control.file &&
          one.range.byteOffset.start >= control.range.byteOffset.start &&
          one.range.byteOffset.end <= control.range.byteOffset.end,
      )
      return { file: control.file, line: control.range.start.line + 1, name: inside[0]?.text }
    })
}

/**
 * 宣言した入力と実装の部品を照合する。
 * 一つの部品を複数のページが描画しうるため、部品のファイルへ到達するページのどれかが
 * その名前を宣言していれば受け入れる。どのページも宣言していない名前だけを拒否する。
 * 静的な name のない部品は照合できないため、ページとファイルごとの件数で返す。
 */
export function validateInputs(
  catalog: UiCatalog,
  reach: ReadonlyMap<string, readonly string[]>,
  controls: readonly UiControl[],
): { findings: string[]; unidentified: Map<string, Record<string, number>> } {
  const findings: string[] = []
  const unidentified = new Map<string, Record<string, number>>()
  const pages = catalog.pages.filter((page) => reach.has(page.route))
  const declared = new Map(
    pages.map((page) => [
      page.id,
      new Set(
        [...page.parameters, ...page.forms.flatMap((form) => form.fields)].map(
          (field) => field.name,
        ),
      ),
    ]),
  )
  for (const control of controls) {
    const owners = pages.filter((page) => reach.get(page.route)!.includes(control.file))
    if (!control.name) {
      for (const page of owners) {
        const counts = unidentified.get(page.id) ?? {}
        counts[control.file] = (counts[control.file] ?? 0) + 1
        unidentified.set(page.id, counts)
      }
    } else if (owners.length && !owners.some((page) => declared.get(page.id)!.has(control.name!)))
      findings.push(
        `実装の入力を宣言したページがない: ${control.name} (${control.file}:${control.line})。候補: ${owners.map((page) => page.id).join(', ')}`,
      )
  }
  return { findings, unidentified }
}
