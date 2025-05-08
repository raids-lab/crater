import { configAtom, initializeConfig } from "@/utils/store/config";
import { useQuery } from "@tanstack/react-query";
import { useAtom } from "jotai";
import { useEffect } from "react";

/**
 * useConfigLoader
 * @description 监听配置变化，更新配置
 */
const useConfigLoader = () => {
  const [appConfig, setAppConfig] = useAtom(configAtom);

  const { data } = useQuery({
    queryKey: ["appConfig"],
    queryFn: initializeConfig,
  });

  useEffect(() => {
    if (data) {
      // check if appConfig deep equal to data
      const isEqual = JSON.stringify(appConfig) === JSON.stringify(data);
      if (!isEqual) {
        setAppConfig(data);
        // refresh page if config changed
        window.location.reload();
      }
    }
  }, [appConfig, data, setAppConfig]);
};

export default useConfigLoader;
