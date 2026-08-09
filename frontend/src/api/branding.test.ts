import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { uploadTenantBrandingAsset } from './branding'

describe('branding API client', () => {
  afterEach(() => restoreGlobals())

  it('multipart uploadでもProblem Detailsのdetailとtypeを伝える', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue({
        ok: false,
        status: 422,
        json: mock().mockResolvedValue({
          type: 'urn:idmagic:error:invalid_branding_asset',
          title: 'Invalid branding asset',
          detail: 'The branding asset format is not supported.',
        }),
      }),
    )

    await expect(
      uploadTenantBrandingAsset('csrf', 'logo', new File(['logo'], 'logo.png')),
    ).rejects.toMatchObject({
      message: 'The branding asset format is not supported.',
      code: 'invalid_branding_asset',
    })
  })
})
