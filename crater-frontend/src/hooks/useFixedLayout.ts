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
import { useSetAtom } from 'jotai'
import { useEffect } from 'react'

import { atomFixedLayout } from '@/utils/store'

/**
 * `useFixedLayout` is a custom hook that sets the global fixed layout state to true when the component mounts,
 * and resets it to false when the component unmounts.
 * @returns {void}
 *
 * @example
 *
 * ```tsx
 * import useFixedLayout from "@/hooks/useFixedLayout";
 *
 * const MyComponent = () => {
 *   useFixedLayout();
 *  return <div>My Component</div>;
 * };
 * ```
 */
const useFixedLayout = (): void => {
  const setFixedLayout = useSetAtom(atomFixedLayout)

  useEffect(() => {
    setFixedLayout(true)
    return () => {
      setFixedLayout(false)
    }
  })
}

export default useFixedLayout
