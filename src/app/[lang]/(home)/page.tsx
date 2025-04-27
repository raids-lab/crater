import { HeroSection } from "@/components/hero-section";
import { WhyChooseSection } from "@/components/why-choose-section";
import { CoreCapabilities } from "@/components/core-capabilities";
import { TechnicalAdvantages } from "@/components/technical-advantages";
import { CustomerScenarios } from "@/components/customer-scenarios";
import { GetStarted } from "@/components/get-started";
import { FaqSection } from "@/components/faq-section";

export async function generateStaticParams() {
  return [{ lang: "cn" }, { lang: "en" }];
}

export default function HomePage() {
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
                © {new Date().getFullYear()} Crater. 保留所有权利。
              </p>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-8">
              <div>
                <h3 className="font-medium mb-4">产品</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      功能
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      定价
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      案例研究
                    </a>
                  </li>
                </ul>
              </div>
              <div>
                <h3 className="font-medium mb-4">资源</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      文档
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      博客
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      社区
                    </a>
                  </li>
                </ul>
              </div>
              <div>
                <h3 className="font-medium mb-4">公司</h3>
                <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      关于我们
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      联系我们
                    </a>
                  </li>
                  <li>
                    <a
                      href="#"
                      className="hover:text-blue-600 dark:hover:text-blue-400"
                    >
                      加入我们
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
