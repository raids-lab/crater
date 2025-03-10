import { ImageListTable } from "@/pages/Portal/Image/Image";
import {
  apiAdminChangeImageDescription,
  apiAdminChangeImagePublicStatus,
  apiAdminChangeImageTaskType,
  apiAdminDeleteImageList,
  apiAdminListImage,
} from "@/services/api/admin/imagepack";
import { type FC } from "react";

export const Component: FC = () => {
  return (
    <ImageListTable
      apiListImage={apiAdminListImage}
      apiDeleteImageList={apiAdminDeleteImageList}
      apiChangeImagePublicStatus={apiAdminChangeImagePublicStatus}
      apiChangeImageDescription={apiAdminChangeImageDescription}
      apiChangeImageTaskType={apiAdminChangeImageTaskType}
      isAdminMode={true}
    />
  );
};
