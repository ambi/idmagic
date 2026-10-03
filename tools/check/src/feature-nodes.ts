/**
 * 新しい形式の機能ノードで、ファイルをまたいで成り立つべきことを確かめる。
 *
 * 要件は機能仕様（`README.md` と章）で宣言し、例は同じ機能ノードの `examples.feature.md` に置く。
 * 一つのファイルだけを読む検証では、付録が宣言のない要件を参照していることが分からない。
 * ここは機能ノードごとに宣言と参照を集めてから照合する。例は任意なので、例のない要件は拒否しない。
 */

import type { DocumentSetView } from '../../workspace/src/document-layout.ts'
import type { SpecificationValidation } from './specification-doc.ts'

/** 旧形式のまま残る Context の一覧。減る方向にしか変えない。 */
export const LEGACY_SPEC_LAYOUT = 'tools/check/legacy-spec-layout.json'

type Declaration = { id: string; title: string; where: string }
type Reference = { id: string; title: string; where: string }

export class FeatureNodeDeclarations {
  private readonly declarations = new Map<string, Declaration[]>()
  private readonly references = new Map<string, Reference[]>()

  add(path: string, result: SpecificationValidation): void {
    const directory = path.slice(0, path.lastIndexOf('/'))
    if (path.endsWith('/examples.feature.md')) {
      const references = this.references.get(directory) ?? []
      references.push(
        ...result.ruleReferences.map((reference) => ({
          id: reference.id,
          title: reference.title,
          where: `${path}:${reference.line}`,
        })),
      )
      this.references.set(directory, references)
      return
    }
    // 旧形式の規則は Gherkin の中で宣言と例がそろうので、ここで照合するのは見出しの宣言だけである。
    if (path.endsWith('/scenarios.feature.md')) return
    const declared = result.scenarioIds.filter((scenario) => scenario.title !== undefined)
    if (declared.length === 0) return
    const declarations = this.declarations.get(directory) ?? []
    declarations.push(
      ...declared.map((scenario) => ({
        id: scenario.id,
        title: scenario.title ?? '',
        where: `${path}:${scenario.line}`,
      })),
    )
    this.declarations.set(directory, declarations)
  }

  verify(view: DocumentSetView): string[] {
    const findings: string[] = []
    for (const [directory, declarations] of this.declarations) {
      const placement = placementProblem(directory, view)
      for (const declaration of declarations) {
        if (placement) findings.push(`${declaration.where}: ${declaration.id} ${placement}`)
      }
    }
    for (const [directory, references] of this.references) {
      const declared = new Map(
        (this.declarations.get(directory) ?? []).map((declaration) => [
          declaration.id,
          declaration,
        ]),
      )
      for (const reference of references) {
        const declaration = declared.get(reference.id)
        if (!declaration) {
          findings.push(
            `${reference.where}: ${reference.id} is not declared by the feature specification in ${directory}/`,
          )
        } else if (declaration.title !== reference.title) {
          findings.push(
            `${reference.where}: ${reference.id} title differs from the feature specification (${declaration.where})`,
          )
        }
      }
    }
    return findings.sort()
  }
}

/** 要件を宣言できない階層なら、その理由を返す。 */
function placementProblem(directory: string, view: DocumentSetView): string | undefined {
  const [, , context = '', ...rest] = directory.split('/')
  if (!view.featureContexts.has(context)) return 'must be declared in scenarios.feature.md'
  if (rest.length === 0 || rest[0] === 'design' || view.parents.has(directory)) {
    return 'must be declared in a feature node'
  }
  return undefined
}

/**
 * 旧形式の一覧と、実際の形式が食い違っていないことを確かめる。一覧が無い作業ツリーでは
 * 何も求めない。新しい Context を旧形式で作ることは、一覧へ載せない限り拒否される。
 */
export function verifyLegacyLayoutList(
  listed: readonly string[] | undefined,
  contexts: readonly string[],
  view: DocumentSetView,
): string[] {
  if (listed === undefined) return []
  const legacy = new Set(listed)
  const findings: string[] = []
  for (const context of contexts) {
    const feature = view.featureContexts.has(context)
    if (feature && legacy.has(context)) {
      findings.push(
        `fail  ${LEGACY_SPEC_LAYOUT}: ${context} uses the feature layout now; remove it from the list`,
      )
    } else if (!feature && !legacy.has(context)) {
      findings.push(
        `fail  docs/domain/${context}: ${context} has no design/README.md; a context outside ${LEGACY_SPEC_LAYOUT} must use the feature layout`,
      )
    }
  }
  const existing = new Set(contexts)
  for (const context of legacy) {
    if (!existing.has(context)) {
      findings.push(
        `fail  ${LEGACY_SPEC_LAYOUT}: ${context} is no longer a context; remove it from the list`,
      )
    }
  }
  return findings
}
