import { Dialog } from '@base-ui/react/dialog'
import { useDictionary } from '../lib/i18n'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { Button } from './ui/button'
import { Card } from './ui/card'

export function ConfirmActionDialog({
  title,
  target,
  description,
  confirmLabel,
  busy = false,
  error,
  onClose,
  onConfirm,
}: {
  title: string
  target: string
  description: string
  confirmLabel: string
  busy?: boolean
  error?: string
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(commonDictionary)
  return (
    <Dialog.Root
      open
      disablePointerDismissal
      onOpenChange={(open) => {
        if (!open && !busy) onClose()
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-slate-950/35 backdrop-blur-[2px]" />
        <Dialog.Popup className="fixed inset-0 z-50 m-auto h-fit w-[calc(100%-2rem)] max-w-lg outline-none">
          <Card className="space-y-4 p-6 shadow-2xl">
            <Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
            <p className="break-all font-medium">{target}</p>
            <Dialog.Description className="text-sm text-slate-600">
              {description}
            </Dialog.Description>
            {error ? (
              <p role="alert" className="text-sm text-red-700">
                {error}
              </p>
            ) : null}
            <div className="flex justify-end gap-2">
              <Button variant="outline" disabled={busy} onClick={onClose}>
                {t.cancel}
              </Button>
              <Button variant="destructive" disabled={busy} onClick={onConfirm}>
                {confirmLabel}
              </Button>
            </div>
          </Card>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
