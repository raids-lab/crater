import { DetailPage } from "@/components/layout/DetailPage";
import RunningJobs from "@/components/custom/UserDetail/RunningJobs";
import LoginHeatmap from "@/components/custom/UserDetail/LoginHeatmap";
import SharedItems from "@/components/custom/UserDetail/SharedItems";
import RecentActivity from "@/components/custom/UserDetail/RecentActivity";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import TipBadge from "@/components/badge/TipBadge";
import { Activity, Calendar, Database, List, User, Users } from "lucide-react";

export default function UserDetail() {
  // User header content
  const header = (
    <div className="flex items-center space-x-4">
      <Avatar className="h-20 w-20">
        <AvatarImage src="/avatar.jpg" alt="User's avatar" />
        <AvatarFallback>JD</AvatarFallback>
      </Avatar>
      <div>
        <h1 className="flex items-center gap-2 text-3xl font-bold">
          John Doe
          <TipBadge />
        </h1>
        <p className="text-muted-foreground">@johndoe</p>
      </div>
    </div>
  );

  // User basic information
  const info = [
    {
      icon: User,
      title: "导师",
      value: "Prof. Jane Smith",
    },
    {
      icon: Users,
      title: "课题组",
      value: "AI Research Lab",
    },
    {
      icon: Calendar,
      title: "加入日期",
      value: "2023-01-15",
    },
  ];

  // Tab configuration
  const tabs = [
    {
      key: "activity",
      icon: Activity,
      label: "用户活跃度",
      children: <LoginHeatmap />,
      scrollable: true,
    },
    {
      key: "jobs",
      icon: List,
      label: "运行中的作业",
      children: <RunningJobs />,
      scrollable: true,
    },
    {
      key: "shared",
      icon: Database,
      label: "分享的资源",
      children: <SharedItems />,
      scrollable: true,
    },
    {
      key: "recent",
      icon: Calendar,
      label: "近期活动",
      children: <RecentActivity />,
      scrollable: true,
    },
  ];

  return <DetailPage header={header} info={info} tabs={tabs} />;
}
