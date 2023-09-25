import { Sidebar, playlists } from "@/components/Sidebar";
import { Outlet } from "react-router-dom";

export default function Layout() {
  return (
    <div className="grid overflow-hidden lg:grid-cols-6">
      <Sidebar playlists={playlists} className="hidden lg:block" />
      <div className="col-span-4 h-screen overflow-auto lg:col-span-5">
        <Outlet />
      </div>
    </div>
  );
}
