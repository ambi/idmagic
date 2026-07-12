import { IconCheck, IconInfoCircle, IconLogin, IconLogout, IconX } from '@tabler/icons-react'
import { tenantURL } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Button } from '../../components/ui/button'
import { cn } from '../../lib/utils'
import { useDictionary } from '../../lib/i18n'
import { statusPageDictionary } from './StatusPage.i18n'

type StatusKey = 'approved' | 'denied' | 'signed-out' | 'authentication-required'

export function StatusPage({ status }: { status: StatusKey }) {
  const t = useDictionary(statusPageDictionary)
  const state = {
    approved: {
      eyebrow: t.approvedEyebrow,
      title: t.approvedTitle,
      text: t.approvedText,
      note: t.approvedNote,
      icon: IconCheck,
      color: 'border-emerald-100 bg-emerald-50 text-emerald-700',
    },
    denied: {
      eyebrow: t.deniedEyebrow,
      title: t.deniedTitle,
      text: t.deniedText,
      note: t.deniedNote,
      icon: IconX,
      color: 'border-slate-200 bg-slate-100 text-slate-700',
    },
    'signed-out': {
      eyebrow: t.signedOutEyebrow,
      title: t.signedOutTitle,
      text: t.signedOutText,
      note: t.signedOutNote,
      icon: IconLogout,
      color: 'border-blue-100 bg-blue-50 text-blue-700',
    },
    'authentication-required': {
      eyebrow: t.authRequiredEyebrow,
      title: t.authRequiredTitle,
      text: t.authRequiredText,
      note: t.authRequiredNote,
      icon: IconLogin,
      color: 'border-amber-200 bg-amber-50 text-amber-700',
    },
  }[status]
  const StatusIcon = state.icon

  return (
    <AuthShell>
      <div className="flex flex-col items-center gap-7 py-4 text-center">
        <div
          className={cn(
            'flex size-16 items-center justify-center rounded-2xl border shadow-xs',
            state.color,
          )}
        >
          <StatusIcon size={30} stroke={2} aria-hidden="true" />
        </div>

        <header className="flex max-w-md flex-col items-center gap-2.5">
          <p className="eyebrow">{state.eyebrow}</p>
          <h2 className="page-title">{state.title}</h2>
          <p className="page-description">{state.text}</p>
        </header>

        <div className="flex w-full items-start gap-3 rounded-xl bg-slate-50 p-3.5 text-left text-xs leading-5 text-slate-600">
          <IconInfoCircle className="mt-0.5 shrink-0 text-slate-500" size={17} aria-hidden="true" />
          <p>{state.note}</p>
        </div>

        {status === 'signed-out' ? (
          <div className="grid w-full gap-2">
            <Button asChild className="w-full">
              <a href={tenantURL('/account')}>{t.signInToAccount}</a>
            </Button>
            <Button asChild variant="outline" className="w-full">
              <a href={tenantURL('/admin')}>{t.signInToAdmin}</a>
            </Button>
          </div>
        ) : null}
      </div>
    </AuthShell>
  )
}
