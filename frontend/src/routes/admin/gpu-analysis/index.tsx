import { createFileRoute } from '@tanstack/react-router'

import GpuAnalysisOverview from './-components/gpu-analysis-table'

export const Route = createFileRoute('/admin/gpu-analysis/')({
  component: Component,
})

function Component() {
  return <GpuAnalysisOverview />
}
