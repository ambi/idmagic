// DemoLoginAffordance を出すかどうかは起動時設定が決める。Vite 開発サーバーで実行している
// あいだは設定なしで出し (REQ-SYSTEM-007)、ビルド済みのフロントエンドを配備した場合は
// `VITE_DEMO_LOGIN_ENABLED` が `true` のときだけ出す (REQ-SYSTEM-006)。
//
// 判定を関数にして既定引数を境界に置いてあるのは、`configuredDefaultLocale` と同じ理由である。
// `import.meta.env` を式の中で直接読むと、起動時設定を入力として与える方法が無くなり、
// 「設定したときに出る」「設定しないときに出ない」のどちらも観測できない。
export interface DemoLoginEnv {
  DEV?: boolean
  VITE_DEMO_LOGIN_ENABLED?: string
}

export function demoLoginEnabled(env: DemoLoginEnv = import.meta.env): boolean {
  return Boolean(env.DEV) || env.VITE_DEMO_LOGIN_ENABLED === 'true'
}
