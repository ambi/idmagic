/**
 * Keep the command map and the pipelines that call it in agreement.
 *
 * A recipe removed during an ordinary refactor takes every caller with it, and
 * a caller living in CI keeps passing locally: `just verify` is green while the
 * pipeline stops at `error: justfile does not contain recipe ...`. Nothing else
 * in the repository can see that gap, so it is checked here.
 *
 * Only the direction that can silently break a pipeline is checked. A recipe no
 * CI job calls is not a defect — the command map serves people too.
 */

export type CommandMapFinding = {
  file: string
  recipe: string
}

/** Recipe names declared by a justfile, including parameterized recipes. */
export function recipeNames(justfile: string): Set<string> {
  const names = new Set<string>()
  for (const line of justfile.split('\n')) {
    // A recipe header starts at column zero and ends with `:`; a variable
    // assignment (`name := value`) and an indented body line never match.
    const match = line.match(/^([a-z][a-z0-9-]*)(?:\s+[^:=]*)?:(?!=)/)
    if (match?.[1]) names.add(match[1])
  }
  return names
}

/** Recipes a workflow invokes as `just <recipe>`. */
export function invokedRecipes(workflow: string): string[] {
  return [...workflow.matchAll(/(?:^|\s)just\s+([a-z][a-z0-9-]*)/g)].map((match) => match[1] ?? '')
}

export function verifyCommandMap(
  justfile: string,
  workflows: Array<{ file: string; source: string }>,
): CommandMapFinding[] {
  const declared = recipeNames(justfile)
  const findings: CommandMapFinding[] = []
  for (const workflow of workflows) {
    for (const recipe of invokedRecipes(workflow.source)) {
      if (!declared.has(recipe) && !findings.some((one) => one.recipe === recipe)) {
        findings.push({ file: workflow.file, recipe })
      }
    }
  }
  return findings
}
