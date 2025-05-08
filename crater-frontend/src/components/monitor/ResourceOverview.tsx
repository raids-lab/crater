import Monitor from "@/pages/Embed/Monitor";
import { configGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const ResourseOverview = () => {
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.schedule} />;
};

export default ResourseOverview;
