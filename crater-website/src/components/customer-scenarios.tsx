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

import type React from "react";
import { School, Building2, Server } from "lucide-react";
import { useTranslations} from "next-intl";

export function CustomerScenarios() {
  const t = useTranslations("CustomerScenarios");
  return (
    <section className="py-20 px-4">
      <div className="container mx-auto max-w-6xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-4">
          {t("heading")}
        </h2>
        <p className="text-center text-gray-600 dark:text-gray-400 mb-16 max-w-3xl mx-auto">
          {t("subheading")}
        </p>

        <div className="grid md:grid-cols-3 gap-8">
          <ScenarioCard
            icon={<School />}
            title={t("cards.education.title")}
            description={t("cards.education.description")}
            features={[
              t("cards.education.features.0"),
              t("cards.education.features.1"),
              t("cards.education.features.2"),
            ]}
          />

          <ScenarioCard
            icon={<Building2 />}
            title={t("cards.enterprise.title")}
            description= {t("cards.enterprise.description")}
            features= {[
              t("cards.enterprise.features.0"),
              t("cards.enterprise.features.1"),
              t("cards.enterprise.features.2"),
            ]}
          />

          <ScenarioCard
            icon={<Server />}
            title= {t("cards.cloudProvider.title")}
            description= {t("cards.cloudProvider.description")}
            features= {[
              t("cards.cloudProvider.features.0"),
              t("cards.cloudProvider.features.1"),
              t("cards.cloudProvider.features.2"),
            ]}
          />
        </div>
      </div>
    </section>
  );
}

function ScenarioCard({
  icon,
  title,
  description,
  features,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  features: string[];
}) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm hover:shadow-md transition-shadow p-6">
      <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg inline-block mb-4">
        <div className="text-blue-600 dark:text-blue-400">{icon}</div>
      </div>
      <h3 className="text-xl font-semibold mb-3">{title}</h3>
      <p className="text-gray-600 dark:text-gray-400 mb-4">{description}</p>
      <ul className="space-y-2">
        {features.map((feature, index) => (
          <li key={index} className="flex items-start gap-2">
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
            <span className="text-gray-700 dark:text-gray-300">{feature}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
