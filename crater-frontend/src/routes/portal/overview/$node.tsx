import { createFileRoute } from '@tanstack/react-router'

import { detailValidateSearch } from '@/components/layout/detail-page'
import NodeDetail from '@/components/node/node-detail'

export const Route = createFileRoute('/portal/overview/$node')({
  validateSearch: detailValidateSearch,
  loader: ({ params }) => {
    const { node } = params
    return {
      crumb: node,
    }
  },
  component: NodeDetailPage,
})

function NodeDetailPage() {
  const nodeName = Route.useParams().node
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <NodeDetail
      nodeName={nodeName}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ to: '.', search: { tab } })}
    />
  )
}
