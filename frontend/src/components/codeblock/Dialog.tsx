import { IResponse } from "@/services/types";
import { useQuery } from "@tanstack/react-query";
import { AxiosResponse } from "axios";
import TipBadge from "../badge/TipBadge";
import SandwichSheet from "../sheet/SandwichSheet";

interface FetchContentProps<T> {
  name: string;
  type: string;
  fetchData: (name: string) => Promise<AxiosResponse<IResponse<T>, unknown>>;
  renderData: (data: T) => React.ReactNode;
}

type FetchDialogProps<T> = React.HTMLAttributes<HTMLDivElement> &
  FetchContentProps<T> & {
    trigger: React.ReactNode;
  };

function CodeContent<T>({
  name,
  type,
  fetchData,
  renderData,
}: FetchContentProps<T>) {
  const { data } = useQuery({
    queryKey: ["code", type, name],
    queryFn: () => fetchData(name),
    select: (res) => res.data.data,
    enabled: !!name,
  });

  if (!data) {
    return <></>;
  }

  return <>{renderData(data)}</>;
}

export function FetchSheet<T>({
  trigger,
  name,
  type,
  fetchData,
  renderData,
}: FetchDialogProps<T>) {
  return (
    <SandwichSheet
      title={
        <div className="flex items-center gap-1.5">
          {name}
          <TipBadge title={type} />
        </div>
      }
      trigger={trigger}
      className="sm:max-w-3xl"
    >
      <CodeContent
        name={name}
        type={type}
        fetchData={fetchData}
        renderData={renderData}
      />
    </SandwichSheet>
  );
}
