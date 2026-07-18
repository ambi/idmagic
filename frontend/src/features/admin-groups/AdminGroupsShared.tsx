export function parseRoles(value: string) {
  return [
    ...new Set(
      value
        .split(',')
        .map((role) => role.trim())
        .filter(Boolean),
    ),
  ]
}

export function optionalValue(value: FormDataEntryValue | null) {
  const normalized = String(value ?? '').trim()
  return normalized || undefined
}
