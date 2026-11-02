# Copyright 2026 The Crater Project Team, RAIDS-Lab
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Supervise TensorBoard and track activity without external dependencies."""

import argparse
import http.client
import http.server
import json
import logging
import os
import signal
import subprocess
import sys
import threading
import time
import urllib.parse

HEALTH_PATH = "/_crater/tensorboard/health"
BUFFER_SIZE = 64 * 1024
HOP_HEADERS = {
    "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
    "te", "trailer", "transfer-encoding", "upgrade",
}


class Activity:
    def __init__(self, idle_timeout, max_runtime, clock=time.monotonic):
        self.clock = clock
        self.started = self.last_access = clock()
        self.idle_timeout = idle_timeout
        self.max_runtime = max_runtime
        self.reason = None
        self.lock = threading.Lock()

    def _check(self, now):
        if self.reason is None:
            if now - self.started >= self.max_runtime:
                self.reason = "runtime_expired"
            elif now - self.last_access >= self.idle_timeout:
                self.reason = "idle_expired"
        return self.reason

    def touch(self):
        with self.lock:
            now = self.clock()
            if self._check(now) is not None:
                return False
            self.last_access = now
            return True

    def expiration_reason(self):
        with self.lock:
            return self._check(self.clock())


class ProxyServer(http.server.ThreadingHTTPServer):
    daemon_threads = True
    request_queue_size = 32

    def __init__(self, address, upstream_port, prefix, activity):
        self.upstream_port = upstream_port
        self.prefix = prefix.rstrip("/")
        self.activity = activity
        self.workers = threading.BoundedSemaphore(32)
        super().__init__(address, ProxyHandler)

    def process_request(self, request, client_address):
        if not self.workers.acquire(blocking=False):
            try:
                request.settimeout(1)
                request.sendall(b"HTTP/1.1 503 Service Unavailable\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
            except OSError:
                pass
            finally:
                self.shutdown_request(request)
            return
        try:
            super().process_request(request, client_address)
        except Exception:
            self.workers.release()
            raise

    def process_request_thread(self, request, client_address):
        try:
            super().process_request_thread(request, client_address)
        finally:
            self.workers.release()


def excluded_headers(headers):
    nominated = {item.strip().lower() for item in headers.get("Connection", "").split(",")}
    return HOP_HEADERS | nominated


class ProxyHandler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    timeout = 60

    def log_message(self, _format, *_args):
        # Do not log request URLs, query strings, cookies, or authorization headers.
        pass

    def _reply(self, status):
        self.send_response(status)
        self.send_header("Content-Length", "0")
        self.send_header("Connection", "close")
        self.end_headers()
        self.close_connection = True

    def _health(self):
        if self.server.activity.expiration_reason() is not None:
            self._reply(503)
            return
        connection = http.client.HTTPConnection("127.0.0.1", self.server.upstream_port, timeout=2)
        try:
            connection.request("HEAD", self.server.prefix + "/")
            response = connection.getresponse()
            self._reply(204 if 200 <= response.status < 400 else 503)
        except (OSError, http.client.HTTPException):
            self._reply(503)
        finally:
            connection.close()

    def _body(self, length, chunked):
        remaining = length
        while chunked or remaining > 0:
            if chunked:
                line = self.rfile.readline(8193)
                if len(line) > 8192 or not line.endswith(b"\r\n"):
                    raise ValueError("invalid chunk header")
                size = int(line.split(b";", 1)[0].strip(), 16)
                if size < 0:
                    raise ValueError("invalid chunk size")
                if size == 0:
                    while True:
                        trailer = self.rfile.readline(8193)
                        if len(trailer) > 8192 or not trailer.endswith(b"\r\n"):
                            raise ValueError("invalid chunk trailer")
                        if trailer == b"\r\n":
                            return
                remaining = size
            while remaining > 0:
                data = self.rfile.read(min(BUFFER_SIZE, remaining))
                if not data:
                    raise ValueError("incomplete request body")
                remaining -= len(data)
                yield data
            if chunked and self.rfile.read(2) != b"\r\n":
                raise ValueError("invalid chunk boundary")

    def _proxy(self):
        path = urllib.parse.urlsplit(self.path).path
        if self.command in ("GET", "HEAD") and path == HEALTH_PATH:
            self._health()
            return
        prefix = self.server.prefix
        if not self.path.startswith("/") or not (path == prefix or path.startswith(prefix + "/")):
            self._reply(404)
            return
        transfer_encoding = self.headers.get("Transfer-Encoding", "").lower()
        lengths = self.headers.get_all("Content-Length", [])
        if len(lengths) > 1 or (transfer_encoding and lengths) or transfer_encoding not in ("", "chunked"):
            self._reply(400)
            return
        try:
            length = int(lengths[0]) if lengths else 0
            if length < 0:
                raise ValueError("invalid content length")
        except ValueError:
            self._reply(400)
            return
        if not self.server.activity.touch():
            self._reply(503)
            return

        connection = http.client.HTTPConnection("127.0.0.1", self.server.upstream_port, timeout=60)
        headers_sent = False
        try:
            excluded = excluded_headers(self.headers)
            headers = {key: value for key, value in self.headers.items() if key.lower() not in excluded}
            headers["Connection"] = "close"
            chunked = transfer_encoding == "chunked"
            body = self._body(length, chunked) if length or chunked else None
            connection.request(self.command, self.path, body=body, headers=headers, encode_chunked=chunked)
            response = connection.getresponse()
            self.send_response(response.status)
            excluded = excluded_headers(response.headers) | {"server", "date"}
            if response.chunked:
                excluded.add("content-length")
            for key, value in response.getheaders():
                if key.lower() not in excluded:
                    self.send_header(key, value)
            self.send_header("Connection", "close")
            self.end_headers()
            headers_sent = True
            if self.command != "HEAD":
                while True:
                    data = response.read1(BUFFER_SIZE)
                    if not data:
                        break
                    self.wfile.write(data)
                    self.wfile.flush()
        except ValueError:
            if not headers_sent:
                self._reply(400)
        except (OSError, http.client.HTTPException):
            if not headers_sent:
                self._reply(502)
        finally:
            self.close_connection = True
            connection.close()

    do_GET = do_HEAD = do_POST = do_PUT = do_PATCH = do_DELETE = do_OPTIONS = _proxy


def stop_process(process, grace=30):
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        process.wait(timeout=grace)
    except subprocess.TimeoutExpired:
        logging.warning("TensorBoard did not stop within the grace period; killing its process group")
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait(timeout=10)


def write_reason(path, reason):
    try:
        with open(path, "w", encoding="utf-8") as output:
            json.dump({"reason": reason}, output)
    except OSError:
        logging.warning("Could not write the container termination reason")


def run(args, command=None):
    stop_requested = threading.Event()
    received_signal = [None]

    def handle_signal(signum, _frame):
        received_signal[0] = signum
        stop_requested.set()

    signal.signal(signal.SIGTERM, handle_signal)
    signal.signal(signal.SIGINT, handle_signal)
    child_command = command or [
        sys.executable, "-m", "tensorboard.main", "--logdir", args.logdir,
        "--host", "127.0.0.1", "--port", str(args.upstream_port),
        "--path_prefix", args.path_prefix,
    ]
    process = subprocess.Popen(child_command, start_new_session=True)
    activity = Activity(args.idle_timeout_seconds, args.max_runtime_seconds)
    server = None
    thread = None
    try:
        server = ProxyServer(("0.0.0.0", args.port), args.upstream_port, args.path_prefix, activity)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        while True:
            code = process.poll()
            if code is not None:
                write_reason(args.termination_log, "completed" if code == 0 else "process_failed")
                return code if code >= 0 else 128 - code
            if stop_requested.is_set():
                write_reason(args.termination_log, "external_stop")
                return 128 + received_signal[0]
            reason = activity.expiration_reason()
            if reason is not None:
                logging.info("Ending TensorBoard: %s", reason)
                stop_process(process)
                write_reason(args.termination_log, reason)
                return 0
            stop_requested.wait(min(5, args.idle_timeout_seconds, args.max_runtime_seconds))
    finally:
        stop_process(process)
        if server is not None:
            if thread is not None:
                server.shutdown()
            server.server_close()


def positive_integer(value):
    number = int(value)
    if number <= 0:
        raise argparse.ArgumentTypeError("must be a positive integer")
    return number


def parse_args():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--logdir", required=True)
    parser.add_argument("--path-prefix", required=True)
    parser.add_argument("--port", type=positive_integer, default=6006)
    parser.add_argument("--upstream-port", type=positive_integer, default=6007)
    parser.add_argument("--idle-timeout-seconds", type=positive_integer, default=16 * 60 * 60)
    parser.add_argument("--max-runtime-seconds", type=positive_integer, default=4 * 24 * 60 * 60)
    parser.add_argument("--termination-log", default="/dev/termination-log")
    return parser.parse_args()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    try:
        sys.exit(run(parse_args()))
    except Exception:
        logging.exception("TensorBoard launcher failed")
        sys.exit(1)
