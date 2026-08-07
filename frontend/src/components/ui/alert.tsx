import type { HTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

type AlertVariant = 'destructive' | 'success'

type AlertProps = HTMLAttributes<HTMLDivElement> & {
  variant?: AlertVariant
}

const variantStyles: Record<AlertVariant, string> = {
  destructive: 'border-red-200 bg-red-50 text-red-950',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
}

export function Alert({ className, variant = 'destructive', ...props }: AlertProps) {
  return (
    <div
      role={variant === 'destructive' ? 'alert' : 'status'}
      className={cn('rounded-md border p-3.5 text-sm', variantStyles[variant], className)}
      {...props}
    />
  )
}
