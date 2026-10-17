/**
 * 機能スライスで、ファイルをまたいで成り立つべきことを確かめる。
 *
 * 要件は機能仕様（`features/<feature>.md` と章）で宣言し、例は同じ機能の `<feature>.feature.md` に置く。
 * 一つのファイルだけを読む検証では、付録が宣言のない要件を参照していることが分からない。
 * ここは機能ごとに宣言と参照を集めてから照合する。例は任意なので、例のない要件は拒否しない。
 */

import {
  featureDocument,
  featureSpecificationPath,
  type DocumentSetView,
} from '../../workspace/src/document-layout.ts'
import type { SpecificationValidation } from './specification-doc.ts'

type Declaration = { id: string; title: string; where: string }
type Reference = { id: string; title: string; where: string }

export class FeatureSliceDeclarations {
  private readonly declarations = new Map<string, Declaration[]>()
  private readonly references = new Map<string, Reference[]>()
  private readonly misplaced: Declaration[] = []

  add(path: string, result: SpecificationValidation): void {
    const feature = featureDocument(path)
    if (feature?.part === 'examples') {
      const key = featureSpecificationPath(feature.module, feature.feature)
      const references = this.references.get(key) ?? []
      references.push(
        ...result.ruleReferences.map((reference) => ({
          id: reference.id,
          title: reference.title,
          where: `${path}:${reference.line}`,
        })),
      )
      this.references.set(key, references)
      return
    }
    const declared = result.scenarioIds
      .filter((scenario) => scenario.title !== undefined)
      .map((scenario) => ({
        id: scenario.id,
        title: scenario.title ?? '',
        where: `${path}:${scenario.line}`,
      }))
    if (declared.length === 0) return
    if (!feature) {
      this.misplaced.push(...declared)
      return
    }
    const key = featureSpecificationPath(feature.module, feature.feature)
    this.declarations.set(key, [...(this.declarations.get(key) ?? []), ...declared])
  }

  verify(): string[] {
    const findings = this.misplaced.map(
      (declaration) =>
        `${declaration.where}: ${declaration.id} must be declared in a feature slice`,
    )
    for (const [specification, references] of this.references) {
      const declared = new Map(
        (this.declarations.get(specification) ?? []).map((declaration) => [
          declaration.id,
          declaration,
        ]),
      )
      for (const reference of references) {
        const declaration = declared.get(reference.id)
        if (!declaration) {
          findings.push(
            `${reference.where}: ${reference.id} is not declared by the feature specification ${specification}`,
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

/**
 * すべてのモジュールが印（`README.md`）を持つことを確かめる。印のないモジュールの段には
 * どの文書も置けず、文書ごとの拒否だけでは原因が伝わらないので、モジュールごとに一件報告する。
 */
export function verifyFeatureLayout(modules: readonly string[], view: DocumentSetView): string[] {
  return modules
    .filter((module) => !view.featureModules.has(module))
    .map(
      (module) =>
        `fail  docs/modules/${module}: ${module} has no README.md; every module must use the feature layout`,
    )
}
