/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// i18n-processed-v1.1.0 (no translatable strings)
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { LucideIcon } from 'lucide-react'
import { useCallback, useMemo } from 'react'
import CardTitle from '../label/CardTitle'

type AccordionCardProps = React.HTMLAttributes<HTMLDivElement> & {
  cardTitle: string
  icon: LucideIcon
  open: boolean
  setOpen?: (open: boolean) => void
}

const AccordionCard = ({
  cardTitle,
  open,
  setOpen,
  children,
  className,
  ...props
}: AccordionCardProps) => {
  // 处理值变化，转换回布尔值
  const handleValueChange = useCallback(
    (newValue: string) => {
      if (setOpen) {
        setOpen(newValue === cardTitle)
      }
    },
    [setOpen, cardTitle]
  )

  // 将 open 布尔值转换为 Accordion 需要的 value 字符串
  const value = useMemo(() => (open ? cardTitle : ''), [open, cardTitle])

  return (
    <Card className={cn(className, 'p-0')} {...props}>
      <Accordion type="single" collapsible value={value} onValueChange={handleValueChange}>
        <AccordionItem value={cardTitle} className="border-b-0 py-2">
          <AccordionTrigger className="mx-6 cursor-pointer leading-none tracking-tight hover:no-underline">
            <CardTitle icon={props.icon}>{cardTitle}</CardTitle>
          </AccordionTrigger>
          <AccordionContent className="px-6">{children}</AccordionContent>
        </AccordionItem>
      </Accordion>
    </Card>
  )
}

export default AccordionCard
