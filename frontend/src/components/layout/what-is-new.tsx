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
import { QrCode, RocketIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocalStorage } from 'usehooks-ts'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

import { MarkdownRenderer } from '@/components/form/markdown-renderer'

// Current app version - update this when you release new features
const CURRENT_VERSION = '1.1.1'

interface WhatsNewDialogProps {
  // You can pass a custom version if needed
  version?: string
}

export function WhatsNewDialog({ version = CURRENT_VERSION }: WhatsNewDialogProps) {
  const { t } = useTranslation()
  const [lastConfirmedVersion, setLastConfirmedVersion] = useLocalStorage<string>(
    'app-last-confirmed-version',
    ''
  )
  const [open, setOpen] = useState(false)

  useEffect(() => {
    // Check if the current version is different from the last confirmed version
    if (lastConfirmedVersion !== version) {
      setOpen(true)
    }
  }, [lastConfirmedVersion, version])

  const handleConfirm = () => {
    setLastConfirmedVersion(version)
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t('whatsNew.title', { version })}</DialogTitle>
          <DialogDescription>{t('whatsNew.description')}</DialogDescription>
        </DialogHeader>

        <ScrollArea className="mt-4 max-h-[300px]">
          <MarkdownRenderer>{t('whatsNew.content', { version })}</MarkdownRenderer>
        </ScrollArea>

        <div className="bg-muted/50 mt-6 rounded-lg border p-4">
          <div className="flex items-start gap-4">
            <div className="bg-primary/10 rounded-lg p-2">
              <QrCode className="text-primary h-8 w-8" />
            </div>
            <div>
              <h4 className="text-sm font-medium">{t('whatsNew.communityTitle')}</h4>
              <p className="text-muted-foreground mt-1 text-sm">
                {t('whatsNew.communityDescription')}
              </p>
            </div>
          </div>
        </div>

        <DialogFooter className="mt-4">
          <Button onClick={handleConfirm}>
            {t('whatsNew.confirm')}
            <RocketIcon />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
