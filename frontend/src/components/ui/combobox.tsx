'use client'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'
import { IconCheck, IconChevronDown, IconX } from '@tabler/icons-react'

import { cn } from '@/lib/utils'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { useDictionary } from '../../lib/i18n'

export type ComboboxOption = { value: string; label: string }

type SearchableSelectProps = {
  value: string
  onValueChange: (value: string) => void
  options: ComboboxOption[]
  placeholder?: string
  emptyMessage?: string
  className?: string
  id?: string
  disabled?: boolean
  'aria-label'?: string
}

// 大量の選択肢（グループ・ユーザー等）から1件を検索して選ぶための薄いラッパー。
// 素の <select>/Select は選択肢数が多いと一覧を目視で探すしかなく破綻するため、
// 入力による絞り込みができる Base UI combobox を使う。
function SearchableSelect({
  value,
  onValueChange,
  options,
  placeholder,
  emptyMessage,
  className,
  id,
  disabled,
  'aria-label': ariaLabel,
}: SearchableSelectProps) {
  const t = useDictionary(commonDictionary)
  const resolvedPlaceholder = placeholder ?? t.searchPlaceholder
  const selected = options.find((option) => option.value === value) ?? null
  return (
    <ComboboxPrimitive.Root
      items={options}
      value={selected}
      onValueChange={(next) => onValueChange((next as ComboboxOption | null)?.value ?? '')}
      disabled={disabled}
    >
      <ComboboxPrimitive.InputGroup
        className={cn(
          'flex h-9 w-full items-center gap-1 rounded-md border border-input bg-transparent pr-1.5 pl-2.5 text-sm shadow-xs transition-[color,box-shadow] focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 dark:bg-input/30',
          className,
        )}
      >
        <ComboboxPrimitive.Input
          id={id}
          placeholder={resolvedPlaceholder}
          aria-label={ariaLabel ?? resolvedPlaceholder}
          className="h-full w-full border-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
        />
        <ComboboxPrimitive.Clear
          className="flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:text-foreground"
          aria-label={t.cancel}
        >
          <IconX className="size-3.5" aria-hidden="true" />
        </ComboboxPrimitive.Clear>
        <ComboboxPrimitive.Trigger
          className="flex size-6 shrink-0 items-center justify-center text-muted-foreground"
          aria-label={resolvedPlaceholder}
        >
          <IconChevronDown className="size-4" aria-hidden="true" />
        </ComboboxPrimitive.Trigger>
      </ComboboxPrimitive.InputGroup>

      <ComboboxPrimitive.Portal>
        <ComboboxPrimitive.Positioner className="z-50 outline-none" sideOffset={4}>
          <ComboboxPrimitive.Popup className="max-h-(--available-height) w-(--anchor-width) min-w-48 overflow-y-auto rounded-md bg-popover text-popover-foreground shadow-sm ring-1 ring-foreground/10 outline-none">
            <ComboboxPrimitive.Empty className="px-3 py-2 text-xs text-muted-foreground">
              {emptyMessage ?? t.noSearchResults}
            </ComboboxPrimitive.Empty>
            <ComboboxPrimitive.List className="p-1">
              {(item: ComboboxOption) => (
                <ComboboxPrimitive.Item
                  key={item.value}
                  value={item}
                  className="relative flex w-full cursor-default items-center gap-2 rounded-sm py-1.5 pr-8 pl-2 text-sm outline-hidden select-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
                >
                  {item.label}
                  <ComboboxPrimitive.ItemIndicator className="absolute right-2 flex size-4 items-center justify-center">
                    <IconCheck className="size-4" aria-hidden="true" />
                  </ComboboxPrimitive.ItemIndicator>
                </ComboboxPrimitive.Item>
              )}
            </ComboboxPrimitive.List>
          </ComboboxPrimitive.Popup>
        </ComboboxPrimitive.Positioner>
      </ComboboxPrimitive.Portal>
    </ComboboxPrimitive.Root>
  )
}

export { SearchableSelect }
