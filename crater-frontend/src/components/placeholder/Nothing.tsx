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

import { cn } from '@/lib/utils'
import { GridIcon } from 'lucide-react'

export const NothingCore = ({ title, className }: { title?: string; className?: string }) => {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-4', className)}>
      <div className="bg-muted rounded-full p-3">
        <GridIcon className="text-muted-foreground h-6 w-6" />
      </div>
      <p className="text-muted-foreground text-sm select-none">{title ?? '暂无数据'}</p>
    </div>
  )
}

const Nothing = ({ title }: { title?: string }) => {
  return (
    <div className="text-muted-foreground/85 flex h-[calc(100vh_-_300px)] w-full items-center justify-center text-center hover:bg-transparent">
      <NothingCore title={title} />
    </div>
  )
}

export default Nothing
