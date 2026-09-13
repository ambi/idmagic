import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { directTasks, parseMiseTasks } from './verification-tasks.ts'

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
const miseToml = await Bun.file(resolve(root, 'mise.toml')).text()
const tasks = parseMiseTasks(miseToml)
const members = (name: string) => directTasks(tasks, name)

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
    for (const task of ['check', 'check-spec', 'check-admin-scopes', 'check-api-compat']) {
      expect(config.tasks?.[task]?.depends).toContain('compile-spec')
    }
  })
})

describe('mise agent-guidance boundary', () => {
  it('runs repository-local guidance checks from the standard check suite', () => {
    expect(String(config.tasks?.['check-agent-guidance']?.run ?? '')).toContain(
      'check/src/runner.ts agent-guidance',
    )
    expect(members('check')).toContain('check-repository')
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

  /** 版が動くと検出も到達性の判定も動くため、可動指定や版範囲を許さない。 */
  it('pins both scanners to an exact version', () => {
    const exactVersion = /^\d+\.\d+\.\d+$/
    expect(config.tools?.['go:golang.org/x/vuln/cmd/govulncheck']).toMatch(exactVersion)
    expect(config.tools?.['aqua:google/osv-scanner']).toMatch(exactVersion)
  })

  it('runs the suppression checker from the standard check suite', () => {
    expect(String(config.tasks?.['check-vulnerability-suppressions']?.run ?? '')).toContain(
      'check/src/runner.ts vulnerability-suppressions',
    )
    expect(members('check')).toContain('check-repository')
  })

  /**
   * 走査は OSV への問い合わせを伴う。オフラインで verify が落ちる形にはしない。
   * verify と verify-serial は同じ一式を別の書き方で持つので、両方を見る。
   */
  it('keeps the network-dependent scans out of both offline verification suites', () => {
    for (const suite of ['verify', 'verify-serial']) {
      expect(members(suite)).not.toContain('audit-dependencies')
      expect(members(suite)).not.toContain('audit-go-reachability')
    }
  })
})

describe('mise Markdown link boundary', () => {
  it('runs the Markdown link checker from the standard check suite', () => {
    expect(String(config.tasks?.['check-links']?.run ?? '')).toContain('check/src/runner.ts links')
    expect(members('check')).toContain('check-repository')
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
  const verifySuite = [...members('verify'), ...members('verify-serial')]

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

  it('names a UI check the parallel suite has that the serial one lacks', () => {
    expect(members('verify')).toContain('build-ui')
    expect(members('verify-serial')).toContain('build-ui')
  })

  it('adds shadcn components with the version pinned in devDependencies', () => {
    expect(String(config.tasks?.['add-ui-component']?.run ?? '')).toContain('bun run add:component')
  })
})

describe('mise aggregate gate reporting', () => {
  /**
   * A gate that stops at the first failure hands back one failure per run, and
   * the next one costs another full run to find. The aggregates therefore run
   * their members with `mise run -c`, which reports every member that failed.
   */
  it('runs every aggregate gate with continue-on-error', () => {
    for (const name of ['check', 'verify', 'verify-spec']) {
      expect(String(config.tasks?.[name]?.run ?? '')).toContain('mise run -c')
    }
  })

  /** 検査構成は check の内側にだけ置き、広い suite はその公開入口を一度だけ呼ぶ。 */
  it('reuses the check aggregate from the wider suites', () => {
    for (const suite of ['verify', 'verify-spec']) {
      expect(members(suite).filter((task) => task === 'check')).toEqual(['check'])
    }
  })

  it('runs the same set of gates in the parallel and serial suites', () => {
    const parallel = members('verify')
    const serial = members('verify-serial')
    expect([...new Set(parallel)].sort()).toEqual([...new Set(serial)].sort())
  })

  /** The timing table walks the same membership the suite runs. */
  it('times a suite whose membership can be read from the command map', () => {
    expect(members('verify').length).toBeGreaterThan(1)
  })
})

describe('mise mutation testing boundary', () => {
  const mutation = String(config.tasks?.['test-go-mutation']?.run ?? '')
  const singleMutation = String(config.tasks?.['test-go-mutation-mutant']?.run ?? '')
  const canary = String(config.tasks?.['check-go-mutation-tool']?.run ?? '')

  it('mutates one Go package through the pinned tool', () => {
    expect(mutation).toContain('gomutants')
  })

  /** ツールの保守が止まったときに動く版が残らないと、証拠の作り方ごと失われる。 */
  it('pins the mutation tool to an exact version', () => {
    expect(config.tools?.['go:github.com/szhekpisov/gomutants']).toBe('0.6.1')
  })

  /**
   * runner の変更と operator の変更を同時に持ち込むと、結果の差をどちらに帰属すべきか
   * 分からない。移行境界では Gremlins の既定集合と同系統の 5 種類へ固定する。
   */
  it('keeps the comparable five-operator boundary', () => {
    for (const operator of [
      'ARITHMETIC_BASE',
      'CONDITIONALS_BOUNDARY',
      'CONDITIONALS_NEGATION',
      'INCREMENT_DECREMENT',
      'INVERT_NEGATIVES',
    ]) {
      expect(mutation).toContain(operator)
    }
  })

  /**
   * gomutants 0.6.1 の cache key は対象 package が import する package の変更を完全には
   * 追跡しない。既定では再利用せず、同一変更を反復するときだけ cache file を明示する。
   */
  it('uses conservative worker and cache defaults', () => {
    expect(mutation).toContain('$' + '{2:-2}')
    expect(mutation).toContain('$' + '{3:-off}')
  })

  it('normalizes a repository-relative directory to a Go package pattern', () => {
    expect(mutation).toContain('target="./$directory"')
  })

  it('writes the report outside the repository by default', () => {
    expect(mutation).toContain('idmagic-mutation-report.json')
    expect(mutation).toContain('--output "$report"')
  })

  it('reruns one stable mutant id without the incremental cache', () => {
    expect(singleMutation).toContain('--run-mutant-id')
    expect(singleMutation).toContain('--cache off')
  })

  it('provides a manual verdict canary', () => {
    expect(canary).toContain('mutation-canary')
  })

  /**
   * 探索的な fuzz と同じ理由でゲートには入れない。対象と無関係な変更でも生成される変異の
   * 集合が変わるうえ、実行時間が対象パッケージではなくリポジトリ全体の大きさに比例する。
   */
  it('keeps mutation testing out of the standard gates', () => {
    for (const suite of ['check', 'verify', 'verify-serial', 'verify-full']) {
      expect(members(suite)).not.toContain('test-go-mutation')
      expect(members(suite)).not.toContain('test-go-mutation-mutant')
      expect(members(suite)).not.toContain('check-go-mutation-tool')
    }
  })
})
