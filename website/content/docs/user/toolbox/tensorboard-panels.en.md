---
title: TensorBoard Panels
description: Create, access, and manage TensorBoard log panels
---

# TensorBoard Panels

Crater can create a TensorBoard panel from a personal directory or logs produced by existing jobs. Start from the personal panel page or a job action menu.

## Writing logs

Single-node, Jupyter, WebIDE, PyTorch DDP, and TensorFlow PS job forms can set `TENSORBOARD_LOGDIR`. After job creation, the backend resolves the default to `/home/<user>/tensorboard-runs/<unique-job-name>`. Jobs with the same display name therefore do not share a log directory. A custom directory must be absolute.

The TensorFlow PS template lets worker-0 (the chief) write summaries by default so workers do not duplicate the same event stream.

## Creating a panel

1. Open the **TensorBoard** creation page. Enter the required **Panel name** (1–40 characters; Unicode is supported). Duplicate names are allowed. The personal list and delete confirmation show the name and independent ID; names do not change log paths, access URLs, or permissions.
2. Enter a personal directory or click **Browse personal directories** to select an existing one. You can also select jobs owned by the current user. Open the source-job dropdown and type a name in its top search box to search all historical jobs. The selector initially shows the latest 50 jobs and shows up to 50 matches per query. Options show the user-defined name and unique job ID, and either field can be searched. IDs distinguish jobs with the same name, and selected names are retained when search results change.
3. Add zero to ten sources. Multiple job sources are mounted under `/tensorboard-runs/<job-name>`, where TensorBoard discovers runs recursively.
4. Submit the panel. It joins your scheduling queue and runs for at most four days after the launcher starts; inactivity can end it earlier. Each user can keep at most ten pending or active panels.

A personal directory must be inside the current user's `/home/<user>` tree. The browser lets you enter subdirectories, go up, return to the personal workspace root, and filter the current directory's children by name. Click **Use current directory** to fill in the path. If browsing fails, retry or cancel and enter a path manually. Creating a panel does not require successful browsing; you may also enter a directory that your training process will create.

A job source may reference only the user's own job and storage mounts that job is allowed to use. Use its suggested log directory or enter one manually; browsing arbitrary job PVCs is not supported. Directory browsing does not check for event files, so ensure your training process writes logs to the selected path.

## Status, access, and cleanup

By default, 16 consecutive hours without access ends the panel normally; administrators can change the idle limit. Access renews the idle timer but never extends the four-day limit. Health probes do not count; browser background refreshes do. After closing the tab, idle time normally starts from its last request. Planned expiry produces Completed both in the cluster and on the page, using the same green badge as regular jobs and explaining idle or four-day expiry; “Latest expiry” shows only the four-day ceiling. Service stops on expiry and resources are normally removed five minutes later. You can delete the panel at any time without deleting training logs. This policy applies only to new panels; recreate older ones.

Panel status is `pending`, `starting`, `ready`, `completed`, `failed`, or `expired`. It is derived from the Volcano Job phase and Pod readiness; `pending` matches the Pending phase used by regular VCJobs. It is not an independent health check of the Service, Ingress, or EndpointSlice.

When a panel is opened, Crater verifies the signed-in user, current panel existence, and ownership before issuing a short-lived access session scoped to that panel path. Knowing the URL alone does not bypass this check. Deleting a panel or automatic cleanup removes its resources and invalidates the URL.

## Troubleshooting

- Stuck at `pending`: the panel is waiting for compatible CPU and memory in your scheduling queue; ask an administrator to inspect queue and node capacity.
- Stuck at `starting`: ask an administrator to inspect image pulls and Pod events.
- `failed`: read the panel error. Only a safe English summary is returned; cluster internals are not exposed.
- The page opens without charts: verify the selected path contains event files and matches the training process output directory.
