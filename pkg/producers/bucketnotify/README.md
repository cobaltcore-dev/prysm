# RGW Bucket Notifications (local or remote producer)

## Overview

The **RGW Bucket Notifications (Prysm Local or Remote Producer)** is a
lightweight HTTP server that handles notifications from Rados Gateway (RGW)
buckets. This tool provides a simple HTTP endpoint to receive and process bucket
notifications, with the capability to forward these notifications to a NATS
server for further processing and distribution. It is designed to integrate
seamlessly into your existing infrastructure, providing a flexible and reliable
solution for handling RGW bucket events.

## Key Features

- **HTTP Endpoint**: Listens for RGW bucket notifications on a configurable HTTP
  port.
- **NATS Integration**: Optionally forwards received notifications to a NATS
  server, enabling real-time event processing and integration with other
  observability tools.
- **Simple Configuration**: Configurable via command-line flags or environment
  variables, allowing easy setup and integration into various environments.

## Usage

To start the Prysm local producer for bucket notifications, use the following
command:

```bash
prysm local-producer bucket-notify [flags]
or
prysm remote-producer bucket-notify [flags]
```

### Example Flags:

- `--nats-url "nats://localhost:4222"`: NATS server URL for publishing
  notifications.
- `--nats-subject "rgw.buckets.notify"`: NATS subject to publish results
  (default is “rgw.buckets.notify”).
- `--port 8080`: HTTP endpoint port to listen for bucket notifications (default
  is 8080).

### Environment Variables

Configuration can also be set through environment variables:

- `BUCKET_NOTIFY_ENDPOINT_PORT`: Port for the HTTP endpoint.
- `NATS_URL`: NATS server URL.
- `NATS_SUBJECT`: NATS subject to publish results.

## Logic and Workflow

The Bucket Notify local producer operates as follows:

### Start the HTTP Server:

- The server starts and listens on the specified port for incoming HTTP POST
  requests to the /notifications endpoint.
- Example: http://localhost:8080/notifications

### Handle Incoming Notifications:

- When a notification is received, the server reads the request body and
  attempts to parse it as JSON. The expected format is a valid RGWNotification
  structure.
- If the JSON is malformed or the request body cannot be read, an appropriate
  error response is returned.

### Process and Forward Notifications:

- If NATS integration is enabled, the parsed notification is forwarded to the
  configured NATS subject.
- If NATS is not enabled, the notification is logged to the console in a
  human-readable format.

### Respond to the Request:

- The server responds with a 200 OK status, indicating that the notification was
  successfully received and processed.

### Example Workflow

Start the server with the desired configuration:

```bash
prysm local-producer bucket-notify --nats-url "nats://localhost:4222" --port 8080
or
prysm remote-producer bucket-notify --nats-url "nats://localhost:4222" --port 8080
```

When an RGW bucket notification is sent to http://localhost:8080/notifications,
the server processes the notification and forwards it to the specified NATS
subject. If NATS is not configured, the notification is printed to the console.

---

> This README is a draft and will be updated as the project continues to evolve.
> Contributions and feedback are welcome to help refine and enhance the
> functionality of Prysm.
