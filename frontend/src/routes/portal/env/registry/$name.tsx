import { createFileRoute } from '@tanstack/react-router'

import RegistryDetail from '@/components/image/registry/registry-detail'
import RegistryDetailError from '@/components/image/registry/registry-detail-error'
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'

import { queryBuildDetail } from '@/services/query/image'

import { atomUserContext } from '@/utils/store'

export const Route = createFileRoute('/portal/env/registry/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  errorComponent: RegistryDetailError,
  loader: ({ params, context: { queryClient, store } }) => {
    const account = store.get(atomUserContext)?.queue
    if (account) {
      void queryClient.prefetchQuery(queryBuildDetail(params.name, false, account))
    }
    return { crumb: params.name }
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
