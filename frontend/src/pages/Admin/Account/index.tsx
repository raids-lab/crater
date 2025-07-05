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

// i18n-processed-v1.1.0
import { useTranslation } from 'react-i18next'
import { IAccount } from '@/services/api/account'
import { useState } from 'react'
import { AccountTable } from './account-table'
import { AccountForm } from './account-form-component'
import SandwichSheet from '@/components/sheet/SandwichSheet'

export const Component = () => {
  const { t } = useTranslation()
  const [isOpen, setIsOpen] = useState(false)
  const [currentAccount, setCurrentAccount] = useState<IAccount | null>(null)

  return (
    <>
      <AccountTable setIsOpen={setIsOpen} setCurrentAccount={setCurrentAccount} />
      <SandwichSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title={currentAccount ? t('accountForm.editTitle') : t('accountForm.createTitle')}
        description={t('accountForm.description')}
        className="sm:max-w-4xl"
      >
        <AccountForm onOpenChange={setIsOpen} account={currentAccount} />
      </SandwichSheet>
    </>
  )
}
