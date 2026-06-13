#!/usr/bin/env python3
import hashlib
import json
import math
import os
import random
import statistics
from pathlib import Path


TOTAL_STEPS = 8
SOURCE_STEPS = 5
LEARNING_RATE = 0.04
MOMENTUM = 0.82
DATASET_VERSION = "synthetic-linear-v1"


def dataset():
    rows = []
    for idx in range(1, 17):
        x1 = idx / 10.0
        x2 = ((idx * idx) % 11) / 10.0
        y = 1.7 * x1 - 0.8 * x2 + 0.35
        rows.append((x1, x2, y))
    return rows


def sha256_json(value):
    raw = json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(raw).hexdigest()


def checkpoint_root():
    return Path(os.environ["CRATER_CHECKPOINT_DIR"])


def checkpoint_path(step):
    return checkpoint_root() / f"checkpoint-{step:04d}"


def initial_state():
    rng = random.Random(20260530)
    return {
        "step": 0,
        "epoch": 0,
        "cursor": 0,
        "weights": [round(rng.uniform(-0.2, 0.2), 8), round(rng.uniform(-0.2, 0.2), 8)],
        "bias": round(rng.uniform(-0.05, 0.05), 8),
        "velocity": [0.0, 0.0],
        "biasVelocity": 0.0,
        "lossHistory": [],
        "datasetVersion": DATASET_VERSION,
        "rngSeed": 20260530,
    }


def train_one_step(state, rows):
    idx = state["cursor"] % len(rows)
    x1, x2, y = rows[idx]
    pred = state["weights"][0] * x1 + state["weights"][1] * x2 + state["bias"]
    err = pred - y
    grad_w = [2 * err * x1, 2 * err * x2]
    grad_b = 2 * err

    for i in range(2):
        state["velocity"][i] = MOMENTUM * state["velocity"][i] + grad_w[i]
        state["weights"][i] -= LEARNING_RATE * state["velocity"][i]
    state["biasVelocity"] = MOMENTUM * state["biasVelocity"] + grad_b
    state["bias"] -= LEARNING_RATE * state["biasVelocity"]

    state["step"] += 1
    state["cursor"] = (state["cursor"] + 1) % len(rows)
    state["epoch"] = state["step"] // len(rows)
    state["lossHistory"].append(round(err * err, 10))


def save_checkpoint(state, rows):
    target = checkpoint_path(state["step"])
    target.mkdir(parents=True, exist_ok=True)
    state_payload = dict(state)
    state_payload["stateChecksum"] = sha256_json({
        "step": state["step"],
        "cursor": state["cursor"],
        "weights": state["weights"],
        "bias": state["bias"],
        "velocity": state["velocity"],
        "biasVelocity": state["biasVelocity"],
        "lossHistory": state["lossHistory"],
        "datasetVersion": state["datasetVersion"],
    })
    (target / "trainer_state.json").write_text(json.dumps(state_payload, indent=2, sort_keys=True))
    manifest = {
        "framework": os.getenv("CRATER_CHECKPOINT_FRAMEWORK", ""),
        "job": os.getenv("CRATER_JOB_NAME", ""),
        "checkpointDir": str(checkpoint_root()),
        "path": str(target),
        "step": state["step"],
        "datasetVersion": DATASET_VERSION,
        "datasetChecksum": sha256_json(rows),
        "stateChecksum": state_payload["stateChecksum"],
        "lossMean": round(statistics.mean(state["lossHistory"]), 10),
        "lossLast": state["lossHistory"][-1],
    }
    (target / "manifest.json").write_text(json.dumps(manifest, indent=2, sort_keys=True))
    return target, manifest


def load_checkpoint(path):
    ckpt = Path(path)
    state_path = ckpt / "trainer_state.json"
    manifest_path = ckpt / "manifest.json"
    if not state_path.exists():
        raise FileNotFoundError(f"missing trainer state: {state_path}")
    if not manifest_path.exists():
        raise FileNotFoundError(f"missing manifest: {manifest_path}")
    state = json.loads(state_path.read_text())
    manifest = json.loads(manifest_path.read_text())
    expected = sha256_json({
        "step": state["step"],
        "cursor": state["cursor"],
        "weights": state["weights"],
        "bias": state["bias"],
        "velocity": state["velocity"],
        "biasVelocity": state["biasVelocity"],
        "lossHistory": state["lossHistory"],
        "datasetVersion": state["datasetVersion"],
    })
    if expected != manifest["stateChecksum"] or expected != state["stateChecksum"]:
        raise ValueError("checkpoint checksum mismatch")
    if state["datasetVersion"] != DATASET_VERSION:
        raise ValueError(f"dataset version mismatch: {state['datasetVersion']}")
    state.pop("stateChecksum", None)
    return state, manifest


def build_continuity_proof(before, after, source_manifest, final_manifest, resume_from):
    return {
        "restored": True,
        "resumeFrom": resume_from,
        "loadedStep": before["step"],
        "continuedToStep": after["step"],
        "cursorBefore": before["cursor"],
        "cursorAfter": after["cursor"],
        "weightsChangedAfterResume": any(
            not math.isclose(a, b, rel_tol=0, abs_tol=1e-12)
            for a, b in zip(before["weights"], after["weights"])
        ) or not math.isclose(before["bias"], after["bias"], rel_tol=0, abs_tol=1e-12),
        "sourceStateChecksum": source_manifest["stateChecksum"],
        "finalStateChecksum": final_manifest["stateChecksum"],
        "datasetChecksum": final_manifest["datasetChecksum"],
        "lossBefore": before["lossHistory"][-1] if before["lossHistory"] else None,
        "lossAfter": after["lossHistory"][-1],
        "job": os.getenv("CRATER_JOB_NAME", ""),
    }


def main():
    root = checkpoint_root()
    root.mkdir(parents=True, exist_ok=True)
    rows = dataset()
    resume_from = os.getenv("CRATER_RESUME_FROM", "").strip()

    if resume_from:
        state, source_manifest = load_checkpoint(resume_from)
        before = json.loads(json.dumps(state))
        if state["step"] >= TOTAL_STEPS:
            raise RuntimeError(f"checkpoint already reached step {state['step']}, expected < {TOTAL_STEPS}")
        while state["step"] < TOTAL_STEPS:
            train_one_step(state, rows)
        final_path, final_manifest = save_checkpoint(state, rows)
        proof = build_continuity_proof(before, state, source_manifest, final_manifest, resume_from)
        (root / "restore-proof-realistic.json").write_text(json.dumps(proof, indent=2, sort_keys=True))
        print(
            "REALISTIC_RESTORE_PROOF "
            f"loaded_step={proof['loadedStep']} "
            f"continued_to={proof['continuedToStep']} "
            f"weights_changed={proof['weightsChangedAfterResume']} "
            f"resume_from={resume_from} "
            f"final={final_path}"
        )
        return

    state = initial_state()
    while state["step"] < SOURCE_STEPS:
        train_one_step(state, rows)
    path, manifest = save_checkpoint(state, rows)
    print(
        "REALISTIC_SOURCE_READY "
        f"checkpoint={path} "
        f"step={state['step']} "
        f"state_checksum={manifest['stateChecksum']} "
        f"dataset_checksum={manifest['datasetChecksum']}"
    )


if __name__ == "__main__":
    main()
