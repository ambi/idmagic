import { describe, expect, it } from 'bun:test'
import { verifyDocumentLayout } from './document-layout-format.ts'

describe('文書配置図', () => {
  it('配置図にない定義済み文書のパスを報告する', () => {
    const findings = verifyDocumentLayout(`# 仕様フォーマット

## 1. 配置

\`\`\`text
docs/
  README.md
  domain/
    README.md
    <context>/
      README.md
\`\`\`
`)

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
      path: 'docs/domain/<context>/standards.md',
      message: '配置図に定義済み文書のパスがない: docs/domain/<context>/standards.md',
    })
  })

  it('機能仕様と内部設計の形式の段を配置図に求め、旧形式の種別は求めない', () => {
    const paths = verifyDocumentLayout('# 仕様フォーマット\n').map((finding) => finding.path)

    expect(paths).toContain('docs/domain/<context>/design/decisions.md')
    expect(paths).toContain('docs/domain/<context>/<group>/README.md')
    expect(paths).toContain('docs/domain/<context>/<group>/<feature>/acceptance.feature.md')
    expect(paths).not.toContain('docs/domain/<context>/internals.md')
  })
})
