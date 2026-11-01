import { useAtomValue } from 'jotai'

import { atomUserContext, atomUserInfo } from '@/utils/store'

export const useInferenceKeys = () => {
  const user = useAtomValue(atomUserInfo)
  const account = useAtomValue(atomUserContext)
  const scope = ['kthena/inference-services', user?.id, account?.space] as const
  return { scope, detail: (name?: string) => [...scope, name] as const }
}
