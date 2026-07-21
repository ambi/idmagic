import { IconChevronRight, IconPencil } from '@tabler/icons-react'
import type { ComponentType } from 'react'
import { Button } from './ui/button'
import { adminPaneActionsDictionary } from './AdminPaneActions.i18n'
import { useDictionary } from '../lib/i18n'
import { cn } from '../lib/utils'

// PaneAction は右ペインの二次アクション (削除・無効化など) を表す記述子。
// tone='danger' は破壊的操作の赤系ボタン。
export type PaneAction = {
  label: string
  icon?: ComponentType<{ size?: number; 'aria-hidden'?: boolean }>
  onClick: () => void
  tone?: 'default' | 'danger'
  disabled?: boolean
}

// AdminPaneActions は一覧画面の右ペイン共通のアクション行 (wi-39)。
// どのエンティティ (ユーザー / アプリケーション / グループ / ロール) でも
// 「詳細」→「編集」→ その他操作 の順で同じ配置・同じ体裁にそろえる。
// 二次アクションは以前 ⋮ メニューに隠していたが、他の一覧画面 (署名鍵・テナント・
// 認可詳細の種類) が直接ボタンで見せているのに合わせ、全画面で直接表示に統一する
// (wi-126 §7、方針: 表示してよい)。編集や操作が無いエンティティでは省略する。
export function AdminPaneActions({
  detailHref,
  editHref,
  onEdit,
  busy = false,
  actions = [],
}: {
  detailHref?: string
  editHref?: string
  onEdit?: () => void
  busy?: boolean
  actions?: PaneAction[]
}) {
  const hasSecondaryAction = Boolean(editHref || onEdit) || actions.length > 0
  const t = useDictionary(adminPaneActionsDictionary)

  // 詳細だけの場合 (例: ロール一覧) は 1 ボタンをそのまま置く。
  if (!hasSecondaryAction) {
    return detailHref ? (
      <Button asChild className="min-w-28">
        <a href={detailHref}>
          {t.detail}
          <IconChevronRight size={16} aria-hidden="true" />
        </a>
      </Button>
    ) : null
  }

  // ボタン総数に応じてレイアウトを切り替える。3 個まで (グループ・アプリの
  // 詳細/編集/削除 など) は 1 行に均等幅で並べる。ユーザーだけは
  // 詳細/編集/無効化/削除 の 4 個になるため、2 列グリッドにして 2×2 に収め、
  // 端数のボタンが 1 つだけ全幅に伸びて悪目立ちしないようにする。
  const count = (detailHref ? 1 : 0) + (editHref || onEdit ? 1 : 0) + actions.length
  const useGrid = count >= 4
  const itemClass = useGrid ? '' : 'flex-1'
  return (
    <div className={useGrid ? 'grid grid-cols-2 gap-2' : 'flex flex-wrap items-center gap-2'}>
      {detailHref ? (
        <Button asChild className={itemClass}>
          <a href={detailHref}>
            {t.detail}
            <IconChevronRight size={16} aria-hidden="true" />
          </a>
        </Button>
      ) : null}
      {editHref ? (
        <Button asChild variant="outline" className={itemClass}>
          <a href={editHref}>
            <IconPencil size={16} aria-hidden="true" />
            {t.edit}
          </a>
        </Button>
      ) : onEdit ? (
        <Button
          type="button"
          variant="outline"
          className={itemClass}
          disabled={busy}
          onClick={onEdit}
        >
          <IconPencil size={16} aria-hidden="true" />
          {t.edit}
        </Button>
      ) : null}
      {actions.map((action) => {
        const Icon = action.icon
        return (
          <Button
            key={action.label}
            type="button"
            variant="outline"
            className={cn(
              itemClass,
              action.tone === 'danger' &&
                'border-red-200 text-red-700 hover:border-red-300 hover:bg-red-50',
            )}
            disabled={busy || action.disabled}
            onClick={action.onClick}
          >
            {Icon ? <Icon size={16} aria-hidden={true} /> : null}
            {action.label}
          </Button>
        )
      })}
    </div>
  )
}
