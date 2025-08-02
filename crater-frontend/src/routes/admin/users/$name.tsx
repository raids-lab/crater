import { createFileRoute } from '@tanstack/react-router'

import { detailValidateSearch } from '@/components/layout/detail-page'
import UserDetail from '@/components/layout/user-detail'

export const Route = createFileRoute('/admin/users/$name')({
  validateSearch: detailValidateSearch,
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
