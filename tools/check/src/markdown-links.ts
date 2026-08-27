import { posix } from 'node:path'
import MarkdownIt from 'markdown-it'
import type Token from 'markdown-it/lib/token.mjs'

export type MarkdownLinkFinding = {
  file: string
  line: number
  target: string
  message: string
}

type MarkdownLink = {
  line: number
  target: string
}

type MarkdownEnvironment = {
  references?: Record<string, { href: string }>
}

const markdown = new MarkdownIt({ html: true })
markdown.validateLink = () => true

function inlineText(token: Token): string {
  return (token.children ?? [])
    .map((child) => {
      if (child.type === 'text' || child.type === 'code_inline') return child.content
      if (child.type === 'image') return child.content
      if (child.type === 'softbreak' || child.type === 'hardbreak') return ' '
      return ''
    })
    .join('')
}

function slugBase(heading: string): string {
  return heading
    .trim()
    .toLocaleLowerCase('en-US')
    .replace(/[^\p{L}\p{M}\p{N}\p{Pc}\s-]/gu, '')
    .replace(/\s/g, '-')
}

function explicitAnchors(token: Token): string[] {
  if (token.type !== 'html_block' && token.type !== 'html_inline') return []
  const result: string[] = []
  const value = String.raw`\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s>]+))`
  for (const match of token.content.matchAll(
    new RegExp(String.raw`<[a-z][^>]*\sid${value}`, 'gi'),
  )) {
    const anchor = match[1] ?? match[2] ?? match[3]
    if (anchor) result.push(anchor)
  }
  for (const match of token.content.matchAll(
    new RegExp(String.raw`<a\b[^>]*\sname${value}`, 'gi'),
  )) {
    const anchor = match[1] ?? match[2] ?? match[3]
    if (anchor) result.push(anchor)
  }
  return result
}

function anchors(tokens: Token[]): Set<string> {
  const result = new Set<string>()

  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index]
    if (!token) continue
    for (const anchor of explicitAnchors(token)) result.add(anchor)
    for (const child of token.children ?? []) {
      for (const anchor of explicitAnchors(child)) result.add(anchor)
    }

    if (token.type !== 'heading_open') continue
    const inline = tokens[index + 1]
    if (inline?.type !== 'inline') continue
    const base = slugBase(inlineText(inline))
    let anchor = base
    let suffix = 0
    while (result.has(anchor)) {
      suffix += 1
      anchor = `${base}-${suffix}`
    }
    result.add(anchor)
  }

  return result
}

function links(tokens: Token[], environment: MarkdownEnvironment): MarkdownLink[] {
  const result: MarkdownLink[] = []
  const usedTargets = new Set<string>()
  for (const token of tokens) {
    if (token.type !== 'inline') continue
    for (const child of token.children ?? []) {
      if (child.type !== 'link_open') continue
      const target = child.attrGet('href')
      if (!target) continue
      result.push({ line: (token.map?.[0] ?? 0) + 1, target })
      usedTargets.add(target)
    }
  }
  for (const reference of Object.values(environment.references ?? {})) {
    if (!usedTargets.has(reference.href)) result.push({ line: 1, target: reference.href })
  }

  return result
}

function decoded(value: string): string | undefined {
  try {
    return decodeURIComponent(value)
  } catch {
    return undefined
  }
}

function splitTarget(target: string): { path: string; anchor?: string } {
  const hash = target.indexOf('#')
  const rawPath = hash === -1 ? target : target.slice(0, hash)
  const rawAnchor = hash === -1 ? undefined : target.slice(hash + 1)
  const query = rawPath.indexOf('?')
  return {
    path: query === -1 ? rawPath : rawPath.slice(0, query),
    anchor: rawAnchor,
  }
}

function resolvedPath(file: string, target: string): string {
  return posix.normalize(posix.join(posix.dirname(file), target)).replace(/\/$/, '')
}

export function verifyMarkdownLinks(
  documents: ReadonlyMap<string, string>,
  existingPaths: ReadonlySet<string>,
): MarkdownLinkFinding[] {
  if (documents.size === 0) {
    return [{ file: '.', line: 1, target: '', message: 'no Markdown documents were collected' }]
  }

  const parsed = new Map<string, { anchors: Set<string>; links: MarkdownLink[] }>()
  for (const [file, source] of documents) {
    const environment: MarkdownEnvironment = {}
    const tokens = markdown.parse(source, environment)
    parsed.set(file, { anchors: anchors(tokens), links: links(tokens, environment) })
  }

  const findings: MarkdownLinkFinding[] = []
  for (const [file, document] of parsed) {
    for (const link of document.links) {
      if (/^file:/i.test(link.target)) {
        findings.push({
          file,
          line: link.line,
          target: link.target,
          message: 'local file URL is not portable',
        })
        continue
      }
      if (
        /^[a-z][a-z0-9+.-]*:/i.test(link.target) ||
        link.target.startsWith('//') ||
        link.target.startsWith('/')
      )
        continue

      const parts = splitTarget(link.target)
      const path = decoded(parts.path)
      const anchor = parts.anchor === undefined ? undefined : decoded(parts.anchor)
      if (path === undefined || (anchor === undefined && parts.anchor !== undefined)) {
        findings.push({
          file,
          line: link.line,
          target: link.target,
          message: 'link contains invalid percent encoding',
        })
        continue
      }

      const targetPath = path === '' ? file : resolvedPath(file, path)
      if (!existingPaths.has(targetPath)) {
        findings.push({
          file,
          line: link.line,
          target: link.target,
          message: `relative link target does not exist: ${targetPath}`,
        })
        continue
      }

      if (anchor && documents.has(targetPath) && !parsed.get(targetPath)?.anchors.has(anchor)) {
        findings.push({
          file,
          line: link.line,
          target: link.target,
          message: `Markdown anchor does not exist in ${targetPath}: #${anchor}`,
        })
      }
    }
  }

  return findings
}
