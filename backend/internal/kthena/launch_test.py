"""Exercise the embedded launcher with deterministic Ray/process fixtures."""
import json
import runpy
import signal
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, patch

SCRIPT = Path(__file__).with_name('launch.py')


class LauncherTest(unittest.TestCase):
    def launch(self, execution='single', entry=True, nodes=None, clock=None):
        self.started = []
        self.commands = []
        self.handlers = {}
        child = Mock()
        child.wait.return_value = 0
        child.poll.return_value = 0
        spec = dict(argv=['engine', '--model', 'opaque;$(value)'],
                    execution=execution, entry=entry, nodes=2, gpus=1, nixl=True)
        ray = SimpleNamespace(init=Mock(), shutdown=Mock(), nodes=Mock(return_value=nodes or []))

        def start(argv):
            self.started.append(argv)
            return child

        self.patches = (
            patch.object(sys, 'argv', [str(SCRIPT), json.dumps(spec)]),
            patch.dict('os.environ', {'POD_IP': '10.0.0.2', 'ENTRY_ADDRESS': 'group-entry'}),
            patch.dict(sys.modules, {'ray': ray}),
            patch('subprocess.Popen', side_effect=start),
            patch('subprocess.run', side_effect=lambda argv, **kw: self.commands.append(argv)),
            patch('signal.signal', side_effect=lambda sig, fn: self.handlers.update({sig: fn})),
            patch('time.sleep'),
            patch('time.monotonic', side_effect=clock or [0, 1, 601]),
        )
        from contextlib import ExitStack
        with ExitStack() as stack:
            for item in self.patches:
                stack.enter_context(item)
            try:
                runpy.run_path(str(SCRIPT), run_name='__main__')
            except SystemExit as exc:
                self.assertEqual(exc.code, 0)
        return child

    def test_single_preserves_opaque_argv(self):
        self.launch()
        self.assertEqual(self.started, [['engine', '--model', 'opaque;$(value)']])
        self.assertEqual(self.commands, [])

    def test_ray_head_waits_for_nodes_then_starts_engine(self):
        self.launch('ray', nodes=[{'Alive': True, 'Resources': {'GPU': 1}}] * 2)
        self.assertIn('--head', self.commands[0])
        self.assertEqual(self.started[0][0], 'engine')
        self.assertEqual(self.commands[-1], ['ray', 'stop', '--force'])

    def test_ray_timeout_never_starts_engine(self):
        with self.assertRaisesRegex(RuntimeError, 'required nodes/GPUs'):
            self.launch('ray', nodes=[{'Alive': True, 'Resources': {'GPU': 1}}])
        self.assertEqual(self.started, [])
        self.assertEqual(self.commands[-1], ['ray', 'stop', '--force'])

    def test_worker_joins_its_entry_and_forwards_termination(self):
        child = self.launch('ray', entry=False)
        self.assertIn('--address=group-entry:6379', self.started[0])
        self.assertIn('--block', self.started[0])
        self.assertFalse(any(cmd[0] == 'engine' for cmd in self.started))
        child.poll.return_value = None
        with self.assertRaises(SystemExit) as result:
            self.handlers[signal.SIGTERM](signal.SIGTERM, None)
        self.assertEqual(result.exception.code, 128 + signal.SIGTERM)
        child.send_signal.assert_called_once_with(signal.SIGTERM)


if __name__ == '__main__':
    unittest.main()
