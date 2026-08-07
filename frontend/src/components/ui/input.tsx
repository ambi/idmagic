import type { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export function Input({ className, type, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      type={type}
      className={cn(
        'h-11 w-full rounded-md border border-slate-300 bg-white px-3.5 py-2 text-[0.875rem] text-slate-950 outline-none transition-[border-color,box-shadow]',
        'placeholder:text-slate-400 hover:border-slate-400 focus:border-accent focus:ring-3 focus:ring-accent/15',
        'disabled:cursor-not-allowed disabled:bg-slate-100 disabled:opacity-60',
        className,
      )}
      {...props}
    />
  )
}
