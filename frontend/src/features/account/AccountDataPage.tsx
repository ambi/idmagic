import { IconDownload, IconFileText } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, exportAccountData } from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { accountDataDictionary } from './AccountDataPage.i18n'

export function AccountDataPage({ username, isAdmin }: { username: string; isAdmin: boolean }) {
  const t = useDictionary(accountDataDictionary)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')

  async function handleExport() {
    setDownloading(true)
    setError('')
    try {
      const data = await exportAccountData()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `account-data-${username}.json`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.exportFailed)
    } finally {
      setDownloading(false)
    }
  }

  return (
    <AccountDataPresentation
      username={username}
      isAdmin={isAdmin}
      downloading={downloading}
      error={error}
      onExport={handleExport}
    />
  )
}

export function AccountDataPresentation({
  username,
  isAdmin,
  downloading,
  error,
  onExport,
}: {
  username: string
  isAdmin: boolean
  downloading: boolean
  error: string
  onExport: () => void
}) {
  const t = useDictionary(accountDataDictionary)
  return (
    <AccountShell
      active="data"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <Card className="flex flex-col gap-4 p-5">
        <div className="flex items-start gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
            <IconFileText size={20} aria-hidden="true" />
          </span>
          <div>
            <p className="text-sm font-semibold text-slate-900">{t.exportTitle}</p>
            <p className="mt-1 text-sm leading-6 text-slate-600">{t.exportDescription}</p>
          </div>
        </div>
        <div>
          <Button type="button" onClick={onExport} disabled={downloading}>
            <IconDownload size={16} aria-hidden="true" />
            {downloading ? t.creating : t.download}
          </Button>
        </div>
      </Card>
    </AccountShell>
  )
}
