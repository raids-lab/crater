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
import { useEffect, useState } from 'react'

import { NamespacedName } from '@/components/codeblock/pod-container-dialog'

/**
 * `useNamespacedState` is a custom hook that manages the state of a dialog based on a namespaced name.
 * @param namespacedName - The namespace and name of Pod
 * @param setNamespacedName - A function to set the namespaced name
 * @returns A tuple containing the dialog open state and a function to set the dialog open state
 */
const useNamespacedState = (
  namespacedName: NamespacedName,
  setNamespacedName: (namespacedName: NamespacedName) => void
) => {
  const [isOpen, setIsOpen] = useState(false)

  // if namespacedName is set, open the dialog
  // if dialog is closed, reset the namespacedName
  useEffect(() => {
    if (namespacedName) {
      setIsOpen(true)
    }
  }, [namespacedName])

  useEffect(() => {
    if (!isOpen) {
      setNamespacedName(undefined)
    }
  }, [isOpen, setNamespacedName])

  return [isOpen, setIsOpen] as const
}

export default useNamespacedState
