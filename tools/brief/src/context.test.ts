import { describe, expect, it } from 'bun:test'
import { resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { deriveContext, measureCandidates, neighborCandidates, renderContext } from './context.ts'

const root = resolve(import.meta.dir, '../../..')
const workspace = () => createWorkspaceSnapshot(root)

describe('task-specific context', () => {
  it('resolves a TypeSpec target to the same canonical module owner', async () => {
    const plan = await deriveContext(
      {
        kind: 'feature',
        targets: ['spec/modules/identity-management/models.tsp'],
        concerns: ['contract'],
      },
      workspace(),
    )
    expect(plan.reading.map((one) => one.location)).toContain(
      'docs/modules/identity-management/README.md#責務と境界',
    )
    expect(plan.tasks.map((one) => one.location)).toContain('check-contract-drift')
  })

  it('offers real module entries as opt-in neighbors and does not read their bodies', async () => {
    const snapshot = workspace()
    const plan = await deriveContext(
      { kind: 'feature', targets: ['backend/idmanagement/user'], concerns: [] },
      snapshot,
    )
    const neighbors = await neighborCandidates(plan, {
      ...snapshot,
      read: (path) => {
        if (path === 'docs/modules/provisioning/README.md')
          throw new Error('neighbor body must stay unopened')
        return snapshot.read(path)
      },
    })
    expect(neighbors.map((one) => one.location)).toContain('docs/modules/provisioning/README.md')
    expect(neighbors.some((one) => one.location.includes('identity-management'))).toBe(false)
    expect(
      neighbors.every((one) => one.basis === 'inferred' && one.origin.endsWith('#責務と境界')),
    ).toBe(true)
  })
  it('routes tooling with no normative reference to code guidance and narrow tools checks', async () => {
    const plan = await deriveContext(
      { kind: 'tooling', targets: ['tools/brief'], concerns: [] },
      workspace(),
    )
    expect(plan.reading.map((one) => one.location)).toContain(
      'docs/development/coding-style.md#使い方',
    )
    expect(plan.reading.map((one) => one.location)).toContain('tools/README.md')
    expect(plan.tasks.map((one) => one.location)).toContain('test-tools-file')
    expect(plan.tasks.map((one) => one.location)).toContain('typecheck-tools')
    expect(plan.reading.some((one) => one.location.startsWith('docs/modules/'))).toBe(false)
    expect(plan.tasks.map((one) => one.location)).not.toContain('test-ui-e2e')
    for (const candidate of [...plan.reading, ...plan.tasks]) {
      expect(candidate.origin).not.toBe('')
      expect(candidate.reason).not.toBe('')
    }
  })

  it('keeps document changes away from production code rules and contracts', async () => {
    const plan = await deriveContext(
      { kind: 'docs', targets: ['docs/development/writing-language.md'], concerns: [] },
      workspace(),
    )
    expect(plan.reading.map((one) => one.location)).toContain(
      'docs/development/writing-language.md',
    )
    expect(plan.tasks.map((one) => one.location)).toContain('check-links')
    expect(plan.tasks.map((one) => one.location)).not.toContain('lint-go')
    expect(plan.reading.some((one) => one.location.includes('coding-style'))).toBe(false)
    expect(plan.reading.some((one) => one.location.startsWith('spec/'))).toBe(false)
    expect(plan.reading.some((one) => one.location.includes('責務と境界'))).toBe(false)
  })

  it('uses the responsibility table for aliased modules and keeps neighbors unopened', async () => {
    const plan = await deriveContext(
      { kind: 'feature', targets: ['backend/idmanagement/user'], concerns: [] },
      workspace(),
    )
    const locations = plan.reading.map((one) => one.location)
    expect(locations).toContain('docs/modules/identity-management/README.md#責務と境界')
    expect(locations).toContain('docs/modules/identity-management/quality.md')
    expect(locations).toContain('docs/development/specification-first-workflow.md#仕様影響の宣言')
    expect(locations).toContain('docs/development/testing.md#主要ユースケース')
    expect(locations).toContain('docs/modules/identity-management/README.md#文書')
    expect(locations.some((one) => one.startsWith('spec/modules/identity-management/'))).toBe(true)
    expect(locations.some((one) => one.includes('docs/modules/oauth2/'))).toBe(false)
    expect(locations.some((one) => one.includes('identity-management/observability.md'))).toBe(
      false,
    )
  })

  it('adds boundary selection and known risks only for an explicit boundary concern', async () => {
    const plan = await deriveContext(
      { kind: 'refactor', targets: ['backend/idmanagement/user'], concerns: ['boundary'] },
      workspace(),
    )
    const locations = plan.reading.map((one) => one.location)
    expect(locations).toContain('docs/design/application/design-guidelines.md#境界を選ぶ判断手順')
    expect(locations).toContain('docs/modules/README.md#モジュール')
    expect(locations).toContain('docs/design/application/backend.md#モジュール間の依存規則')
    expect(locations).toContain('docs/modules/identity-management/risks.md')
    expect(plan.tasks.map((one) => one.location)).toContain('check-boundaries')
  })

  it('follows the effect index to the owner of commit and notification semantics', async () => {
    const plan = await deriveContext(
      { kind: 'refactor', targets: ['backend/idmanagement/user'], concerns: ['effects'] },
      workspace(),
    )
    expect(plan.reading.map((one) => one.location)).toContain(
      'docs/design/application/design-guidelines.md#一貫性の境界と永続化',
    )
    expect(plan.reading).toContainEqual({
      location: 'docs/modules/identity-management/observability.md#経路',
      origin: expect.stringContaining('docs/modules/identity-management/README.md:'),
      reason: expect.stringContaining('確定'),
      basis: 'inferred',
    })
    expect(plan.reading.some((one) => one.location.includes('docs/modules/provisioning/'))).toBe(
      false,
    )
  })

  it('does not match a similarly named path outside a declared prefix', async () => {
    const plan = await deriveContext(
      { kind: 'tooling', targets: ['toolshed'], concerns: [] },
      workspace(),
    )
    expect(plan.tasks.map((one) => one.location)).not.toContain('test-tools-file')
    expect(plan.reading.map((one) => one.location)).not.toContain('tools/README.md')
  })

  it('derives a new owner link from the canonical index without a code registry', async () => {
    const snapshot = workspace()
    const index = 'docs/modules/identity-management/README.md'
    const plan = await deriveContext(
      { kind: 'refactor', targets: ['backend/idmanagement/user'], concerns: ['effects'] },
      {
        ...snapshot,
        read: (path) =>
          path === index
            ? Promise.resolve(
                [
                  '# IdManagement',
                  '',
                  '## 機能',
                  '',
                  '| 機能 | 内容 |',
                  '| --- | --- |',
                  '| [通知の設定](features/user.md) | 通知の確定 |',
                  '',
                  '## 文書',
                  '',
                  '| 文書 | 説明 |',
                  '| --- | --- |',
                  '| [確定の所有](data.md) | 通知と確定の境界 |',
                  '',
                ].join('\n'),
              )
            : snapshot.read(path),
      },
    )
    expect(plan.reading.map((one) => one.location)).toContain(
      'docs/modules/identity-management/data.md',
    )
    // 文書の節の外にある機能の表は、作用の索引として読まない。
    expect(plan.reading.map((one) => one.location)).not.toContain(
      'docs/modules/identity-management/features/user.md',
    )
    expect(plan.reading.map((one) => one.location)).not.toContain(
      'docs/modules/identity-management/observability.md#経路',
    )
  })
})

describe('reading size', () => {
  it('measures the lead of a section and lists its children with their sizes', async () => {
    const snapshot = workspace()
    const plan = await deriveContext(
      { kind: 'refactor', targets: ['backend/authentication'], concerns: [] },
      snapshot,
    )
    const sizes = await measureCandidates(plan, snapshot)
    const cycle = sizes.get('docs/development/specification-first-workflow.md#3-開発サイクル')!
    const source = await snapshot.read('docs/development/specification-first-workflow.md')
    const section = source.slice(source.indexOf('## 3. 開発サイクル'), source.indexOf('## 4. '))
    expect(cycle.bytes).toBeGreaterThan(0)
    const childBytes = cycle.children.reduce((sum, one) => sum + one.bytes, 0)
    // 本体は子節を含まない。本体と子節を足すと節の全体になる。
    expect(cycle.bytes + childBytes).toBeLessThanOrEqual(Buffer.byteLength(section))
    expect(cycle.children.map((one) => one.anchor)).toContain('仕様にない振る舞いの分類')
    expect(cycle.children.every((one) => one.bytes > 0)).toBe(true)

    const rendered = renderContext(plan, sizes).join('\n')
    expect(rendered).toMatch(/3-開発サイクル \[explicit\] \d+ KB/)
    expect(rendered).toMatch(/ {2}- #仕様にない振る舞いの分類 \d+ KB/)
    expect(rendered).toMatch(/Lead sections total \d+ KB/)
  })

  it('leaves TypeSpec out when the change declares no specification impact', async () => {
    const withoutImpact = await deriveContext(
      { kind: 'refactor', targets: ['backend/authentication'], concerns: [], specification: false },
      workspace(),
    )
    expect(withoutImpact.reading.some((one) => one.location.startsWith('spec/'))).toBe(false)
    expect(withoutImpact.reading.map((one) => one.location)).toContain(
      'docs/modules/authentication/README.md#責務と境界',
    )

    const contract = await deriveContext(
      {
        kind: 'refactor',
        targets: ['backend/authentication'],
        concerns: ['contract'],
        specification: false,
      },
      workspace(),
    )
    expect(contract.reading.some((one) => one.location.startsWith('spec/'))).toBe(true)
  })
})
