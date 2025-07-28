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

import { HeroSection } from "@/components/hero-section";
import { WhyChooseSection } from "@/components/why-choose-section";
import { CoreCapabilities } from "@/components/core-capabilities";
import { TechnicalAdvantages } from "@/components/technical-advantages";
import { CustomerScenarios } from "@/components/customer-scenarios";
import { GetStarted } from "@/components/get-started";
import { FaqSection } from "@/components/faq-section";
import {use} from 'react';
import {setRequestLocale} from 'next-intl/server';
import {useTranslations } from "next-intl";

export async function generateStaticParams() {
  return [{ lang: "zh" }, { lang: "en" }];
}

export default function HomePage({ params }: { params: Promise<{ lang: string }> }) {
  const { lang } = use(params);
  setRequestLocale(lang);
  const tFooter = useTranslations("Footer");

  return (
    <div className="min-h-screen text-gray-900 dark:text-gray-100">
      <main>
        <HeroSection />
        <WhyChooseSection />
        <CoreCapabilities />
        <TechnicalAdvantages />
        <CustomerScenarios />
        <GetStarted />
        <FaqSection />
      </main>

      <footer className="bg-gray-100 dark:bg-gray-900 py-12 mt-20">
        <div className="container mx-auto px-4">
          <div className="flex flex-col md:flex-row justify-between items-center">
            <div className="mb-6 md:mb-0">
              <div className="flex items-center gap-2 mb-4">
                <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-blue-600 to-cyan-500"></div>
                <span className="text-lg font-bold">Crater</span>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                {tFooter("copyright", {year: new Date().getFullYear()})}
              </p>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-8">
              <div>
                <h3 className="font-medium mb-4">{tFooter("product.title")}</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("product.features")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("product.pricing")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("product.caseStudies")}
                    </a>
                  </li>
                </ul>
              </div>
              <div>
                <h3 className="font-medium mb-4">{tFooter("resources.title")}</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("resources.documentation")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("resources.blog")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("resources.community")}
                    </a>
                  </li>
                </ul>
              </div>
              <div>
                <h3 className="font-medium mb-4">{tFooter("company.title")}</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("company.aboutUs")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("company.contactUs")}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      {tFooter("company.careers")}
                    </a>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
