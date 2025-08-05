import { createFileRoute } from '@tanstack/react-router'

import FileSystem from '@/components/file/FileSystemTable'

import { apiGetFiles } from '@/services/api/file'

export const Route = createFileRoute('/portal/files/$')({
  component: RouteComponent,
})

function RouteComponent() {
  const path = Route.useParams()._splat
  return <FileSystem apiGetFiles={apiGetFiles} path={path || ''} />
}
