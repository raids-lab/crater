import { useTheme } from "@/utils/theme";
import { type FC } from "react";

const GPU_DASHBOARD = import.meta.env.VITE_GRAFANA_GPU_DASHBOARD;

export const GrafanaIframe = ({ baseSrc }: { baseSrc: string }) => {
  const { theme } = useTheme();
  return (
    <iframe
      title="grafana"
      src={`${baseSrc}?theme=${theme}&kiosk`}
      height={"100%"}
      width={"100%"}
    />
  );
};

const Monitor: FC = () => {
  return (
    <div className="h-[calc(100vh_-_80px)] w-full">
      <GrafanaIframe baseSrc={GPU_DASHBOARD} />
    </div>
  );
};

export default Monitor;
