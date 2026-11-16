import { createFileRoute } from '@tanstack/react-router'

import RegistryDetail from '@/components/image/registry/registry-detail'
import RegistryDetailError from '@/components/image/registry/registry-detail-error'
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'

import { queryBuildDetail } from '@/services/query/image'

export const Route = createFileRoute('/admin/env/registry/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  errorComponent: RegistryDetailError,
  loader: async ({ params, context: { queryClient } }) => {
    const { data } = await queryClient.ensureQueryData(queryBuildDetail(params.name, true))
    return {
      crumb: data?.description || params.name,
    }
  },
})

function RouteComponent() {
  const name = Route.useParams().name
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <RegistryDetail
      name={name}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab))}
    />
  )
}
