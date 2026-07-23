import { IconChevronDown } from '@tabler/icons-react'
import type { ApiTokenScope } from '../../types'
import type { AdminSettingsDictionary } from './AdminSettingsPage.i18n'

type DictionaryKey = keyof AdminSettingsDictionary

type ScopePermission = {
  scope: ApiTokenScope
  action: 'read' | 'write'
}

type ScopeResource = {
  labelKey: DictionaryKey
  descriptionKey?: DictionaryKey
  permissions: ScopePermission[]
}

type ScopeGroup = {
  headingKey: DictionaryKey
  descriptionKey: DictionaryKey
  resources: ScopeResource[]
}

const managementResources: ScopeResource[] = [
  resource('usersScopeResourceLabel', 'users:read', 'users:write'),
  resource('groupsScopeResourceLabel', 'groups:read', 'groups:write'),
  resource('agentsScopeResourceLabel', 'agents:read', 'agents:write'),
  resource('sessionsScopeResourceLabel', 'sessions:read', 'sessions:write'),
  resource('consentsScopeResourceLabel', 'consents:read', 'consents:write'),
  resource(
    'lifecycleWorkflowsScopeResourceLabel',
    'lifecycle-workflows:read',
    'lifecycle-workflows:write',
  ),
  resource('tenantsScopeResourceLabel', 'tenants:read', 'tenants:write'),
  resource('settingsScopeResourceLabel', 'settings:read', 'settings:write'),
  resource('signingKeysScopeResourceLabel', 'signing-keys:read', 'signing-keys:write'),
  resource('auditScopeResourceLabel', 'audit:read'),
  resource('applicationsScopeResourceLabel', 'applications:read', 'applications:write'),
  resource('oauthClientsScopeResourceLabel', 'oauth-clients:read', 'oauth-clients:write'),
  resource(
    'authorizationDetailTypesScopeResourceLabel',
    'authorization-detail-types:read',
    'authorization-detail-types:write',
  ),
  resource(
    'mcpResourceServersScopeResourceLabel',
    'mcp-resource-servers:read',
    'mcp-resource-servers:write',
  ),
  resource('samlScopeResourceLabel', 'saml:read', 'saml:write'),
  resource('wsfedScopeResourceLabel', 'wsfed:read', 'wsfed:write'),
  resource('provisioningScopeResourceLabel', 'provisioning:read', 'provisioning:write'),
]

const scopeGroups: ScopeGroup[] = [
  {
    headingKey: 'managementScopesHeading',
    descriptionKey: 'managementScopesDescription',
    resources: managementResources,
  },
  {
    headingKey: 'scimScopesHeading',
    descriptionKey: 'scimScopesDescription',
    resources: [
      resource('usersScopeResourceLabel', 'scim:users:read', 'scim:users:write'),
      resource('groupsScopeResourceLabel', 'scim:groups:read', 'scim:groups:write'),
    ],
  },
  {
    headingKey: 'accountScopesHeading',
    descriptionKey: 'accountScopesDescription',
    resources: [
      {
        ...resource('accountProfileScopeLabel', 'account:read', 'account:write'),
        descriptionKey: 'accountProfileScopeDescription',
      },
      {
        labelKey: 'accountMfaScopeLabel',
        descriptionKey: 'accountMfaScopeDescription',
        permissions: [{ scope: 'account:mfa:write', action: 'write' }],
      },
      {
        labelKey: 'accountSessionsScopeLabel',
        descriptionKey: 'accountSessionsScopeDescription',
        permissions: [{ scope: 'account:sessions:write', action: 'write' }],
      },
      {
        labelKey: 'accountConsentsScopeLabel',
        descriptionKey: 'accountConsentsScopeDescription',
        permissions: [{ scope: 'account:consents:write', action: 'write' }],
      },
      {
        labelKey: 'accountPasswordScopeLabel',
        descriptionKey: 'accountPasswordScopeDescription',
        permissions: [{ scope: 'account:password:write', action: 'write' }],
      },
    ],
  },
]

function resource(
  labelKey: DictionaryKey,
  readScope: ApiTokenScope,
  writeScope?: ApiTokenScope,
): ScopeResource {
  return {
    labelKey,
    permissions: [
      { scope: readScope, action: 'read' },
      ...(writeScope ? [{ scope: writeScope, action: 'write' } as const] : []),
    ],
  }
}

export function ApiTokenScopePicker({
  t,
  selectedScopes,
  onChange,
}: {
  t: AdminSettingsDictionary
  selectedScopes: ApiTokenScope[]
  onChange: (scopes: ApiTokenScope[]) => void
}) {
  function toggle(scope: ApiTokenScope, checked: boolean) {
    onChange(
      checked
        ? [...selectedScopes, scope]
        : selectedScopes.filter((candidate) => candidate !== scope),
    )
  }

  return (
    <fieldset className="mt-4 rounded-md border border-slate-200 p-3">
      <legend className="px-1 text-sm font-semibold text-slate-800">{t.tokenScopesLabel}</legend>
      <p className="mt-1 text-sm text-slate-600">{t.tokenScopesHelp}</p>
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        <div className="rounded-md bg-slate-50 px-3 py-2">
          <p className="text-sm font-medium text-slate-800">{t.readScopeLabel}</p>
          <p className="mt-0.5 text-xs text-slate-500">{t.readScopeDescription}</p>
        </div>
        <div className="rounded-md bg-slate-50 px-3 py-2">
          <p className="text-sm font-medium text-slate-800">{t.writeScopeLabel}</p>
          <p className="mt-0.5 text-xs text-slate-500">{t.writeScopeDescription}</p>
        </div>
      </div>
      <div className="mt-3 grid gap-2">
        {scopeGroups.map((group) => {
          const groupScopes = group.resources.flatMap((item) =>
            item.permissions.map((permission) => permission.scope),
          )
          const selectedCount = groupScopes.filter((scope) => selectedScopes.includes(scope)).length
          return (
            <details
              key={group.headingKey}
              className="group rounded-md border border-slate-200 bg-white"
            >
              <summary className="cursor-pointer list-none px-3 py-3 marker:hidden">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-semibold text-slate-900">{t[group.headingKey]}</p>
                    <p className="mt-0.5 text-xs text-slate-500">{t[group.descriptionKey]}</p>
                  </div>
                  <span className="flex shrink-0 items-center gap-2">
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                      {t.selectedScopesCount.replace('{count}', String(selectedCount))}
                    </span>
                    <IconChevronDown
                      size={16}
                      className="text-slate-500 transition-transform group-open:rotate-180"
                      aria-hidden="true"
                    />
                  </span>
                </div>
              </summary>
              <div className="border-t border-slate-200 px-3">
                {group.resources.map((item) => (
                  <div
                    key={item.labelKey}
                    className="grid gap-2 border-b border-slate-100 py-3 last:border-b-0 sm:grid-cols-[minmax(10rem,1fr)_minmax(0,2fr)]"
                  >
                    <div>
                      <p className="text-sm font-medium text-slate-800">{t[item.labelKey]}</p>
                      {item.descriptionKey ? (
                        <p className="mt-0.5 text-xs text-slate-500">{t[item.descriptionKey]}</p>
                      ) : null}
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2">
                      {item.permissions.map((permission) => (
                        <label
                          key={permission.scope}
                          aria-label={permission.scope}
                          className={`flex cursor-pointer items-start gap-2 rounded-md border border-slate-200 px-2.5 py-2 hover:bg-slate-50 ${
                            item.permissions.length === 1 && permission.action === 'write'
                              ? 'sm:col-start-2'
                              : ''
                          }`}
                        >
                          <input
                            type="checkbox"
                            name="api-token-scopes"
                            value={permission.scope}
                            checked={selectedScopes.includes(permission.scope)}
                            onChange={(event) => toggle(permission.scope, event.target.checked)}
                            className="mt-0.5"
                          />
                          <span className="min-w-0">
                            <span className="block text-xs font-medium text-slate-700">
                              {permission.action === 'read' ? t.readScopeLabel : t.writeScopeLabel}
                            </span>
                            <code className="block break-all text-[11px] text-slate-500">
                              {permission.scope}
                            </code>
                          </span>
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </details>
          )
        })}
      </div>
    </fieldset>
  )
}
