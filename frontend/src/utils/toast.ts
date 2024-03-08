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
      } catch (e) {
        toast.error(`${error.message}`);
      }
    } else {
      toast.error(`${error.message}`);
    }
  } else {
    toast.error((error as Error).message);
  }
};
