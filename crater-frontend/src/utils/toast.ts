import { toast } from "sonner";
import { IErrorResponse } from "@/services/types";
import { isAxiosError } from "axios";

export const showErrorToast = (error: unknown) => {
  if (isAxiosError(error)) {
    if (error.response?.data) {
      try {
        const errorResponse = error.response.data as IErrorResponse;
        if (errorResponse.msg) {
          toast.error(`${errorResponse.msg}`);
        } else {
          toast.error(`${error.message}`);
        }
      } catch {
        toast.error(`${error.message}`);
      }
    } else {
      toast.error(`${error.message}`);
    }
  } else if (typeof error === "string") {
    toast.error(error);
  } else {
    toast.error((error as Error).message);
  }
};
