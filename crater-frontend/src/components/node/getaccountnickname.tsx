import { useCallback, useState, useRef } from "react";
import { apiAccountGetByName } from "@/services/api/account";
import { toast } from "sonner";
/**
 * 获取账户昵称的工具函数
 */
export const useAccountNameLookup = () => {
  // 使用 ref 存储缓存，避免触发重新渲染
  const nicknameCache = useRef<Record<string, string>>({});
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const pendingRequests = useRef<Set<string>>(new Set());

  const getNicknameByName = useCallback(
    (name: string): string | undefined => {
      // 如果缓存中有，直接返回
      if (nicknameCache.current[name]) {
        return nicknameCache.current[name];
      }

      // 如果正在请求中，返回 undefined
      if (pendingRequests.current.has(name)) {
        return undefined;
      }

      // 缓存中没有且不在请求中，则异步获取
      if (name) {
        pendingRequests.current.add(name);
        setIsLoading(true);

        apiAccountGetByName(name)
          .then((response) => {
            const nickname = response.data?.data?.nickname;
            if (nickname) {
              // 存储到 ref 中，不触发重新渲染
              nicknameCache.current[name] = nickname;
            }
          })
          .catch((err) => {
            setError(err as Error);
            toast.error(`获取账户 ${name} 的昵称失败:`, err);
          })
          .finally(() => {
            pendingRequests.current.delete(name);
            setIsLoading(false);
          });
      }

      return undefined;
    },
    [], // 无依赖项，避免死循环
  );

  return {
    getNicknameByName,
    isLoading,
    error,
  };
};

export default useAccountNameLookup;
