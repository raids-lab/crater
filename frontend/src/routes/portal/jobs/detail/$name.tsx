import { createFileRoute } from '@tanstack/react-router'

import BaseCore from '@/components/job/detail'
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'
import NotFound from '@/components/placeholder/not-found'

import { queryJobDetail } from '@/services/query/job'

export const Route = createFileRoute('/portal/jobs/detail/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  errorComponent: () => <NotFound />,
  loader: async ({ params, location, context: { queryClient } }) => {
    const id = detailValidateSearch(location.search).id
    const { data } = await queryClient.ensureQueryData(queryJobDetail(params.name, id))
    return {
      crumb: data.name ?? params.name,
    }
  },
})

function RouteComponent() {
  const jobName = Route.useParams().name
  const { tab, id } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <BaseCore
      jobName={jobName}
      jobID={id}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab, id))}
    />
  )
}
