/**
 * Dangling-ADR-link detection.
 *
 * Design records, work items and the format docs point at ADRs by path. Nothing
 * used to verify those paths, so renaming or merging an ADR left silent dead
 * links (ADR-143). This module extracts the references and reports the ones that
 * resolve to no file; it stays pure so the CLI owns all filesystem access.
 */

/**
 * A markdown link pointing at `…/ADR-NNN-*.md`. Path characters exclude `*` so a
 * prose glob such as `decisions/ADR-092-*.md` is not mistaken for a link.
 */
const ADR_PATH_RE =
  /(?:^|[\s("'`[\]])((?:\.{1,2}\/|decisions\/)[^\s)"'`\]*]*ADR-\d+[^\s)"'`\]*]*\.md)/gi

export type AdrLinkSource = {
  /** Workspace-relative path of the referencing file. */
  path: string
  text: string
  /** Extra references not present in the text, e.g. work-item frontmatter lists. */
  declared?: readonly string[]
}

export type AdrLinkFinding = {
  path: string
  message: string
}

export function extractAdrLinks(text: string): string[] {
  const links = new Set<string>()
  for (const match of text.matchAll(ADR_PATH_RE)) {
    if (match[1]) links.add(match[1])
  }
  return [...links].sort()
}

/**
 * Report references that resolve to no file.
 *
 * `resolvesFrom` receives the referencing file's path and the raw reference; it
 * returns true when the reference resolves either against that file's directory
 * or against the workspace root, because both spellings are in use.
 */
export function checkAdrLinks(
  sources: readonly AdrLinkSource[],
  resolvesFrom: (fromPath: string, reference: string) => boolean,
): AdrLinkFinding[] {
  const findings: AdrLinkFinding[] = []
  for (const source of sources) {
    const references = [...extractAdrLinks(source.text), ...(source.declared ?? [])]
    for (const reference of [...new Set(references)].sort()) {
      if (resolvesFrom(source.path, reference)) continue
      findings.push({
        path: source.path,
        message: `dangling ADR link '${reference}' resolves to no file`,
      })
    }
  }
  return findings
}
