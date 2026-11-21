import {
  createContext,
  useContext,
  useEffect,
  useState,
  useRef,
  useCallback,
  type CSSProperties,
  type ReactNode,
} from 'react'
import { getBranding, tenantBasePath } from '../api'
import { adminThemeStyle } from '../lib/adminTheme'
import type { Branding } from '../types'

const AdminThemeContext = createContext<{
  branding: Branding
  style?: CSSProperties
  update?: (branding: Branding) => void
}>({
  branding: {},
})

export function AdminTheme({ children }: { children: ReactNode }) {
  const realm = tenantBasePath()
  const [loaded, setLoaded] = useState<{ realm: string; branding: Branding }>()
  const revision = useRef(0)
  const update = useCallback(
    (branding: Branding) => {
      if (tenantBasePath() !== realm) return
      revision.current += 1
      setLoaded({ realm, branding })
    },
    [realm],
  )
  useEffect(() => {
    let cancelled = false
    const startedAt = revision.current
    getBranding()
      .then((branding) => {
        if (!cancelled && revision.current === startedAt)
          setLoaded({ realm, branding: branding ?? {} })
      })
      .catch(() => {
        // 外観の取得失敗で管理操作を止めない。
        if (!cancelled && revision.current === startedAt) setLoaded({ realm, branding: {} })
      })
    return () => {
      cancelled = true
    }
  }, [realm])
  // effect の後始末より先の描画でも、前の realm の設定を見せない。
  const branding = loaded?.realm === realm ? loaded.branding : {}
  const style = adminThemeStyle(branding)
  return (
    <AdminThemeContext value={{ branding, style, update }}>
      <div data-admin-theme="" style={style}>
        {children}
      </div>
    </AdminThemeContext>
  )
}

export function useAdminBranding(): Branding {
  return useContext(AdminThemeContext).branding
}

export function useUpdateAdminBranding() {
  return useContext(AdminThemeContext).update
}

// ポータルは DOM の祖先を失うので、React context から描画先へテーマを渡す。
export function useAdminThemeProps() {
  const { style } = useContext(AdminThemeContext)
  return { style, 'data-admin-theme': style ? '' : undefined }
}
