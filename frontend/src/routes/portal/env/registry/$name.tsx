import { createFileRoute } from '@tanstack/react-router'

import RegistryDetail from '@/components/image/registry/registry-detail'
import { detailValidateSearch } from '@/components/layout/detail-page'
import NotFound from '@/components/placeholder/not-found'

export const Route = createFileRoute('/portal/env/registry/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  errorComponent: () => <NotFound />,
  loader: ({ params }) => ({ crumb: params.name }),
})

function RouteComponent() {
  const name = Route.useParams().name
  const { tab, id } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <RegistryDetail
      name={name}
      id={id}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ search: { tab, id } })}
    />
  )
}
