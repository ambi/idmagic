import {
  MAX_IMAGE_UPLOAD_BYTES,
  isSupportedImageType,
  safeInternalAssetURL,
  validateImageUploadFile,
  type ImageUploadValidationError,
} from './imageUpload'

const TENANT_BRANDING_ASSET_PATH_SEGMENT = '/tenant-branding-assets/'

export const MAX_TENANT_BRANDING_ASSET_BYTES = MAX_IMAGE_UPLOAD_BYTES

export type TenantBrandingAssetFileValidationError = ImageUploadValidationError

export function isTenantBrandingAssetFile(file: File): boolean {
  return isSupportedImageType(file)
}

export async function validateTenantBrandingAssetFile(
  file: File,
): Promise<TenantBrandingAssetFileValidationError | null> {
  return validateImageUploadFile(file, MAX_TENANT_BRANDING_ASSET_BYTES)
}

export function safeTenantBrandingAssetURL(value?: string): string {
  return safeInternalAssetURL(value, TENANT_BRANDING_ASSET_PATH_SEGMENT)
}

const HEX_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/

export function isValidHexColor(value: string): boolean {
  return HEX_COLOR_PATTERN.test(value)
}

const HTTPS_URL_PATTERN = /^https:\/\//

export function isHTTPSURL(value: string): boolean {
  return HTTPS_URL_PATTERN.test(value)
}
