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
import { Link, isMatch, useMatches } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { Fragment, useMemo } from 'react'

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'

import { atomBreadcrumb } from '@/utils/store'

import { cn } from '@/lib/utils'

export const NavBreadcrumb = ({ className }: { className: string }) => {
  const customItems = useAtomValue(atomBreadcrumb)
  const matches = useMatches()

  const items = useMemo(() => {
    if (customItems.length > 0) {
      return customItems
    }
    const matchesWithCrumbs = matches.filter((match) => isMatch(match, 'loaderData.crumb'))
    return matchesWithCrumbs.map(({ pathname, loaderData }) => {
      return {
        href: pathname,
        label: loaderData?.crumb,
      }
    })
  }, [matches, customItems])

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList>
        {items.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && <BreadcrumbSeparator key={`bread-separator-${index}`} />}
              {!item.href && (
                <BreadcrumbPage
                  className={cn({
                    'text-muted-foreground': items.length > 1,
                  })}
                >
                  {item.label}
                </BreadcrumbPage>
              )}
              {item.href && (
                <BreadcrumbItem key={`bread-item-${index}`}>
                  {item.href && index !== items.length - 1 ? (
                    <BreadcrumbLink asChild>
                      <Link to={item.href}>{item.label}</Link>
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage>{item.label}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
              )}
            </Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
