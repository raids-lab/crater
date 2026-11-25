import { useAtomValue, useStore } from 'jotai'

import { getAccountListScope } from '@/services/query/account-list-scope'

import { atomUserContext, atomUserInfo } from '@/utils/store'

export function useAccountListScope() {
  const store = useStore()
  const userId = useAtomValue(atomUserInfo)?.id
  const context = useAtomValue(atomUserContext)

  return {
    scope: getAccountListScope(userId, context),
    isCurrentScope: () =>
      store.get(atomUserInfo)?.id === userId && store.get(atomUserContext) === context,
  }
}
