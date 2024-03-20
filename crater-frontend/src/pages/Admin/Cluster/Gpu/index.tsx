import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { FC } from "react";

export const Component: FC = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>加速资源管理界面</CardTitle>
      </CardHeader>
      <CardContent>
        <p>1. 与加速资源相关的标签（架构、Model、性能优先级）增删查改</p>
      </CardContent>
    </Card>
  );
};
