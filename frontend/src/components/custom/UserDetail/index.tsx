import { DetailPage } from "@/components/layout/DetailPage";
import RunningJobs from "@/components/custom/UserDetail/RunningJobs";
import LoginHeatmap from "@/components/custom/UserDetail/LoginHeatmap";
import SharedItems from "@/components/custom/UserDetail/SharedItems";
import RecentActivity from "@/components/custom/UserDetail/RecentActivity";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import TipBadge from "@/components/badge/TipBadge";
import { Activity, Calendar, Database, List, User, Users } from "lucide-react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiGetUser } from "@/services/api/user";
import { Skeleton } from "@/components/ui/skeleton";
import { Role } from "@/services/api/auth";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useEffect } from "react";
import { TimeDistance } from "../TimeDistance";

export default function UserDetail() {
  const setBreadcrumb = useBreadcrumb();

  useEffect(() => {
    setBreadcrumb([{ title: "用户详情" }]);
  }, [setBreadcrumb]);

  // Get username from URL parameters
  const { name } = useParams<{ name: string }>();

  // Fetch user data
  const {
    data: user,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["user", name],
    queryFn: () => apiGetUser(name || ""),
    select: (data) => data.data.data,
    enabled: !!name,
  });

  // Loading state
  if (isLoading) {
    return (
      <DetailPage
        header={
          <div className="flex items-center space-x-4">
            <Skeleton className="h-20 w-20 rounded-full" />
            <div>
              <Skeleton className="mb-2 h-8 w-40" />
              <Skeleton className="h-4 w-20" />
            </div>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    );
  }

  // Error state
  if (error || !user) {
    return (
      <DetailPage
        header={
          <div>
            <h1 className="text-2xl font-bold text-red-500">
              Error loading user data
            </h1>
            <p className="text-muted-foreground">
              {(error as Error)?.message || "User not found"}
            </p>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    );
  }

  // User header content with actual data
  const header = (
    <div className="flex items-center space-x-4">
      <Avatar className="h-20 w-20">
        <AvatarImage
          src={user.Attributes.avatar || "/avatar.jpg"}
          alt={`${user.Nickname}'s avatar`}
        />
        <AvatarFallback>
          {user.Nickname?.substring(0, 2) || user.Name?.substring(0, 2) || "??"}
        </AvatarFallback>
      </Avatar>
      <div>
        <h1 className="flex items-center gap-2 text-3xl font-bold">
          {user.Nickname || user.Name}
          {user.role === Role.Admin && <TipBadge />}
        </h1>
        <p className="text-muted-foreground">@{user.Name}</p>
      </div>
    </div>
  );

  // User basic information
  const info = [
    {
      icon: User,
      title: "导师",
      value: user.Attributes.teacher || "未设置",
    },
    {
      icon: Users,
      title: "课题组",
      value: user.Attributes.group || "未设置",
    },
    {
      icon: Calendar,
      title: "加入日期",
      value: <TimeDistance date={user.CreatedAt} />,
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
