import { Sidebar, SidebarItem } from "@/components/Sidebar";
import { Outlet } from "react-router-dom";

const items: SidebarItem[] = [
  {
    title: "集群资源查看",
    path: "cluster",
    children: [{ title: "PVC", path: "pvc" }],
  },
  {
    title: "训练任务管理",
    path: "training",
    children: [
      { title: "训练任务列表", path: "list" },
      { title: "新建训练任务", path: "new" },
      { title: "任务监控", path: "monitor" },
    ],
  },
  {
    title: "资源配额",
    path: "resource",
    children: [
      { title: "训练任务列表", path: "list" },
      { title: "新建训练任务", path: "new" },
      { title: "任务监控", path: "monitor" },
    ],
  },
  {
    title: "Jupyter 管理",
    path: "jupyter",
    children: [
      { title: "训练任务列表", path: "list" },
      { title: "新建训练任务", path: "new" },
      { title: "任务监控", path: "monitor" },
    ],
  },
];

export default function Layout() {
  return (
    <div className="grid overflow-hidden lg:grid-cols-6">
      <Sidebar sidebarItems={items} className="hidden lg:block" />
      <div className="col-span-4 h-screen overflow-auto lg:col-span-5">
        <Outlet />
      </div>
    </div>
  );
}
