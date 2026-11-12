import { tenantURL } from '../../api'

export const XML_FEDERATION_KEY_PATH = '/admin/keys/xml-federation'

export function xmlFederationKeyURL(): string {
  return tenantURL(XML_FEDERATION_KEY_PATH)
}
