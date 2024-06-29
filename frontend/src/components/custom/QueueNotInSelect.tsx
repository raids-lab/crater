import Select, { OnChangeValue } from "react-select";
import { useQuery } from "@tanstack/react-query";
import { apiListQueuesNotInDataset } from "@/services/api/dataset";
interface QueueOption {
  value: string;
  id: number;
  label: string;
}
interface QueueSelectProps {
  id: number;
  onChange: (newValue: OnChangeValue<QueueOption, true>) => void;
}
export function QueueNotInSelect({ id, onChange }: QueueSelectProps) {
  const { data: queueList } = useQuery({
    queryKey: ["dataset", "queueOutList", { id }],
    queryFn: () => apiListQueuesNotInDataset(id),
    select: (res) => {
      return res.data.data.map((queue) => {
        return {
          value: queue.name,
          label: queue.name,
          id: queue.id,
        };
      });
    },
  });

  return (
    <Select
      isMulti
      name="queues"
      options={queueList}
      className="basic-multi-select col-span-3"
      classNamePrefix="select"
      onChange={onChange}
    />
  );
}
