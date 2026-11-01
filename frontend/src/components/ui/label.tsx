'use client'

import type * as React from 'react'

import { cn } from '@/lib/utils'
import { useDictionary } from '@/lib/i18n'
import { commonDictionary } from '@/lib/i18n/common.i18n'

function Label({
  className,
  required,
  optional,
  children,
  ...props
}: React.ComponentProps<'label'> & { required?: boolean; optional?: boolean }) {
  const t = useDictionary(commonDictionary)
  const label = (
    // biome-ignore lint/a11y/noLabelWithoutControl: htmlFor は呼び出し側から受け取る
    <label
      data-slot="label"
      className={cn(
        'flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50',
        className,
      )}
      {...props}
    >
      {children}
    </label>
  )
  if (!required && !optional) return label
  return (
    <span className="flex items-center gap-2">
      {label}
      <span aria-hidden="true" className="text-xs font-normal text-slate-500">
        {required ? t.required : t.optional}
      </span>
    </span>
  )
}

export { Label }
