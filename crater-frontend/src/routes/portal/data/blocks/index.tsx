import { createFileRoute } from '@tanstack/react-router'

import { DataView } from '@/components/file/data-view'

import { apiGetDataset } from '@/services/api/dataset'

export const Route = createFileRoute('/portal/data/blocks/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <DataView apiGetDataset={apiGetDataset} sourceType="sharefile" />
}
