---
name: IdMagic
colors:
  paper: '#faf9f6'
  surface: '#ffffff'
  line: '#e2e4e9'
  foreground: '#020617'
  foreground-secondary: '#475569'
  foreground-tertiary: '#64748b'
  foreground-faint: '#94a3b8'
  ink: '#020617'
  ink-foreground: '#ffffff'
  accent: '#0f6f65'
  accent-soft: '#eaf2f0'
  inverse-surface: '#020617'
  success: '#10b981'
  success-bg: '#ecfdf5'
  success-text: '#047857'
  destructive: '#dc2626'
  destructive-bg: '#fef2f2'
  destructive-text: '#b91c1c'
  warning: '#f59e0b'
  warning-bg: '#fffbeb'
  warning-text: '#b45309'
typography:
  aside-title:
    fontFamily: Inter
    fontSize: clamp(1.5rem, 2.2vw, 2rem)
    fontWeight: '600'
    lineHeight: '1.3'
    letterSpacing: normal
  page-title:
    fontFamily: Inter
    fontSize: 1.375rem
    fontWeight: '600'
    lineHeight: '1.2'
    letterSpacing: normal
  section-title:
    fontFamily: Inter
    fontSize: 1.25rem
    fontWeight: '600'
    lineHeight: '1.2'
    letterSpacing: normal
  heading-sm:
    fontFamily: Inter
    fontSize: 0.875rem
    fontWeight: '600'
    lineHeight: '1.3'
    letterSpacing: normal
  stat-value:
    fontFamily: Inter
    fontSize: 1.875rem
    fontWeight: '600'
    lineHeight: '1.15'
    letterSpacing: -0.01em
  body-base:
    fontFamily: Inter
    fontSize: 0.875rem
    fontWeight: '400'
    lineHeight: '1.6'
    letterSpacing: normal
  body-sm:
    fontFamily: Inter
    fontSize: 0.8125rem
    fontWeight: '400'
    lineHeight: '1.5'
    letterSpacing: normal
  label-eyebrow:
    fontFamily: Inter
    fontSize: 0.6875rem
    fontWeight: '600'
    lineHeight: '1.2'
    letterSpacing: normal
    textTransform: uppercase
  meta-micro:
    fontFamily: Inter
    fontSize: 0.75rem
    fontWeight: '500'
    lineHeight: '1.4'
    letterSpacing: normal
  data-mono:
    fontFamily: ui-monospace
    fontSize: 0.75rem
    fontWeight: '400'
    lineHeight: '1.5'
    letterSpacing: normal
rounded:
  sm: 0.3rem
  DEFAULT: 0.5rem
  md: 0.4rem
  lg: 0.5rem
  xl: 0.7rem
  full: 9999px
spacing:
  unit: 4px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  gutter: 20px
  content-max: 1500px
  auth-container-max: 1120px
  sidebar-width: 248px
---

## 1. Visual Theme & Atmosphere

IdMagic is an identity & access management (IAM) admin console — a sibling to
Okta or Microsoft Entra — plus the end-user auth flows (login, MFA, consent)
that sit in front of it. This redesign deliberately moves away from that
genre's usual "SaaS dashboard" look (gradients, glassmorphism, bento grids
of colored tiles, a badge on everything) toward the restraint of ChatGPT and
Claude: a warm, low-saturation paper canvas, one quiet accent color, and
typography/spacing/hierarchy doing the work that decoration used to do. It
is **not** a chat UI — it keeps every explicit control the product needs
(forms, tables, nav, dialogs) — it just stops dressing them up.

Two colors carry all the meaning, and each has exactly one job. **Ink**
(`#020617`, the same near-black used everywhere already) is the one solid
fill for every primary action: the default `Button`, the tenant-branding
CTA fallback, the active sidebar item, avatar circles. **Accent** — a
quiet, deep teal (`#0f6f65`) — is reserved for text-level interaction only:
links, focus rings, the eyebrow label, selected/verified states. It was
chosen over "another blue" because a muted teal is already latent in the
product's own mark (the small verified-dot on the fingerprint logo) —
so the one accent this whole redesign spends is one the brand already
owns, not a generic SaaS blue. Status color (emerald/red/amber) stays
functional-only, expressed as a small dot + colored text, never a filled
pill.

Structurally, chrome is quiet almost to the point of disappearing: flat
paper background throughout (no gradients, no backdrop-blur, no dot-grid
textures), hairline borders only where they separate real structural zones
(sidebar from content, header from page, one table row from the next), and
no shadows beyond the minimum a floating popover needs to read as "above"
the page. Headings are sized to be found, not admired — nothing above
1.375rem outside the auth hero. Cards, when content truly needs a
boundary, are a single hairline border and nothing else — no nested cards,
no nested cards inside cards, no nested containers for their own sake.

## 2. Color Palette & Roles

### Foundation

- **Paper** `#faf9f6` — the page background everywhere (`--background`). Warm and low-saturation rather than clinical white or cool slate — the canvas a document sits on, not a "surface" trying to look elevated.
- **Surface White** `#ffffff` — reserved for the few things that are genuinely floating above the page: popovers, dropdown menus, the rare card. Distinct from paper by exactly one functional step, not by shadow or blur.
- **Hairline** `#e2e4e9` — the only border weight in the system. Used where it's structurally load-bearing (sidebar/content split, header/content split, table row dividers) and nowhere else.
- **Ink** `#020617` (`slate-950`) — primary text, and the one solid-fill color for every primary action (see below).
- Secondary/tertiary/faint text stay on the existing Tailwind slate scale (`slate-600` / `slate-500` / `slate-400`) — deliberately not reinvented, so the hundreds of untouched call sites across the app stay visually consistent with the redesigned core.

### Accent & Interactive

- **Ink — Primary/CTA** `#020617` — the single solid-fill color for every primary action: `Button` "default", the tenant-branding CTA fallback (`.tenant-primary-cta`), the active sidebar indicator's implied weight, avatar circles. Hover lightens via opacity, no separate pressed shade, no colored button anywhere in the system.
- **Accent — Teal** `#0f6f65` — the one restrained accent, used only for things you can act on: link text, the `Button` "link" variant, focus rings (`--ring`), the eyebrow label, selected-state icon tints (`accent-soft` `#eaf2f0` background + accent-colored icon/text). Never a solid button fill.
- **Ink Navy** `#020617`/`slate-950` — the flat dark panel used for the auth aside. No gradient, no glow, no texture — just the same ink as everywhere else, inverted.

### Functional States

- **Success — Emerald** dot `#10b981` / text `#047857` — expressed as a small dot + plain colored text, not a filled pill.
- **Destructive — Red** dot `#ef4444` / text `#b91c1c` — same idiom.
- **Warning — Amber** dot `#f59e0b` / text `#b45309` — same idiom.
- Functional backgrounds (`success-bg` / `destructive-bg` / `warning-bg`) exist only for the rare full-width `Alert`, never for a small inline badge.

## 3. Typography Rules

### Hierarchy & Weights

Font: **Inter** (variable weight), with `"Noto Sans JP"` and system UI
fonts as fallback — the product is bilingual (EN/JA), which is exactly why
this redesign keeps a single quiet sans rather than reaching for a
decorative display face: most characterful display faces have no Japanese
coverage, and ChatGPT/Claude's own restraint comes from *not* needing a
second face to carry personality. The one deliberate second role is a
monospace face for identifiers — user IDs, client IDs, tokens — which
already existed in a few detail rows and is now the formal "data" register:
in an IAM console, looking exact and auditable *is* the personality.

| Role | Size | Weight | Notes |
|---|---|---|---|
| Auth hero title | `clamp(1.5rem, 2.2vw, 2rem)` | 600 | the largest text in the product; still modest |
| App/page title | 1.375rem | 600 | admin shell header, auth section titles |
| Section heading | 0.875rem | 600 | inside a page, above a list or form group |
| Stat value | 1.875rem | 600 | dashboard KPI numbers, plain type, no card |
| Body | 0.875rem | 400 | default UI text |
| Small body / table cell | 0.8125rem | 400–500 | tables, form hints |
| Eyebrow label | 0.6875rem | 600, uppercase | accent-colored, used sparingly above a heading |
| Meta / micro label | 0.75rem | 500 | breadcrumbs, timestamps, captions |
| Data / identifier | 0.75rem | mono | user IDs, client IDs, tokens |

### Spacing Principles

No heading in the product exceeds 1.375rem outside the auth hero — hierarchy
comes from weight and color (ink vs. slate-600 vs. slate-500) more than from
size jumps. Line-height stays generous for body copy (1.5–1.6) and tight for
headings (1.2–1.3). Letter-spacing stays at `normal` everywhere, including
uppercase eyebrow labels — weight and the accent color carry emphasis, not
tracking.

## 4. Component Stylings

### Buttons

`rounded-md` (0.4rem), `h-9` (36px) default height. `default` is a solid
ink fill with white text — the only colored button in the system, and it's
never blue. `outline`/`secondary`/`ghost` stay unfilled or near-unfilled
until hover. `destructive` is a low-opacity red tint, not a solid red
block. `link` is accent-teal text with an underline on hover — the one
place the accent color fills real pixels rather than a thin ring. No
button anywhere carries a drop shadow; the only motion cue is a 1px
press-down on click and a 3px accent-tinted focus ring on keyboard focus.

### Cards & Panels

Used sparingly, and only where content genuinely needs a boundary (a form
section, a table wrapper). A card is one hairline border on a white
surface — no shadow, no blur, no translucency, no nested cards. Most
"card-shaped" content from the previous design (dashboard metrics, security
recommendations, quota usage) was rebuilt as plain typographic blocks
separated by a hairline divider and whitespace instead, per "can hierarchy
be expressed without another container?"

### Navigation

A quiet 248px sidebar, flush with the page (no background fill, no blur) —
just a hairline border separating it from content. Nav items carry no
filled background at all, active or not: the active item is ink-colored
text at semibold weight with a 2px accent-colored rule on its left edge;
inactive items are slate-600, going to ink on hover. The header is flat
paper with a single hairline bottom border — no blur, no translucency, no
inset highlight. Secondary chrome that used to live in a bordered chip
(the "Default organization" pill, the "Account portal" badge, the auth
flow's "Protected" badge) is now plain text — the container wasn't adding
information, so it was removed rather than quieted.

### Inputs & Forms

`h-11` rounded-`md` inputs, a plain white fill, a `slate-300` border. Focus
turns the border and a soft ring accent-teal (`ring-accent/15`) — the same
accent used for links and focus everywhere else, so "this is now active"
always reads the same color across the whole product. Labels sit above
fields.

### Status

No pill, no badge background. Status is a 6px colored dot followed by
plain colored text (`● Active`, `● Disabled`, `● Pending`) at the same
size and weight as surrounding text — present, not shouting. Role/tag
chips (an actual small set of discrete values a user attached, as opposed
to a status) keep a light bordered treatment since they *are* a container
of real content, not decoration.

### Auth Flow

Two columns on a flat paper background — no boxed "frame," no border, no
shadow, no backdrop-blur around the pair. The promotional aside is a flat
ink panel (no gradient, no glow, no dot-grid) with the same restrained
teal accent on its checklist icons and eyebrow. Below 900px the aside
drops and the form runs full-width, single column.

## 5. Layout Principles

### Grid & Structure

- Admin console: `max-w-[1500px]` content, `248px` fixed sidebar at `lg`.
- Auth flow: `max-w-[1120px]` outer container; two columns split roughly
  0.9fr / 1fr (promo : form) above 900px, single column below it.
- Standard Tailwind breakpoints (`sm`640 / `md`768 / `lg`1024 / `xl`1280),
  plus the auth flow's own `900px` / `520px` steps.

### Whitespace Strategy

Sections separate with whitespace and, where truly structural, a single
hairline divider (`border-t border-slate-100`) — not a box. Page padding
runs `p-5` → `p-6` → `p-8` as the viewport grows; content sections stack
with `gap-7`–`gap-8` between them rather than being boxed apart.

### Alignment & Visual Balance

Left-aligned, text-first admin layouts. The auth flow is the one
centered/symmetric exception. Icon+text pairing (Tabler, 14–20px, stroke
1.7–1.8) stays the connective micro-pattern across nav, buttons, and list
rows — icons are never colored chips, just inline glyphs.

### Responsive Behavior & Touch

Desktop-first, degrading gracefully: sidebar and auth aside both drop below
their breakpoints, multi-column grids collapse to 1–2 columns. Control
height stays 36–44px, sized for mouse/trackpad rather than oversized touch
targets.

## 6. Design System Notes for Stitch Generation

### Language to Use

Describe this system as: *"calm, content-first enterprise IAM console in
the register of ChatGPT and Claude — warm low-saturation paper canvas, one
ink-black solid color for every primary action, one quiet deep-teal accent
for links/focus/selection only, status shown as a colored dot and plain
text rather than a pill, hairline borders only where structurally
necessary, no gradients, no glassmorphism, no card-on-card, generous
whitespace, Inter typeface, a monospace register for identifiers, Tabler
line icons."* Before adding any visible element, ask whether it needs to be
permanently visible, whether hierarchy can be expressed without another
container, and whether a secondary action can appear contextually instead.

### Color References

- Background: Paper `#faf9f6`
- Surface (popovers/rare cards only): White `#ffffff`
- Border: Hairline `#e2e4e9`
- Primary text & every solid CTA fill: Ink `#020617`
- The one accent (links, focus, selection): Teal `#0f6f65`
- Success / Destructive / Warning: Emerald `#10b981` / Red `#dc2626` / Amber `#f59e0b`

### Component Prompts

- *"A primary button: solid ink-black fill, white text, rounded-md corners,
  no shadow, no gradient — a 1px press-down on click is the only motion."*
- *"A status row: a 6px colored dot, then plain colored text at body size —
  no pill, no background fill."*
- *"A dashboard summary: four plain typographic stat blocks (icon, label,
  large number, one line of detail) in a grid separated by whitespace and
  a single hairline divider underneath — no cards, no gradient hero, no
  circular gauge."*
- *"A sidebar nav item: no background fill ever; active state is
  semibold ink text with a 2px teal rule on the left edge, inactive is
  slate-600 text that darkens on hover."*
- *"An auth screen: two plain columns on flat paper, no boxed frame, no
  shadow, no blur — a flat ink promo panel on one side, a plain form on
  the other."*

### Incremental Iteration

When refining generated screens: if a button, tile, or panel has a
gradient, a glow, backdrop-blur, or a shadow heavier than a 1px definition
line, remove it — that's the previous design language leaking back in. If
two colors are both trying to signal "primary," collapse them into ink.
Prefer deleting a container to styling it more quietly. Keep emerald/red/
amber strictly functional and never decorative.

## 7. Known Scope of This Pass

This redesign rebuilt the shared foundation — `styles.css` tokens, `Button`,
`Card`, `Input`, `Alert`, `Select`/`DropdownMenu` popovers, `AdminShell`/
`AccountShell`/`SystemShell` chrome, `AuthShell`, the `StatusBadge` +
`UserAvatar` primitives in `admin-users`, and `AdminDashboardPage` as the
flagship example — because those are shared across most of the product and
carry the redesign to every screen that uses them. It did **not** sweep the
~40 remaining `features/admin-*` pages individually: several still have
their own local bordered/tinted "status badge" components (duplicated
per-feature rather than sharing `StatusBadge`), bespoke metric tiles, or
hardcoded `blue-600`/`blue-700` text that wasn't touched. Those read
correctly today because they reuse the redesigned `Card`/`Button`/`Input`
primitives, but a full pass would still find and quiet their local
one-off decoration.
