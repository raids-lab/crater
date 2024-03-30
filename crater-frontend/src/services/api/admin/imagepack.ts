import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { ImagePackCreate, ImagePackInfoResponse } from "../imagepack";

export const apiAdminImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/create",
    imagepack,
  );
  return response.data;
};

export const apiAdminImagePackList = (type: number) =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(
    VERSION + "/admin/images/list",
    {
      params: {
        type,
      },
    },
  );

export const apiAdminImagePackDelete = async (id: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/delete",
    {
      id,
    },
  );
  return response.data;
};
