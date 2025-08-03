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
import { VariantProps } from 'class-variance-authority'
import { useAtomValue } from 'jotai'
import { BookOpenIcon } from 'lucide-react'

import { Button, buttonVariants } from '@/components/ui/button'

import { configUrlWebsiteBaseAtom } from '@/utils/store/config'

import { cn } from '@/lib/utils'

type DocsButtonProps = React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    title: string
    url: string
  }

const DocsButton = ({ title, url, variant, className, ...props }: DocsButtonProps) => {
  const website = useAtomValue(configUrlWebsiteBaseAtom)
  return (
    <Button
      variant={variant ?? 'secondary'}
      className={cn('cursor-pointer', className)}
      {...props}
      asChild
    >
      <a href={`${website}/docs/user/${url}`} target="_blank" rel="noopener noreferrer">
        <BookOpenIcon />
        {title}
      </a>
    </Button>
  )
}

export default DocsButton
