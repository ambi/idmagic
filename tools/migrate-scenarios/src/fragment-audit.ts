/**
 * 旧 `scenarios.md` から新しい `scenarios.feature.md` への一度きりの移行で、
 * 条件・結果・拒否時に防いだ効果のどれも落ちていないことを機械的に確かめる比較。
 *
 * 308 件の規則と 405 件の `ALT` を移すため、文書と検査が同時に新形式へ移ると
 * 断片の欠落に気付けない。旧文書から取り出した断片の一覧と、新文書を展開した
 * 本文とを別々に作って突き合わせる。
 */

import { AstBuilder, compile, GherkinInMarkdownTokenMatcher, Parser } from '@cucumber/gherkin'
import { IdGenerator } from '@cucumber/messages'

/** 空白の有無だけの差で移行の取りこぼしを報告しないよう、比較前に空白を落とす。 */
export function normalize(text: string): string {
  return text.replace(/\s+/g, '')
}

/** 旧文書が `ALT` と `THEN` で名指ししていたエラー型。 */
export function legacyErrorTypes(source: string): Set<string> {
  const types = new Set<string>()
  let inScenario = false
  for (const line of source.split('\n')) {
    if (/^### REQ-[A-Z0-9]+-\d+/.test(line)) {
      inScenario = true
      continue
    }
    if (!inScenario || (!/^\s+- ALT /.test(line) && !/^- THEN /.test(line))) continue
    for (const match of line.matchAll(/\b([A-Z][A-Za-z0-9]*Error)\b/g)) types.add(match[1] ?? '')
  }
  return types
}

/**
 * 旧文書の経路を構成する断片。`ALT` は条件と結果を `→` で直列化していたので、
 * 一行を断片へ割り、条件と結果のどちらが落ちても報告できるようにする。
 */
export function legacyFragments(source: string): string[] {
  const fragments: string[] = []
  for (const line of source.split('\n')) {
    const step = line.match(/^- (?:GIVEN|WHEN|THEN) (.+)$/)?.[1]
    if (step) fragments.push(step)
    const alternative = line.match(/^  - ALT (.+)$/)?.[1]
    if (alternative) fragments.push(...alternative.split(' → '))
  }
  return fragments
}

/**
 * `Scenario Outline` のステップは表の値を代入して初めて旧ステップの文になるため、
 * 生の本文だけでなく Pickle へ展開した各行のステップ本文も照合対象にする。
 */
export function migratedText(source: string): string {
  const parser = new Parser(
    new AstBuilder(IdGenerator.incrementing()),
    new GherkinInMarkdownTokenMatcher('en'),
  )
  const pickles = compile(parser.parse(source), 'scenarios.feature.md', IdGenerator.incrementing())
  return [source, ...pickles.flatMap((pickle) => pickle.steps.map((step) => step.text))]
    .map(normalize)
    .join('\n')
}

/** 新文書のどこにも残らなかった旧断片。空であることが移行の受け入れ条件になる。 */
export function missingFragments(oldSource: string, newSource: string): string[] {
  const migrated = migratedText(newSource)
  return legacyFragments(oldSource).filter((fragment) => !migrated.includes(normalize(fragment)))
}
