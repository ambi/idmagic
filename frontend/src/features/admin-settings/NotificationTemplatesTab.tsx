import { IconPencil } from '@tabler/icons-react'
import { useCallback, useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  getNotificationTemplate,
  listNotificationTemplates,
  previewNotificationTemplate,
  resetNotificationTemplate,
  sendTestNotification,
  updateNotificationTemplate,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type {
  NotificationTemplateDetail,
  NotificationTemplateKey,
  NotificationTemplatePreview,
  NotificationTemplateSummary,
} from '../../types'
import {
  type NotificationTemplatesTabDictionary,
  notificationTemplatesTabDictionary,
} from './NotificationTemplatesTab.i18n'

// templateKeyLabel / localeLabel は SCL の enum 値と locale タグを利用者向けの語に
// 落とす。未知の値はそのまま出す (カタログが増えたときに空欄にならない)。
function templateKeyLabel(key: NotificationTemplateKey, t: NotificationTemplatesTabDictionary) {
  switch (key) {
    case 'password_reset':
      return t.templatePasswordReset
    case 'email_verification':
      return t.templateEmailVerification
    case 'email_change_confirmation':
      return t.templateEmailChangeConfirmation
    case 'account_security_alert':
      return t.templateAccountSecurityAlert
    case 'lifecycle_workflow_notification':
      return t.templateLifecycleWorkflowNotification
    default:
      return key
  }
}

function localeLabel(locale: string, t: NotificationTemplatesTabDictionary) {
  if (locale === 'ja') return t.localeJa
  if (locale === 'en') return t.localeEn
  return locale
}

type EditorState = {
  subject: string
  bodyText: string
  bodyHtml: string
  fromDisplayName: string
}

function toEditorState(detail: NotificationTemplateDetail): EditorState {
  return {
    subject: detail.subject,
    bodyText: detail.body_text,
    bodyHtml: detail.body_html,
    fromDisplayName: detail.from_display_name ?? '',
  }
}

export function NotificationTemplatesTab({ csrfToken }: { csrfToken: string }) {
  const t = useDictionary(notificationTemplatesTabDictionary)
  const [summaries, setSummaries] = useState<NotificationTemplateSummary[]>([])
  const [detail, setDetail] = useState<NotificationTemplateDetail | null>(null)
  const [draft, setDraft] = useState<EditorState | null>(null)
  const [preview, setPreview] = useState<NotificationTemplatePreview | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const loadList = useCallback(async () => {
    try {
      setSummaries((await listNotificationTemplates()).templates)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.loadFailedError)
    }
  }, [t.loadFailedError])

  useEffect(() => {
    void loadList()
  }, [loadList])

  function failed(cause: unknown, fallback: string) {
    setError(cause instanceof AuthenticationAPIError ? cause.message : fallback)
  }

  function applyDetail(next: NotificationTemplateDetail) {
    setDetail(next)
    setDraft(toEditorState(next))
    setPreview(null)
  }

  async function openEditor(summary: NotificationTemplateSummary) {
    setError('')
    setNotice('')
    try {
      applyDetail(await getNotificationTemplate(summary.template_key, summary.locale))
    } catch (cause) {
      failed(cause, t.loadFailedError)
    }
  }

  function closeEditor() {
    setDetail(null)
    setDraft(null)
    setPreview(null)
    setError('')
    setNotice('')
  }

  async function handleSave() {
    if (!detail || !draft) return
    setError('')
    setNotice('')
    // 件名 / テキスト / HTML は 3 点セット。片方だけの上書きは作れない (ADR-142 決定 4)。
    if (!draft.subject.trim() || !draft.bodyText.trim() || !draft.bodyHtml.trim()) {
      setError(t.requiredFieldsError)
      return
    }
    setBusy(true)
    try {
      const saved = await updateNotificationTemplate(
        csrfToken,
        detail.template_key,
        detail.locale,
        {
          subject: draft.subject,
          body_text: draft.bodyText,
          body_html: draft.bodyHtml,
          from_display_name: draft.fromDisplayName,
        },
      )
      applyDetail(saved)
      setNotice(t.savedNotice)
      await loadList()
    } catch (cause) {
      failed(cause, t.saveFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleReset() {
    if (!detail) return
    if (!window.confirm(t.resetConfirm)) return
    setError('')
    setNotice('')
    setBusy(true)
    try {
      applyDetail(await resetNotificationTemplate(csrfToken, detail.template_key, detail.locale))
      setNotice(t.resetNotice)
      await loadList()
    } catch (cause) {
      failed(cause, t.resetFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handlePreview() {
    if (!detail || !draft) return
    setError('')
    setNotice('')
    setBusy(true)
    try {
      setPreview(
        await previewNotificationTemplate(csrfToken, detail.template_key, detail.locale, {
          subject: draft.subject,
          body_text: draft.bodyText,
          body_html: draft.bodyHtml,
          from_display_name: draft.fromDisplayName,
        }),
      )
    } catch (cause) {
      failed(cause, t.previewFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleSendTest() {
    if (!detail) return
    setError('')
    setNotice('')
    setBusy(true)
    try {
      // 宛先はサーバが操作者本人に固定する。UI から渡す余地を作らない。
      const result = await sendTestNotification(csrfToken, detail.template_key, detail.locale)
      const message = result.delivered ? t.sendTestNotice : t.sendTestUndeliveredNotice
      setNotice(message.replace('{to}', result.to))
    } catch (cause) {
      failed(cause, t.sendTestFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.heading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.description}</p>
      </header>

      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        {detail && draft ? (
          <Editor
            t={t}
            detail={detail}
            draft={draft}
            preview={preview}
            busy={busy}
            onChange={setDraft}
            onBack={closeEditor}
            onSave={handleSave}
            onReset={handleReset}
            onPreview={handlePreview}
            onSendTest={handleSendTest}
          />
        ) : (
          <TemplateList t={t} summaries={summaries} onEdit={openEditor} />
        )}
      </div>
    </Card>
  )
}

function TemplateList({
  t,
  summaries,
  onEdit,
}: {
  t: NotificationTemplatesTabDictionary
  summaries: NotificationTemplateSummary[]
  onEdit: (summary: NotificationTemplateSummary) => void
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[36rem] text-left text-sm">
        <caption className="sr-only">{t.listCaption}</caption>
        <thead className="text-xs text-slate-500">
          <tr>
            <th scope="col" className="px-3 py-2 font-medium">
              {t.templateColumn}
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              {t.localeColumn}
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              {t.statusColumn}
            </th>
            <th scope="col" className="px-3 py-2 font-medium">
              {t.subjectColumn}
            </th>
            <th scope="col" className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {summaries.map((summary) => (
            <tr
              key={`${summary.template_key}/${summary.locale}`}
              className="border-t border-slate-200/80"
            >
              <td className="px-3 py-2.5 font-medium text-slate-900">
                {templateKeyLabel(summary.template_key, t)}
              </td>
              <td className="px-3 py-2.5 text-slate-600">{localeLabel(summary.locale, t)}</td>
              <td className="px-3 py-2.5">
                <span
                  className={
                    summary.customized
                      ? 'rounded-md bg-blue-50 px-1.5 py-0.5 text-xs font-medium text-blue-700'
                      : 'rounded-md bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-600'
                  }
                >
                  {summary.customized ? t.customizedBadge : t.defaultBadge}
                </span>
              </td>
              <td className="max-w-xs truncate px-3 py-2.5 text-slate-600">{summary.subject}</td>
              <td className="px-3 py-2.5 text-right">
                <Button type="button" variant="outline" onClick={() => onEdit(summary)}>
                  <IconPencil size={16} aria-hidden="true" />
                  {t.edit}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Editor({
  t,
  detail,
  draft,
  preview,
  busy,
  onChange,
  onBack,
  onSave,
  onReset,
  onPreview,
  onSendTest,
}: {
  t: NotificationTemplatesTabDictionary
  detail: NotificationTemplateDetail
  draft: EditorState
  preview: NotificationTemplatePreview | null
  busy: boolean
  onChange: (next: EditorState) => void
  onBack: () => void
  onSave: () => void
  onReset: () => void
  onPreview: () => void
  onSendTest: () => void
}) {
  const textareaClass = 'w-full rounded-md border border-slate-300 bg-white p-3 font-mono text-sm'
  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-900">
          {templateKeyLabel(detail.template_key, t)} / {localeLabel(detail.locale, t)}
        </h3>
        <Button type="button" variant="ghost" onClick={onBack}>
          {t.back}
        </Button>
      </div>

      <section className="rounded-lg border border-slate-200/80 bg-slate-50/60 p-3">
        <h4 className="text-xs font-semibold text-slate-700">{t.placeholdersHeading}</h4>
        <ul className="mt-2 flex flex-wrap gap-1.5">
          {detail.placeholders.map((placeholder) => (
            <li
              key={placeholder}
              className="rounded-md bg-white px-1.5 py-0.5 font-mono text-xs text-slate-700 ring-1 ring-slate-200"
            >
              {`{{${placeholder}}}`}
            </li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-slate-500">{t.placeholdersHelp}</p>
      </section>

      <div className="grid gap-1.5">
        <Label htmlFor="notification-subject">{t.subjectLabel}</Label>
        <Input
          id="notification-subject"
          value={draft.subject}
          maxLength={200}
          onChange={(event) => onChange({ ...draft, subject: event.target.value })}
        />
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="notification-body-text">{t.bodyTextLabel}</Label>
        <textarea
          id="notification-body-text"
          value={draft.bodyText}
          maxLength={8000}
          onChange={(event) => onChange({ ...draft, bodyText: event.target.value })}
          className={`${textareaClass} min-h-40`}
        />
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="notification-body-html">{t.bodyHtmlLabel}</Label>
        <textarea
          id="notification-body-html"
          value={draft.bodyHtml}
          maxLength={20000}
          onChange={(event) => onChange({ ...draft, bodyHtml: event.target.value })}
          className={`${textareaClass} min-h-40`}
        />
        <p className="text-xs text-slate-500">{t.bodyHtmlHelp}</p>
      </div>

      <div className="grid gap-1.5">
        <Label htmlFor="notification-from-display-name">{t.fromDisplayNameLabel}</Label>
        <Input
          id="notification-from-display-name"
          value={draft.fromDisplayName}
          maxLength={80}
          onChange={(event) => onChange({ ...draft, fromDisplayName: event.target.value })}
        />
        <p className="text-xs text-slate-500">{t.fromDisplayNameHelp}</p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" disabled={busy} onClick={onSave}>
          {busy ? t.saving : t.save}
        </Button>
        <Button type="button" variant="outline" disabled={busy} onClick={onPreview}>
          {t.preview}
        </Button>
        <Button type="button" variant="outline" disabled={busy} onClick={onSendTest}>
          {busy ? t.sending : t.sendTest}
        </Button>
        {detail.customized ? (
          <Button type="button" variant="ghost" disabled={busy} onClick={onReset}>
            {t.resetToDefault}
          </Button>
        ) : null}
      </div>
      <p className="text-xs text-slate-500">{t.sendTestHelp}</p>

      {preview ? (
        <section className="grid gap-2 rounded-lg border border-slate-200/80 bg-white p-3">
          <h4 className="text-xs font-semibold text-slate-700">{t.previewHeading}</h4>
          <div>
            <p className="text-xs text-slate-500">{t.previewSubjectLabel}</p>
            <p className="text-sm font-medium text-slate-900">{preview.subject}</p>
          </div>
          <div>
            <p className="text-xs text-slate-500">{t.previewTextLabel}</p>
            <pre className="mt-1 overflow-x-auto whitespace-pre-wrap rounded-md bg-slate-50 p-3 text-xs text-slate-800">
              {preview.body_text}
            </pre>
          </div>
          <div>
            <p className="text-xs text-slate-500">{t.previewHtmlLabel}</p>
            {/*
              描画済み HTML はサーバのレンダラが差し込み値をエスケープした結果だが、
              テンプレート本体はテナント管理者が書いた markup なので、管理画面の DOM へ
              流し込まず iframe に隔離して表示する。sandbox 無指定 (= 全部禁止) により
              スクリプト実行も同一オリジンアクセスも起きない。
            */}
            <iframe
              title={t.previewHtmlLabel}
              sandbox=""
              srcDoc={preview.body_html}
              className="mt-1 h-64 w-full rounded-md border border-slate-200 bg-white"
            />
          </div>
        </section>
      ) : null}
    </div>
  )
}
