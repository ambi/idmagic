import { type FormEvent, useState } from 'react'
import { updateAdminApplicationProvisioning } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import { messageOf, parseList, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import {
  CredentialFieldsEditor,
  credentialInputFrom,
  emptyCredentialFields,
} from './AdminApplicationProvisioningShared'
import type {
  AttributeMappingRule,
  DeprovisionPolicy,
  GroupPushConfig,
  ProvisioningAuthMethod,
  ProvisioningConnection,
  ProvisioningFeatureFlags,
  ProvisioningGroupSelection,
  ProvisioningScope,
} from '../../types'

function defaultDeprovisionPolicy(): DeprovisionPolicy {
  return {
    on_unassign: 'deactivate',
    on_delete: 'deactivate',
    on_group_deleted_or_unassigned: 'none',
    grace_period_days: 0,
    accidental_deletion_count_threshold: null,
    accidental_deletion_percent_threshold: null,
  }
}

export function ConnectionSettingsForm({
  csrfToken,
  applicationID,
  connection,
  onSaved,
}: {
  csrfToken: string
  applicationID: string
  connection: ProvisioningConnection
  onSaved: (conn: ProvisioningConnection) => void
}) {
  const t = useDictionary(provisioningDictionary)
  const [baseURL, setBaseURL] = useState(connection.base_url)
  const [flags, setFlags] = useState<ProvisioningFeatureFlags>(connection.feature_flags)
  const [scope, setScope] = useState<ProvisioningScope>(connection.scope)
  const [groupPushEnabled, setGroupPushEnabled] = useState(connection.group_push != null)
  const [groupPush, setGroupPush] = useState<GroupPushConfig>(
    connection.group_push ?? {
      selection: 'assigned_groups',
      explicit_group_ids: [],
      display_name_source: '',
    },
  )
  const [mappingJSON, setMappingJSON] = useState(
    JSON.stringify(connection.attribute_mappings, null, 2),
  )
  const [conflictMatchAttribute, setConflictMatchAttribute] = useState(
    connection.matching.conflict_match_attribute,
  )
  const [policy, setPolicy] = useState<DeprovisionPolicy>(
    connection.deprovision_policy ?? defaultDeprovisionPolicy(),
  )
  const [rateLimit, setRateLimit] = useState(String(connection.rate_limit_per_minute))
  const [maxAttempts, setMaxAttempts] = useState(String(connection.max_attempts))
  const [notificationEmail, setNotificationEmail] = useState(connection.notification_email ?? '')
  const [quarantineThreshold, setQuarantineThreshold] = useState(
    String(connection.quarantine_after_consecutive_failures),
  )
  const [rotateCredential, setRotateCredential] = useState(false)
  const [authMethod, setAuthMethod] = useState<ProvisioningAuthMethod>(
    connection.credential.auth_method,
  )
  const [credFields, setCredFields] = useState(emptyCredentialFields())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    setNotice('')
    let attributeMappings: AttributeMappingRule[]
    try {
      const parsed = JSON.parse(mappingJSON)
      if (!Array.isArray(parsed)) throw new Error('not an array')
      attributeMappings = parsed as AttributeMappingRule[]
    } catch {
      setError(t.invalidAttributeMappingJsonError)
      setSaving(false)
      return
    }
    try {
      const conn = await updateAdminApplicationProvisioning(csrfToken, applicationID, {
        base_url: baseURL,
        feature_flags: flags,
        scope,
        group_push: groupPushEnabled
          ? {
              selection: groupPush.selection,
              explicit_group_ids: groupPush.explicit_group_ids,
              display_name_source: groupPush.display_name_source,
            }
          : null,
        attribute_mappings: attributeMappings,
        matching: { conflict_match_attribute: conflictMatchAttribute },
        deprovision_policy: policy,
        rate_limit_per_minute: Number.parseInt(rateLimit, 10) || 0,
        max_attempts: Number.parseInt(maxAttempts, 10) || 1,
        notification_email: notificationEmail || null,
        quarantine_after_consecutive_failures: Number.parseInt(quarantineThreshold, 10) || 1,
        credential: rotateCredential ? credentialInputFrom(authMethod, credFields) : undefined,
      })
      onSaved(conn)
      setRotateCredential(false)
      setNotice(t.savedNotice)
    } catch (cause) {
      setError(messageOf(cause, t.saveFailedError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="grid gap-6 p-5">
      <SectionTitle>{t.connectionSettingsHeading}</SectionTitle>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {notice ? <Alert variant="success">{notice}</Alert> : null}
      <form className="grid gap-6" onSubmit={(e) => void submit(e)}>
        <div className="grid gap-1.5">
          <Label>{t.baseUrlFieldLabel}</Label>
          <Input required value={baseURL} onChange={(e) => setBaseURL(e.target.value)} />
        </div>

        <section className="grid gap-2 border-t border-slate-100 pt-5">
          <SectionTitle>{t.featureFlagsHeading}</SectionTitle>
          <div className="grid gap-2 sm:grid-cols-2">
            {(
              [
                ['create_users', t.createUsersLabel],
                ['update_users', t.updateUsersLabel],
                ['deactivate_users', t.deactivateUsersLabel],
                ['delete_users', t.deleteUsersLabel],
                ['push_groups', t.pushGroupsLabel],
              ] as [keyof ProvisioningFeatureFlags, string][]
            ).map(([key, label]) => (
              <label key={key} className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  className="size-4 rounded border-slate-300"
                  checked={flags[key]}
                  onChange={(e) => setFlags({ ...flags, [key]: e.target.checked })}
                />
                {label}
              </label>
            ))}
          </div>
        </section>

        <section className="grid gap-1.5 border-t border-slate-100 pt-5">
          <Label>{t.scopeFieldLabel}</Label>
          <Select
            value={scope}
            onValueChange={(v) => setScope(v as ProvisioningScope)}
            options={[
              { value: 'assigned_only', label: t.scopeAssignedOnlyLabel },
              { value: 'all_users', label: t.scopeAllUsersLabel },
            ]}
            className="max-w-sm"
          />
        </section>

        <GroupPushSection
          groupPushEnabled={groupPushEnabled}
          setGroupPushEnabled={setGroupPushEnabled}
          groupPush={groupPush}
          setGroupPush={setGroupPush}
        />

        <section className="grid gap-1.5 border-t border-slate-100 pt-5">
          <SectionTitle>{t.attributeMappingHeading}</SectionTitle>
          <p className="text-xs text-slate-500">{t.attributeMappingHelp}</p>
          <Label>{t.attributeMappingJsonFieldLabel}</Label>
          <textarea
            className="min-h-40 rounded-lg border border-slate-300 bg-white/92 px-3.5 py-2 font-mono text-xs text-slate-950 outline-none focus:border-blue-600 focus:ring-3 focus:ring-blue-600/10"
            value={mappingJSON}
            onChange={(e) => setMappingJSON(e.target.value)}
          />
        </section>

        <section className="grid gap-1.5 border-t border-slate-100 pt-5">
          <SectionTitle>{t.matchingHeading}</SectionTitle>
          <Label>{t.conflictMatchAttributeFieldLabel}</Label>
          <Input
            value={conflictMatchAttribute}
            onChange={(e) => setConflictMatchAttribute(e.target.value)}
            className="max-w-sm"
          />
          <p className="text-xs text-slate-500">{t.conflictMatchAttributeHelp}</p>
        </section>

        <DeprovisionPolicySection policy={policy} setPolicy={setPolicy} />

        <section className="grid gap-3 border-t border-slate-100 pt-5">
          <SectionTitle>{t.reliabilityHeading}</SectionTitle>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label>{t.rateLimitFieldLabel}</Label>
              <Input
                type="number"
                min={1}
                value={rateLimit}
                onChange={(e) => setRateLimit(e.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.maxAttemptsFieldLabel}</Label>
              <Input
                type="number"
                min={1}
                value={maxAttempts}
                onChange={(e) => setMaxAttempts(e.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.notificationEmailFieldLabel}</Label>
              <Input
                type="email"
                value={notificationEmail}
                onChange={(e) => setNotificationEmail(e.target.value)}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.quarantineThresholdFieldLabel}</Label>
              <Input
                type="number"
                min={1}
                value={quarantineThreshold}
                onChange={(e) => setQuarantineThreshold(e.target.value)}
              />
            </div>
          </div>
        </section>

        <section className="grid gap-3 border-t border-slate-100 pt-5">
          <SectionTitle>{t.rotateCredentialHeading}</SectionTitle>
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              className="size-4 rounded border-slate-300"
              checked={rotateCredential}
              onChange={(e) => setRotateCredential(e.target.checked)}
            />
            {t.rotateCredentialToggle}
          </label>
          {rotateCredential ? (
            <>
              <p className="text-xs text-slate-500">{t.rotateCredentialHelp}</p>
              <CredentialFieldsEditor
                authMethod={authMethod}
                setAuthMethod={setAuthMethod}
                fields={credFields}
                setFields={setCredFields}
              />
            </>
          ) : null}
        </section>

        <div>
          <Button type="submit" disabled={saving}>
            {saving ? t.saving : t.saveButton}
          </Button>
        </div>
      </form>
    </Card>
  )
}

function GroupPushSection({
  groupPushEnabled,
  setGroupPushEnabled,
  groupPush,
  setGroupPush,
}: {
  groupPushEnabled: boolean
  setGroupPushEnabled: (v: boolean) => void
  groupPush: GroupPushConfig
  setGroupPush: (v: GroupPushConfig) => void
}) {
  const t = useDictionary(provisioningDictionary)
  return (
    <section className="grid gap-3 border-t border-slate-100 pt-5">
      <SectionTitle>{t.groupPushHeading}</SectionTitle>
      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input
          type="checkbox"
          className="size-4 rounded border-slate-300"
          checked={groupPushEnabled}
          onChange={(e) => setGroupPushEnabled(e.target.checked)}
        />
        {t.groupPushEnableLabel}
      </label>
      {groupPushEnabled ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label>{t.groupSelectionFieldLabel}</Label>
            <Select
              value={groupPush.selection}
              onValueChange={(v) =>
                setGroupPush({ ...groupPush, selection: v as ProvisioningGroupSelection })
              }
              options={[
                { value: 'assigned_groups', label: t.groupSelectionAssignedLabel },
                { value: 'explicit', label: t.groupSelectionExplicitLabel },
              ]}
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t.displayNameSourceFieldLabel}</Label>
            <Input
              value={groupPush.display_name_source ?? ''}
              onChange={(e) => setGroupPush({ ...groupPush, display_name_source: e.target.value })}
            />
          </div>
          {groupPush.selection === 'explicit' ? (
            <div className="grid gap-1.5 sm:col-span-2">
              <Label>{t.explicitGroupIdsFieldLabel}</Label>
              <textarea
                className="min-h-24 rounded-lg border border-slate-300 bg-white/92 px-3.5 py-2 font-mono text-xs text-slate-950 outline-none focus:border-blue-600 focus:ring-3 focus:ring-blue-600/10"
                value={(groupPush.explicit_group_ids ?? []).join('\n')}
                onChange={(e) =>
                  setGroupPush({ ...groupPush, explicit_group_ids: parseList(e.target.value) })
                }
              />
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

function DeprovisionPolicySection({
  policy,
  setPolicy,
}: {
  policy: DeprovisionPolicy
  setPolicy: (p: DeprovisionPolicy) => void
}) {
  const t = useDictionary(provisioningDictionary)
  return (
    <section className="grid gap-3 border-t border-slate-100 pt-5">
      <SectionTitle>{t.deprovisionPolicyHeading}</SectionTitle>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="grid gap-1.5">
          <Label>{t.onUnassignFieldLabel}</Label>
          <Select
            value={policy.on_unassign}
            onValueChange={(v) =>
              setPolicy({ ...policy, on_unassign: v as DeprovisionPolicy['on_unassign'] })
            }
            options={[
              { value: 'deactivate', label: t.deprovisionActionDeactivate },
              { value: 'delete', label: t.deprovisionActionDelete },
              { value: 'none', label: t.deprovisionActionNone },
            ]}
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t.onDeleteFieldLabel}</Label>
          <Select
            value={policy.on_delete}
            onValueChange={(v) =>
              setPolicy({ ...policy, on_delete: v as DeprovisionPolicy['on_delete'] })
            }
            options={[
              { value: 'deactivate', label: t.deprovisionActionDeactivate },
              { value: 'delete', label: t.deprovisionActionDelete },
              { value: 'none', label: t.deprovisionActionNone },
            ]}
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t.onGroupDeletedFieldLabel}</Label>
          <Select
            value={policy.on_group_deleted_or_unassigned || 'none'}
            onValueChange={(v) =>
              setPolicy({
                ...policy,
                on_group_deleted_or_unassigned:
                  v as DeprovisionPolicy['on_group_deleted_or_unassigned'],
              })
            }
            options={[
              { value: 'delete', label: t.deprovisionActionDelete },
              { value: 'none', label: t.deprovisionActionNone },
            ]}
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t.gracePeriodDaysFieldLabel}</Label>
          <Input
            type="number"
            min={0}
            value={String(policy.grace_period_days)}
            onChange={(e) =>
              setPolicy({ ...policy, grace_period_days: Number.parseInt(e.target.value, 10) || 0 })
            }
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t.accidentalDeletionCountThresholdFieldLabel}</Label>
          <Input
            type="number"
            min={0}
            value={policy.accidental_deletion_count_threshold?.toString() ?? ''}
            onChange={(e) =>
              setPolicy({
                ...policy,
                accidental_deletion_count_threshold:
                  e.target.value === '' ? null : Number.parseInt(e.target.value, 10),
              })
            }
          />
        </div>
        <div className="grid gap-1.5">
          <Label>{t.accidentalDeletionPercentThresholdFieldLabel}</Label>
          <Input
            type="number"
            min={1}
            max={100}
            value={policy.accidental_deletion_percent_threshold?.toString() ?? ''}
            onChange={(e) =>
              setPolicy({
                ...policy,
                accidental_deletion_percent_threshold:
                  e.target.value === '' ? null : Number.parseInt(e.target.value, 10),
              })
            }
          />
        </div>
      </div>
    </section>
  )
}
