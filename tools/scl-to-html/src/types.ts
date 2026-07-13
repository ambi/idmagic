/**
 * Types for every artifact the tool renders: SCL document, ADRs (incl.
 * CONCEPTION), and work items with optional completion records.
 *
 * SCL types follow SPECIFICATION_CORE_LANGUAGE.md §2–§3. Change types
 * mirror the JSON Schemas under tools/yaml-check/schemas/.
 */

// ─── SCL ───────────────────────────────────────────────────────────

export const SECTION_KINDS = [
  'standards',
  'context_map',
  'glossary',
  'models',
  'interfaces',
  'states',
  'authorization',
  'objectives',
  'scenarios',
  'flows',
] as const

export type SectionKind = (typeof SECTION_KINDS)[number]

export interface SclDocument {
  system: string
  spec_version: string
  context?: string
  annotations?: Record<string, unknown>
  standards?: Record<string, Standard>
  context_map?: Record<string, ContextMapEntry>
  glossary?: Record<string, GlossaryEntry>
  models?: Record<string, Model>
  interfaces?: Record<string, Interface>
  states?: Record<string, StateMachine>
  authorization?: Authorization
  objectives?: Record<string, Objective>
  scenarios?: Record<string, Scenario>
  flows?: Record<string, Flow>
}

export interface ContextMapEntry {
  path?: string
  description?: string
  publishes?: string[]
  depends_on?: Record<string, { uses?: string[]; via?: string; reason?: string }>
  annotations?: Record<string, unknown>
}

export interface SclContextDocument {
  name: string
  path: string
  document: SclDocument
}

export interface SclBundle {
  root: SclDocument
  contexts: SclContextDocument[]
}

export interface Standard {
  title?: string
  version?: string
  url?: string
  roles?: string[]
  scope?: string
  requirements?: StandardRequirement[]
}

export interface StandardRequirement {
  id?: string
  section?: string
  strength?: string
  adoption?: 'required' | 'optional' | 'excluded'
  statement?: string
  reason?: string
  refs?: string[]
}

export interface GlossaryEntry {
  definition?: string
  description?: string
  aliases?: string[]
  context?: string
  not_to_confuse_with?: Array<{ term?: string; reason?: string }>
  annotations?: Record<string, unknown>
}

export interface Field {
  type?: unknown
  fields?: Record<string, Field>
  optional?: boolean
  default?: unknown
  constraints?: unknown[]
  description?: string
  annotations?: Record<string, unknown>
}

export interface Model {
  kind?: string
  description?: string
  identity?: string | string[]
  annotations?: Record<string, unknown>
  values?: string[]
  fields?: Record<string, Field>
  payload?: Record<string, Field>
  constraints?: string[]
}

export interface Interface {
  description?: string
  steps?: string[]
  input?: Record<string, Field>
  output?: Record<string, Field>
  errors?: string[]
  emits?: string[]
  requires?: string[]
  ensures?: string[]
  access?: Access
  idempotent?: boolean
  read_only?: boolean
  bindings?: Binding[]
  annotations?: Record<string, unknown>
}

export type Binding = { kind: string; description?: string } & Record<string, unknown>

export type Access =
  | 'public'
  | 'internal'
  | {
      policies: string[]
      resource: { type: string; id: string }
    }

export interface StateMachine {
  description?: string
  annotations?: Record<string, unknown>
  target?: string
  initial?: string
  terminal?: string[]
  transitions?: Array<{
    from?: string
    to?: string
    event?: string
    on?: string
    guard?: unknown
    effect?: string[]
  }>
}

export interface Authorization {
  resources?: Record<string, { description?: string }>
  principals?: Record<
    string,
    { type: string; matches: string[]; description?: string; annotations?: Record<string, unknown> }
  >
  policies?: Record<
    string,
    {
      effect: 'permit' | 'forbid'
      principal: string
      when?: string
      description?: string
      annotations?: Record<string, unknown>
    }
  >
}

export interface Scenario {
  description?: string
  annotations?: Record<string, unknown>
  tags?: string[]
  actor: string
  given?: string[]
  main_success?: string[]
  extensions?: Array<{ at?: string | number; condition?: string; steps?: string[] }>
}

export interface Objective {
  description?: string
  annotations?: Record<string, unknown>
  interface?: string
  indicator: string
  target: number
  window: string
  budgeting?: 'occurrences' | 'timeslices'
  slice?: string
}

export interface Flow {
  description?: string
  annotations?: Record<string, unknown>
  entry: string
  transitions: Array<{
    from: string
    action: string
    interface?: string
    to?: string
    external?: boolean
  }>
}

// ─── Decisions (CONCEPTION + ADR) ──────────────────────────────────

export interface DecisionDoc {
  /** Stable slug used as the in-page anchor (e.g. "adr-001-...", "conception"). */
  id: string
  /** Display title parsed from the first markdown heading. */
  title: string
  /** Document kind drives the navigation grouping. */
  kind: 'conception' | 'adr'
  /** Source filename, kept for "view source" links. */
  filename: string
  /** Raw markdown body (heading line dropped). */
  body: string
  /** Best-effort ADR number, parsed from the filename when applicable. */
  number?: number
}

// ─── Work items with optional completion records ────────────────────

export interface WorkItem {
  id: string
  title?: string
  status?: 'pending' | 'in_progress' | 'completed' | 'cancelled'
  created_at?: string
  authors?: string[]
  risk?: 'low' | 'medium' | 'high' | 'critical'
  depends_on?: string[]
  motivation?: string
  scope?: unknown
  out_of_scope?: unknown
  plan?: string
  tasks?: string
  affected_guarantees?: unknown
  verification?: unknown
  risk_notes?: string
  completion?: Completion
  [k: string]: unknown
}

export interface Completion {
  completed_at?: string
  summary?: string
  verification?: unknown
  evidence?: unknown
  affected_guarantees_state?: unknown
  remaining_guarantees_state?: unknown
  residual_risk?: unknown
  semantic_diff?: unknown
  traceability?: unknown
  human_decisions?: unknown
  approver_note?: string
  [k: string]: unknown
}

export interface ChangeEntry {
  /** File stem under `work-items/` (also the in-page anchor). */
  id: string
  work_item: WorkItem
}

// ─── Architecture (second-layer current-state map) ──────────────────

export interface ArchitectureModule {
  path?: string
  responsibility?: string
  realizes?: string[]
}

export interface ArchitectureContext {
  root?: string
  summary?: string
}

export interface ArchitectureDoc {
  context?: string
  updated_at?: string
  contexts?: Record<string, ArchitectureContext>
  modules?: Record<string, ArchitectureModule>
  /** Prose body sections (rendered as markdown). */
  overview?: string
  structure?: string
  stack?: string
  structural_decisions?: string
  cross_cutting_concerns?: string
  diagrams?: string
}

// ─── Top-level page input ──────────────────────────────────────────

export interface SiteInput {
  scl: SclDocument | SclBundle
  decisions: DecisionDoc[]
  work_items: ChangeEntry[]
  /** Optional second-layer Architecture map (ARCHITECTURE.md). */
  architecture?: ArchitectureDoc | null
  /** Optional override for the document <title> and page header. */
  title?: string
}
