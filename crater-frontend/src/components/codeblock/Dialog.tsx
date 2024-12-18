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
import TipBadge from "../badge/TipBadge";

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

export function FetchDialog<T>({
  trigger,
  name,
  type,
  fetchData,
  renderData,
}: FetchDialogProps<T>) {
  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="h-[calc(100vh_-200px)] w-[calc(100vw_-200px)] max-w-full gap-5">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center gap-1 font-semibold">
            <span className="font-mono">{name}</span>
            <TipBadge title={type} />
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

interface NormalDialogProps {
  trigger: React.ReactNode;
  name: string;
  type: string;
  content: React.ReactNode;
}

export function NormalDialog({
  trigger,
  name,
  type,
  content,
}: NormalDialogProps) {
  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className="h-[calc(100vh_-200px)] w-[calc(100vw_-200px)] max-w-full gap-5"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center gap-1.5 font-semibold">
            <span className="font-mono">{name}</span>
            <TipBadge title={type} />
          </DialogTitle>
        </DialogHeader>
        <div className="relative h-[calc(100vh_-292px)] w-[calc(100vw_-252px)] overflow-y-auto">
          {content}
        </div>
      </DialogContent>
    </Dialog>
  );
}
