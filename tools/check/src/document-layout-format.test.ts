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
      path: 'docs/design/product-overview.md',
      message: '配置図に定義済み文書のパスがない: docs/design/product-overview.md',
    })
    expect(findings).toContainEqual({
      path: 'docs/domain/<context>/standards.md',
      message: '配置図に定義済み文書のパスがない: docs/domain/<context>/standards.md',
    })
  })
})
