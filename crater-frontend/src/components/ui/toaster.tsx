// https://stackoverflow.com/questions/76812372/shadcn-toasts-inspired-by-react-hot-toast-library-duration-issues
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from "@/components/ui/toast";
import { TOAST_REMOVE_DELAY, useToast } from "@/components/ui/use-toast";
import { useEffect, useState } from "react";

const TIME_INTERVAL = 20;

const TimerBar = ({ max }: { max: number }) => {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setProgress((progress) => {
        if (progress < max) {
          return progress + TIME_INTERVAL;
        }
        return progress;
      });
    }, TIME_INTERVAL);

    return () => clearInterval(interval);
  }, [max]);

  return (
    <div className="absolute bottom-0 left-0 right-0 h-1">
      <div
        className="h-full bg-foreground/20"
        style={{ width: `${(progress / max) * 100}%` }}
      />
    </div>
  );
};

export const Toaster = () => {
  const { toasts } = useToast();

  return (
    <ToastProvider duration={TOAST_REMOVE_DELAY}>
      {toasts.map(({ id, title, description, action, duration, ...props }) => {
        return (
          <Toast key={id} duration={duration} {...props} className="relative">
            <div className="flex w-full flex-col">
              <div className="mr-4 flex w-full items-center justify-start gap-2 p-1">
                <div className="grid gap-1">
                  {title && <ToastTitle>{title}</ToastTitle>}
                  {description && (
                    <ToastDescription>{description}</ToastDescription>
                  )}
                </div>
                {action}
              </div>
              <ToastClose />
              <TimerBar max={duration || TOAST_REMOVE_DELAY} />
            </div>
          </Toast>
        );
      })}
      <ToastViewport />
    </ToastProvider>
  );
};
