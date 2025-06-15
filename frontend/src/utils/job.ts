import { JobType } from "@/services/api/vcjob";

/**
 * Get the URL for creating a new job based on an existing job type
 * @param jobType The type of job
 * @returns The URL path for creating a new job
 */
export const getNewJobUrl = (jobType: JobType): string => {
  switch (jobType) {
    case JobType.Jupyter:
      return "/portal/job/inter/new-jupyter-vcjobs";
    case JobType.Custom:
      return "/portal/job/batch/new-vcjobs";
    case JobType.Tensorflow:
      return "/portal/job/batch/new-tensorflow";
    case JobType.Pytorch:
      return "/portal/job/batch/new-pytorch";
    default:
      return "/portal/job/batch/new-vcjobs";
  }
};
