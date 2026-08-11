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

    const catalog = extractTypeSpecCatalog(host.program, new Set(['Demo.PublicRecord']))

    expect(catalog.map((symbol) => symbol.name)).toEqual([
      'Example.Demo.Identifier',
      'Example.Demo.InternalRecord',
      'Example.Demo.PublicRecord',
      'Example.Demo.Result',
      'Example.Demo.Status',
    ])
    const publicRecord = catalog.find((symbol) => symbol.shortName === 'PublicRecord')
    expect(publicRecord?.apiExposed).toBe(true)
    expect(publicRecord?.properties[0]).toMatchObject({
      name: 'id',
      optional: false,
      doc: 'Stable identifier.',
      constraints: ['minLength: 3'],
    })
    expect(publicRecord?.references).toContain('Example.Demo.InternalRecord')
    expect(catalog.find((symbol) => symbol.shortName === 'InternalRecord')?.apiExposed).toBe(false)
  })
})
