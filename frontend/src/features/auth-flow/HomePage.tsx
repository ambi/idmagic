import { IconArrowRight, IconInfoCircle } from '@tabler/icons-react'
import { startDemoAuthorization } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import { informationalPagesDictionary } from './InformationalPages.i18n'

export function HomePage({ demoEnabled }: { demoEnabled: boolean }) {
  const t = useDictionary(informationalPagesDictionary)
  return (
    <AuthShell>
      <div className="flex flex-col gap-7 py-4">
        <header className="flex flex-col gap-2.5">
          <p className="eyebrow">{t.homeEyebrow}</p>
          <h2 className="page-title">{t.homeTitle}</h2>
          <p className="page-description">{t.homeDescription}</p>
        </header>

        <div className="flex items-start gap-3 rounded-xl border border-blue-100 bg-blue-50 p-4 text-sm leading-6 text-blue-950">
          <IconInfoCircle className="mt-0.5 shrink-0 text-blue-700" size={18} aria-hidden="true" />
          <p>{t.homeDirectLogin}</p>
        </div>

        {demoEnabled ? (
          <Button size="lg" onClick={() => void startDemoAuthorization()}>
            {t.startDemo}
            <IconArrowRight size={18} aria-hidden="true" />
          </Button>
        ) : (
          <p className="text-sm leading-6 text-slate-600">{t.startFromApplication}</p>
        )}

        {demoEnabled && (
          <p className="text-center text-xs leading-5 text-slate-500">
            {t.demoUser} <code>alice</code> / <code>demo-password-1234</code>
          </p>
        )}
      </div>
    </AuthShell>
  )
}
