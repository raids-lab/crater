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
import {
  Database,
  Code,
  Cpu,
  BarChart3,
  GitBranch,
  Monitor,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslations} from "next-intl";

export function CoreCapabilities() {
  const t = useTranslations("CoreCapabilities");

  return (
    <section
      id="capabilities"
      className="py-24 px-4 bg-gradient-to-b from-transparent to-gray-50 dark:to-gray-900/30"
    >
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-20">
          <h2 className="text-4xl md:text-4xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-gray-900 to-gray-600 dark:from-white dark:to-gray-400">
            {t("title")}
          </h2>
          <p className="text-lg text-gray-600 dark:text-gray-400 max-w-3xl mx-auto">
            {t("description")}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          <CapabilityCard
            icon={<Database className="text-white" />}
            title={t("cards.dataManagement.title")}
            features={[
              t("cards.dataManagement.features.0"),
              t("cards.dataManagement.features.1"),
              t("cards.dataManagement.features.2"),
            ]}
            color="from-blue-600 to-blue-400"
          />

          <CapabilityCard
            icon={<Code className="text-white" />}
            title= {t("cards.environmentSetup.title")}
            features={[
              t("cards.environmentSetup.features.0"),
              t("cards.environmentSetup.features.1"),
              t("cards.environmentSetup.features.2"),
            ]}
            color="from-cyan-600 to-cyan-400"
          />

          <CapabilityCard
            icon={<Cpu className="text-white" />}
            title= {t("cards.modelTraining.title")}
            features={[
              t("cards.modelTraining.features.0"),
              t("cards.modelTraining.features.1"),
              t("cards.modelTraining.features.2"),
            ]}
            color="from-purple-600 to-purple-400"
          />

          <CapabilityCard
            icon={<BarChart3 className="text-white" />}
            title= {t("cards.performanceMonitoring.title")}
            features={[
              t("cards.performanceMonitoring.features.0"),
              t("cards.performanceMonitoring.features.1"),
              t("cards.performanceMonitoring.features.2"),
            ]}
            color="from-green-600 to-green-400"
          />

          <CapabilityCard
            icon={<GitBranch className="text-white" />}
            title= {t("cards.versionControl.title")}
            features={[
              t("cards.versionControl.features.0"),
              t("cards.versionControl.features.1"),
              t("cards.versionControl.features.2"),
            ]}
            color="from-orange-600 to-orange-400"
          />

          <CapabilityCard
            icon={<Monitor className="text-white" />}
            title= {t("cards.modelDeployment.title")}
            features={[
              t("cards.modelDeployment.features.0"),
              t("cards.modelDeployment.features.1"),
              t("cards.modelDeployment.features.2"),
            ]}
            color="from-red-600 to-red-400"
          />
        </div>
      </div>
    </section>
  );
}

function CapabilityCard({
  icon,
  title,
  features,
  color,
}: {
  icon: React.ReactNode;
  title: string;
  features: string[];
  color: string;
}) {
  return (
    <div className="group bg-white dark:bg-gray-800/80 rounded-xl shadow-lg hover:shadow-xl duration-300 overflow-hidden border border-gray-100 dark:border-gray-700 backdrop-blur-sm">
      <div className={cn("h-2 bg-gradient-to-r", color)}></div>
      <div className="p-8">
        <div className="flex items-center gap-4 mb-6">
          <div
            className={cn(
              "p-3 rounded-xl bg-gradient-to-br shadow-sm",
              color,
              "duration-300"
            )}
          >
            {icon}
          </div>
          <h3 className="text-xl font-bold">{title}</h3>
        </div>
        <ul className="space-y-3">
          {features.map((feature, index) => (
            <li key={index} className="flex items-start gap-3">
              <span
                className={cn(
                  "inline-block w-2 h-2 rounded-full bg-gradient-to-r mt-2",
                  color
                )}
              ></span>
              <span className="text-gray-700 dark:text-gray-300">
                {feature}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
