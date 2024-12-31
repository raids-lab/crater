import { useTheme } from "@/utils/theme";
import { useMemo, type FC } from "react";

const GRAFANA_NODE = import.meta.env.VITE_GRAFANA_NODE;

const Monitor: FC = () => {
  const { theme } = useTheme();

  const grafanaTheme = useMemo(() => {
    if (theme === "dark") {
      return "dark";
    }
    return "light";
  }, [theme]);

  return (
    <div className="h-[calc(100vh_-_80px)] w-full">
      <iframe
        title="monitor"
        src={`${GRAFANA_NODE}?orgId=1&refresh=5m&var-datasource=prometheus&theme=${grafanaTheme}&kiosk`}
        height={"100%"}
        width={"100%"}
      />
    </div>
  );
};

export default Monitor;
