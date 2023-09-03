import type { FC } from "react";
import { useEffect } from "react";
import log from "@/utils/loglevel";
import { Sidebar, playlists } from "@/components/sidebar";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ScrollArea } from "@radix-ui/react-scroll-area";

const Home: FC = () => {
  useEffect(() => {
    log.debug("test");
  }, []);

  return (
    <div className="grid overflow-hidden lg:grid-cols-7">
      <Sidebar playlists={playlists} className="hidden bg-slate-100 lg:block" />
      <div className="col-span-5 h-screen space-y-4 overflow-auto bg-slate-100 lg:col-span-6">
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
