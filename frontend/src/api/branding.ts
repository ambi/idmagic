import type { Branding, BrandingUpdateInput, TenantBrandingAssetKind } from '../types'
import { AuthenticationAPIError, adminRequest, request } from './core'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { getCurrentLocale } from '../lib/i18n/currentLocale'

const uiFallback = () => commonDictionary[getCurrentLocale()].networkError

// GetTenantBranding は解決済みテナントの hosted UI branding を返す公開 endpoint。
// 未認証の login / consent / device 画面からも呼ぶため認証を要求しない。
export async function getBranding(): Promise<Branding> {
  return request<Branding>('/api/branding')
}

export async function updateBranding(
  csrfToken: string,
  input: BrandingUpdateInput,
): Promise<Branding> {
  return request('/api/admin/v1/tenant/branding', adminRequest(csrfToken, 'PUT', input))
}

export async function uploadTenantBrandingAsset(
  csrfToken: string,
  kind: TenantBrandingAssetKind,
  file: File,
): Promise<Branding> {
  const form = new FormData()
  form.set('file', file)
  // 本文は multipart なので Content-Type をブラウザーに任せ、Bearer は共通の処理が付ける。
  const body = await request<{ branding?: Branding }>(
    `/api/admin/v1/tenant/branding/assets/${kind}`,
    { method: 'POST', headers: { 'X-CSRF-Token': csrfToken }, body: form },
  )
  if (!body?.branding) {
    throw new AuthenticationAPIError(uiFallback())
  }
  return body.branding
}

export async function deleteTenantBrandingAsset(
  csrfToken: string,
  kind: TenantBrandingAssetKind,
): Promise<Branding> {
  return (
    await request<{ branding: Branding }>(
      `/api/admin/v1/tenant/branding/assets/${kind}`,
      adminRequest(csrfToken, 'DELETE'),
    )
  ).branding
}
