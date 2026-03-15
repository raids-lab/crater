/**
 * Copyright 2025 Crater
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export const CHART_YAML_URL =
  'https://github.com/raids-lab/crater/blob/main/charts/crater/Chart.yaml';

const PLACEHOLDER = '<chart-version>';

/**
 * Returns the Crater Helm Chart version from build-time env.
 * Empty or placeholder value means "not injected" (e.g. build without Chart.yaml).
 */
export function getChartVersion(): string {
  const v = process.env.NEXT_PUBLIC_CRATER_CHART_VERSION;
  if (!v || v === PLACEHOLDER) return '';
  return v;
}

export function isChartVersionInjected(): boolean {
  return getChartVersion() !== '';
}
