import { ShieldBanIcon, ShieldCheckIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

export function UserBanStatusBadge({ banned }: { banned: boolean }) {
  const { t } = useTranslation()
  return banned ? (
    <Badge variant="destructive" className="whitespace-nowrap">
      <ShieldBanIcon className="mr-1 size-3" />
      {t('userBan.status.banned')}
    </Badge>
  ) : (
    <Badge variant="secondary" className="whitespace-nowrap">
      <ShieldCheckIcon className="mr-1 size-3" />
      {t('userBan.status.normal')}
    </Badge>
  )
}
