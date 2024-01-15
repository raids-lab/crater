import { type FC } from "react";
import Tasks from "./Tasks";
import Status from "./Status";
import "./index.css";
import Quota from "./Quota";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export const Component: FC = () => {
  return (
    <div className="grid grid-flow-row-dense gap-6 md:grid-cols-2">
      <Status />
      <Quota />
      <Card className="col-span-full">
        <CardHeader>
          <CardTitle>资源使用情况 [WIP]</CardTitle>
          <CardDescription>
            <span>1. 当前整体集群的资源使用情况</span>
            <br />
            <span>
              2. 查看整体集群的历史资源使用情况 (详细信息对接prometheus)
            </span>
            <br />
            <span>3. 查看本用户的历史资源使用情况 (汇总统计)</span>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tasks />
        </CardContent>
      </Card>
    </div>
  );
};
