import Monitor from "@/pages/Embed/Monitor";
import { asyncGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const NvidiaOverview = () => {
  const grafanaOverview = useAtomValue(asyncGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.main} />;
};

export default NvidiaOverview;
