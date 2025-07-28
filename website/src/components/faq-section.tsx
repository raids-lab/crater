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

import { Accordion, Accordions } from "fumadocs-ui/components/accordion";
import { useTranslations} from "next-intl";

type FaqItem = {
  question: string;
  answer: string;
};

export function FaqSection() {
  const t = useTranslations("FaqSection");

  const faqs: FaqItem[] = [
    {
      question: t("faqs.0.question"),
      answer: t("faqs.0.answer"),
    },
    {
      question: t("faqs.1.question"),
      answer: t("faqs.1.answer"),
    },
    {
      question: t("faqs.2.question"),
      answer: t("faqs.2.answer"),
    },
    {
      question: t("faqs.3.question"),
      answer: t("faqs.3.answer"),
    },
    {
      question: t("faqs.4.question"),
      answer: t("faqs.4.answer"),
    },
    {
      question: t("faqs.5.question"),
      answer: t("faqs.5.answer"),
    },
  ];

  return (
    <section id="faq" className="py-20 px-4">
      <div className="container mx-auto max-w-4xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-16">
          {t("heading")}
        </h2>
        <Accordions type="single" className="bg-fd-accent/50">
          {faqs.map((faq, index) => (
            <Accordion key={index} title={faq.question}>
              {faq.answer}
            </Accordion>
          ))}
        </Accordions>
      </div>
    </section>
  );
}
