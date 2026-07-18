import { useState } from 'react'
import { rotateApplicationClientSecret } from '../../api'
import { Button } from '../../components/ui/button'
import { CopyableField, messageOf } from './AdminApplicationsShared'

export function ClientSecretRotationPanel({
  applicationID,
  csrfToken,
  onError,
}: {
  applicationID: string
  csrfToken: string
  onError: (message: string) => void
}) {
  const [secret, setSecret] = useState('')
  const [copied, setCopied] = useState(false)

  async function rotate(graceDays: number) {
    onError('')
    try {
      const result = await rotateApplicationClientSecret(csrfToken, applicationID, graceDays)
      setSecret(result.client_secret)
      setCopied(false)
    } catch (cause) {
      onError(messageOf(cause, 'Client secret の rotation に失敗しました'))
    }
  }

  return (
    <div className="grid gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3">
      <p className="text-sm text-amber-900">
        Client secret を回すと、旧 secret は選択した猶予期間の後に無効になります。
      </p>
      <div className="flex flex-wrap gap-2">
        {[1, 7, 14, 30].map((days) => (
          <Button key={days} type="button" variant="destructive" onClick={() => void rotate(days)}>
            {days} 日の猶予で回す
          </Button>
        ))}
        <Button type="button" variant="destructive" onClick={() => void rotate(0)}>
          即時 revoke
        </Button>
      </div>
      {secret ? (
        <div className="grid gap-2">
          <CopyableField
            label="新しい client secret（この画面を離れると再表示できません）"
            value={secret}
          />
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
            コピー済み
          </label>
          {copied ? (
            <Button type="button" variant="outline" onClick={() => setSecret('')}>
              閉じる
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
