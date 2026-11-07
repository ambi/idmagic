import { cn } from '../../lib/utils'

export function ReadSetting({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="rounded-lg border border-slate-200/80 bg-white/70 px-3 py-2.5">
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className={cn('mt-0.5 text-sm font-medium text-slate-900', mono && 'font-mono')}>
        {value}
      </dd>
    </div>
  )
}
