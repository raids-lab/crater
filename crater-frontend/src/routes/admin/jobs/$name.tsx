import { createFileRoute } from '@tanstack/react-router'

import BaseCore from '@/components/job/detail'
import { detailValidateSearch } from '@/components/layout/detail-page'

export const Route = createFileRoute('/admin/jobs/$name')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
  loader: ({ params }) => ({ crumb: params.name }),
})

function RouteComponent() {
  const jobName = Route.useParams().name
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <BaseCore
      jobName={jobName}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ to: '.', search: { tab } })}
    />
  )
}
