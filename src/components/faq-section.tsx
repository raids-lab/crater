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

type FaqItem = {
  question: string;
  answer: string;
};

const faqs: FaqItem[] = [
  {
    question: "Crater 与 OpenPAI / Kubeflow 的区别？",
    answer:
      "Crater 深度整合了调度、数据和环境配置组件，提供更贴近生产环境的开箱即用体验。相比 OpenPAI 和 Kubeflow，Crater 更注重用户体验和资源利用率，降低了使用门槛，同时保持了高度的可扩展性。",
  },
  {
    question: "是否支持私有化部署？",
    answer:
      "是的，Crater 完全支持私有化部署，可以在裸金属服务器、VMware、OpenStack 以及主流云厂商的 Kubernetes 服务上部署。我们提供完整的部署文档和技术支持，确保您的部署顺利进行。",
  },
  {
    question: "如何保证数据安全？",
    answer:
      "Crater 采用多层次的安全机制保护您的数据，包括网络隔离、访问控制、数据加密等。所有数据处理都在您的私有环境中进行，不会上传到外部服务器，确保数据安全和隐私。",
  },
  {
    question: "支持哪些深度学习框架？",
    answer:
      "Crater 支持主流的深度学习框架，包括但不限于 PyTorch、TensorFlow、JAX、MXNet 等。我们提供这些框架的预配置环境模板，您可以根据需要选择和定制。",
  },
  {
    question: "如何进行资源配额管理？",
    answer:
      "Crater 提供细粒度的资源配额管理功能，管理员可以为不同的用户组或项目设置 GPU、CPU、内存等资源的使用限制，确保资源的合理分配和使用。",
  },
  {
    question: "是否提供商业支持？",
    answer:
      "是的，我们提供企业级的商业支持服务，包括技术咨询、定制开发、培训和 7x24 小时故障响应等。请通过官方渠道联系我们了解详情。",
  },
];

export function FaqSection() {
  return (
    <section id="faq" className="py-20 px-4">
      <div className="container mx-auto max-w-4xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-16">
          常见问题
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
