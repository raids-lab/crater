import { useCallback, useState } from "react";
import { apiAccountGetByName } from "@/services/api/account";
import { toast } from "sonner";

/**
 * 获取账户昵称的工具函数
 */
export const useAccountNameLookup = () => {
  // 使用本地缓存存储已查询的账户昵称
  const [nicknameCache, setNicknameCache] = useState<Record<string, string>>(
    {},
  );
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  /**
   * 通过账户名获取对应的昵称
   * @param name 账户名
   * @returns 对应的昵称，如果未找到则返回undefined
   */
  const getNicknameByName = useCallback(
    (name: string): string | undefined => {
      // 如果缓存中有，直接返回
      if (nicknameCache[name]) {
        return nicknameCache[name];
      }

      // 缓存中没有，则异步获取并更新缓存（但这里返回同步结果）
      if (name) {
        setIsLoading(true);
        apiAccountGetByName(name)
          .then((response) => {
            const nickname = response.data?.data?.nickname;
            if (nickname) {
              setNicknameCache((prev) => ({ ...prev, [name]: nickname }));
            }
          })
          .catch((err) => {
            setError(err as Error);
            // toast.error(`获取账户 ${name} 的昵称失败: ${err.message}`);
          })
          .finally(() => {
            setIsLoading(false);
          });
      }

      // 第一次调用时返回undefined，后续依赖缓存更新触发重新渲染
      return undefined;
    },
    [nicknameCache],
  );

  // 提供一个预加载方法，可以在组件挂载时批量预加载账户昵称
  const preloadNicknames = useCallback(
    (names: string[]) => {
      names.forEach((name) => {
        if (!nicknameCache[name] && name) {
          apiAccountGetByName(name)
            .then((response) => {
              const nickname = response.data?.data?.nickname;
              if (nickname) {
                setNicknameCache((prev) => ({ ...prev, [name]: nickname }));
              }
            })
            .catch((err) => {
              toast.error(`预加载账户 ${name} 昵称失败:`, err);
            });
        }
      });
    },
    [nicknameCache],
  );

  return {
    getNicknameByName,
    preloadNicknames,
    nicknameCache,
    isLoading,
    error,
  };
};

export default useAccountNameLookup;
