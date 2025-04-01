import Monitor from "@/pages/Embed/Monitor";
import { asyncGrafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const ResourseOverview = () => {
  const grafanaOverview = useAtomValue(asyncGrafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.schedule} />;
};

export default ResourseOverview;
