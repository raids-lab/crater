import { createFileRoute } from '@tanstack/react-router'

import { KanikoListTable } from '@/components/image/registry'

import { apiAdminDeleteKanikoList, apiAdminListKaniko } from '@/services/api/admin/imagepack'

export const Route = createFileRoute('/admin/env/registry/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <KanikoListTable
      apiListKaniko={apiAdminListKaniko}
      apiRemoveKanikoList={apiAdminDeleteKanikoList}
      isAdminMode={true}
    />
  )
}
