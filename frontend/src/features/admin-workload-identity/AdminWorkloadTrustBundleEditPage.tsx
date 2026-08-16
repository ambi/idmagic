import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateWorkloadTrustBundle } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { WorkloadTrustBundle } from '../../types'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { detailURL, toBundleForm, TrustBundleFormFields } from './AdminWorkloadIdentityShared'
import { parseAudiences, parseInlineJwks } from './presentation'

// issuer と trust_domain は更新リクエストが受け付けないため入力欄を無効化する (locked)。
// 変更するには登録し直すしかない、という仕様上の非対称をフォームで示す。
export function AdminWorkloadTrustBundleEditPage({
  csrfToken,
  actorUsername,
  trustBundle,
}: {
  csrfToken: string
  actorUsername?: string
  trustBundle: WorkloadTrustBundle
}) {
  const [form, setForm] = useState(() => toBundleForm(trustBundle))
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const t = useDictionary(adminWorkloadIdentityDictionary)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const jwks = parseInlineJwks(form.jwks)
    if (!jwks.ok) {
      setError(t.jwksInvalidError)
      return
    }
    setSaving(true)
    try {
      await updateWorkloadTrustBundle(csrfToken, trustBundle.id, {
        name: form.name,
        // jwks_uri は非 nil なら空文字でも上書きされるため、変更が無ければ送らない。常に送ると
        // インライン JWKS だけのバンドルで名前を直しただけの更新が取得元設定を空にしてしまう。
        // 逆に、意図して空にした場合は空文字を送る必要がある (それが唯一の消し方である)。
        ...(form.jwksUri === (trustBundle.jwks_uri ?? '') ? {} : { jwks_uri: form.jwksUri }),
        // インライン JWKS は空欄が「差し替えない」であって「消す」ではないので送らない。
        ...(jwks.value ? { jwks: jwks.value } : {}),
        accepted_audiences: parseAudiences(form.acceptedAudiences),
        max_subject_token_ttl_seconds: Number(form.maxTtlSeconds),
      })
      window.location.assign(detailURL(trustBundle.id))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="workload-identity"
      actorUsername={actorUsername}
      title={trustBundle.name}
      description={t.pageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={detailURL(trustBundle.id)} />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl p-4">
        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <TrustBundleFormFields form={form} onChange={setForm} locked={true} t={t} />
          <div className="flex gap-2.5">
            <Button type="submit" disabled={saving}>
              {t.update}
            </Button>
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={detailURL(trustBundle.id)} />}
            >
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}
