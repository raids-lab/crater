/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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

export function formatParameterCount(value?: number): string {
  if (!value || value <= 0) return ''

  const units = [
    { threshold: 1e12, suffix: 'T' },
    { threshold: 1e9, suffix: 'B' },
    { threshold: 1e6, suffix: 'M' },
    { threshold: 1e3, suffix: 'K' },
  ]
  const unit = units.find(({ threshold }) => value >= threshold)
  if (!unit) return value.toLocaleString()

  const scaled = value / unit.threshold
  const digits = scaled >= 100 || Number.isInteger(scaled) ? 0 : 1
  return `${scaled.toFixed(digits)}${unit.suffix}`
}
