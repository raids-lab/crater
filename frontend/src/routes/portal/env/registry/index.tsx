import { createFileRoute } from '@tanstack/react-router'

import { KanikoListTable } from '@/components/image/registry'

import { apiUserListKaniko, apiUserRemoveKanikoList } from '@/services/api/imagepack'

export const Route = createFileRoute('/portal/env/registry/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <KanikoListTable
      apiListKaniko={apiUserListKaniko}
      apiRemoveKanikoList={apiUserRemoveKanikoList}
      isAdminMode={false}
    />
  )
}
