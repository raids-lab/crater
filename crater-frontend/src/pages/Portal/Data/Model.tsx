import type { FC } from "react";
import { apiGetDataset } from "@/services/api/dataset";

import { ModelTable } from "@/components/custom/ModelTable";

const ModelDetail: FC = () => {
  return <ModelTable apiGetDataset={apiGetDataset} />;
};

export default ModelDetail;
