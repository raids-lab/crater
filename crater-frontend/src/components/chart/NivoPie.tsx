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

import useNivoTheme from '@/hooks/useNivoTheme'
import { MayHaveLabel, PieSvgProps, ResponsivePie } from '@nivo/pie'

const NivoPie = <RawDatum extends MayHaveLabel>(
  props: Omit<PieSvgProps<RawDatum>, 'width' | 'height'>
) => {
  const { nivoTheme, theme } = useNivoTheme()

  return (
    <ResponsivePie
      innerRadius={0.5}
      padAngle={0.7}
      cornerRadius={3}
      activeOuterRadiusOffset={8}
      borderColor={{
        from: 'color',
        modifiers: [['darker', 0.6]],
      }}
      arcLinkLabelsSkipAngle={10}
      arcLinkLabel={(d) => `${d.label || d.id}`}
      arcLabelsSkipAngle={10}
      colors={{ scheme: theme === 'dark' ? 'category10' : 'paired' }}
      theme={nivoTheme}
      {...props}
    />
  )
}

export default NivoPie
