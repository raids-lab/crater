import { QueryFunction } from "@tanstack/react-query";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { logger } from "@/utils/loglevel";

interface LabelInfo {
  ID: number;
  Name: string;
  Priority: number;
  Type: string;
}

export const apiNodeLabelsList: QueryFunction<
  IResponse<LabelInfo[]>
> = async () => {
  const response = await instance.get<IResponse<LabelInfo[]>>(
    `${VERSION}/labels`,
  );
  return response.data;
};

export const apiNodeLabelsCreate = (
  name: string,
  priority: number,
  type: string,
) => {
  const numericType = parseInt(type, 10); // 将type字段从字符串转换为数字
  return instance.post<IResponse<LabelInfo>>(`${VERSION}/admin/labels`, {
    name,
    priority,
    type: numericType, // 传递转换后的数字类型
  });
};

export const apiNodeLabelsUpdate = (
  id: number,
  name: string,
  priority: number,
  type: string,
) => {
  if (typeof id !== "number" || isNaN(id)) {
    logger.error("Invalid id value");
  }
  const numericType = parseInt(type, 10);
  return instance.put<IResponse<LabelInfo>>(`${VERSION}/admin/labels/${id}`, {
    name,
    priority,
    type: numericType,
  });
};

export const apiNodeLabelsDelete = (id: number) => {
  return instance.delete<IResponse<LabelInfo>>(`${VERSION}/admin/labels/${id}`);
};
