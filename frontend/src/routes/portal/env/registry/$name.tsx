import { createFileRoute } from '@tanstack/react-router'

import RegistryDetail from '@/components/image/registry/registry-detail'
import { detailValidateSearch } from '@/components/layout/detail-page'
import NotFound from '@/components/placeholder/not-found'

import { queryBuildDetail } from '@/services/query/image'

export const Route = createFileRoute('/portal/env/registry/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  loader: async ({ params, context: { queryClient } }) => {
    const { data } = await queryClient.ensureQueryData(queryBuildDetail(params.name))
    return {
      crumb: data?.description || params.name,
    }
  },
  errorComponent: () => <NotFound />,
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
