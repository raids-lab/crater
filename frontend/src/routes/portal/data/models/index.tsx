import { createFileRoute } from '@tanstack/react-router'

import { DataView } from '@/components/file/data-view'

import { apiGetDataset } from '@/services/api/dataset'

export const Route = createFileRoute('/portal/data/models/')({
  validateSearch: (search: Record<string, unknown>) => ({
    organization: typeof search.organization === 'string' ? search.organization : undefined,
  }),
  component: RouteComponent,
})

function RouteComponent() {
  const { organization } = Route.useSearch()
  return <DataView apiGetDataset={apiGetDataset} sourceType="model" organization={organization} />
}
