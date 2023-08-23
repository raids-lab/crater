import type { FC } from "react";
import { useEffect } from "react";
import log from "@/utils/loglevel";
import { Sidebar, playlists } from "@/components/sidebar";

const Home: FC = () => {
  useEffect(() => {
    log.debug("test");
  }, []);

  return (
    <div className="grid lg:grid-cols-5">
      <Sidebar playlists={playlists} className="hidden lg:block" />
      <div className="col-span-3 bg-slate-100 lg:col-span-4 lg:border-l"></div>
    </div>
  );
};

export default Home;
