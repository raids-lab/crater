import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtomValue } from 'jotai'

import VolcanoOverview from '@/components/job/overview/custom-jobs'
import ColocateOverview from '@/components/job/overview/emias-jobs'

import { globalJobUrl } from '@/utils/store'

export const Route = createFileRoute('/portal/jobs/custom/')({
  loader: () => {
    return {
      crumb: t('navigation.customJobs'),
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  const jobType = useAtomValue(globalJobUrl)
  return jobType === 'aijobs' ? <ColocateOverview /> : <VolcanoOverview />
}
