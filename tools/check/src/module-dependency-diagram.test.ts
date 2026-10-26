import { describe, expect, it } from 'bun:test'
import {
  MODULE_DEPENDENCY_DIAGRAM_END,
  MODULE_DEPENDENCY_DIAGRAM_START,
  moduleDependencyDiagram,
  moduleGroups,
  withModuleDependencyDiagram,
} from './module-dependency-diagram.ts'

const HEADER =
  '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |\n| --- | --- | --- | --- | --- |'

const README = [
  '# モジュール設計文書',
  '',
  '### 基盤',
  '',
  HEADER,
  '| [System](system/README.md) | 組み立て地点 | なし | `backend/cmd` | 起動 |',
  '| [Tenancy](tenancy/README.md) | `internal` | なし | `backend/tenancy` | テナント |',
  '',
  '### 連携プロトコル',
  '',
  HEADER,
  '| [OAuth2](oauth2/README.md) | `internal` | なし | `backend/oauth2` | OAuth |',
  '',
  '| 文書 | 内容 |',
  '| --- | --- |',
  '| [用語集](glossary.md) | 語 |',
  '',
  MODULE_DEPENDENCY_DIAGRAM_START,
  'stale',
  MODULE_DEPENDENCY_DIAGRAM_END,
  '',
].join('\n')

describe('module dependency diagram', () => {
  // 区分は直前の H3 で決まり、責務表でない表の行と組み立て地点は図に入らない。
  it('groups modules by the heading above their responsibility table', () => {
    expect(moduleGroups(README, 'System')).toEqual([
      { name: '基盤', modules: ['Tenancy'] },
      { name: '連携プロトコル', modules: ['OAuth2'] },
    ])
  })

  it('draws each group as a subgraph and each import as an edge', () => {
    const diagram = moduleDependencyDiagram(moduleGroups(README, 'System'), [
      { sourceModule: 'OAuth2', targetModule: 'Tenancy' },
    ])
    expect(diagram.split('\n')).toEqual([
      '```mermaid',
      'flowchart LR',
      '  subgraph group0["基盤"]',
      '    Tenancy',
      '  end',
      '  subgraph group1["連携プロトコル"]',
      '    OAuth2',
      '  end',
      '  OAuth2 --> Tenancy',
      '```',
    ])
  })

  it('replaces only the text between the markers, and reports missing markers', () => {
    const replaced = withModuleDependencyDiagram(README, 'DIAGRAM')
    expect(replaced).toContain(
      `${MODULE_DEPENDENCY_DIAGRAM_START}\n\nDIAGRAM\n\n${MODULE_DEPENDENCY_DIAGRAM_END}`,
    )
    expect(replaced).not.toContain('stale')
    expect(replaced?.startsWith('# モジュール設計文書')).toBe(true)
    expect(withModuleDependencyDiagram('# 印のない文書\n', 'DIAGRAM')).toBeUndefined()
  })

  it('図の関係ラベルを保持し、コード上の参照がない関係は破線で描く', () => {
    const groups = moduleGroups(README, 'System')
    const dependencies = [{ sourceModule: 'OAuth2', targetModule: 'Tenancy' }]
    const source = withModuleDependencyDiagram(
      README,
      '```mermaid\nflowchart LR\n  OAuth2 -->|"Tenant を使う"| Tenancy\n  Tenancy -.->|"設定の変更を通知"| OAuth2\n  Tenancy --> OAuth2\n```',
    )!
    const diagram = moduleDependencyDiagram(groups, dependencies, source)
    expect(diagram).toContain('OAuth2 -->|"Tenant を使う"| Tenancy')
    expect(diagram).toContain('Tenancy -.->|"設定の変更を通知"| OAuth2')
    expect(diagram).not.toContain('Tenancy --> OAuth2')
    expect(
      moduleDependencyDiagram(groups, dependencies, withModuleDependencyDiagram(source, diagram)!),
    ).toBe(diagram)
    expect(moduleDependencyDiagram(groups, [], source)).toContain(
      'OAuth2 -.->|"Tenant を使う"| Tenancy',
    )
  })

  it('責務表にないモジュールと、同じ組に重複した関係ラベルを拒否する', () => {
    const groups = moduleGroups(README, 'System')
    for (const relations of [
      '  OAuth2 -.->|"公開 HTTP"| Unknown',
      '  OAuth2 -->|"設定"| Tenancy\n  OAuth2 -.->|"通知"| Tenancy',
    ]) {
      const source = withModuleDependencyDiagram(README, `\`\`\`mermaid\n${relations}\n\`\`\``)!
      expect(() => moduleDependencyDiagram(groups, [], source)).toThrow()
    }
  })
})
