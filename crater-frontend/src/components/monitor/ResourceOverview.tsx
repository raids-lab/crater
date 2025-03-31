import Monitor from "@/pages/Embed/Monitor";
import { grafanaOverviewAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";

const ResourseOverview = () => {
  const grafanaOverview = useAtomValue(grafanaOverviewAtom);
  return <Monitor baseSrc={grafanaOverview.schedule} />;
};

export default ResourseOverview;
