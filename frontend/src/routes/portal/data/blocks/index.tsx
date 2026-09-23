import { createFileRoute } from '@tanstack/react-router'

import { DataView } from '@/components/file/data-view'

import { apiGetDataset, apiGetDatasetPaged } from '@/services/api/dataset'

export const Route = createFileRoute('/portal/data/blocks/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <DataView
      apiGetDataset={apiGetDataset}
      apiGetDatasetPaged={apiGetDatasetPaged}
      sourceType="sharefile"
    />
  )
}
