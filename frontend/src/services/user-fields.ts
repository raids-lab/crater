export const userFieldKeys = [
  'nickname',
  'email',
  'identity',
  'group',
  'teacher',
  'phone',
  'expiredAt',
  'createdAt',
  'lastLoginAt',
] as const

export type UserFieldKey = (typeof userFieldKeys)[number]
export type EditableUserField = Exclude<UserFieldKey, 'expiredAt' | 'createdAt' | 'lastLoginAt'>

export interface UserFieldPresentation {
  showInTable: boolean
  showInForm: boolean
  labels?: Record<string, string>
}
export type UserFields = Partial<Record<UserFieldKey, UserFieldPresentation>>

// Use an exact locale override first, then its base language. Never fall back to
// an unrelated deployment label: the caller supplies the built-in translation.
export function userFieldLabel(
  fields: UserFields,
  key: UserFieldKey,
  language: string,
  fallback: string
): string {
  const normalize = (value: string) => value.replace(/[-_]/g, '').toLowerCase()
  const locale = normalize(language)
  const base = language.split(/[-_]/)[0].toLowerCase().slice(0, 2)
  const labels = Object.entries(fields[key]?.labels ?? {})
  return (
    labels.find(([lang, label]) => normalize(lang) === locale && label.trim())?.[1] ??
    labels.find(([lang, label]) => normalize(lang) === base && label.trim())?.[1] ??
    fallback
  )
}

export function visibleUserFields(fields: UserFields, surface: 'showInTable' | 'showInForm') {
  return userFieldKeys.filter((key) => fields[key]?.[surface])
}
