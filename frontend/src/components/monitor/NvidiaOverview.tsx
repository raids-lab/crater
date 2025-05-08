import Monitor from "@/pages/Embed/Monitor";
import { configGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const NvidiaOverview = () => {
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.main} />;
};

export default NvidiaOverview;
