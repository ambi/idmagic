import { IconAlertTriangle, IconBan, IconTrash, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import type { AdminUser } from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'

// REQUIRE_USERNAME_CONFIRMATION は削除確認としてユーザー名の再入力を求める
// オプション機能のスイッチ。既定では無効。誤操作の最終防御を強めたい運用では
// true にすると、削除前に対象のユーザー名タイピングを要求する。
const REQUIRE_USERNAME_CONFIRMATION: boolean = false

// DeleteUserDialog は削除前の確認ダイアログ。mode='soft' は削除予約 (復元可能)、
// mode='purge' は完全削除 (匿名化・不可逆)。ユーザー名の再入力確認は
// REQUIRE_USERNAME_CONFIRMATION が true のときだけ求める (既定は無効)。
export function DeleteUserDialog({
  user,
  busy,
  mode,
  onClose,
  onConfirm,
}: {
  user: AdminUser
  busy: boolean
  mode: 'soft' | 'purge'
  onClose: () => void
  onConfirm: () => void
}) {
  const [confirmName, setConfirmName] = useState('')
  const canConfirm = !REQUIRE_USERNAME_CONFIRMATION || confirmName === user.preferred_username
  const purge = mode === 'purge'
  const t = useDictionary(adminUsersDictionary)

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canConfirm) return
    onConfirm()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-user-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span
              className={cn(
                'flex size-9 shrink-0 items-center justify-center rounded-full',
                purge ? 'bg-red-50 text-red-700' : 'bg-amber-50 text-amber-700',
              )}
            >
              <IconAlertTriangle size={18} aria-hidden="true" />
            </span>
            <div>
              <p
                className={cn(
                  'text-xs font-bold uppercase tracking-[0.12em]',
                  purge ? 'text-red-700' : 'text-amber-700',
                )}
              >
                {purge ? t.irreversibleAction : t.reversibleFor30Days}
              </p>
              <h2 id="delete-user-title" className="mt-1 text-xl font-semibold">
                {purge ? t.deleteUserPermanently : t.deleteUser}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {user.name || user.preferred_username} (@{user.preferred_username})
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="grid gap-5 p-6">
            {purge ? (
              <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-xs leading-5 text-red-900">
                <p className="font-semibold">{t.purgeConsequencesHeading}</p>
                <ul className="mt-1.5 list-disc pl-5">
                  <li>{t.purgeConsequenceConsents}</li>
                  <li>{t.purgeConsequenceSessions}</li>
                  <li>{t.purgeConsequenceMfa}</li>
                  <li>{t.purgeConsequenceDeviceAuth}</li>
                </ul>
                <p className="mt-2">
                  {t.purgeSubNoteLead} <code>sub</code> {t.purgeSubNoteMid}
                  <strong>{t.purgeSubNoteStrong}</strong>
                </p>
              </div>
            ) : (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-xs leading-5 text-amber-900">
                <p className="font-semibold">{t.softDeleteRestorableNotice}</p>
                <p className="mt-1.5">{t.softDeleteDescription}</p>
              </div>
            )}

            {REQUIRE_USERNAME_CONFIRMATION && (
              <div className="grid gap-2">
                <Label htmlFor="delete-user-confirm">
                  {t.confirmUsernamePrefix}{' '}
                  <span className="font-mono text-slate-700">{user.preferred_username}</span>{' '}
                  {t.confirmUsernameSuffix}
                </Label>
                <Input
                  id="delete-user-confirm"
                  value={confirmName}
                  onChange={(event) => setConfirmName(event.target.value)}
                  autoFocus
                  autoComplete="off"
                />
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
              {t.cancel}
            </Button>
            <Button type="submit" variant="destructive" disabled={busy || !canConfirm}>
              <IconTrash size={16} aria-hidden="true" />
              {purge ? t.purgeConfirm : t.confirmDelete}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

// DisableUserDialog は無効化 (disable) 前に挟む軽い確認ダイアログ。削除と違い
// 復元可能なため username typing は求めず、影響と復元動線の説明だけで確定させる
// (enable 方向はダイアログ無しで即時実行する)。
export function DisableUserDialog({
  user,
  busy,
  onClose,
  onConfirm,
}: {
  user: AdminUser
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="disable-user-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-700">
              <IconBan size={18} aria-hidden="true" />
            </span>
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.12em] text-red-700">
                {t.accountAccessBadge}
              </p>
              <h2 id="disable-user-title" className="mt-1 text-xl font-semibold">
                {t.disableAccount}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {user.name || user.preferred_username} (@{user.preferred_username})
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>

        <div className="grid gap-5 p-6">
          <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-xs leading-5 text-red-900">
            <p className="font-semibold">{t.disableConsequencesHeading}</p>
            <ul className="mt-1.5 list-disc pl-5">
              <li>{t.disableConsequenceLogin}</li>
              <li>{t.disableConsequenceSessions}</li>
              <li>{t.disableConsequenceRefresh}</li>
            </ul>
            <p className="mt-2">
              {t.disableUndoNotePrefix}{' '}
              <span className="font-semibold">{t.disableUndoNoteEmphasis}</span>{' '}
              {t.disableUndoNoteSuffix}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" variant="destructive" disabled={busy} onClick={onConfirm}>
            <IconBan size={16} aria-hidden="true" />
            {t.disableConfirm}
          </Button>
        </div>
      </Card>
    </div>
  )
}
