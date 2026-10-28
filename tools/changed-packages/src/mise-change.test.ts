import { describe, expect, it } from 'bun:test'
import { miseChange } from './mise-change.ts'

const BASE = `
[tools]
go = "1.26.0"

[tasks.lint-go]
run = "golangci-lint run ./..."

[tasks.test-go]
run = "bun run tools/verification/src/main.ts -- ./..."

[tasks.install-ui]
run = "bun install"

[tasks.lint-ui]
depends = ["install-ui"]
run = "bun run lint"

[tasks.verify-ui]
depends = ["lint-ui"]

[tasks.setup-tools]
run = "bun install"

[tasks.test-tools]
depends = ["setup-tools"]
run = "bun test"

[tasks.typecheck-tools]
run = "bun run typecheck"

[tasks.lint-tools]
run = "bun run lint"

[tasks.deploy-k8s]
run = "kubectl apply -k infra/k8s"
`

describe('miseChange', () => {
  it('selects every layer when a table other than the tasks changes', () => {
    const change = miseChange(BASE, BASE.replace('go = "1.26.0"', 'go = "1.26.1"'))
    expect(change.all).toBe(true)
    expect(change.reasons).toEqual(['mise.toml changes [tools], which may affect every layer'])
  })

  it('selects the layers whose checks reach a changed task', () => {
    expect(miseChange(BASE, BASE.replace('-- ./...', '-- -race ./...')).layers).toEqual(['go'])
    expect(
      miseChange(
        BASE,
        BASE.replace(
          'run = "bun install"\n\n[tasks.lint-ui]',
          'run = "bun install --frozen-lockfile"\n\n[tasks.lint-ui]',
        ),
      ).layers,
    ).toEqual(['ui'])
  })

  it('selects no layer for a task no check reaches', () => {
    const change = miseChange(BASE, BASE.replace('kubectl apply', 'kubectl apply --prune'))
    expect(change).toEqual({ all: false, reasons: [], layers: [] })
  })

  // 起点の定義が変わる場合も、その層を選ぶ。
  it('selects the layer of a check whose dependencies were taken out', () => {
    const after = BASE.replace('depends = ["setup-tools"]\nrun = "bun test"', 'run = "bun test"')
    expect(miseChange(BASE, after).layers).toEqual(['tools'])
  })

  it('selects every layer when either side cannot be read', () => {
    expect(miseChange(BASE, '[tasks\n').all).toBe(true)
    expect(miseChange('[tasks\n', BASE).all).toBe(true)
  })

  it('コメントとキー順の違いでは層を選ばない', () => {
    const before =
      '[tools]\ngo = "1"\nbun = "2"\n[tasks.test-go]\nrun = "test"\ndescription = "test"'
    const after =
      '# comment\n[tools]\nbun = "2"\ngo = "1"\n[tasks.test-go]\ndescription = "test"\nrun = "test"'
    expect(miseChange(before, after)).toEqual({ all: false, reasons: [], layers: [] })
  })

  it('mise run の推移的な呼び出しと共有依存の変更を拾う', () => {
    const before =
      '[tasks.verify-ui]\nrun = "mise run typecheck-tools"\n[tasks.typecheck-tools]\ndepends = ["setup-tools"]\n[tasks.setup-tools]\nrun = "old"'
    const after = before.replace('"old"', '"new"')
    expect(miseChange(before, after).layers).toEqual(['ui', 'tools'])
  })

  it('タスクの追加と削除を対称に扱う', () => {
    const configurations = [
      ['[tasks.verify-ui]\ndepends = ["child"]', '[tasks.child]\nrun = "new"', ['ui']],
      ['', '[tasks.test-go]\nrun = "new"', ['go']],
      ['', '[tasks.unused]\nrun = "new"', []],
    ] as const
    for (const [before, added, layers] of configurations) {
      const after = `${before}\n${added}`
      expect(miseChange(before, after)).toEqual({ all: false, reasons: [], layers: [...layers] })
      expect(miseChange(after, before)).toEqual(miseChange(before, after))
    }
  })
})
