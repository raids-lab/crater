import { createFileRoute } from '@tanstack/react-router'

import RegistryDetail from '@/components/image/registry/registry-detail'
import { detailValidateSearch } from '@/components/layout/detail-page'

export const Route = createFileRoute('/portal/env/registry/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
})

function RouteComponent() {
  const name = Route.useParams().name
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <RegistryDetail
      name={name}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ to: '.', search: { tab } })}
    />
  )
}
