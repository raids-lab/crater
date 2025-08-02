import { createFileRoute } from '@tanstack/react-router'

import { DetailPageSearch } from '@/components/layout/detail-page'
import UserDetail from '@/components/layout/user-detail'

export const Route = createFileRoute('/portal/users/$name')({
  validateSearch: (search): DetailPageSearch => {
    return {
      tab: (search.tab as string) || '',
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  const userName = Route.useParams().name
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <UserDetail
      name={userName}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ to: '.', search: { tab } })}
    />
  )
}
