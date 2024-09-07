import type { FC } from "react";
import { apiGetDataset } from "@/services/api/dataset";

import { DatasetTable } from "@/components/custom/DatasetTable";

const DatasetDetail: FC = () => {
  return <DatasetTable apiGetDataset={apiGetDataset} />;
};

export default DatasetDetail;
