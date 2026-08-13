import { IconChecklist, IconCheck, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  decideMyApprovalRequest,
  type AccountApprovalRequest,
} from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { accountApprovalsDictionary } from './AccountApprovalsPage.i18n'

export function AccountApprovalsPage({
  csrfToken,
  username,
  approvalRequests: initial,
  isAdmin,
}: {
  csrfToken: string
  username: string
  approvalRequests: AccountApprovalRequest[]
  isAdmin: boolean
}) {
  const t = useDictionary(accountApprovalsDictionary)
  const [approvalRequests, setApprovalRequests] = useState(initial)
  const [pending, setPending] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const { guard, dialog } = useStepUpGuard(csrfToken)

  async function handleDecision(request: AccountApprovalRequest, decision: 'approve' | 'deny') {
    setPending(request.id)
    setError('')
    setNotice('')
    try {
      await guard(() => decideMyApprovalRequest(csrfToken, request.id, decision))
      setApprovalRequests((current) => current.filter((item) => item.id !== request.id))
      const message = decision === 'approve' ? t.approved : t.denied
      setNotice(message.replace('{name}', request.client_name))
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setPending('')
    }
  }

  return (
    <>
      <AccountApprovalsPresentation
        username={username}
        approvalRequests={approvalRequests}
        isAdmin={isAdmin}
        pending={pending}
        error={error}
        notice={notice}
        onDismissNotice={() => setNotice('')}
        onDecision={handleDecision}
      />
      {dialog}
    </>
  )
}

export function AccountApprovalsPresentation({
  username,
  approvalRequests,
  isAdmin,
  pending,
  error,
  notice,
  onDismissNotice,
  onDecision,
}: {
  username: string
  approvalRequests: AccountApprovalRequest[]
  isAdmin: boolean
  pending: string
  error: string
  notice: string
  onDismissNotice: () => void
  onDecision: (request: AccountApprovalRequest, decision: 'approve' | 'deny') => void
}) {
  const t = useDictionary(accountApprovalsDictionary)
  const { locale } = useLocale()
  const formatDateTime = (value: string) => {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale)
  }

  return (
    <AccountShell
      active="approvals"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <Toast message={notice} onDismiss={onDismissNotice} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      {approvalRequests.length === 0 ? (
        <Card className="flex flex-col items-center gap-2 p-10 text-center">
          <IconChecklist size={28} className="text-slate-300" aria-hidden="true" />
          <p className="text-sm text-slate-500">{t.empty}</p>
        </Card>
      ) : (
        <div className="grid gap-4">
          {approvalRequests.map((request) => (
            <Card key={request.id} className="grid gap-4 p-5">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                    {t.requestedBy}
                  </p>
                  <h2 className="mt-1 text-base font-semibold text-slate-900">
                    {request.client_name}
                  </h2>
                  {request.agent_name ? (
                    <p className="mt-1 text-sm text-slate-600">
                      {t.agent}: {request.agent_name}
                    </p>
                  ) : null}
                </div>
                <p className="text-xs text-slate-500">
                  {t.expires.replace('{date}', formatDateTime(request.expires_at))}
                </p>
              </div>

              {request.binding_message ? (
                <div className="rounded-md border border-amber-200 bg-amber-50 p-3">
                  <p className="text-xs font-semibold text-amber-900">{t.bindingMessage}</p>
                  <p className="mt-1 text-sm text-amber-950">{request.binding_message}</p>
                </div>
              ) : null}

              <div>
                <p className="text-xs font-semibold text-slate-700">{t.scopes}</p>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {request.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
              </div>

              {request.authorization_details?.length ? (
                <div>
                  <p className="text-xs font-semibold text-slate-700">{t.details}</p>
                  <pre className="mt-2 max-h-64 overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-100">
                    {JSON.stringify(request.authorization_details, null, 2)}
                  </pre>
                </div>
              ) : null}

              <div className="flex justify-end gap-2 border-t border-slate-100 pt-4">
                <Button
                  type="button"
                  variant="destructive"
                  disabled={pending === request.id}
                  onClick={() => onDecision(request, 'deny')}
                >
                  <IconX aria-hidden="true" />
                  {pending === request.id ? t.deciding : t.deny}
                </Button>
                <Button
                  type="button"
                  disabled={pending === request.id}
                  onClick={() => onDecision(request, 'approve')}
                >
                  <IconCheck aria-hidden="true" />
                  {pending === request.id ? t.deciding : t.approve}
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </AccountShell>
  )
}
