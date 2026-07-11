import { useEffect, useState, type CSSProperties } from 'react'
import { getBranding } from '../api'
import type { Branding } from '../types'

// useTenantBranding は hosted UI (login / consent / device / account portal) が
// 消費する branding を取得する (wi-89, ADR-096)。取得に失敗しても例外を投げず、
// 呼び出し側は空の Branding (システム既定) を使い続ける — branding の解決失敗で
// 画面自体を止めない。
export function useTenantBranding(): Branding {
  const [branding, setBranding] = useState<Branding>({})
  useEffect(() => {
    let cancelled = false
    getBranding()
      .then((result) => {
        if (!cancelled) setBranding(result)
      })
      .catch(() => {
        // no-op: システム既定のまま表示する。
      })
    return () => {
      cancelled = true
    }
  }, [])
  return branding
}

// tenantBrandStyle は primary/accent color を CSS custom properties としてのみ
// 注入する (ADR-096 決定 3: 任意プロパティ名・任意宣言は受け付けない)。
export function tenantBrandStyle(branding: Branding): CSSProperties {
  const style: Record<string, string> = {}
  if (branding.primary_color) style['--tenant-brand-primary'] = branding.primary_color
  if (branding.accent_color) style['--tenant-brand-accent'] = branding.accent_color
  return style as CSSProperties
}
