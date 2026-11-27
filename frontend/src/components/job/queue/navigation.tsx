import { Link } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { ListOrdered } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { TabsList, TabsTrigger } from '@/components/ui/tabs'

import { globalJobUrl } from '@/utils/store'

export function QueueShortcut() {
  const { t } = useTranslation()
  const jobUrl = useAtomValue(globalJobUrl)
  if (jobUrl !== 'vcjobs') return null
  return (
    <Button asChild variant="outline">
      <Link to="/portal/overview" search={{ tab: 'queue', mine: true }}>
        <ListOrdered />
        {t('queueBoard.viewQueue')}
      </Link>
    </Button>
  )
}

export function JobPageTabs({
  items,
  label,
}: {
  items: { value: string; label: string }[]
  label: string
}) {
  return (
    <div className="border-b">
      <TabsList
        aria-label={label}
        className="h-auto max-w-full flex-wrap justify-start gap-x-3 rounded-none bg-transparent p-0 sm:gap-x-6"
      >
        {items.map((item) => (
          <TabsTrigger
            key={item.value}
            value={item.value}
            className="text-muted-foreground data-[state=active]:text-primary data-[state=active]:border-b-primary rounded-none border-0 border-b-2 px-0 py-2.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none dark:data-[state=active]:bg-transparent"
          >
            {item.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </div>
  )
}
