import { IconCopy } from '@tabler/icons-react'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import type { AdminSamlIDPProfile } from '../../types'
import type { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'

type Dictionary = (typeof adminSamlIDPProfilesDictionary)['en']

function CopyableField({
  label,
  value,
  copyLabel,
}: {
  label: string
  value: string
  copyLabel: string
}) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <div className="flex items-center gap-2">
        <code className="min-w-0 flex-1 break-all rounded-md bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800">
          {value}
        </code>
        <Button
          type="button"
          variant="outline"
          className="shrink-0"
          aria-label={copyLabel}
          onClick={() => void navigator.clipboard?.writeText(value)}
        >
          <IconCopy size={16} aria-hidden="true" />
          <span className="hidden sm:inline">{copyLabel}</span>
        </Button>
      </div>
    </div>
  )
}

export function SamlIDPProfileFields({ entry, t }: { entry: AdminSamlIDPProfile; t: Dictionary }) {
  return (
    <div className="grid gap-4">
      <CopyableField label={t.entityId} value={entry.entity_id} copyLabel={t.copy} />
      <CopyableField label={t.metadataUrl} value={entry.metadata_url} copyLabel={t.copy} />
      <CopyableField label={t.ssoUrl} value={entry.sso_url} copyLabel={t.copy} />
      <CopyableField label={t.sloUrl} value={entry.slo_url} copyLabel={t.copy} />
      <CopyableField
        label={t.certificateUrl}
        value={entry.signing_certificate_url}
        copyLabel={t.copy}
      />
      <CopyableField
        label={t.fingerprint}
        value={entry.signing_certificate_fingerprint_sha256}
        copyLabel={t.copy}
      />
    </div>
  )
}
