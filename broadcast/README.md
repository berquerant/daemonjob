# broadcast

This directory provides the broadcast container image.

The broadcast container should create the worker jobs.
A single worker job is provisioned for each node.

The worker job's specification is derived from the template in `DaemonJob`, `DaemonCronJob`.
