import Monitor from "@/pages/Embed/Monitor";
import { configGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const NetworkOverview = () => {
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.network} />;
};

export default NetworkOverview;
