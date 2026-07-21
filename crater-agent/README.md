# Crater Agent

Crater Agent is the Python service used by the Crater AI assistant. It receives chat requests from the Go backend, calls platform tools through internal backend APIs, and streams agent responses back to the browser.

The service is packaged as a container image and deployed by the Crater Helm chart when `agent.enabled` is true.
