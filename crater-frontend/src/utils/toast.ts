import { toast } from "@/components/ui/use-toast";
import { IErrorResponse } from "@/services/types";
import { isAxiosError } from "axios";
import { logger } from "./loglevel";

export const showErrorToast = (title: string, error: unknown) => {
  if (isAxiosError(error)) {
    if (error.response?.data) {
      const responseData = error.response.data as IErrorResponse;
      if ("error" in responseData) {
        toast({
          title: title,
          description: `${responseData.error}`,
          variant: "destructive",
        });
      }
    } else {
      toast({
        title: title,
        description: `${error.message}`,
        variant: "destructive",
      });
    }
  } else {
    logger.error(error);
    toast({
      title: title,
      description: `Received unknown error, please contact admin.`,
      variant: "destructive",
    });
  }
};
