import { type FC } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("cluster", "node");
  return (
    <div className="space-y-4">
      <div className="grid h-48 grid-cols-5 space-x-4">
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>Cluster Node Status</CardTitle>
            <CardDescription>Card Description</CardDescription>
          </CardHeader>
          <CardContent>
            {/* <p>{isLoading ? "isLoading" : "loaded"}</p> */}
          </CardContent>
        </Card>
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>Component Status</CardTitle>
            <CardDescription>Card Description</CardDescription>
          </CardHeader>
          <CardContent>
            <p>Card Content</p>
          </CardContent>
        </Card>
      </div>
      <Card className="h-72">
        <CardHeader>
          <CardTitle>Cluster Resource Usage</CardTitle>
          <CardDescription>Card Description</CardDescription>
        </CardHeader>
        <CardContent>
          <p>Card Content</p>
        </CardContent>
      </Card>
    </div>
  );
};
