import { Dialog } from '@base-ui/react/dialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'

export function AgentActionConfirmation({
  title,
  description,
  busy,
  error,
  onConfirm,
  onCancel,
}: {
  title: string
  description: string
  busy: boolean
  error?: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const t = useDictionary(adminAgentsDictionary)
  return (
    <Dialog.Root
      open
      onOpenChange={(open) => {
        if (!open && !busy) onCancel()
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-slate-900/40" />
        <Dialog.Popup className="fixed top-1/2 left-1/2 z-50 w-[calc(100%-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl bg-white p-6 shadow-xl">
          <Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
          <Dialog.Description className="mt-3 text-sm text-slate-600">
            {description}
          </Dialog.Description>
          {error ? (
            <Alert variant="destructive" className="mt-4">
              {error}
            </Alert>
          ) : null}
          <div className="mt-6 flex justify-end gap-2">
            <Button variant="outline" disabled={busy} onClick={onCancel}>
              {t.cancel}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={onConfirm}>
              {title}
            </Button>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
