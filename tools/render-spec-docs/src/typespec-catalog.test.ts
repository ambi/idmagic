import { describe, expect, it } from 'bun:test'
import { createTestHost } from '@typespec/compiler/testing'
import { extractTypeSpecCatalog } from './typespec-catalog.ts'

describe('extractTypeSpecCatalog', () => {
  it('includes repository-owned symbols that are not exposed by HTTP', async () => {
    const host = await createTestHost()
    host.addTypeSpecFile(
      'main.tsp',
      `namespace Example.Demo;

@doc("Public model.")
model PublicRecord {
  @doc("Stable identifier.")
  @minLength(3)
  id: string;
  internal: InternalRecord;
}

@doc("Internal model.")
model InternalRecord {
  secret: string;
}

enum Status { Ready, Done }
union Result { ok: PublicRecord, empty: null }
scalar Identifier extends string;

op inspect(): { ephemeral: string };

namespace Operations {
  model InspectHttpResponse { body: PublicRecord; }
}
`,
    )
    await host.compile('main.tsp')

    const catalog = extractTypeSpecCatalog(host.program, new Set(['Demo.PublicRecord']), '/test')

    expect(catalog.symbols.map((symbol) => symbol.name)).toEqual([
      'Example.Demo.Identifier',
      'Example.Demo.InternalRecord',
      'Example.Demo.PublicRecord',
      'Example.Demo.Result',
      'Example.Demo.Status',
    ])
    const publicRecord = catalog.symbols.find((symbol) => symbol.shortName === 'PublicRecord')
    expect(publicRecord?.apiExposed).toBe(true)
    expect(publicRecord?.properties[0]).toMatchObject({
      name: 'id',
      optional: false,
      doc: 'Stable identifier.',
      constraints: ['minLength: 3'],
    })
    expect(publicRecord?.references).toContain('Example.Demo.InternalRecord')
    expect(
      catalog.symbols.find((symbol) => symbol.shortName === 'InternalRecord')?.apiExposed,
    ).toBe(false)
  })

  it('reads the owning context and its API tag from the declaring directory', async () => {
    const host = await createTestHost()
    host.addTypeSpecFile(
      'spec/contexts/demo/models.tsp',
      `namespace Example.Contract {
  model DemoRecord { id: string; }
}
`,
    )
    host.addTypeSpecFile(
      'spec/contexts/demo/main.tsp',
      `import "./models.tsp";

@tag("Demo")
namespace Example.Demo {
}
`,
    )
    host.addTypeSpecFile('main.tsp', 'import "./spec/contexts/demo/main.tsp";\n')
    await host.compile('main.tsp')

    const catalog = extractTypeSpecCatalog(host.program, new Set(), '/test')

    expect(catalog.symbols.find((symbol) => symbol.shortName === 'DemoRecord')?.context).toBe(
      'demo',
    )
    expect(catalog.contextTags).toEqual({ demo: ['Demo'] })
  })

  it('leaves symbols declared outside a context directory unowned', async () => {
    const host = await createTestHost()
    host.addTypeSpecFile('main.tsp', 'namespace Example.Loose;\nmodel Free { id: string; }\n')
    await host.compile('main.tsp')

    const catalog = extractTypeSpecCatalog(host.program, new Set(), '/test')

    expect(catalog.symbols[0]?.context).toBeUndefined()
    expect(catalog.contextTags).toEqual({})
  })
})
