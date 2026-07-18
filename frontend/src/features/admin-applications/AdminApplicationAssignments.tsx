import { IconUserPlus, IconX } from '@tabler/icons-react'
import { type FormEvent, useEffect, useMemo, useState } from 'react'
import {
  assignApplication,
  listAdminGroups,
  listAdminUsers,
  listApplicationAssignments,
  unassignApplication,
} from '../../api'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import {
  adminApplicationsDictionary,
  type AdminApplicationsDictionary,
} from './AdminApplicationsPage.i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import type { AdminGroup, AdminUser, ApplicationAssignment } from '../../types'

function useAssignmentData(
  appID: string,
  onError: (msg: string) => void,
  t: AdminApplicationsDictionary,
) {
  const [assignments, setAssignments] = useState<ApplicationAssignment[]>([])
  const [users, setUsers] = useState<AdminUser[]>([])
  const [groups, setGroups] = useState<AdminGroup[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let cancelled = false
    void Promise.all([listApplicationAssignments(appID), listAdminUsers(), listAdminGroups()])
      .then(([a, u, g]) => {
        if (cancelled) return
        setAssignments(a)
        setUsers(u)
        setGroups(g)
        setLoaded(true)
      })
      .catch((cause) => onError(messageOf(cause, t.assignmentFetchFailedError)))
    return () => {
      cancelled = true
    }
  }, [appID, onError, t.assignmentFetchFailedError])

  return { assignments, setAssignments, users, groups, loaded }
}

function useDisplayName(users: AdminUser[], groups: AdminGroup[]) {
  const userName = useMemo(() => new Map(users.map((u) => [u.id, u.preferred_username])), [users])
  const groupName = useMemo(() => new Map(groups.map((g) => [g.id, g.name])), [groups])
  return (a: ApplicationAssignment): string => {
    if (a.subject_type === 'user') return userName.get(a.subject_id) ?? a.subject_id
    return groupName.get(a.subject_id) ?? a.subject_id
  }
}

function AssignmentChip({ a, displayName }: { a: ApplicationAssignment; displayName: string }) {
  const t = useDictionary(adminApplicationsDictionary)
  return (
    <span className="flex items-center gap-2">
      <span
        className={`rounded px-1.5 py-0.5 text-xs ${
          a.subject_type === 'user' ? 'bg-blue-50 text-blue-700' : 'bg-violet-50 text-violet-700'
        }`}
      >
        {a.subject_type === 'user' ? t.userTypeLabel : t.groupTypeLabel}
      </span>
      <span className="font-medium text-slate-800">{displayName}</span>
    </span>
  )
}

export function AssignmentList({
  appID,
  onError,
}: {
  appID: string
  onError: (msg: string) => void
}) {
  const t = useDictionary(adminApplicationsDictionary)
  const { assignments, users, groups, loaded } = useAssignmentData(appID, onError, t)
  const displayName = useDisplayName(users, groups)

  if (!loaded) return <p className="text-xs text-slate-400">{t.loadingNotice}</p>
  if (assignments.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-400">
        {t.noAssignmentsNotice}
      </p>
    )
  }
  return (
    <ul className="grid gap-2">
      {assignments.map((a) => (
        <li
          key={`${a.subject_type}:${a.subject_id}`}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
        >
          <AssignmentChip a={a} displayName={displayName(a)} />
        </li>
      ))}
    </ul>
  )
}

export function AssignmentManager({
  appID,
  csrfToken,
  onError,
}: {
  appID: string
  csrfToken: string
  onError: (msg: string) => void
}) {
  const t = useDictionary(adminApplicationsDictionary)
  const { assignments, setAssignments, users, groups, loaded } = useAssignmentData(
    appID,
    onError,
    t,
  )
  const displayName = useDisplayName(users, groups)
  const [subjectType, setSubjectType] = useState<'user' | 'group'>('user')
  const [subjectID, setSubjectID] = useState('')
  const [busy, setBusy] = useState(false)

  const assignedKeys = useMemo(
    () => new Set(assignments.map((a) => `${a.subject_type}:${a.subject_id}`)),
    [assignments],
  )

  const options: SelectOption[] = useMemo(() => {
    const source =
      subjectType === 'user'
        ? users.map((u) => ({ value: u.id, label: u.preferred_username }))
        : groups.map((g) => ({ value: g.id, label: g.name }))
    return source.filter((o) => !assignedKeys.has(`${subjectType}:${o.value}`))
  }, [subjectType, users, groups, assignedKeys])

  async function add(event: FormEvent) {
    event.preventDefault()
    if (!subjectID) return
    setBusy(true)
    try {
      const created = await assignApplication(csrfToken, appID, {
        subject_type: subjectType,
        subject_id: subjectID,
      })
      setAssignments((current) => [...current, created])
      setSubjectID('')
    } catch (cause) {
      onError(messageOf(cause, t.assignmentAddFailedError))
    } finally {
      setBusy(false)
    }
  }

  async function remove(a: ApplicationAssignment) {
    try {
      await unassignApplication(csrfToken, appID, a.subject_type, a.subject_id)
      setAssignments((current) =>
        current.filter(
          (x) => !(x.subject_type === a.subject_type && x.subject_id === a.subject_id),
        ),
      )
    } catch (cause) {
      onError(messageOf(cause, t.assignmentRemoveFailedError))
    }
  }

  return (
    <section className="grid gap-3">
      <SectionTitle>{t.assignmentsHeading}</SectionTitle>
      <p className="text-xs text-slate-500">{t.assignmentsManagerHelp}</p>
      {!loaded ? (
        <p className="text-xs text-slate-400">{t.loadingNotice}</p>
      ) : assignments.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-400">
          {t.noAssignmentsShortNotice}
        </p>
      ) : (
        <ul className="grid gap-2">
          {assignments.map((a) => (
            <li
              key={`${a.subject_type}:${a.subject_id}`}
              className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            >
              <AssignmentChip a={a} displayName={displayName(a)} />
              <Button
                variant="ghost"
                className="text-rose-700 hover:bg-rose-50"
                onClick={() => remove(a)}
              >
                <IconX size={14} aria-hidden="true" />
                {t.unassign}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <form className="flex flex-wrap items-end gap-2" onSubmit={add}>
        <div className="grid gap-1.5">
          <Label>{t.targetFieldLabel}</Label>
          <Select
            value={subjectType}
            onValueChange={(v) => {
              setSubjectType(v as 'user' | 'group')
              setSubjectID('')
            }}
            options={[
              { value: 'user', label: t.userTypeLabel },
              { value: 'group', label: t.groupTypeLabel },
            ]}
            className="w-32"
          />
        </div>
        <div className="grid flex-1 gap-1.5">
          <Label>{subjectType === 'user' ? t.selectUserFieldLabel : t.selectGroupFieldLabel}</Label>
          <Select
            value={subjectID}
            onValueChange={setSubjectID}
            options={options}
            placeholder={options.length === 0 ? t.noTargetsNotice : t.selectPlaceholder}
            className="w-full"
            disabled={options.length === 0}
          />
        </div>
        <Button type="submit" disabled={busy || !subjectID}>
          <IconUserPlus size={16} aria-hidden="true" />
          {t.assign}
        </Button>
      </form>
    </section>
  )
}
