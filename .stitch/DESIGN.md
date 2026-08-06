---
name: IdMagic
colors:
  background: '#f8fafc'
  surface: '#ffffff'
  surface-translucent: 'rgb(255 255 255 / 90%)'
  surface-muted: '#f6f8fb'
  border: '#e2e8f0'
  border-strong: '#cbd5e1'
  foreground: '#020617'
  foreground-secondary: '#475569'
  foreground-tertiary: '#64748b'
  foreground-faint: '#94a3b8'
  primary: '#2563eb'
  primary-hover: '#1d4ed8'
  primary-foreground: '#ffffff'
  secondary-accent: '#2dd4bf'
  inverse-surface: '#0a1020'
  inverse-surface-deep: '#020617'
  success: '#10b981'
  success-bg: '#ecfdf5'
  success-text: '#047857'
  destructive: '#dc2626'
  destructive-bg: '#fef2f2'
  destructive-text: '#b91c1c'
  warning: '#f59e0b'
  warning-bg: '#fffbeb'
  warning-text: '#b45309'
  info-violet: '#6d28d9'
  info-violet-bg: '#eef2ff'
typography:
  display-hero:
    fontFamily: Inter
    fontSize: clamp(2rem, 3.6vw, 3.05rem)
    fontWeight: '600'
    lineHeight: '1.12'
    letterSpacing: normal
  page-title:
    fontFamily: Inter
    fontSize: 1.9rem
    fontWeight: '600'
    lineHeight: '1.2'
    letterSpacing: normal
  section-title:
    fontFamily: Inter
    fontSize: 1.7rem
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
    fontWeight: '700'
    lineHeight: '1.15'
    letterSpacing: -0.01em
  body-base:
    fontFamily: Inter
    fontSize: 0.925rem
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
    fontSize: 0.7rem
    fontWeight: '700'
    lineHeight: '1.2'
    letterSpacing: normal
    textTransform: uppercase
  meta-micro:
    fontFamily: Inter
    fontSize: 0.68rem
    fontWeight: '600'
    lineHeight: '1.4'
    letterSpacing: normal
rounded:
  sm: 0.45rem
  DEFAULT: 0.75rem
  md: 0.6rem
  lg: 0.75rem
  xl: 1.05rem
  2xl: 1.35rem
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
  auth-container-max: 1160px
  sidebar-width: 248px
---

## 1. Visual Theme & Atmosphere

IdMagic is an identity & access management (IAM) admin console — a sibling to
products like Okta or Microsoft Entra. The visual language is **clean
enterprise SaaS**: a bright, cool-neutral canvas (slate-50/white) punctuated by
a single confident **blue accent**, with functional color reserved strictly
for status communication (success/warning/danger). The overall feel is
trustworthy, precise, and quiet — the UI recedes so that identity data
(users, roles, sessions, security posture) stays legible.

Two atmospheres coexist. The **light "operations" surface** (admin console,
dashboards, tables) uses a soft slate-to-white gradient background, hairline
borders, and gentle diffuse shadows to imply depth without heaviness — cards
feel like frosted glass floating a few pixels above the page
(`backdrop-blur` + low-opacity white fills are everywhere: headers, sidebars,
cards). The **dark "brand" surface** (auth screens' aside panel, the admin
dashboard's security-score hero card) uses a near-black navy
(`#0a1020`–`#020617`) with subtle blue/teal glow gradients and a faint dot-grid
texture — this is where the product asserts its identity and security
credibility. Whitespace is generous but not loose (16–32px rhythm); density is
moderate-to-high in data tables and moderate in dashboards, favoring clarity
over minimalism.

## 2. Color Palette & Roles

### Primary Foundation

- **Cool Barely-There Slate** `#f8fafc` (`slate-50`) — page background (`<html>` and most feature pages).
- **Soft Layered Gradient** `linear-gradient(180deg, #f8fbff, #f2f6fb 38%, #f8fafc)` — the `.app-surface` background behind the admin shell, giving depth without a flat fill.
- **Pure White** `#ffffff` — card and panel surfaces, almost always at 90–92% opacity with `backdrop-blur` over the slate gradient.
- **Hairline Border Slate** `#e2e8f0` (`slate-200`, often at 70–80% opacity) — the default border for cards, headers, sidebars, table rows.
- **Deep Ink** `#020617` (`slate-950`) — primary text color and the default fill for high-emphasis UI chips (active nav item, avatar badges, sidebar "org" pill).

### Accent & Interactive

- **Confident Blue** `#2563eb` (`blue-600`) — the single primary accent: primary CTAs' intended color, links, focus rings (`ring-blue-600/10-30`), active icons, progress bars, the "eyebrow" label on auth screens.
- **Blue Hover/Pressed** `#1d4ed8` (`blue-700`) — hover/active state for blue text and links.
- **Teal Spark** `#2dd4bf` (`teal-400`) — a minimal secondary accent, used as a single "online/verified" indicator dot on the brand mark and in the auth aside's top glow gradient. Used sparingly, never as a fill.
- **Ink Navy** `#0a1020` — the dark brand surface for the auth split-screen aside and dashboard hero card, with layered low-opacity blue/teal radial gradients for atmosphere.

### Typography & Text Hierarchy

- **Primary Text — Deep Ink** `#020617` (`slate-950`) — headings, primary values, high-emphasis labels.
- **Secondary Text — Slate** `#475569` (`slate-600`) — body copy, descriptions.
- **Tertiary Text — Muted Slate** `#64748b` (`slate-500`) — field labels, table meta, timestamps, breadcrumbs.
- **Faint Text — Pale Slate** `#94a3b8` (`slate-400`) — placeholders, disabled/empty states, decorative icons.
- **Inverse Text — White** `#ffffff` / `slate-300` — text on the dark navy brand surface.

### Functional States

- **Success — Emerald** bg `#ecfdf5` / text `#047857` / dot `#10b981` — active status, granted consents, positive metrics.
- **Destructive — Red** bg `#fef2f2` / text `#b91c1c` / accent `#dc2626` — errors, disabled/danger actions, sign-out.
- **Warning — Amber** bg `#fffbeb` / text `#b45309` / dot `#f59e0b` — pending states (e.g. pending deletion), caution.
- **Info/Alt Accent — Violet** bg `#eef2ff`(-ish `indigo-50`) / text `#6d28d9` — a fourth metric "tone" used to differentiate dashboard tiles (applications) from the primary blue tone; not used for real alerts.

> `src/styles.css` previously defined `--color-primary: #2563eb` in a plain
> `@theme` block, but a later `@theme inline` block re-declared the same
> variable as `var(--primary)`, and CSS's last-write-wins cascade made the
> shadcn scaffold's near-black `:root` value win instead — so the base
> `Button` "default"/"link" variants silently rendered near-black rather
> than brand blue. This has been fixed by setting the underlying `--primary`
> (and the matching `--muted`, `--border`, `--destructive`, `--radius`
> tokens, which had the same dead-override problem) directly in `:root`/
> `.dark`, and removing the now-redundant `@theme` overrides. **Blue-600
> (`#2563eb`) is the confirmed brand primary**, both in the token system and
> in hand-built screens.

## 3. Typography Rules

### Hierarchy & Weights

Font: **Inter** (variable weight, via `@fontsource-variable/inter`), with
`"Noto Sans JP"` and system UI fonts as fallback — the product ships a
bilingual (EN/JA) UI. Inter's neutral, high-legibility character suits a
data-dense admin console; there is no display/serif pairing — this is a
single-typeface system.

| Role | Size | Weight | Line-height | Notes |
|---|---|---|---|---|
| Hero display (auth aside title) | `clamp(2rem, 3.6vw, 3.05rem)` | 600 | 1.12 | fluid, tight leading |
| App page title | 1.9rem | 600 | tight | admin shell page header |
| Auth/section page title | 1.7rem | 600 | tight | auth flow, settings pages |
| Section heading | 0.875rem (`text-sm`) | 600 | 1.3 | card/section headers |
| Stat value | 1.875–2.25rem (`text-3xl/4xl`) | 700–800 | 1.15 | dashboard metric numbers |
| Body | 0.925rem | 400 | 1.6 (leading-6/7) | descriptions, paragraph copy |
| Small body / table cell | 0.8125rem (`text-sm`/13px) | 400–500 | 1.5 | table rows, form hints |
| Eyebrow label | 0.7rem | 700, uppercase | 1.2 | small accent label above headings, colored blue |
| Meta / micro label | 0.625–0.68rem | 600–700 | 1.4 | badges, breadcrumbs, card captions |

### Spacing Principles

Letter-spacing is left at `normal`/tracking-tight defaults rather than wide
tracking — the one exception is uppercase eyebrow/meta labels, which rely on
`uppercase` + weight rather than added letter-spacing for emphasis. Line-height
is generous for body copy (1.5–1.6) to support long-form descriptions and
comfortable for headings (tight, ~1.1–1.3) to keep large numerals and titles
compact.

## 4. Component Stylings

### Buttons

Rounded-`md` (0.5rem) corners, `h-9` (36px) default height, border-transparent
by default. Variants: `default` (solid brand fill, white text),
`outline` (bordered, transparent bg, hover fills muted), `secondary` (subtle
gray fill), `ghost` (no fill until hover), `destructive` (low-opacity red
fill, not solid — red-10%/20% tint rather than a hard red block), `link`
(text-only, underline on hover). Active state nudges the button down 1px
(`active:translate-y-px`) for tactile feedback instead of a shadow change.
Focus is a 3px `ring-ring/50` halo, not an outline.

### Cards & Panels

Two consistent radii: `rounded-lg` (0.75rem) for standard cards,
`rounded-xl`/`rounded-2xl` (1rem+) for hero/feature panels like the auth
frame. Cards are near-white (`bg-white/90`–`/92`) over the gradient page
background, always with a hairline `border-slate-200/80` and a soft, large,
low-opacity shadow rather than a tight drop shadow — e.g.
`shadow-[0_18px_50px_-36px_rgb(15_23_42/42%)]`. Many cards add
`backdrop-blur-sm/xl`, reinforcing the "frosted glass over gradient" motif.
Hover-interactive cards (dashboard metrics) lift slightly
(`hover:-translate-y-0.5`) and darken their border/shadow.

### Navigation

The admin shell uses a fixed 248px left sidebar (desktop) with a sticky,
blurred, translucent header (`bg-white/82 backdrop-blur-xl`). Nav items are
`h-10` rounded-lg rows; the **active** item gets a solid deep-ink
(`bg-slate-950`) fill with white text and a soft shadow, inactive items are
muted slate text that goes white-on-hover. Breadcrumbs are tiny
(`text-xs font-semibold text-slate-500`) with `/` separators and blue-hover
links.

### Inputs & Forms

`h-12` (48px) rounded-`lg` inputs with a `slate-300` border, white/92%
background, and a distinctive focus treatment: border turns **blue-600** and
gains a soft `ring-blue-600/10` halo (not the generic gray ring shadcn
defaults to). Labels sit above fields, `text-sm font-medium`. Disabled state
is a flat `slate-100` fill at 60% opacity.

### Status Badges & Pills

A recurring pattern across admin list pages: a small `rounded-full` pill
combining a 6px colored dot + tinted background + colored text
(`bg-emerald-50 text-emerald-700` / red / amber), e.g. `active` / `disabled`
/ `pending_deletion` user status. Role/tag chips use a squarer
`rounded-md border border-slate-200 bg-white` treatment instead, to visually
distinguish "status" (pill) from "attribute" (chip).

### Auth Split-Screen (domain-specific)

The signed-out auth flow (login, MFA, consent, forgot-password) uses a
two-column "frame": a promotional **dark navy aside** (headline + copy, hidden
below 900px) beside a **white form column** (max 33rem). The frame itself is
`rounded-2xl`, bordered in translucent white, with a large soft shadow and
`backdrop-blur-xl` — collapses to a single full-bleed column with squared
corners below 520px (mobile).

## 5. Layout Principles

### Grid & Structure

- Admin console content: `max-w-[1500px]`, `248px` fixed sidebar + fluid
  main column (`grid-cols-[248px_minmax(0,1fr)]` at `lg` breakpoint).
- Auth flow: `max-w-[1160px]` outer container; inner frame splits
  `minmax(0,0.92fr) / minmax(440px,1.08fr)` (promo : form).
- Standard Tailwind breakpoints (no custom `screens` override): `sm`640 /
  `md`768 / `lg`1024 / `xl`1280 / `2xl`1536, plus two bespoke auth-only
  breakpoints at `900px` (drop the promo aside) and `520px` (go full-bleed
  mobile).

### Whitespace Strategy

Base unit is Tailwind's 4px scale. Page/section padding runs `p-5` (mobile)
→ `p-6` → `p-8` (`lg`) in the admin main area; cards use `p-4`–`p-6`
internally. Section-to-section gaps are `gap-4`–`gap-6`. Auth form padding is
notably generous (`px-14 py-12` desktop) to give the single-column form room
to breathe.

### Alignment & Visual Balance

Left-aligned, text-heavy admin layouts (tables, forms, dashboards) rather
than centered marketing-style layouts. The one centered/symmetric exception
is the auth flow, which centers its frame within the viewport. Icon+text
pairing is a near-universal micro-pattern (Tabler icon at 14–22px, `stroke`
1.7–1.8, followed by a label) for nav items, buttons, metric tiles, and
detail rows.

### Responsive Behavior & Touch

Mobile-first isn't the priority — this is a desktop-first admin tool that
degrades gracefully: the sidebar disappears below `lg`, the auth aside
disappears below `900px`, and dense multi-column grids collapse to 1–2
columns. Standard control height is `36–48px`, adequate for mouse/trackpad
use rather than oversized touch targets.

## 6. Design System Notes for Stitch Generation

### Language to Use

Describe this system as: *"clean enterprise IAM console, cool slate-and-white
canvas, one confident blue accent, functional-only color (emerald/red/amber),
frosted-glass translucent cards over a soft gradient background, dark navy
brand surfaces reserved for hero/promo moments, Inter typeface throughout,
Tabler line icons."* Avoid describing it as colorful, playful, or
warm — the palette is deliberately restrained.

### Color References

- Background: Cool Barely-There Slate `#f8fafc`
- Surface: White (90–92% opacity, blurred) `#ffffff`
- Border: Hairline Slate `#e2e8f0`
- Primary text: Deep Ink `#020617`
- Primary accent: Confident Blue `#2563eb` (hover `#1d4ed8`)
- Secondary accent: Teal Spark `#2dd4bf` (sparing use only)
- Dark brand surface: Ink Navy `#0a1020`
- Success / Destructive / Warning: Emerald `#10b981` / Red `#dc2626` / Amber `#f59e0b`

### Component Prompts

- *"A dashboard metric card: white rounded-xl card with a soft diffuse
  shadow, a rounded-xl icon badge tinted blue-50/blue-700, a large bold
  slate-950 number, a small slate-500 label beneath, and a thin progress bar
  in blue-600 at the bottom."*
- *"An admin sidebar nav item: full-width rounded-lg row, 40px tall, Tabler
  icon + label; active state solid deep-ink (#020617) fill with white text
  and a soft shadow, inactive state muted slate text turning white-on-hover."*
- *"A status pill: rounded-full badge with a small colored dot and tinted
  background — emerald for active, red for disabled, amber for pending —
  paired with bold small-caps text."*

### Incremental Iteration

When refining generated screens, push toward: less color (blue + slate should
dominate; keep emerald/red/amber strictly functional), softer/larger shadows
instead of hard drop-shadows, and translucent/blurred surfaces over solid
fills wherever a panel sits above the gradient page background.
