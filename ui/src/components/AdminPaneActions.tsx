import { IconChevronRight, IconPencil } from '@tabler/icons-react'
import type { ComponentType } from 'react'
import { Button } from './ui/button'
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
  onEdit,
  busy = false,
  actions = [],
}: {
  detailHref?: string
  onEdit?: () => void
  busy?: boolean
  actions?: PaneAction[]
}) {
  const hasSecondaryAction = Boolean(onEdit) || actions.length > 0
  return (
    <div className="flex flex-wrap items-center gap-2">
      {detailHref ? (
        <Button asChild className={hasSecondaryAction ? 'flex-1' : 'min-w-28'}>
          <a href={detailHref}>
            詳細
            <IconChevronRight size={16} aria-hidden="true" />
          </a>
        </Button>
      ) : null}
      {onEdit ? (
        <Button type="button" variant="outline" className="flex-1" disabled={busy} onClick={onEdit}>
          <IconPencil size={16} aria-hidden="true" />
          編集
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
              'flex-1',
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
