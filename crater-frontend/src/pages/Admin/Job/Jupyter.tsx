import { apiAdminTaskListByType } from "@/services/api/admin/task";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

const Jupyter: FC = () => {
  const { data: taskList, isLoading } = useQuery({
    queryKey: ["admin", "tasklist"],
    queryFn: () => apiAdminTaskListByType("jupyter"),
    select: (res) => res.data.data.Tasks,
  });

  return (
    <div>
      {isLoading ? (
        <div>Loading...</div>
      ) : (
        <ul>
          {taskList?.map((task) => (
            <li key={task.id}>
              {task.taskName},{task.jobName},{task.isDeleted.toString()},
              {task.userName},{task.status}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default Jupyter;
