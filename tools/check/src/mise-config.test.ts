import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'

type MiseConfig = {
  tools?: Record<string, unknown>
  env?: Record<string, unknown>
  tasks?: Record<string, { depends?: string[]; run?: unknown; tools?: Record<string, unknown> }>
}

const root = resolve(import.meta.dir, '../../..')
const config = Bun.TOML.parse(await Bun.file(resolve(root, 'mise.toml')).text()) as MiseConfig
const { packageManager } = (await Bun.file(resolve(root, 'frontend/package.json')).json()) as {
  packageManager?: string
}
const dockerfile = await Bun.file(resolve(root, 'frontend/Dockerfile')).text()

describe('mise operational tool boundary', () => {
  it('does not provision PostgreSQL client tools', () => {
    expect(config.tools?.postgres).toBeUndefined()
    expect(Object.keys(config.env ?? {}).filter((name) => name.startsWith('POSTGRES_'))).toEqual([])
    for (const task of Object.values(config.tasks ?? {})) {
      expect(task.tools?.postgres).toBeUndefined()
    }
  })
})

describe('mise generated OpenAPI dependencies', () => {
  it('compiles the specification before every parallel verification consumer', () => {
    for (const task of ['check-spec', 'check-admin-scopes', 'check-api-compat']) {
      expect(config.tasks?.[task]?.depends).toContain('compile-spec')
    }
  })
})

describe('mise agent-guidance boundary', () => {
  it('runs repository-local guidance checks from the standard check suite', () => {
    expect(config.tasks?.['check-agent-guidance']?.run).toBeDefined()
    expect(config.tasks?.check?.depends).toContain('check-agent-guidance')
  })
})

describe('mise dependency audit boundary', () => {
  const auditDependencies = String(config.tasks?.['audit-dependencies']?.run ?? '')

  /** 引数の値まで取り出す。部分一致だと `tools/bun.lock.disabled` のような取り違えを通す。 */
  const flagValues = (name: string) =>
    [...auditDependencies.matchAll(new RegExp(`--${name}=(\\S+)`, 'g'))].map(
      (match) => match[1] ?? '',
    )

  it('names every lockfile the repository owns, and only those', () => {
    expect(flagValues('lockfile').sort()).toEqual(['frontend/bun.lock', 'go.mod', 'tools/bun.lock'])
  })

  it('applies the suppression configuration the checker validates', () => {
    expect(flagValues('config')).toEqual(['osv-scanner.toml'])
  })

  /**
   * OSV-Scanner は go/types を自分のバイナリに焼き込むので、go.mod の言語版を配布バイナリの
   * ビルド Go が下回ると解析が壊れ、それが検出結果には現れない。到達性は govulncheck が持つ。
   * `--call-analysis=none` は none という言語名として一般エラーになるだけで無効化にならない。
   */
  it('disables Go call analysis with the flag that actually disables it', () => {
    expect(flagValues('no-call-analysis')).toEqual(['go'])
    expect(auditDependencies).not.toContain('--call-analysis=none')
  })

  it('hands Go reachability to govulncheck', () => {
    expect(String(config.tasks?.['audit-go-reachability']?.run ?? '')).toContain('govulncheck')
  })

  /** 版が動くと検出も到達性の判定も動く。Design の議論はこの 2 つの版を前提にしている。 */
  it('pins both scanners to an exact version', () => {
    expect(config.tools?.['go:golang.org/x/vuln/cmd/govulncheck']).toBe('1.7.0')
    expect(config.tools?.['aqua:google/osv-scanner']).toBe('2.5.1')
  })

  it('runs the suppression checker from the standard check suite', () => {
    expect(config.tasks?.['check-vulnerability-suppressions']?.run).toBeDefined()
    expect(config.tasks?.check?.depends).toContain('check-vulnerability-suppressions')
  })

  /**
   * 走査は OSV への問い合わせを伴う。オフラインで verify が落ちる形にはしない。
   * verify と verify-serial は同じ一式を別の書き方で持つので、両方を見る。
   */
  it('keeps the network-dependent scans out of both offline verification suites', () => {
    const suite = [
      ...(config.tasks?.verify?.depends ?? []),
      ...[config.tasks?.['verify-serial']?.run ?? []].flat().map(String),
    ].join('\n')
    expect(suite).not.toContain('audit-dependencies')
    expect(suite).not.toContain('audit-go-reachability')
  })
})

describe('mise Markdown link boundary', () => {
  it('runs the Markdown link checker from the standard check suite', () => {
    expect(config.tasks?.['check-links']?.run).toBeDefined()
    expect(config.tasks?.check?.depends).toContain('check-links')
  })
})

describe('Bun version boundary', () => {
  /**
   * Bun は三か所が同じ版を指して初めて再現する。mise が手元と CI の実行系を選び、
   * `packageManager` が bun install の版を宣言し、Dockerfile が配信用イメージを構築する。
   * 一つだけ更新しても各所は動き続け、ずれは別の版が壊れたときにしか現れない。
   */
  const declared = String(config.tools?.bun ?? '')

  it('pins an exact version rather than a range', () => {
    expect(declared).toMatch(/^\d+\.\d+\.\d+$/)
  })

  it('declares the same version in the UI package manifest', () => {
    expect(packageManager).toBe(`bun@${declared}`)
  })

  it('builds the UI container from the same version', () => {
    expect(dockerfile).toContain(`FROM oven/bun:${declared}-alpine AS build`)
  })
})

describe('mise UI verification boundary', () => {
  /**
   * `verify-ui` は UI だけを触るときの入口で、`verify` はリポジトリ全体の入口である。
   * 片方にしか無い検査は、その入口を使った人にだけ通ったように見える。
   */
  const verifySuite = [
    ...(config.tasks?.verify?.depends ?? []),
    ...[config.tasks?.['verify-serial']?.run ?? []].flat().map(String),
  ].join('\n')

  it('checks dependency declarations and types from the UI entry point', () => {
    expect(config.tasks?.['check-ui-dependencies']?.run).toBeDefined()
    expect(config.tasks?.['verify-ui']?.depends).toContain('check-ui-dependencies')
    expect(config.tasks?.['verify-ui']?.depends).toContain('typecheck-ui')
  })

  it('runs every UI check of verify-ui from both repository-wide suites', () => {
    for (const task of config.tasks?.['verify-ui']?.depends ?? []) {
      expect(verifySuite).toContain(task)
    }
  })

  it('adds shadcn components with the version pinned in devDependencies', () => {
    expect(String(config.tasks?.['add-ui-component']?.run ?? '')).toContain('bun run add:component')
  })
})
