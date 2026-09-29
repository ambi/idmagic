/**
 * 規則一件の書式と、仕様の木の形を確かめる。
 *
 * どの関数もファイルを読まない。読み取りは `check-specification-rules.ts` が受け持ち、
 * ここは渡された文字列と一覧だけから判定する。
 */

import { ruleBodies, type RuleBody } from './gherkin-scenarios.ts'

export type Finding = { path: string; line: number; message: string }

/** `backend/` の Go で宣言された名前。`qualified` は `型.メソッド` と `パッケージ名.名前`。 */
export type GoDeclarations = { names: Set<string>; qualified: Set<string> }

/** `backend/<context>/<name>/` の直下に層のディレクトリを持つ機能スライス。 */
export type FeatureSlice = { context: string; name: string; path: string }

export type FeatureNodeDebt = {
  /** コードの Context 名から、ハイフンを除いても一致しない文書の Context 名への対応。 */
  contextAliases: Record<string, string>
  /** 導入時点で機能ノードを持たなかったスライス。増やさない。 */
  unmappedSlices: string[]
}

const FIELD = /^- (?:\*\*)?(理由|担保手段|上位の規則|例|要判断)(?:\*\*)?[：:]\s*(.*)$/

/**
 * 規則文から外す語。どれも、書き手が決めていない条件を読み手に委ねる。
 * INCOSE の Guide to Writing Requirements が曖昧な語として挙げるものから、日本語の規則文に
 * 現れるものを採った。
 */
const VAGUE_WORDS = ['適切に', '適宜', 'など', '必要に応じて', '可能な限り', '十分に']

const LIFECYCLE_SECTIONS = ['生成', '有効性', '利用', '効果の範囲', '失効と変更', '保持と削除']
const API_SECTIONS = ['対象の操作', '入力', '結果', '拒否', '作用']

const LAYER_DIRECTORIES = new Set(['domain', 'usecases'])

type RuleFields = {
  statements: Array<{ line: number; text: string }>
  guarantees: Array<{ line: number; symbol: string }>
  parents: Array<{ line: number; value: string }>
  openQuestions: Array<{ line: number; text: string }>
}

/** 規則の本文を、欄と規則文に分ける。箇条書きでも表でもない行は、どちらにも数えない。 */
function ruleFields(body: RuleBody): RuleFields {
  const fields: RuleFields = { statements: [], guarantees: [], parents: [], openQuestions: [] }
  for (const { line, text } of body.lines) {
    const field = text.match(FIELD)
    if (field) {
      const [, name, value = ''] = field
      if (name === '担保手段') {
        for (const symbol of value.matchAll(/`([^`]+)`/g)) {
          fields.guarantees.push({ line, symbol: symbol[1] ?? '' })
        }
      } else if (name === '上位の規則') fields.parents.push({ line, value })
      else if (name === '要判断') fields.openQuestions.push({ line, text: value })
      continue
    }
    if (text.startsWith('- ') || text.startsWith('|')) fields.statements.push({ line, text })
  }
  return fields
}

/**
 * 担保手段、上位の規則、曖昧な語を確かめる。`resolveLink` は、`from` の文書から見た
 * リンク先（アンカーを含む）が存在するかを答える。
 */
export function verifyRuleFields(
  path: string,
  source: string,
  declarations: GoDeclarations,
  resolveLink: (from: string, target: string) => boolean,
): Finding[] {
  const findings: Finding[] = []
  for (const body of ruleBodies(source)) {
    const fields = ruleFields(body)
    for (const { line, symbol } of fields.guarantees) {
      if (!isDeclared(symbol, declarations)) {
        findings.push({
          path,
          line,
          message: `${body.id} guarantee \`${symbol}\` is not declared in backend/`,
        })
      }
    }
    for (const { line, value } of fields.parents) {
      const target = value.match(/\[[^\]]+\]\(([^)\s]+)\)/)?.[1]
      if (!target) {
        findings.push({
          path,
          line,
          message: `${body.id} parent rule must be a Markdown link to the rule it departs from`,
        })
      } else if (!resolveLink(path, target)) {
        findings.push({ path, line, message: `${body.id} parent rule does not resolve: ${target}` })
      }
    }
    for (const { line, text } of [...fields.statements, ...fields.openQuestions]) {
      const prose = text.replaceAll(/`[^`]*`/g, '')
      for (const word of VAGUE_WORDS) {
        if (prose.includes(word)) {
          findings.push({
            path,
            line,
            message: `${body.id} uses vague wording 「${word}」; state the condition it stands for`,
          })
        }
      }
    }
  }
  return findings
}

function isDeclared(symbol: string, declarations: GoDeclarations): boolean {
  return symbol.includes('.') ? declarations.qualified.has(symbol) : declarations.names.has(symbol)
}

/**
 * Go のテスト以外のソースから、担保手段として書ける名前を集める。構文解析はせず、
 * gofmt が整えた行頭の宣言だけを読む。
 */
export function goDeclarations(files: Array<{ path: string; source: string }>): GoDeclarations {
  const names = new Set<string>()
  const qualified = new Set<string>()
  for (const { source } of files) {
    const pkg = source.match(/^package (\w+)/m)?.[1]
    const declare = (name: string) => {
      names.add(name)
      if (pkg) qualified.add(`${pkg}.${name}`)
    }
    let block: 'type' | 'value' | undefined
    for (const line of source.split('\n')) {
      if (block) {
        if (line === ')') block = undefined
        else {
          const member = line.match(/^\t(\w+)/)?.[1]
          if (member) declare(member)
        }
        continue
      }
      const method = line.match(/^func \((?:\w+ )?\*?(\w+)(?:\[[^\]]*\])?\) (\w+)/)
      if (method?.[1] && method[2]) {
        names.add(method[2])
        qualified.add(`${method[1]}.${method[2]}`)
        continue
      }
      const single = line.match(/^(?:func|type|var|const) (\w+)/)?.[1]
      if (single) {
        declare(single)
        continue
      }
      if (/^type \($/.test(line)) block = 'type'
      else if (/^(?:var|const) \($/.test(line)) block = 'value'
    }
  }
  return { names, qualified }
}

/**
 * 機能ノードの `scenarios.feature.md` で、節の見出しがライフサイクルか API のどちらか一方の
 * 語彙を、その順で使い、すべての規則がいずれかの節の下にあることを確かめる。
 * Context のルートの規則は機能をまたぐので、節を求めない。
 */
export function verifySectionOrder(path: string, source: string): Finding[] {
  if (!/^docs\/domain\/[^/]+\/[^/]+\/scenarios\.feature\.md$/.test(path)) return []
  const findings: Finding[] = []
  let vocabulary: string[] | undefined
  let position = -1
  let previous = ''
  let inSection = false
  for (const [index, text] of source.split('\n').entries()) {
    const line = index + 1
    const rule = text.match(/^#{2,6} Rule: (REQ-[A-Z0-9-]+)/)
    if (rule) {
      if (!inSection || text.startsWith('## ')) {
        findings.push({
          path,
          line,
          message: `${rule[1]} must sit under a section of the feature node`,
        })
      }
      continue
    }
    const heading = text.match(/^## (.+)$/)?.[1]?.trim()
    if (heading === undefined) continue
    inSection = true
    const lifecycle = LIFECYCLE_SECTIONS.indexOf(heading)
    const api = API_SECTIONS.indexOf(heading)
    if (lifecycle < 0 && api < 0) {
      findings.push({
        path,
        line,
        message:
          `section ${heading} is neither a lifecycle section (${LIFECYCLE_SECTIONS.join(', ')}) ` +
          `nor an API section (${API_SECTIONS.join(', ')})`,
      })
      continue
    }
    const own = lifecycle >= 0 ? LIFECYCLE_SECTIONS : API_SECTIONS
    vocabulary ??= own
    if (own !== vocabulary) {
      findings.push({
        path,
        line,
        message:
          own === API_SECTIONS
            ? `section ${heading} mixes the API sections into the lifecycle sections`
            : `section ${heading} mixes the lifecycle sections into the API sections`,
      })
      continue
    }
    const current = own.indexOf(heading)
    if (current <= position) {
      findings.push({
        path,
        line,
        message:
          current === position
            ? `section ${heading} appears twice`
            : `section ${heading} must come before ${previous}`,
      })
      continue
    }
    position = current
    previous = heading
  }
  return findings
}

/**
 * `backend/` の下のディレクトリ一覧から機能スライスを取り出す。`directories` は
 * リポジトリ相対のパスで、層のディレクトリ（`domain`、`usecases`）まで含む。
 */
export function featureSlices(directories: string[]): FeatureSlice[] {
  const slices = new Map<string, FeatureSlice>()
  for (const directory of directories) {
    const [root, context, name, layer] = directory.split('/')
    if (root !== 'backend' || !context || !name || !layer || !LAYER_DIRECTORIES.has(layer)) continue
    if (LAYER_DIRECTORIES.has(name)) continue
    const path = `backend/${context}/${name}`
    slices.set(path, { context, name, path })
  }
  return [...slices.values()].sort((left, right) => left.path.localeCompare(right.path))
}

/**
 * 各機能スライスに、名前が対応する機能ノードがあることを確かめる。機能ノードの名前から
 * ハイフンを除いた名前がスライスの名前と一致すれば対応とみなす。Context の名前も同じ規則で
 * 対応させ、一致しないものだけを `contextAliases` で引く。
 */
export function verifyFeatureNodes(
  slices: FeatureSlice[],
  nodes: Set<string>,
  debt: FeatureNodeDebt,
): Finding[] {
  const flatten = (name: string) => name.replaceAll('-', '')
  const nodeKeys = new Set(
    [...nodes].map((node) => {
      const [, , context = '', feature = ''] = node.split('/')
      return `${flatten(context)}/${flatten(feature)}`
    }),
  )
  const unmapped = new Set(debt.unmappedSlices)
  const findings: Finding[] = []
  for (const slice of slices) {
    const context = debt.contextAliases[slice.context] ?? slice.context
    const mapped = nodeKeys.has(`${flatten(context)}/${slice.name}`)
    if (mapped && unmapped.has(slice.path)) {
      findings.push({
        path: slice.path,
        line: 1,
        message: `${slice.path} has a feature node now; remove it from tools/check/feature-node-debt.json`,
      })
    } else if (!mapped && !unmapped.has(slice.path)) {
      findings.push({
        path: slice.path,
        line: 1,
        message: `${slice.path} has no feature node under docs/domain/${context}/`,
      })
    }
  }
  const existing = new Set(slices.map((slice) => slice.path))
  for (const path of debt.unmappedSlices) {
    if (!existing.has(path)) {
      findings.push({
        path,
        line: 1,
        message: `${path} is no longer a feature slice; remove it from tools/check/feature-node-debt.json`,
      })
    }
  }
  return findings
}
