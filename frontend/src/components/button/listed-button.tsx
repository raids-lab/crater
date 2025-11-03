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
import { ChevronDownIcon } from 'lucide-react'
import { ReactNode, useEffect } from 'react'
import { useLocalStorage } from 'usehooks-ts'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'

export interface SplitButtonItem {
  key: string
  title: string
  action: () => void
  disabled?: boolean
}

interface SplitButtonProps {
  icon: ReactNode
  renderTitle: (itemTitle?: string) => ReactNode
  itemTitle: string
  items: SplitButtonItem[]
  cacheKey: string
}

const ListedButton = ({ icon, renderTitle, itemTitle, items, cacheKey }: SplitButtonProps) => {
  const [position, setPosition] = useLocalStorage(`split-button-${cacheKey}`, items[0].key)

  useEffect(() => {
    // if position is not in items, or item is disabled, set position to first not disabled
    if (
      !items.find((item) => item.key === position) ||
      items.find((item) => item.key === position)?.disabled
    ) {
      setPosition(items.find((item) => !item.disabled)?.key || items[0].key)
    }
  }, [items, position, setPosition])

  return (
    <div className="bg-primary hover:bg-primary/90 flex h-9 w-fit items-center rounded-md transition-colors">
      <Button
        className="text-primary-foreground hover:text-primary-foreground cursor-pointer rounded-r-none pr-3 capitalize shadow-none hover:bg-transparent dark:hover:bg-transparent"
        variant="ghost"
        onClick={() => items.find((item) => item.key === position)?.action()}
      >
        {icon}
        {renderTitle(items.find((item) => item.key === position)?.title)}
      </Button>
      <Separator orientation="vertical" className="bg-background" />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            className="text-primary-foreground cursor-pointer rounded-l-none pr-3 pl-2 shadow-none hover:bg-transparent focus:ring-0 dark:hover:bg-transparent"
            variant="ghost"
          >
            <ChevronDownIcon className="text-primary-foreground size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" alignOffset={-5} className="w-[200px]" forceMount>
          <DropdownMenuLabel className="text-muted-foreground text-xs">
            {itemTitle}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup value={position} onValueChange={setPosition}>
            {items.map((item) => (
              <DropdownMenuRadioItem key={item.key} value={item.key} disabled={item.disabled}>
                {item.title}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

export default ListedButton
