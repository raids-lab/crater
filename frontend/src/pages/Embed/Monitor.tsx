// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import useFixedLayout from "@/hooks/useFixedLayout";
import { useTheme } from "@/utils/theme";
import { useMemo, type FC } from "react";

export const GrafanaIframe = ({ baseSrc }: { baseSrc: string }) => {
  const { t } = useTranslation();
  const { theme } = useTheme();

  // if baseURL does not have parameter, add ?, else if baseURL does not ends with &, add &
  const base = useMemo(() => {
    if (baseSrc.indexOf("?") === -1) {
      return `${baseSrc}?`;
    } else if (baseSrc.endsWith("&") === false) {
      return `${baseSrc}&`;
    } else {
      return baseSrc;
    }
  }, [baseSrc]);

  return (
    <iframe
      title={t("grafanaIframe.title")}
      src={`${base}theme=${theme}&kiosk&timezone=Asia%2FShanghai`}
      height={"100%"}
      width={"100%"}
    />
  );
};

const Monitor: FC<{ baseSrc: string }> = ({ baseSrc }: { baseSrc: string }) => {
  useFixedLayout();
  return (
    <div className="h-[calc(100vh_-_80px)] w-full">
      <GrafanaIframe baseSrc={baseSrc || ""} />
    </div>
  );
};

export default Monitor;
