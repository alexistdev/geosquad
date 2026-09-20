"""Venv terpisah untuk workspace, dikelola runner dan bukan oleh agent.

Kenapa tidak berbagi venv dengan runner:
    requirements.txt ditulis oleh agent. Kalau venv-nya sama dengan yang
    menjalankan CrewAI, satu pin dari agent bisa menyeret turun dependency
    runner dan mematikan squad di tengah run. Ini bukan teori: fastapi==0.110.0
    menarik starlette ke 0.36.3, padahal sse-starlette (lewat mcp -> crewai)
    butuh >=0.49.1.

Kenapa venv-nya DI LUAR workspace:
    write_project_file mengizinkan tulis ke mana pun di dalam workspace,
    termasuk .venv/bin/python. Kalau venv diletakkan di dalam, agent bisa
    menimpa interpreter yang nanti dieksekusi run_tests, dan "menulis file"
    berubah jadi "menjalankan kode". SKIP_DIRS tidak menutup ini karena hanya
    dipakai untuk menyembunyikan dari listing, bukan untuk menolak tulis.

Stamp hash juga disimpan di dalam venv, di luar jangkauan agent, supaya agent
tidak bisa memalsukannya agar install dilewati.
"""

import hashlib
import shutil
import subprocess
import sys
from pathlib import Path

from squad.config import (
    ALLOW_DEPS,
    DEPS_TIMEOUT,
    REQUIREMENTS,
    VENV_DIR,
    VENV_PYTHON,
    WORKSPACE_PY_VERSION,
)

STAMP = VENV_DIR / ".squad-requirements.sha256"


def _run(args: list[str], timeout: int) -> tuple[int, str]:
    try:
        proc = subprocess.run(
            args, capture_output=True, text=True, timeout=timeout
        )
    except FileNotFoundError:
        return 127, f"Perintah tidak ditemukan: {args[0]}"
    except subprocess.TimeoutExpired:
        return 124, f"Lewat batas {timeout} detik: {' '.join(args)}"
    return proc.returncode, (proc.stdout + proc.stderr).strip()


def _uv() -> str | None:
    return shutil.which("uv")


def _has(module: str) -> bool:
    if not VENV_PYTHON.is_file():
        return False
    code, _ = _run([str(VENV_PYTHON), "-c", f"import {module}"], 60)
    return code == 0


def ensure_venv() -> str:
    """Bikin venv workspace kalau belum ada. Idempotent."""
    if VENV_PYTHON.is_file():
        return ""

    VENV_DIR.parent.mkdir(parents=True, exist_ok=True)
    uv = _uv()
    if uv:
        code, out = _run(
            [uv, "venv", str(VENV_DIR), "--python", WORKSPACE_PY_VERSION], 300
        )
    else:
        code, out = _run(
            [sys.executable, "-m", "venv", str(VENV_DIR)], 300
        )
    if code != 0:
        raise RuntimeError(f"Gagal membuat venv workspace di {VENV_DIR}:\n{out}")

    msg = f"[venv] dibuat: {VENV_DIR}"
    # pytest adalah alat ukur SQA, bukan dependency aplikasi. Selalu ada,
    # tidak bergantung pada apakah agent ingat menulisnya di requirements.txt.
    if not _has("pytest"):
        code, out = _run(_install_args(["pytest"]), DEPS_TIMEOUT)
        if code != 0:
            raise RuntimeError(f"Gagal memasang pytest di venv workspace:\n{out}")
        msg += " + pytest"
    return msg


def _install_args(spec: list[str]) -> list[str]:
    uv = _uv()
    if uv:
        return [uv, "pip", "install", "--python", str(VENV_PYTHON), *spec]
    return [str(VENV_PYTHON), "-m", "pip", "install", *spec]


def _digest() -> str:
    if not REQUIREMENTS.is_file():
        return "tidak-ada"
    return hashlib.sha256(REQUIREMENTS.read_bytes()).hexdigest()


def sync_requirements() -> str:
    """Pasang requirements.txt workspace kalau isinya berubah sejak sync terakhir."""
    if not ALLOW_DEPS:
        return ""
    current = _digest()
    if current == "tidak-ada":
        return ""
    if STAMP.is_file() and STAMP.read_text().strip() == current:
        return ""

    code, out = _run(_install_args(["-r", str(REQUIREMENTS)]), DEPS_TIMEOUT)
    if code != 0:
        # Sengaja tidak raise: biar SQA melihatnya sebagai kegagalan yang bisa
        # ditindak, bukan crash yang menghentikan seluruh run.
        return f"[deps] GAGAL memasang requirements.txt (exit {code}):\n{out[:1500]}"

    STAMP.write_text(current)
    if not _has("pytest"):
        _run(_install_args(["pytest"]), DEPS_TIMEOUT)
    return f"[deps] requirements.txt tersinkron ke {VENV_DIR.name}"


def ensure_ready() -> str:
    """Dipanggil sebelum test Python jalan. Gabungan bikin venv + sync dependency.

    Hanya relevan untuk stack Python. Project Go, Java, atau Rust mengambil
    dependency-nya sendiri lewat toolchain masing-masing, dan membuatkan venv
    Python untuk mereka cuma membuang waktu serta ruang disk.
    """
    lines = [ensure_venv(), sync_requirements()]
    return "\n".join(line for line in lines if line)


def describe() -> str:
    status = "siap" if VENV_PYTHON.is_file() else "belum dibuat"
    return f"venv workspace: {VENV_DIR} ({status})"
