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
  showToAll?: boolean
  editable?: boolean
  ldapManaged?: boolean
  labels?: Record<string, string>
}
export type UserFields = Partial<Record<UserFieldKey, UserFieldPresentation>>

// Web and CLI use normalized locale keys in lexical order for a stable fallback.
export function userFieldLabel(
  fields: UserFields,
  key: UserFieldKey,
  language: string,
  fallback: string
): string {
  const normalize = (value: string) => value.replace(/[-_]/g, '').toLowerCase()
  const locale = normalize(language)
  const base = language.split(/[-_]/)[0].toLowerCase().slice(0, 2)
  const labels = Object.entries(fields[key]?.labels ?? {}).filter(([, label]) => label.trim())
  labels.sort(([a], [b]) => {
    const left = normalize(a),
      right = normalize(b)
    return left < right ? -1 : left > right ? 1 : a < b ? -1 : a > b ? 1 : 0
  })
  return (
    labels.find(([lang]) => normalize(lang) === locale)?.[1] ??
    labels.find(([lang]) => normalize(lang) === base)?.[1] ??
    labels[0]?.[1] ??
    fallback
  )
}
export function visibleUserFields(fields: UserFields, editableOnly = false) {
  return userFieldKeys.filter((key) => fields[key] && (!editableOnly || fields[key]?.editable))
}
