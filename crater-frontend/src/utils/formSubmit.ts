import { logger } from "./loglevel";

export function asyncHelper(func: Promise<void>) {
  return () => {
    func.catch((error) => {
      logger.error(error);
    });
  };
}
