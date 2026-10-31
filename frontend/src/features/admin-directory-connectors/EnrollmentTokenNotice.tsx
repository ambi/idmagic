import { CopyableValue } from '../../components/CopyableValue'
import { Label } from '../../components/ui/label'
import type { Dictionary } from './presentation'

// EnrollmentTokenNotice は発行の応答を受けた画面でだけ描画する。トークンは呼び出し元の
// useState にだけ置き、URL、保存領域、ルートのデータへ渡さない (REQ-SOURCING-023)。
export function EnrollmentTokenNotice({ token, t }: { token: string; t: Dictionary }) {
  return (
    <div className="grid gap-2 rounded-lg border border-emerald-200 bg-emerald-50 p-4">
      <Label>{t.tokenLabel}</Label>
      <CopyableValue value={token} copyLabel={t.copyAria} />
      <p className="text-xs leading-5 text-emerald-900">{t.tokenNotice}</p>
    </div>
  )
}
