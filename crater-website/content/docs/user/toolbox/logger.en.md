---
title: Logger
description: Documentation for Logger
---

# Logging and Diagnostics

For logging and diagnostics, we provide some tools to help you better understand and debug your jobs. These tools are available on the [Job Details Page](../quick-start/interactive#Job Details Page).

## Job-related Events

On the Job Details Page, click the Job Events button to view events from the past hour.

![events](./image/events.webp)

Since these events are retrieved from the cluster, they are displayed in English by default. You can focus on the following:

1. The time when the event occurred, for example, the PodGroupPending warning in the screenshot. After that, the job was successfully scheduled and executed, so it can be ignored.
2. Normal events and Warning events: Pay more attention to Warning events.

## Job-related Logs

After the job is successfully scheduled and started, you can view the log information for the corresponding Pod of the job.

![log](./image/log.webp)

For possible issues you might encounter, please refer to [Common Issues](../category/Common Issues/).