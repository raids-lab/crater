import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { IResponse } from "@/services/types";
import { useQuery } from "@tanstack/react-query";
import { AxiosResponse } from "axios";

interface ConfigContentProps<T> {
  name: string;
  type: string;
  fetchData: (name: string) => Promise<AxiosResponse<IResponse<T>, unknown>>;
  renderData: (data: T) => React.ReactNode;
}

type ConfigDialogProps<T> = React.HTMLAttributes<HTMLDivElement> &
  ConfigContentProps<T> & {
    trigger: React.ReactNode;
  };

function CodeContent<T>({
  name,
  type,
  fetchData,
  renderData,
}: ConfigContentProps<T>) {
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

export function CodeDialog<T>({
  trigger,
  name,
  type,
  fetchData,
  renderData,
}: ConfigDialogProps<T>) {
  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className="h-[calc(100vh_-200px)] w-[calc(100vw_-200px)] max-w-full gap-5"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle className="font-semibold">
            <span className="font-mono">{name}</span>
          </DialogTitle>
        </DialogHeader>
        <div className="relative h-[calc(100vh_-292px)] w-[calc(100vw_-252px)]">
          <CodeContent
            name={name}
            type={type}
            fetchData={fetchData}
            renderData={renderData}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
