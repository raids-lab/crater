import type { FC } from "react";
import { apiGetDataset } from "@/services/api/dataset";

import { ModelTable } from "@/components/custom/ModelTable";

const ModelDetail: FC<{ sourceType?: "model" | "sharefile" | "dataset" }> = ({
  sourceType,
}) => {
  return <ModelTable apiGetDataset={apiGetDataset} sourceType={sourceType} />;
};

export default ModelDetail;
