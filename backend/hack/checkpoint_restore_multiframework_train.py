#!/usr/bin/env python3
import hashlib
import json
import math
import os
import random
from pathlib import Path


SOURCE_STEPS = 4
TOTAL_STEPS = 7
LEARNING_RATE = 0.035
MOMENTUM = 0.78
DATASET_VERSION = "synthetic-multiframework-v1"


def rows():
    return [
        (idx / 13.0, ((idx * 7) % 17) / 17.0, 0.9 * (idx / 13.0) - 1.1 * (((idx * 7) % 17) / 17.0) + 0.2)
        for idx in range(1, 18)
    ]


def sha256_json(value):
    raw = json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(raw).hexdigest()


def framework():
    return os.getenv("CRATER_CHECKPOINT_FRAMEWORK", "custom").strip()


def checkpoint_root():
    return Path(os.environ["CRATER_CHECKPOINT_DIR"])


def initial_state():
    rng = random.Random(260530)
    return {
        "step": 0,
        "cursor": 0,
        "weights": [round(rng.uniform(-0.3, 0.3), 8), round(rng.uniform(-0.3, 0.3), 8)],
        "bias": round(rng.uniform(-0.1, 0.1), 8),
        "velocity": [0.0, 0.0],
        "biasVelocity": 0.0,
        "lossHistory": [],
        "datasetVersion": DATASET_VERSION,
        "rngSeed": 260530,
    }


def train_one_step(state, dataset):
    x1, x2, y = dataset[state["cursor"] % len(dataset)]
    pred = state["weights"][0] * x1 + state["weights"][1] * x2 + state["bias"]
    err = pred - y
    grads = [2 * err * x1, 2 * err * x2]
    bias_grad = 2 * err
    for idx in range(2):
        state["velocity"][idx] = MOMENTUM * state["velocity"][idx] + grads[idx]
        state["weights"][idx] -= LEARNING_RATE * state["velocity"][idx]
    state["biasVelocity"] = MOMENTUM * state["biasVelocity"] + bias_grad
    state["bias"] -= LEARNING_RATE * state["biasVelocity"]
    state["step"] += 1
    state["cursor"] = (state["cursor"] + 1) % len(dataset)
    state["lossHistory"].append(round(err * err, 10))


def state_checksum(state):
    return sha256_json({
        "step": state["step"],
        "cursor": state["cursor"],
        "weights": state["weights"],
        "bias": state["bias"],
        "velocity": state["velocity"],
        "biasVelocity": state["biasVelocity"],
        "lossHistory": state["lossHistory"],
        "datasetVersion": state["datasetVersion"],
    })


def checkpoint_name(fw, step):
    if fw == "deepspeed":
        return f"global_step{step:04d}"
    if fw == "verl":
        return f"global_step_{step:04d}"
    if fw == "lightning":
        return f"epoch=0-step_{step:04d}.ckpt"
    return f"checkpoint-{step:04d}"


def write_text(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(payload)


def save_checkpoint(state, dataset):
    fw = framework()
    root = checkpoint_root()
    root.mkdir(parents=True, exist_ok=True)
    checksum = state_checksum(state)
    manifest = {
        "framework": fw,
        "job": os.getenv("CRATER_JOB_NAME", ""),
        "step": state["step"],
        "checkpointDir": str(root),
        "datasetVersion": DATASET_VERSION,
        "datasetChecksum": sha256_json(dataset),
        "stateChecksum": checksum,
        "lossLast": state["lossHistory"][-1],
    }
    state_payload = dict(state)
    state_payload["stateChecksum"] = checksum

    name = checkpoint_name(fw, state["step"])
    path = root / name
    if fw == "lightning":
        payload = {"manifest": manifest, "state": state_payload, "format": "lightning-ckpt-json"}
        path.write_text(json.dumps(payload, indent=2, sort_keys=True))
        manifest["path"] = str(path)
        return path, manifest

    path.mkdir(parents=True, exist_ok=True)
    manifest["path"] = str(path)
    write_text(path / "manifest.json", json.dumps(manifest, indent=2, sort_keys=True))
    write_text(path / "trainer_state.json", json.dumps(state_payload, indent=2, sort_keys=True))

    if fw == "pytorch":
        write_text(path / "model_state.pt", json.dumps({"weights": state["weights"], "bias": state["bias"]}))
        write_text(path / "optimizer_state.pt", json.dumps({"velocity": state["velocity"], "biasVelocity": state["biasVelocity"]}))
    elif fw == "hf-trainer":
        write_text(path / "pytorch_model.bin", json.dumps({"weights": state["weights"], "bias": state["bias"]}))
        write_text(path / "optimizer.pt", json.dumps({"velocity": state["velocity"], "biasVelocity": state["biasVelocity"]}))
        write_text(path / "scheduler.pt", json.dumps({"lr": LEARNING_RATE, "step": state["step"]}))
    elif fw == "deepspeed":
        write_text(path / "mp_rank_00_model_states.pt", json.dumps({"module": state["weights"], "bias": state["bias"]}))
        write_text(path / "zero_pp_rank_0_mp_rank_00_optim_states.pt", json.dumps({"optimizer_state": state["velocity"]}))
        write_text(root / "latest", name)
    elif fw == "verl":
        write_text(path / "actor/model.pt", json.dumps({"weights": state["weights"]}))
        write_text(path / "critic/model.pt", json.dumps({"bias": state["bias"]}))
        write_text(path / "optimizer.pt", json.dumps({"velocity": state["velocity"], "biasVelocity": state["biasVelocity"]}))
    return path, manifest


def load_checkpoint(path):
    source = Path(path)
    if source.is_file():
        payload = json.loads(source.read_text())
        manifest = payload["manifest"]
        state = payload["state"]
    else:
        manifest = json.loads((source / "manifest.json").read_text())
        state = json.loads((source / "trainer_state.json").read_text())
    expected = state_checksum(state)
    if expected != manifest["stateChecksum"] or expected != state["stateChecksum"]:
        raise ValueError(f"state checksum mismatch for {source}")
    if manifest["datasetVersion"] != DATASET_VERSION:
        raise ValueError(f"dataset version mismatch for {source}")
    state.pop("stateChecksum", None)
    return state, manifest


def proof(before, after, source_manifest, final_manifest, resume_from):
    return {
        "framework": framework(),
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
    dataset = rows()
    resume_from = os.getenv("CRATER_RESUME_FROM", "").strip()
    if resume_from:
        state, source_manifest = load_checkpoint(resume_from)
        before = json.loads(json.dumps(state))
        while state["step"] < TOTAL_STEPS:
            train_one_step(state, dataset)
        final_path, final_manifest = save_checkpoint(state, dataset)
        restore_proof = proof(before, state, source_manifest, final_manifest, resume_from)
        proof_path = checkpoint_root() / f"restore-proof-{framework()}.json"
        proof_path.write_text(json.dumps(restore_proof, indent=2, sort_keys=True))
        print(
            "MF_RESTORE_PROOF "
            f"framework={framework()} "
            f"loaded_step={restore_proof['loadedStep']} "
            f"continued_to={restore_proof['continuedToStep']} "
            f"weights_changed={restore_proof['weightsChangedAfterResume']} "
            f"final={final_path}"
        )
        return

    state = initial_state()
    while state["step"] < SOURCE_STEPS:
        train_one_step(state, dataset)
    path, manifest = save_checkpoint(state, dataset)
    print(
        "MF_SOURCE_READY "
        f"framework={framework()} "
        f"checkpoint={path} "
        f"step={state['step']} "
        f"state_checksum={manifest['stateChecksum']}"
    )


if __name__ == "__main__":
    main()
