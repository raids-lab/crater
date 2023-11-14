import { toast } from "@/components/ui/use-toast";
import { IErrorResponse } from "@/services/types";
import { isAxiosError } from "axios";

export const showErrorToast = (title: string, error: unknown) => {
  if (isAxiosError(error)) {
    if (error.response?.data) {
      try {
        const errorResponse = error.response.data as IErrorResponse;
        if (errorResponse.error) {
          toast({
            title: title,
            description: `${errorResponse.error}`,
            variant: "destructive",
          });
        } else {
          toast({
            title: title,
            description: `${error.message}`,
            variant: "destructive",
          });
        }
      } catch (e) {
        toast({
          title: title,
          description: `${error.message}`,
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
    toast({
      title: title,
      description: `${(error as Error).message}`,
      variant: "destructive",
    });
  }
};
