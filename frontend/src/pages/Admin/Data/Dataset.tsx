import { type FC } from "react";
import { apiAdminGetDataset } from "@/services/api/dataset";

import { DatasetTable } from "@/components/custom/DatasetTable";

const AdminDataset: FC = () => {
  return <DatasetTable apiGetDataset={apiAdminGetDataset} />;
};
export default AdminDataset;
