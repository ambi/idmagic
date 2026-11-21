import type { CSSProperties } from 'react'
import type { Branding } from '../types'
import { isValidHexColor } from './tenantBranding'

export function adminThemeStyle(branding: Branding): CSSProperties {
  const primary = validColor(branding.primary_color, '#020617')
  const accent = validColor(branding.accent_color, '#0f6f65')
  const surface = `#${channels(accent)
    .map((value) =>
      Math.round(value * 0.08 + 255 * 0.92)
        .toString(16)
        .padStart(2, '0'),
    )
    .join('')}`
  const surfaces = ['#ffffff', '#faf9f6', surface]
  const readable = (color: string) =>
    surfaces.every((background) => contrast(color, background) >= 4.5) ? color : '#020617'
  const onColor =
    contrast(primary, '#000000') >= contrast(primary, '#ffffff') ? '#000000' : '#ffffff'
  return {
    '--admin-theme-surface': surface,
    '--admin-theme-emphasis': readable(primary),
    '--admin-theme-interactive': primary,
    '--admin-theme-on-color': onColor,
    '--admin-theme-link': readable(accent),
    '--admin-theme-focus': readable(accent),
    '--primary': primary,
    '--primary-foreground': onColor,
    '--accent': surface,
    '--accent-foreground': readable(accent),
    '--ring': readable(accent),
  } as CSSProperties
}

function validColor(color: string | undefined, fallback: string): string {
  return color && isValidHexColor(color) ? color : fallback
}

function channels(color: string): number[] {
  return [1, 3, 5].map((offset) => Number.parseInt(color.slice(offset, offset + 2), 16))
}

// sRGB の相対輝度とコントラスト比: https://www.w3.org/WAI/WCAG21/Understanding/contrast-minimum
function luminance(color: string): number {
  const [r = 0, g = 0, b = 0] = channels(color).map((value) => {
    const channel = value / 255
    return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
  })
  return r * 0.2126 + g * 0.7152 + b * 0.0722
}

function contrast(a: string, b: string): number {
  const first = luminance(a)
  const second = luminance(b)
  return (Math.max(first, second) + 0.05) / (Math.min(first, second) + 0.05)
}
