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
import { Event as KubernetesEvent } from 'kubernetes-types/core/v1'

import TipBadge from '@/components/badge/tip-badge'
import {
  Timeline,
  TimelineBody,
  TimelineConnector,
  TimelineHeader,
  TimelineIcon,
  TimelineItem,
  TimelineTitle,
} from '@/components/layout/timeline/timeline'

import { cn } from '@/lib/utils'

import { TimeDistance } from '../../custom/time-distance'

interface TimelineLayoutProps {
  items: KubernetesEvent[] // Replace any[] with the actual type of items.
}

export const EventTimeline = ({ items }: TimelineLayoutProps) => {
  return (
    <Timeline>
      {items
        .sort((a, b) => {
          if (!a.lastTimestamp) {
            return -1
          }
          if (!b.lastTimestamp) {
            return 1
          }
          return new Date(b.lastTimestamp).getTime() - new Date(a.lastTimestamp).getTime()
        })
        .map((item, index) => (
          <TimelineItem key={index}>
            {index < items.length - 1 && <TimelineConnector />}
            <TimelineHeader>
              <TimelineIcon>
                <div className="bg-primary h-2 w-2 rounded-full" />
              </TimelineIcon>
              <TimelineTitle>
                {item.reason && <>{item.reason}</>}
                {item.lastTimestamp && (
                  <span className="">
                    <span className="text-primary mx-1">-</span>
                    <TimeDistance date={item.lastTimestamp} />
                  </span>
                )}
              </TimelineTitle>
            </TimelineHeader>
            <TimelineBody>
              <div className="mb-2 flex flex-row items-center gap-1.5">
                <TipBadge
                  title={item.type}
                  className={cn({
                    'bg-secondary text-secondary-foreground hover:bg-accent capitalize':
                      item.type === 'Normal',
                  })}
                />
                {item.involvedObject.kind && (
                  <TipBadge
                    title={item.involvedObject.kind}
                    className={cn({
                      'bg-secondary text-secondary-foreground hover:bg-accent capitalize':
                        item.type === 'Normal',
                    })}
                  />
                )}
                {item.involvedObject.name && (
                  <TipBadge
                    title={item.involvedObject.name}
                    className={cn({
                      'bg-secondary text-secondary-foreground hover:bg-accent':
                        item.type === 'Normal',
                    })}
                  />
                )}
                {item.reportingComponent && (
                  <TipBadge
                    title={item.reportingComponent}
                    className={cn({
                      'bg-secondary text-secondary-foreground hover:bg-accent':
                        item.type === 'Normal',
                    })}
                  />
                )}
              </div>
              <p className="text-muted-foreground font-mono text-sm">{item.message}</p>
            </TimelineBody>
          </TimelineItem>
        ))}
    </Timeline>
  )
}
