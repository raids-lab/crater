// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { DetailPage } from "@/components/layout/DetailPage";
import RunningJobs from "@/components/custom/UserDetail/RunningJobs";
import LoginHeatmap from "@/components/custom/UserDetail/LoginHeatmap";
import SharedItems from "@/components/custom/UserDetail/SharedItems";
import RecentActivity from "@/components/custom/UserDetail/RecentActivity";
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
import { UserAvatar } from "./UserAvatar";
import GPUIcon from "@/components/icon/GPUIcon";
import { GrafanaIframe } from "@/pages/Embed/Monitor";
import { configGrafanaUserAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";
import { globalHideUsername } from "@/utils/store";
import { getUserPseudonym } from "@/utils/pseudonym";

export default function UserDetail() {
  const { t } = useTranslation();
  const setBreadcrumb = useBreadcrumb();
  const hideUsername = useAtomValue(globalHideUsername);

  useEffect(() => {
    setBreadcrumb([{ title: t("userDetail.breadcrumb.title") }]);
  }, [setBreadcrumb, t]);

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
  const grafanaUser = useAtomValue(configGrafanaUserAtom);

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
              {t("userDetail.header.errorTitle")}
            </h1>
            <p className="text-muted-foreground">
              {(error as Error)?.message || t("userDetail.header.userNotFound")}
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
      <UserAvatar user={user} className="size-20" size={80} />
      <div>
        <h1 className="flex items-center gap-2 text-3xl font-bold">
          {hideUsername
            ? getUserPseudonym(user.name)
            : user.nickname || user.name}
          {user.role === Role.Admin && <TipBadge />}
        </h1>
        <p className="text-muted-foreground">
          @{hideUsername ? getUserPseudonym(user.name) : user.name}
        </p>
      </div>
    </div>
  );

  // User basic information
  const info = [
    {
      icon: User,
      title: t("userDetail.info.advisor.title"),
      value: user.teacher || t("userDetail.info.notSet"),
    },
    {
      icon: Users,
      title: t("userDetail.info.researchGroup.title"),
      value: user.group || t("userDetail.info.notSet"),
    },
    {
      icon: Calendar,
      title: t("userDetail.info.joinDate.title"),
      value: <TimeDistance date={user.createdAt} />,
    },
  ];

  // Tab configuration
  const tabs = [
    {
      key: "gpu",
      icon: GPUIcon,
      label: t("userDetail.tabs.gpuMonitoring"),
      children: (
        <GrafanaIframe
          baseSrc={`${grafanaUser.nvidia}?var-user=${user.name}`}
        />
      ),
    },
    {
      key: "activity",
      icon: Activity,
      label: t("userDetail.tabs.userActivity"),
      children: <LoginHeatmap />,
      scrollable: true,
    },
    {
      key: "jobs",
      icon: List,
      label: t("userDetail.tabs.runningJobs"),
      children: <RunningJobs />,
      scrollable: true,
    },
    {
      key: "shared",
      icon: Database,
      label: t("userDetail.tabs.sharedResources"),
      children: <SharedItems />,
      scrollable: true,
    },
    {
      key: "recent",
      icon: Calendar,
      label: t("userDetail.tabs.recentActivity"),
      children: <RecentActivity />,
      scrollable: true,
    },
  ];

  return <DetailPage header={header} info={info} tabs={tabs} />;
}
