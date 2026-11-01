import { AccountShell } from '../../components/AccountShell'
import { useDictionary } from '../../lib/i18n'
import { adminSettingsDictionary } from '../admin-settings/AdminSettingsPage.i18n'
import { ApiTokensTab } from '../admin-settings/ApiTokensTab'

export function AccountApiTokensPage({
  csrfToken,
  username,
  isAdmin,
  create = false,
}: {
  csrfToken: string
  username: string
  isAdmin: boolean
  create?: boolean
}) {
  const t = useDictionary(adminSettingsDictionary)
  return (
    <AccountShell
      active="api-tokens"
      username={username}
      isAdmin={isAdmin}
      title={create ? t.newTokenHeading : t.personalTokensHeading}
    >
      <ApiTokensTab csrfToken={csrfToken} personal screen={create ? 'create' : 'list'} />
    </AccountShell>
  )
}
