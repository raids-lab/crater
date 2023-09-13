import type { FC } from "react";
import { Sidebar, playlists } from "@/components/sidebar";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const Home: FC = () => {
  return (
    <div className="grid overflow-hidden lg:grid-cols-6">
      <Sidebar playlists={playlists} className="hidden bg-slate-100 lg:block" />
      <div className="col-span-4 h-screen space-y-4 overflow-auto bg-slate-100 lg:col-span-5">
        <div className="space-y-4 py-8 pr-6">
          <div className="grid h-48 grid-cols-5 space-x-4">
            <Card className="col-span-2">
              <CardHeader>
                <CardTitle>Cluster Node Status</CardTitle>
                <CardDescription>Card Description</CardDescription>
              </CardHeader>
              <CardContent>
                <p>Cluster Node Status</p>
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
      </div>
    </div>
  );
};

export default Home;
