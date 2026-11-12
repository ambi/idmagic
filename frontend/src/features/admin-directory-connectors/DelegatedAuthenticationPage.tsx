import { useState, type FormEvent } from 'react'
import { configureDirectoryDelegation, testDirectoryDelegation } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { DirectoryConnector, DirectoryPasswordResult } from '../../types'
import { delegatedAuthenticationDictionary } from './DelegatedAuthenticationPage.i18n'
import { detailURL } from './presentation'

export function DelegatedAuthenticationPage({
  csrfToken,
  actorUsername,
  connector: initialConnector,
}: {
  csrfToken: string
  actorUsername?: string
  connector: DirectoryConnector
}) {
  const t = useDictionary(delegatedAuthenticationDictionary)
  const [connector, setConnector] = useState(initialConnector)
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [userID, setUserID] = useState('')
  const [password, setPassword] = useState('')
  const [result, setResult] = useState<DirectoryPasswordResult>()
  const [testing, setTesting] = useState(false)
  const [testError, setTestError] = useState('')

  async function configure() {
    setBusy(true)
    setError('')
    try {
      setConnector(
        await configureDirectoryDelegation(
          csrfToken,
          connector.id,
          !connector.delegated_authentication_enabled,
        ),
      )
      setConfirming(false)
    } catch {
      setError(t.settingFailed)
    } finally {
      setBusy(false)
    }
  }

  async function test(event: FormEvent) {
    event.preventDefault()
    setTesting(true)
    setTestError('')
    setResult(undefined)
    const submittedPassword = password
    setPassword('')
    try {
      setResult(
        (await testDirectoryDelegation(csrfToken, connector.id, userID, submittedPassword)).result,
      )
    } catch {
      setTestError(t.testFailed)
    } finally {
      setTesting(false)
    }
  }

  const enabled = connector.delegated_authentication_enabled ?? false
  const capable = connector.status === 'active' && !!connector.encryption_key_id
  return (
    <AdminShell
      active="directory-connectors"
      actorUsername={actorUsername}
      title={t.title}
      description={t.description}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={detailURL(connector.id)} />}
        >
          {t.back}
        </Button>
      }
    >
      <Card className="space-y-4 p-4">
        <h2 className="font-semibold">{connector.display_name}</h2>
        <p>
          {t.status}: {enabled ? t.enabled : t.disabled}
        </p>
        <p>{capable ? t.capable : t.unavailable}</p>
        <Button
          variant="outline"
          disabled={busy || (!enabled && !capable)}
          onClick={() => {
            setError('')
            setConfirming(true)
          }}
        >
          {busy ? t.working : enabled ? t.disable : t.enable}
        </Button>
        {confirming ? (
          <ConfirmActionDialog
            title={t.title}
            target={connector.display_name}
            description={t.confirmDescription}
            confirmLabel={t.confirm}
            busy={busy}
            error={error}
            onClose={() => setConfirming(false)}
            onConfirm={() => void configure()}
          />
        ) : null}
      </Card>
      <Card className="space-y-4 p-4">
        <h2 className="font-semibold">{t.testTitle}</h2>
        <p>{t.testDescription}</p>
        <form onSubmit={test} className="admin-form space-y-4">
          <div>
            <Label htmlFor="delegation-user-id" required>
              {t.userID}
            </Label>
            <Input
              id="delegation-user-id"
              value={userID}
              onChange={(e) => setUserID(e.target.value)}
              required
              disabled={testing}
            />
          </div>
          <div>
            <Label htmlFor="delegation-password" required>
              {t.password}
            </Label>
            <Input
              id="delegation-password"
              type="password"
              autoComplete="off"
              maxLength={4096}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={testing}
            />
          </div>
          <Button type="submit" disabled={testing || busy || !capable}>
            {testing ? t.testing : t.test}
          </Button>
        </form>
        {testError ? <Alert variant="destructive">{testError}</Alert> : null}
        {result ? <p role="status">{t[result]}</p> : null}
      </Card>
    </AdminShell>
  )
}
