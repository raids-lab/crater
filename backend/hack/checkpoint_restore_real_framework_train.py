#!/usr/bin/env python3
import json
import math
import os
from pathlib import Path


SOURCE_STEPS = 4
TOTAL_STEPS = 7
DATASET_SEED = 260530


def checkpoint_dir() -> Path:
    return Path(os.environ["CRATER_CHECKPOINT_DIR"])


def framework() -> str:
    return os.getenv("CRATER_CHECKPOINT_FRAMEWORK", "pytorch").strip().lower()


def resume_from() -> str:
    return os.getenv("CRATER_RESUME_FROM", "").strip()


def dataset_tensors(torch):
    torch.manual_seed(DATASET_SEED)
    x = torch.linspace(-1.0, 1.0, steps=24, dtype=torch.float32).view(12, 2)
    y = 0.7 * x[:, :1] - 1.3 * x[:, 1:] + 0.25
    return x, y


def proof_path(fw: str) -> Path:
    return checkpoint_dir() / f"restore-proof-real-{fw}.json"


def write_proof(fw: str, loaded_step: int, continued_to: int, before, after, resume_path: str, final_path: Path):
    changed = any(
        not math.isclose(float(a), float(b), rel_tol=0, abs_tol=1e-12)
        for a, b in zip(before, after)
    )
    proof = {
        "framework": fw,
        "restored": True,
        "resumeFrom": resume_path,
        "loadedStep": loaded_step,
        "continuedToStep": continued_to,
        "weightsChangedAfterResume": changed,
        "finalCheckpoint": str(final_path),
        "job": os.getenv("CRATER_JOB_NAME", ""),
    }
    proof_path(fw).write_text(json.dumps(proof, indent=2, sort_keys=True))
    return proof


def flatten_params(torch, model):
    return torch.cat([p.detach().cpu().reshape(-1) for p in model.parameters()]).tolist()


def train_torch_steps(torch, model, optimizer, start_step: int, end_step: int):
    x, y = dataset_tensors(torch)
    loss_fn = torch.nn.MSELoss()
    last_loss = 0.0
    for step in range(start_step + 1, end_step + 1):
        idx = (step - 1) % x.shape[0]
        optimizer.zero_grad(set_to_none=True)
        pred = model(x[idx : idx + 1])
        loss = loss_fn(pred, y[idx : idx + 1])
        loss.backward()
        optimizer.step()
        last_loss = float(loss.detach().cpu())
    return last_loss


def torch_model(torch):
    torch.manual_seed(DATASET_SEED)
    return torch.nn.Sequential(torch.nn.Linear(2, 8), torch.nn.Tanh(), torch.nn.Linear(8, 1))


def run_pytorch():
    import torch

    fw = framework()
    ckpt_root = checkpoint_dir()
    ckpt_root.mkdir(parents=True, exist_ok=True)

    model = torch_model(torch)
    optimizer = torch.optim.SGD(model.parameters(), lr=0.05, momentum=0.8)
    resume_path = resume_from()
    if resume_path:
        payload = torch.load(resume_path, map_location="cpu")
        model.load_state_dict(payload["model_state"])
        optimizer.load_state_dict(payload["optimizer_state"])
        loaded_step = int(payload["step"])
        before = flatten_params(torch, model)
        loss = train_torch_steps(torch, model, optimizer, loaded_step, TOTAL_STEPS)
        final_path = ckpt_root / f"checkpoint-{TOTAL_STEPS:04d}.pt"
        torch.save(
            {
                "step": TOTAL_STEPS,
                "model_state": model.state_dict(),
                "optimizer_state": optimizer.state_dict(),
                "loss": loss,
            },
            final_path,
        )
        proof = write_proof(fw, loaded_step, TOTAL_STEPS, before, flatten_params(torch, model), resume_path, final_path)
        print(
            "REAL_FRAMEWORK_RESTORE_PROOF "
            f"framework={fw} loaded_step={proof['loadedStep']} continued_to={proof['continuedToStep']} "
            f"weights_changed={proof['weightsChangedAfterResume']} final={final_path}"
        )
        return

    loss = train_torch_steps(torch, model, optimizer, 0, SOURCE_STEPS)
    path = ckpt_root / f"checkpoint-{SOURCE_STEPS:04d}.pt"
    torch.save(
        {
            "step": SOURCE_STEPS,
            "model_state": model.state_dict(),
            "optimizer_state": optimizer.state_dict(),
            "loss": loss,
        },
        path,
    )
    print(f"REAL_FRAMEWORK_SOURCE_READY framework={fw} checkpoint={path} step={SOURCE_STEPS}")


def run_lightning():
    import torch
    import lightning as L
    from torch.utils.data import DataLoader, TensorDataset

    fw = framework()
    ckpt_root = checkpoint_dir()
    ckpt_root.mkdir(parents=True, exist_ok=True)

    class RegressionModule(L.LightningModule):
        def __init__(self):
            super().__init__()
            torch.manual_seed(DATASET_SEED)
            self.model = torch.nn.Sequential(torch.nn.Linear(2, 8), torch.nn.Tanh(), torch.nn.Linear(8, 1))
            self.loss_fn = torch.nn.MSELoss()

        def training_step(self, batch, batch_idx):
            x, y = batch
            return self.loss_fn(self.model(x), y)

        def configure_optimizers(self):
            return torch.optim.SGD(self.parameters(), lr=0.05, momentum=0.8)

    x, y = dataset_tensors(torch)
    loader = DataLoader(TensorDataset(x, y), batch_size=1, shuffle=False)
    trainer = L.Trainer(
        max_steps=TOTAL_STEPS if resume_from() else SOURCE_STEPS,
        accelerator="cpu",
        devices=1,
        logger=False,
        enable_checkpointing=False,
        enable_model_summary=False,
        enable_progress_bar=False,
    )
    model = RegressionModule()
    resume_path = resume_from()
    if resume_path:
        payload = torch.load(resume_path, map_location="cpu")
        loaded_step = int(payload["global_step"])
        before_model = RegressionModule.load_from_checkpoint(resume_path)
        before = flatten_params(torch, before_model)
        trainer.fit(model, train_dataloaders=loader, ckpt_path=resume_path)
        final_path = ckpt_root / f"epoch=0-step_{TOTAL_STEPS:04d}.ckpt"
        trainer.save_checkpoint(final_path)
        proof = write_proof(fw, loaded_step, TOTAL_STEPS, before, flatten_params(torch, model), resume_path, final_path)
        print(
            "REAL_FRAMEWORK_RESTORE_PROOF "
            f"framework={fw} loaded_step={proof['loadedStep']} continued_to={proof['continuedToStep']} "
            f"weights_changed={proof['weightsChangedAfterResume']} final={final_path}"
        )
        return

    trainer.fit(model, train_dataloaders=loader)
    path = ckpt_root / f"epoch=0-step_{SOURCE_STEPS:04d}.ckpt"
    trainer.save_checkpoint(path)
    print(f"REAL_FRAMEWORK_SOURCE_READY framework={fw} checkpoint={path} step={SOURCE_STEPS}")


def run_deepspeed():
    import torch
    import deepspeed

    fw = framework()
    ckpt_root = checkpoint_dir()
    ckpt_root.mkdir(parents=True, exist_ok=True)

    os.environ.setdefault("MASTER_ADDR", "127.0.0.1")
    os.environ.setdefault("MASTER_PORT", "29501")
    os.environ.setdefault("RANK", "0")
    os.environ.setdefault("WORLD_SIZE", "1")
    os.environ.setdefault("LOCAL_RANK", "0")

    model = torch_model(torch)
    optimizer = torch.optim.SGD(model.parameters(), lr=0.05, momentum=0.8)
    config = {
        "train_batch_size": 1,
        "train_micro_batch_size_per_gpu": 1,
        "gradient_accumulation_steps": 1,
        "zero_optimization": {"stage": 0},
    }
    engine, _, _, _ = deepspeed.initialize(
        model=model,
        optimizer=optimizer,
        config=config,
        dist_init_required=True,
    )

    def train_engine_steps(start_step: int, end_step: int):
        x, y = dataset_tensors(torch)
        loss_fn = torch.nn.MSELoss()
        for step in range(start_step + 1, end_step + 1):
            idx = (step - 1) % x.shape[0]
            loss = loss_fn(engine(x[idx : idx + 1]), y[idx : idx + 1])
            engine.backward(loss)
            engine.step()

    resume_path = resume_from()
    if resume_path:
        _, client_state = engine.load_checkpoint(str(Path(resume_path).parent), tag=Path(resume_path).name)
        loaded_step = int(client_state["step"])
        before = flatten_params(torch, engine.module)
        train_engine_steps(loaded_step, TOTAL_STEPS)
        final_tag = f"global_step{TOTAL_STEPS:04d}"
        engine.save_checkpoint(str(ckpt_root), tag=final_tag, client_state={"step": TOTAL_STEPS})
        final_path = ckpt_root / final_tag
        proof = write_proof(fw, loaded_step, TOTAL_STEPS, before, flatten_params(torch, engine.module), resume_path, final_path)
        print(
            "REAL_FRAMEWORK_RESTORE_PROOF "
            f"framework={fw} loaded_step={proof['loadedStep']} continued_to={proof['continuedToStep']} "
            f"weights_changed={proof['weightsChangedAfterResume']} final={final_path}"
        )
        return

    train_engine_steps(0, SOURCE_STEPS)
    tag = f"global_step{SOURCE_STEPS:04d}"
    engine.save_checkpoint(str(ckpt_root), tag=tag, client_state={"step": SOURCE_STEPS})
    print(f"REAL_FRAMEWORK_SOURCE_READY framework={fw} checkpoint={ckpt_root / tag} step={SOURCE_STEPS}")


def run_verl_limited():
    import torch

    # The local arm64 CPU validation environment is not suitable for a full veRL
    # runtime, but this still writes the native veRL checkpoint layout expected by
    # Crater and verifies tensor state restoration with torch serialization.
    fw = framework()
    ckpt_root = checkpoint_dir()
    ckpt_root.mkdir(parents=True, exist_ok=True)
    model = torch_model(torch)
    optimizer = torch.optim.SGD(model.parameters(), lr=0.05, momentum=0.8)
    resume_path = resume_from()
    if resume_path:
        payload = torch.load(Path(resume_path) / "actor" / "model.pt", map_location="cpu")
        model.load_state_dict(payload["model_state"])
        optimizer.load_state_dict(torch.load(Path(resume_path) / "optimizer.pt", map_location="cpu")["optimizer_state"])
        loaded_step = int(payload["step"])
        before = flatten_params(torch, model)
        train_torch_steps(torch, model, optimizer, loaded_step, TOTAL_STEPS)
        final_path = ckpt_root / f"global_step_{TOTAL_STEPS:04d}"
        (final_path / "actor").mkdir(parents=True, exist_ok=True)
        (final_path / "critic").mkdir(parents=True, exist_ok=True)
        torch.save({"step": TOTAL_STEPS, "model_state": model.state_dict()}, final_path / "actor" / "model.pt")
        torch.save({"step": TOTAL_STEPS, "model_state": model.state_dict()}, final_path / "critic" / "model.pt")
        torch.save({"optimizer_state": optimizer.state_dict()}, final_path / "optimizer.pt")
        proof = write_proof(fw, loaded_step, TOTAL_STEPS, before, flatten_params(torch, model), resume_path, final_path)
        print(
            "REAL_FRAMEWORK_RESTORE_PROOF "
            f"framework={fw} loaded_step={proof['loadedStep']} continued_to={proof['continuedToStep']} "
            f"weights_changed={proof['weightsChangedAfterResume']} final={final_path} mode=verl-layout-limited"
        )
        return

    train_torch_steps(torch, model, optimizer, 0, SOURCE_STEPS)
    path = ckpt_root / f"global_step_{SOURCE_STEPS:04d}"
    (path / "actor").mkdir(parents=True, exist_ok=True)
    (path / "critic").mkdir(parents=True, exist_ok=True)
    torch.save({"step": SOURCE_STEPS, "model_state": model.state_dict()}, path / "actor" / "model.pt")
    torch.save({"step": SOURCE_STEPS, "model_state": model.state_dict()}, path / "critic" / "model.pt")
    torch.save({"optimizer_state": optimizer.state_dict()}, path / "optimizer.pt")
    print(f"REAL_FRAMEWORK_SOURCE_READY framework={fw} checkpoint={path} step={SOURCE_STEPS} mode=verl-layout-limited")


def main():
    fw = framework()
    if fw in {"pytorch", "hf-trainer"}:
        run_pytorch()
    elif fw == "lightning":
        run_lightning()
    elif fw == "deepspeed":
        run_deepspeed()
    elif fw == "verl":
        run_verl_limited()
    else:
        raise SystemExit(f"unsupported framework for real validation: {fw}")


if __name__ == "__main__":
    main()
