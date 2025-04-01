import Monitor from "@/pages/Embed/Monitor";
import { asyncGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const NetworkOverview = () => {
  const grafanaOverview = useAtomValue(asyncGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.network} />;
};

export default NetworkOverview;
