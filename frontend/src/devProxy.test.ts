import { describe, expect, it } from 'bun:test'
import viteConfig from '../vite.config'

function proxyMatches(path: string) {
  const proxy = viteConfig.server?.proxy ?? {}
  return Object.keys(proxy).some((context) =>
    context.startsWith('^') ? new RegExp(context).test(path) : path.startsWith(context),
  )
}

describe('development gateway', () => {
  it.each([
    '/realms/default/saml/metadata',
    '/realms/default/saml/idp/partner/metadata',
    '/realms/default/saml/signing-certificate.pem',
    '/realms/default/federationmetadata/2007-06/federationmetadata.xml',
    '/realms/default/wsfed',
    '/realms/default/trust/mex',
    '/realms/default/scim/v2',
  ])('proxies the published integration endpoint %s', (path) => {
    expect(proxyMatches(path)).toBe(true)
  })
})
