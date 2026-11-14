# Embedded in the generated Pod command. All user options are argv, never shell.
import json
import os
import signal
import subprocess
import sys
import time

spec = json.loads(sys.argv[1])
children = []


def stop(signum, _frame):
    for child in children:
        if child.poll() is None:
            child.send_signal(signum)
    raise SystemExit(128 + signum)


signal.signal(signal.SIGTERM, stop)
signal.signal(signal.SIGINT, stop)


def run(argv):
    child = subprocess.Popen(argv)
    children.append(child)
    return child


def main():
    ip = os.environ['POD_IP']
    os.environ['PYTHONHASHSEED'] = '1047'
    os.environ['VLLM_HOST_IP'] = ip
    if spec['nixl']:
        os.environ['VLLM_NIXL_SIDE_CHANNEL_HOST'] = ip
        os.environ['VLLM_NIXL_SIDE_CHANNEL_PORT'] = '5558'
        os.environ['VLLM_WORKER_MULTIPROC_METHOD'] = 'spawn'
    if spec['execution'] == 'ray':
        args = ['ray', 'start', '--node-ip-address=' + ip,
                '--num-gpus=' + str(spec['gpus']), '--disable-usage-stats']
        if not spec['entry']:
            # Kthena injects the entry's stable DNS name; Ray may start later.
            address = os.environ['ENTRY_ADDRESS'] + ':6379'
            deadline = time.monotonic() + 600
            while time.monotonic() < deadline:
                child = run(args + ['--address=' + address, '--block'])
                code = child.wait()
                if code == 0:
                    return 0
                time.sleep(5)
            raise RuntimeError('Ray worker could not join entry within 600s')
        subprocess.run(args + ['--head', '--port=6379'], check=True)
        import ray
        ray.init(address='127.0.0.1:6379')
        deadline = time.monotonic() + 600
        while True:
            nodes = [n for n in ray.nodes() if n['Alive']]
            if len(nodes) >= spec['nodes'] and sum(n['Resources'].get('GPU', 0) for n in nodes) >= spec['nodes'] * spec['gpus']:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError('Ray cluster did not acquire all required nodes/GPUs within 600s')
            time.sleep(2)
        ray.shutdown()
    argv = spec['argv']
    # SGLang's transfer protocol advertises its actual reachable Pod address.
    argv = [ip if arg == '__POD_IP__' else arg for arg in argv]
    return run(argv).wait()


try:
    sys.exit(main())
finally:
    for child in children:
        if child.poll() is None:
            child.terminate()
    if spec['execution'] == 'ray':
        subprocess.run(['ray', 'stop', '--force'], timeout=15, check=False)
