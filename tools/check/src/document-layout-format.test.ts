import { describe, expect, it } from 'bun:test'
import { verifyDocumentLayout } from './document-layout-format.ts'

describe('文書配置図', () => {
  it('配置図にない定義済み文書のパスを報告する', () => {
    const findings = verifyDocumentLayout(`# 文書ガイド

### 配置

\`\`\`text
docs/
  README.md
  modules/
    README.md
    <module>/
      README.md
\`\`\`
`)
    const reported = findings.map((finding) => finding.path)

    // 配置図に書いた文書は報告しない。配置図を見つけられなければ、これらも欠落として報告される。
    expect(reported).not.toContain('docs/README.md')
    expect(reported).not.toContain('docs/modules/README.md')
    expect(reported).not.toContain('docs/modules/<module>/README.md')
    expect(findings).toContainEqual({
      path: 'docs/requirements/product-overview.md',
      message: '配置図に定義済み文書のパスがない: docs/requirements/product-overview.md',
    })
    expect(findings).toContainEqual({
      path: 'docs/design/architecture/system-boundary.md',
      message: '配置図に定義済み文書のパスがない: docs/design/architecture/system-boundary.md',
    })
    expect(findings).toContainEqual({
      path: 'docs/design/verification/system-acceptance.md',
      message: '配置図に定義済み文書のパスがない: docs/design/verification/system-acceptance.md',
    })
    expect(findings).toContainEqual({
      path: 'docs/modules/<module>/standards.md',
      message: '配置図に定義済み文書のパスがない: docs/modules/<module>/standards.md',
    })
  })

  it('機能仕様と内部設計の形式の段を配置図に求め、旧形式の種別は求めない', () => {
    const paths = verifyDocumentLayout('# 仕様フォーマット\n').map((finding) => finding.path)

    expect(paths).toContain('docs/modules/<module>/design/decisions.md')
    expect(paths).toContain('docs/modules/<module>/<group>/README.md')
    expect(paths).toContain('docs/modules/<module>/<group>/<feature>/acceptance.feature.md')
    expect(paths).not.toContain('docs/modules/<module>/internals.md')
  })
})
