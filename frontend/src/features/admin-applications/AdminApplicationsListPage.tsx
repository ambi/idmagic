import { IconApps, IconExternalLink, IconPlus, IconRefresh, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { deleteAdminApplication, listAdminApplications } from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import {
  AppIcon,
  detailURL,
  kindLabel,
  KindBadge,
  messageOf,
  ReadOnlyField,
  StatusBadge,
} from './AdminApplicationsShared'
import { CreateApplicationDialog } from './CreateApplicationDialog'
import type { AdminApplication } from '../../types'

export function AdminApplicationsPage({
  csrfToken,
  actorUsername,
  applications: initial,
}: {
  csrfToken: string
  actorUsername?: string
  applications: AdminApplication[]
}) {
  const [applications, setApplications] = useState(initial)
  const [selectedID, setSelectedID] = useState<string>(() => initial[0]?.application_id ?? '')
  const [showCreate, setShowCreate] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminApplicationsDictionary)

  const selected = applications.find((a) => a.application_id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminApplications()
    setApplications(next)
    setSelectedID(
      next.find((a) => a.application_id === preferredID)?.application_id ??
        next[0]?.application_id ??
        '',
    )
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(messageOf(cause, t.genericOpError))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="applications"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadAriaLabel}
            onClick={() => run(() => refresh(), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button onClick={() => setShowCreate(true)} disabled={busy}>
            <IconPlus size={16} aria-hidden="true" />
            {t.addApplication}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,420px)]">
        <Card className="overflow-hidden">
          {applications.length === 0 ? (
            <div className="flex min-h-48 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconApps size={28} className="text-slate-300" aria-hidden="true" />
              <p className="mt-3">{t.emptyApplicationsNotice}</p>
            </div>
          ) : (
            <ul>
              {applications.map((app) => (
                <li key={app.application_id}>
                  <button
                    type="button"
                    onClick={() => setSelectedID(app.application_id)}
                    className={`flex w-full items-center gap-3 border-t border-slate-100 px-4 py-3 text-left first:border-t-0 hover:bg-slate-50 ${
                      selectedID === app.application_id ? 'bg-blue-50/60' : ''
                    }`}
                  >
                    <AppIcon app={app} size="sm" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate font-semibold text-slate-900">{app.name}</span>
                        <StatusBadge status={app.status} />
                      </div>
                      <div className="mt-0.5">
                        <KindBadge app={app} />
                      </div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <ApplicationSummaryCard
          key={selectedID || 'none'}
          app={selected}
          busy={busy}
          onDelete={(id) =>
            run(async () => {
              await deleteAdminApplication(csrfToken, id)
              await refresh()
            }, t.applicationDeletedNotice)
          }
        />
      </div>

      {showCreate ? (
        <CreateApplicationDialog
          csrfToken={csrfToken}
          onClose={() => setShowCreate(false)}
          onCreated={(id) => {
            window.location.assign(detailURL(id))
          }}
        />
      ) : null}
    </AdminShell>
  )
}

function ApplicationSummaryCard({
  app,
  busy,
  onDelete,
}: {
  app: AdminApplication | null
  busy: boolean
  onDelete: (id: string) => void
}) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)
  const { locale } = useLocale()

  if (!app) {
    return (
      <Card className="flex min-h-48 items-center justify-center p-6 text-sm text-slate-500">
        {t.selectApplicationPrompt}
      </Card>
    )
  }

  return (
    <Card className="overflow-hidden">
      <div className="border-b border-slate-200 p-5">
        <div className="flex items-start gap-3">
          <AppIcon app={app} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold text-slate-950">{app.name}</h2>
              <StatusBadge status={app.status} />
            </div>
            <div className="mt-1">
              <KindBadge app={app} />
            </div>
          </div>
        </div>
        <div className="mt-4">
          <AdminPaneActions
            detailHref={detailURL(app.application_id)}
            busy={busy}
            actions={[
              {
                label: t.deleteApplication,
                icon: IconTrash,
                onClick: () => setConfirmDelete(true),
                tone: 'danger',
              },
            ]}
          />
        </div>
      </div>
      {confirmDelete ? (
        <Alert
          variant="destructive"
          className="m-5 flex flex-wrap items-center justify-between gap-2"
        >
          <span>{t.confirmDeleteAppPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
              {t.dismissConfirm}
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => onDelete(app.application_id)}
            >
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}
      <dl className="grid gap-4 p-5">
        <ReadOnlyField label={t.kindFieldLabel}>{kindLabel(app, t)}</ReadOnlyField>

        <ReadOnlyField label={t.statusFieldLabel}>
          <StatusBadge status={app.status} />
        </ReadOnlyField>

        <ReadOnlyField label={t.categoryFieldLabel}>
          {app.category_names && app.category_names.length > 0 ? (
            <div className="flex flex-wrap gap-1">
              {app.category_names.map((name) => (
                <span key={name} className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700">
                  {name}
                </span>
              ))}
            </div>
          ) : (
            <span className="text-slate-400">{t.noCategoryNotice}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.bindingFieldLabel}>
          {app.binding_summaries && app.binding_summaries.length > 0 ? (
            <div className="flex flex-col gap-1 font-mono text-xs text-slate-700">
              {app.binding_summaries.map((summary, idx) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static list
                <span key={idx}>{summary}</span>
              ))}
            </div>
          ) : (
            <span className="text-slate-400">{t.notSetLabel}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.assignmentStatusFieldLabel}>
          {app.assigned_subject_count > 0 ? (
            <span className="text-slate-700">
              {t.assignedCount.replace('{count}', String(app.assigned_subject_count))}
            </span>
          ) : (
            <span className="text-slate-400">{t.noAssignmentNotice}</span>
          )}
        </ReadOnlyField>

        <ReadOnlyField label={t.signInPolicyFieldLabel}>
          {app.sign_in_policy_summary ? (
            <span className="text-slate-700">{app.sign_in_policy_summary}</span>
          ) : (
            <span className="text-slate-400">{t.notSetLabel}</span>
          )}
        </ReadOnlyField>

        {app.kind === 'service' ? (
          <ReadOnlyField label={t.serviceDescriptionFieldLabel}>
            <p className="text-xs text-slate-500">{t.serviceM2mDescription}</p>
          </ReadOnlyField>
        ) : (
          <ReadOnlyField label={t.launchUrlFieldLabel}>
            {app.launch_url ? (
              <a
                href={app.launch_url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 break-all font-mono text-xs text-blue-700 hover:underline"
              >
                {app.launch_url}
                <IconExternalLink size={13} aria-hidden="true" />
              </a>
            ) : (
              <span className="text-slate-400">{t.notSetLabel}</span>
            )}
          </ReadOnlyField>
        )}

        <ReadOnlyField label={t.registeredUpdatedFieldLabel}>
          <div className="text-xs text-slate-500">
            <div>
              {t.registeredLabel.replace(
                '{date}',
                app.created_at
                  ? new Date(app.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                  : t.unknownDate,
              )}
            </div>
            <div className="mt-0.5">
              {t.updatedLabel.replace(
                '{date}',
                app.updated_at
                  ? new Date(app.updated_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
                  : t.unknownDate,
              )}
            </div>
          </div>
        </ReadOnlyField>
      </dl>
    </Card>
  )
}
