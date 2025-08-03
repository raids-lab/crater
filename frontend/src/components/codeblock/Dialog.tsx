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
import { useQuery } from '@tanstack/react-query'

import { IResponse } from '@/services/types'

import TipBadge from '../badge/TipBadge'
import Nothing from '../placeholder/nothing'
import SandwichSheet from '../sheet/SandwichSheet'

interface FetchContentProps<T> {
  name: string
  type: string
  fetchData: (name: string) => Promise<IResponse<T>>
  renderData: (data: T) => React.ReactNode
}

type FetchDialogProps<T> = React.HTMLAttributes<HTMLDivElement> &
  FetchContentProps<T> & {
    trigger: React.ReactNode
  }

export function LazyContent<T>({ name, type, fetchData, renderData }: FetchContentProps<T>) {
  const { data } = useQuery({
    queryKey: ['code', type, name],
    queryFn: () => fetchData(name),
    select: (res) => res.data,
    enabled: !!name,
  })

  if (!data) {
    return <Nothing />
  }

  return <>{renderData(data)}</>
}

export function FetchSheet<T>({ trigger, name, type, fetchData, renderData }: FetchDialogProps<T>) {
  return (
    <SandwichSheet
      title={
        <div className="flex items-center gap-1.5">
          {name}
          <TipBadge title={type} />
        </div>
      }
      trigger={trigger}
      className="sm:max-w-3xl"
    >
      <LazyContent name={name} type={type} fetchData={fetchData} renderData={renderData} />
    </SandwichSheet>
  )
}
