/**
 * 機能スライスで、ファイルをまたいで成り立つべきことを確かめる。
 *
 * 要件は機能仕様（`README.md` と章）で宣言し、例は同じ機能スライスの `acceptance.feature.md` に置く。
 * 一つのファイルだけを読む検証では、付録が宣言のない要件を参照していることが分からない。
 * ここは機能スライスごとに宣言と参照を集めてから照合する。例は任意なので、例のない要件は拒否しない。
 */

import type { DocumentSetView } from '../../workspace/src/document-layout.ts'
import type { SpecificationValidation } from './specification-doc.ts'

type Declaration = { id: string; title: string; where: string }
type Reference = { id: string; title: string; where: string }

export class FeatureSliceDeclarations {
  private readonly declarations = new Map<string, Declaration[]>()
  private readonly references = new Map<string, Reference[]>()

  add(path: string, result: SpecificationValidation): void {
    const directory = path.slice(0, path.lastIndexOf('/'))
    if (path.endsWith('/acceptance.feature.md')) {
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
  const [, , , ...rest] = directory.split('/')
  if (rest.length === 0 || rest[0] === 'design' || view.parents.has(directory)) {
    return 'must be declared in a feature slice'
  }
  return undefined
}

/**
 * すべての Context が印（`design/README.md`）を持つことを確かめる。印のない Context の段には
 * どの文書も置けず、文書ごとの拒否だけでは原因が伝わらないので、Context ごとに一件報告する。
 */
export function verifyFeatureLayout(contexts: readonly string[], view: DocumentSetView): string[] {
  return contexts
    .filter((context) => !view.featureContexts.has(context))
    .map(
      (context) =>
        `fail  docs/domain/${context}: ${context} has no design/README.md; every context must use the feature layout`,
    )
}
