import { createFileRoute } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { useTranslation } from 'react-i18next'

import { Tabs, TabsContent } from '@/components/ui/tabs'

import AdminJobOverview from '@/components/job/overview/admin-jobs'
import { QueueBoard } from '@/components/job/queue/board'
import { JobPageTabs } from '@/components/job/queue/navigation'
import PageTitle from '@/components/layout/page-title'

import { globalJobUrl } from '@/utils/store'

export const Route = createFileRoute('/admin/jobs/')({
  validateSearch: (raw: Record<string, unknown>): { tab?: 'queue' } => ({
    tab: raw.tab === 'queue' ? 'queue' : undefined,
  }),
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  const volcano = useAtomValue(globalJobUrl) === 'vcjobs'
  if (!volcano) return <AdminJobOverview />
  return (
    <Tabs
      value={tab ?? 'all'}
      onValueChange={(value) =>
        void navigate({ search: { tab: value === 'queue' ? 'queue' : undefined } })
      }
      className="min-w-0 gap-4"
    >
      <PageTitle title={t('queueBoard.management')} />
      <JobPageTabs
        label={t('queueBoard.management')}
        items={[
          { value: 'all', label: t('queueBoard.allJobs') },
          { value: 'queue', label: t('queueBoard.queueJobs') },
        ]}
      />
      <TabsContent value="all">
        <AdminJobOverview hideHeading />
      </TabsContent>
      <TabsContent value="queue">
        <QueueBoard admin />
      </TabsContent>
    </Tabs>
  )
}
