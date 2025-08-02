import { createFileRoute } from '@tanstack/react-router'

import { KanikoListTable } from '@/components/image/registry'

import { apiUserDeleteKanikoList, apiUserListKaniko } from '@/services/api/imagepack'

export const Route = createFileRoute('/portal/env/registry/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <KanikoListTable
      apiListKaniko={apiUserListKaniko}
      apiDeleteKanikoList={apiUserDeleteKanikoList}
      isAdminMode={false}
    />
  )
}
