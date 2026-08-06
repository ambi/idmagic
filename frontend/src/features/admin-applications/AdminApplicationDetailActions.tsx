import { IconArrowLeft, IconPencil, IconTrash } from '@tabler/icons-react'
import { Button } from '../../components/ui/button'
import type { AdminApplication } from '../../types'
import { ProvisioningNavButton } from './AdminApplicationProvisioningShared'
import type { AdminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import { editURL, listURL } from './AdminApplicationsShared'

export function AdminApplicationDetailActions({
  app,
  busy,
  onDelete,
  t,
}: {
  app: AdminApplication
  busy: boolean
  onDelete: () => void
  t: AdminApplicationsDictionary
}) {
  return (
    <div className="flex items-center gap-2">
      <a
        href={listURL()}
        className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
      >
        <IconArrowLeft size={16} aria-hidden="true" />
        {t.backToList}
      </a>
      <ProvisioningNavButton app={app} />
      <Button nativeButton={false} render={<a href={editURL(app.id)} />}>
        <IconPencil size={16} aria-hidden="true" />
        {t.edit}
      </Button>
      <Button type="button" variant="destructive" disabled={busy} onClick={onDelete}>
        <IconTrash size={16} aria-hidden="true" />
        {t.delete}
      </Button>
    </div>
  )
}
