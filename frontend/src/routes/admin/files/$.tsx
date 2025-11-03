import { createFileRoute } from '@tanstack/react-router'

import FileSystem from '@/components/file/file-system-table'

import { apiGetAdminFiles } from '@/services/api/file'

export const Route = createFileRoute('/admin/files/$')({
  component: RouteComponent,
})

function RouteComponent() {
  const path = Route.useParams()._splat
  return <FileSystem apiGetFiles={apiGetAdminFiles} path={path || ''} />
}
