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

"use client";

import { Zap, Shield, Layers, RefreshCw } from "lucide-react";
import { useTranslations} from "next-intl";

export function TechnicalAdvantages() {
  const t = useTranslations("TechnicalAdvantages");

  return (
    <section id="advantages" className="py-20 px-4 bg-gray-50 dark:bg-gray-900">
      <div className="container mx-auto max-w-6xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-16">
          {t("heading")}
        </h2>

        <div className="grid md:grid-cols-2 gap-8">
          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                <Zap className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <h3 className="text-xl font-semibold">{t("cards.performance.title")}</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              {t("cards.performance.description")}
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.performance.features.0")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.performance.features.1")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.performance.features.2")}
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
                <Shield className="h-6 w-6 text-cyan-600 dark:text-cyan-400" />
              </div>
              <h3 className="text-xl font-semibold">{t("cards.security.title")}</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              {t("cards.security.description")}
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.security.features.0")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.security.features.1")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.security.features.2")}
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                <Layers className="h-6 w-6 text-purple-600 dark:text-purple-400" />
              </div>
              <h3 className="text-xl font-semibold">{t("cards.integration.title")}</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              {t("cards.integration.description")}
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.integration.features.0")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.integration.features.1")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.integration.features.2")}
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                <RefreshCw className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <h3 className="text-xl font-semibold">{t("cards.scalability.title")}</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              {t("cards.scalability.description")}
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.scalability.features.0")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.scalability.features.1")}
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  {t("cards.scalability.features.2")}
                </span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}
