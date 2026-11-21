import { describe, expect, it } from 'bun:test'
import { adminThemeStyle } from './adminTheme'

// 製品の色計算を使わず、sRGB の定義で出力のコントラストを検算する。
function luminance(hex: string): number {
  const rgb = [1, 3, 5].map((offset) => Number.parseInt(hex.slice(offset, offset + 2), 16) / 255)
  const linear = rgb.map((value) =>
    value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4,
  )
  return linear[0]! * 0.2126 + linear[1]! * 0.7152 + linear[2]! * 0.0722
}
function contrast(a: string, b: string): number {
  const [low, high] = [luminance(a), luminance(b)].sort((x, y) => x - y)
  return (high! + 0.05) / (low! + 0.05)
}

describe('adminThemeStyle', () => {
  //spec:covers REQ-FRONTEND-005: 淡色、中間色、原色でも彩色面、本文と focus のコントラストを保つ。
  it('keeps arbitrary brand colors readable', () => {
    const colors = [
      '#ffffff',
      '#000000',
      '#ffff00',
      '#00ff00',
      '#0000ff',
      '#777777',
      '#757575',
      '#767676',
      '#663399',
    ]
    // RGB 立方体の各軸の端と中間を含む決定的なサンプルを追加する。
    for (const r of ['00', '40', '80', 'c0', 'ff']) {
      for (const g of ['00', '40', '80', 'c0', 'ff']) {
        for (const b of ['00', '40', '80', 'c0', 'ff']) colors.push(`#${r}${g}${b}`)
      }
    }
    for (const color of colors) {
      const style = adminThemeStyle({ primary_color: color, accent_color: color }) as Record<
        string,
        string
      >
      expect(style['--admin-theme-interactive']).toBe(color)
      expect(contrast(color, style['--admin-theme-on-color']!)).toBeGreaterThanOrEqual(4.5)
      for (const surface of ['#ffffff', '#faf9f6', style['--admin-theme-surface']!]) {
        expect(contrast(surface, style['--admin-theme-emphasis']!)).toBeGreaterThanOrEqual(4.5)
        expect(contrast(surface, style['--admin-theme-link']!)).toBeGreaterThanOrEqual(4.5)
        expect(contrast(surface, style['--admin-theme-focus']!)).toBeGreaterThanOrEqual(3)
      }
    }
  })

  //spec:covers REQ-FRONTEND-005: 未設定と不正な色は既定表示に退避し、任意の CSS を渡さない。
  it('uses defaults for absent and invalid colors', () => {
    expect(adminThemeStyle({ primary_color: 'red;display:none', accent_color: '#fff' })).toEqual(
      adminThemeStyle({}),
    )
    expect((adminThemeStyle({}) as Record<string, string>)['--admin-theme-interactive']).toBe(
      '#020617',
    )
  })
})
