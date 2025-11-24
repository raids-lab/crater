import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'

import { DetailPageSearch, detailValidateSearch } from '@/components/layout/detail-page'
import { ModelDownloadDetail } from '@/components/model/model-download-detail'

export const Route = createFileRoute('/portal/data/datasets/downloads/$id')({
  component: RouteComponent,
  validateSearch: detailValidateSearch,
})

function RouteComponent() {
  const { tab } = Route.useSearch() as DetailPageSearch
  const [currentTab, setCurrentTab] = useState(tab)

  return <ModelDownloadDetail currentTab={currentTab} setCurrentTab={setCurrentTab} />
}
