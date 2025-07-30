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

import { FileText, Github, Mail } from "lucide-react";
import { useTranslations } from "next-intl";
import { Link } from '@/i18n/navigation';

export function GetStarted() {
  const t = useTranslations("GetStarted");
  

  return (
    <section className="py-20 px-4 bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-950">
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            {t("heading")}
          </h2>
          <p className="text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
            {t("subheading")}
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          <Link
            href="/docs/admin/deployment/overview"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <FileText className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">{t("cards.docs.title")}</h3>
            <p className="text-gray-600 dark:text-gray-400">
              {t("cards.docs.description")}
            </p>
          </Link>

          <a
            href="https://github.com/raids-lab/crater"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <Github className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">{t("cards.github.title")}</h3>
            <p className="text-gray-600 dark:text-gray-400">
              {t("cards.github.description")}
            </p>
          </a>

          <a
            // href="mailto:sales@crater.ai"
            href="https://github.com/raids-lab/crater/issues"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <Mail className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">{t("cards.contact.title")}</h3>
            <p className="text-gray-600 dark:text-gray-400">
              {t("cards.contact.description")}
            </p>
          </a>
        </div>
      </div>
    </section>
  );
}
