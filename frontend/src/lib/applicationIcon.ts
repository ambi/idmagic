import {
  MAX_IMAGE_UPLOAD_BYTES,
  isSupportedImageType,
  safeInternalAssetURL,
  validateImageUploadFile,
  type ImageUploadValidationError,
} from './imageUpload'

const APPLICATION_ICON_PATH_SEGMENT = '/application-icons/'

export const MAX_APPLICATION_ICON_BYTES = MAX_IMAGE_UPLOAD_BYTES

export type ApplicationIconFileValidationError = ImageUploadValidationError

export function isApplicationIconFile(file: File): boolean {
  return isSupportedImageType(file)
}

export async function validateApplicationIconFile(
  file: File,
): Promise<ApplicationIconFileValidationError | null> {
  return validateImageUploadFile(file, MAX_APPLICATION_ICON_BYTES)
}

export function safeApplicationIconURL(value?: string): string {
  return safeInternalAssetURL(value, APPLICATION_ICON_PATH_SEGMENT)
}
