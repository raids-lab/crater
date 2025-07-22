// src/components/internal-link-updater.tsx

"use client";

import { useEffect } from "react";
import { checkInternalNetwork } from "@/lib/utils";

// 定义你设计的特殊分隔符
const SEPARATOR = '|||';

// 这个函数用于获取当前页面的语言代码
function getCurrentLocale(): string {
  const pathSegments = window.location.pathname.split('/');
  if (pathSegments[1] && (pathSegments[1] === 'en' || pathSegments[1] === 'cn')) {
    return pathSegments[1];
  }
  return 'en'; 
}

export function InternalLinkUpdater() {
  useEffect(() => {
    const runUpdater = async () => {
      // 1. 查找所有可能需要处理的链接
      const linksToProcess = Array.from(document.querySelectorAll<HTMLAnchorElement>('a'))
        .filter(link => link.getAttribute('href')?.includes(SEPARATOR));

      if (linksToProcess.length === 0) return;
      
      // 2. 收集所有不重复的内网主机地址，以进行一次性的并行网络检查
      const internalHosts = new Set<string>();
      linksToProcess.forEach(link => {
        // 取spilt的第二部分作为内网地址
        // 注意：这里假设链接格式为 "externalUrl|||internalUrl"


        const hrefAttr = link.getAttribute('href');
        if (!hrefAttr) return;
        const parts = hrefAttr.split(SEPARATOR);
        if (parts.length !== 2) return;
        const internalUrl = parts[1].trim();
        if (internalUrl) {
          try {
            console.log(`Checking internal URL: ${internalUrl}`);
            internalHosts.add(new URL(internalUrl).origin);
          } catch (e) { /* 忽略无效的 URL */ }
        }
      });
      
      // 3. 并行检查所有内网主机的可达性
      const hostCheckPromises = Array.from(internalHosts).map(host => 
        checkInternalNetwork(host).then(isReachable => ({ host, isReachable }))
      );
      const results = await Promise.all(hostCheckPromises);
      const reachableHosts = new Set(results.filter(r => r.isReachable).map(r => r.host));

      const locale = getCurrentLocale();

      // 4. 遍历并修复每一个链接
      for (const link of linksToProcess) {
        const hrefAttr = link.getAttribute('href');
        if (!hrefAttr) continue;
        
        const parts = hrefAttr.split(SEPARATOR);
        if (parts.length !== 2) continue;
        
        const [externalUrlTemplate, internalUrl] = parts;

        try {
          const internalHost = new URL(internalUrl).origin;
          
          // 根据网络状态选择最终的 URL
          if (reachableHosts.has(internalHost)) {
            // 内网可达，使用内网 URL
            link.href = internalUrl;
          } else {
            // 内网不可达，使用外网 URL (并替换语言占位符)
            link.href = externalUrlTemplate.replace('${locale}', locale);
          }
        } catch (e) {
          // 如果 URL 无效，则恢复为外网地址以防万一
          link.href = externalUrlTemplate.replace('${locale}', locale);
        }
      }
    };

    runUpdater();
  }, []); // 仅在客户端挂载时运行一次

  return null;
}