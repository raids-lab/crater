import { useTheme } from "@/utils/theme";
import { type FC } from "react";

const GPU_DASHBOARD = import.meta.env.VITE_GRAFANA_GPU_DASHBOARD;

const Monitor: FC = () => {
  const { theme } = useTheme();

  return (
    <div className="h-[calc(100vh_-_80px)] w-full">
      <iframe
        title="monitor"
        src={`${GPU_DASHBOARD}?theme=${theme}&kiosk`}
        height={"100%"}
        width={"100%"}
      />
    </div>
  );
};

export default Monitor;
