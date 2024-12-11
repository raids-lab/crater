import UserProfile from "@/components/custom/UserDetail/UserProfile";
import RunningJobs from "@/components/custom/UserDetail/RunningJobs";
import LoginHeatmap from "@/components/custom/UserDetail/LoginHeatmap";
import SharedItems from "@/components/custom/UserDetail/SharedItems";
import RecentActivity from "@/components/custom/UserDetail/RecentActivity";

export default function UserDetail() {
  return (
    <>
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <UserProfile />
        <RunningJobs />
      </div>
      <LoginHeatmap />
      <SharedItems />
      <RecentActivity />
    </>
  );
}
