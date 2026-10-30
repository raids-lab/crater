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

import gzip
import http.client
import http.server
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest

import launcher

PREFIX = "/ingress/alice-test"


class ActivityTests(unittest.TestCase):
    def setUp(self):
        self.now = 0
        self.activity = launcher.Activity(16 * 3600, 4 * 86400, clock=lambda: self.now)

    def test_idle_expiration_is_not_revived_by_late_access(self):
        self.now = 15 * 3600
        self.assertTrue(self.activity.touch())
        self.now += 16 * 3600
        self.assertEqual(self.activity.expiration_reason(), "idle_expired")
        self.assertFalse(self.activity.touch())

    def test_access_does_not_extend_maximum_runtime(self):
        for hour in range(1, 97):
            self.now = hour * 3600
            if hour < 96:
                self.assertTrue(self.activity.touch())
        self.assertEqual(self.activity.expiration_reason(), "runtime_expired")
        self.assertFalse(self.activity.touch())


class Upstream(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *_args):
        pass

    def do_HEAD(self):
        self.send_response(200)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_GET(self):
        if self.path.endswith("/compressed"):
            body = gzip.compress(b"tensorboard" * 20000)
            self.send_response(200)
            self.send_header("Content-Encoding", "gzip")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        elif self.path.endswith("/chunks"):
            self.send_response(200)
            self.send_header("Transfer-Encoding", "chunked")
            self.end_headers()
            for part in (b"first", b"second"):
                self.wfile.write(f"{len(part):x}\r\n".encode() + part + b"\r\n")
                self.wfile.flush()
            self.wfile.write(b"0\r\n\r\n")
        else:
            body = json.dumps({"path": self.path, "host": self.headers.get("Host"),
                               "hop": self.headers.get("X-Hop")}).encode()
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Set-Cookie", "test=1; HttpOnly")
            self.end_headers()
            self.wfile.write(body)

    def do_POST(self):
        if self.headers.get("Transfer-Encoding") == "chunked":
            chunks = []
            while True:
                size = int(self.rfile.readline().strip(), 16)
                if size == 0:
                    self.rfile.readline()
                    break
                chunks.append(self.rfile.read(size))
                self.rfile.read(2)
            body = b"".join(chunks)
        else:
            body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


class ProxyTests(unittest.TestCase):
    def setUp(self):
        self.now = 0
        self.activity = launcher.Activity(10, 100, clock=lambda: self.now)
        self.upstream = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Upstream)
        self.proxy = launcher.ProxyServer(("127.0.0.1", 0), self.upstream.server_port, PREFIX, self.activity)
        for server in (self.upstream, self.proxy):
            threading.Thread(target=server.serve_forever, daemon=True).start()
            self.addCleanup(server.server_close)
            self.addCleanup(server.shutdown)

    def request(self, path, method="GET", body=None, headers=None, chunked=False):
        connection = http.client.HTTPConnection("127.0.0.1", self.proxy.server_port, timeout=5)
        try:
            connection.request(method, path, body=body, headers=headers or {}, encode_chunked=chunked)
            response = connection.getresponse()
            return response.status, dict(response.getheaders()), response.read()
        finally:
            connection.close()

    def test_probe_and_invalid_routes_do_not_renew_activity(self):
        self.now = 5
        self.assertEqual(self.request(launcher.HEALTH_PATH)[0], 204)
        self.assertEqual(self.request("/other")[0], 404)
        self.assertEqual(self.activity.last_access, 0)
        self.now = 10
        self.assertEqual(self.request(launcher.HEALTH_PATH)[0], 503)
        self.assertEqual(self.request(PREFIX + "/")[0], 503)

    def test_user_request_preserves_path_headers_and_renews_activity(self):
        self.now = 5
        status, headers, body = self.request(PREFIX + "/data?a=%2F", headers={
            "Host": "crater.example.com", "Connection": "X-Hop", "X-Hop": "remove-me",
        })
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(body), {"path": PREFIX + "/data?a=%2F", "host": "crater.example.com", "hop": None})
        self.assertEqual(headers["Set-Cookie"], "test=1; HttpOnly")
        self.assertEqual(self.activity.last_access, 5)

    def test_binary_and_chunked_responses_are_preserved(self):
        status, headers, body = self.request(PREFIX + "/compressed")
        self.assertEqual(status, 200)
        self.assertEqual(headers["Content-Encoding"], "gzip")
        self.assertEqual(gzip.decompress(body), b"tensorboard" * 20000)
        status, headers, body = self.request(PREFIX + "/chunks")
        self.assertEqual(status, 200)
        self.assertNotIn("Transfer-Encoding", headers)
        self.assertEqual(body, b"firstsecond")

    def test_post_bodies_are_streamed(self):
        payload = b"data" * 40000
        self.assertEqual(self.request(PREFIX + "/data", "POST", payload)[2], payload)
        self.assertEqual(self.request(PREFIX + "/data", "POST", iter([b"one", b"two"]), chunked=True)[2], b"onetwo")

    def test_head_and_ambiguous_request_framing(self):
        self.now = 5
        self.assertEqual(self.request(PREFIX + "/", "HEAD")[0], 200)
        self.assertEqual(self.activity.last_access, 5)
        self.now = 6
        self.assertEqual(self.request(PREFIX + "/data", "POST", headers={
            "Content-Length": "0", "Transfer-Encoding": "chunked",
        })[0], 400)
        self.assertEqual(self.activity.last_access, 5)


def free_port():
    with socket.socket() as connection:
        connection.bind(("127.0.0.1", 0))
        return connection.getsockname()[1]


class SupervisorTests(unittest.TestCase):
    def launch(self, idle, maximum, child_exit=None):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        directory = Path(temporary.name)
        reason_file = directory / "reason.json"
        pid_file = directory / "child.pid"
        port = free_port()
        child_code = (f"import os,time;open({str(pid_file)!r},'w').write(str(os.getpid()));time.sleep(300)"
                      if child_exit is None else f"import sys;sys.exit({child_exit})")
        command = [sys.executable, "-c", child_code]
        args = {"logdir": "/logs", "path_prefix": PREFIX, "port": port, "upstream_port": free_port(),
                "idle_timeout_seconds": idle, "max_runtime_seconds": maximum, "termination_log": str(reason_file)}
        code = f"import argparse,launcher,sys;sys.exit(launcher.run(argparse.Namespace(**{args!r}), {command!r}))"
        process = subprocess.Popen([sys.executable, "-B", "-c", code], cwd=Path(__file__).parent,
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.addCleanup(self.cleanup_process, process)
        return process, reason_file, pid_file, port

    @staticmethod
    def cleanup_process(process):
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=10)

    def test_idle_and_maximum_expiry_exit_zero_and_reap_child(self):
        for idle, maximum, reason in ((1, 10, "idle_expired"), (10, 1, "runtime_expired")):
            with self.subTest(reason=reason):
                process, reason_file, pid_file, _port = self.launch(idle, maximum)
                self.assertEqual(process.wait(timeout=10), 0)
                self.assertEqual(json.loads(reason_file.read_text())["reason"], reason)
                with self.assertRaises(ProcessLookupError):
                    os.kill(int(pid_file.read_text()), 0)

    def test_real_failure_is_not_completed(self):
        process, reason_file, _pid_file, _port = self.launch(10, 10, child_exit=7)
        self.assertEqual(process.wait(timeout=10), 7)
        self.assertEqual(json.loads(reason_file.read_text())["reason"], "process_failed")

    def test_successful_child_exit_is_completed(self):
        process, reason_file, _pid_file, _port = self.launch(10, 10, child_exit=0)
        self.assertEqual(process.wait(timeout=10), 0)
        self.assertEqual(json.loads(reason_file.read_text())["reason"], "completed")

    def test_external_termination_is_forwarded(self):
        process, reason_file, pid_file, port = self.launch(10, 10)
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            try:
                with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                    if pid_file.exists() and pid_file.read_text():
                        break
            except OSError:
                pass
            time.sleep(0.02)
        else:
            self.fail("launcher did not start")
        process.send_signal(signal.SIGTERM)
        self.assertEqual(process.wait(timeout=10), 128 + signal.SIGTERM)
        self.assertEqual(json.loads(reason_file.read_text())["reason"], "external_stop")
        with self.assertRaises(ProcessLookupError):
            os.kill(int(pid_file.read_text()), 0)


if __name__ == "__main__":
    unittest.main()
