import { createFileRoute } from '@tanstack/react-router'

import StorageManagementPage from './index'

export const Route = createFileRoute('/admin/storage')({
  component: StorageManagementPage,
})
