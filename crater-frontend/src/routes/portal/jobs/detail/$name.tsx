import { createFileRoute } from '@tanstack/react-router'

import BaseCore from '@/components/job/detail'
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'
import NotFound from '@/components/placeholder/not-found'

import { queryJobDetail } from '@/services/query/job'

export const Route = createFileRoute('/portal/jobs/detail/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  errorComponent: () => <NotFound />,
  loader: async ({ params, context: { queryClient } }) => {
    const { data } = await queryClient.ensureQueryData(queryJobDetail(params.name))
    return {
      crumb: data.name ?? params.name,
    }
  },
})

function RouteComponent() {
  const jobName = Route.useParams().name
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <BaseCore
      jobName={jobName}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab))}
    />
  )
}
