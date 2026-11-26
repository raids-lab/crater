import { createFileRoute } from '@tanstack/react-router'

import { KanikoListTable } from '@/components/image/registry'

import { apiUserListKanikoPage, apiUserRemoveKanikoList } from '@/services/api/imagepack'

export const Route = createFileRoute('/portal/env/registry/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <KanikoListTable
      apiListKaniko={apiUserListKanikoPage}
      apiRemoveKanikoList={apiUserRemoveKanikoList}
      isAdminMode={false}
    />
  )
}
