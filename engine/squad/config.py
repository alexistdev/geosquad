"""Konfigurasi terpusat: path, batas keamanan, dan LLM per peran."""

import os
import re
import shlex
import sys
from datetime import datetime
from pathlib import Path

from crewai import LLM
from dotenv import load_dotenv

load_dotenv()

ROOT = Path(__file__).resolve().parent.parent


def _ticket() -> str:
    """Nomor tiket yang memisahkan satu task dari task lain.

    Normalnya diberikan API lewat SQUAD_TICKET (geo-1001, geo-1002, ...) yang
    bersumber dari AUTO_INCREMENT MySQL sehingga tidak mungkin bentrok. Saat
    engine dijalankan manual dari CLI tidak ada API yang memberi nomor, jadi
    dipakai penanda waktu -- diberi awalan "local" supaya jelas bukan tiket
    yang tercatat di database.
    """
    given = (os.getenv("SQUAD_TICKET") or "").strip()
    if given:
        # Tiket ikut jadi nama folder, jadi karakternya dibatasi. Tanpa ini,
        # nilai berisi "../" bisa membuat workspace mendarat di luar engine.
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", given):
            raise RuntimeError(
                f"SQUAD_TICKET tidak valid: {given!r}. "
                "Pakai huruf, angka, titik, minus, atau garis bawah saja."
            )
        return given
    return f"local-{datetime.now():%Y%m%d-%H%M%S}"


TICKET = _ticket()

# Tiap tiket punya folder sendiri. Sebelum ini semua task berbagi satu
# workspace, sehingga file antar-task menumpuk dan laporan saling menimpa.
WORKSPACE_ROOT = Path(os.getenv("SQUAD_WORKSPACE") or (ROOT / "workspace")).resolve()
OUTPUT_ROOT = Path(os.getenv("SQUAD_OUTPUT") or (ROOT / "output")).resolve()
VENVS_ROOT = Path(os.getenv("SQUAD_VENVS_DIR") or (ROOT / ".venvs")).resolve()

# Folder tempat squad membangun aplikasi. Sengaja dipisah dari project runner
# ini supaya agent tidak pernah bisa mengedit kode yang sedang menjalankannya.
WORKSPACE = WORKSPACE_ROOT / TICKET
OUTPUT = OUTPUT_ROOT / TICKET
DEPLOY_SCRIPT = ROOT / "deploy.sh"

# Venv workspace sengaja DI LUAR WORKSPACE. write_project_file mengizinkan tulis
# ke mana pun di dalam workspace, termasuk .venv/bin/python. Kalau venv ada di
# dalam, agent bisa menimpa interpreter yang dieksekusi run_tests.
#
# Per tiket juga, supaya requirements.txt satu task tidak menyeret versi
# dependency task lain.
VENV_DIR = Path(os.getenv("SQUAD_VENV_DIR") or (VENVS_ROOT / TICKET)).resolve()
if VENV_DIR == WORKSPACE or WORKSPACE in VENV_DIR.parents:
    raise RuntimeError(
        f"SQUAD_VENV_DIR ({VENV_DIR}) berada di dalam workspace. "
        "Agent bisa menimpa interpreternya. Letakkan di luar workspace."
    )
VENV_PYTHON = VENV_DIR / "bin" / "python"
REQUIREMENTS = WORKSPACE / "requirements.txt"
WORKSPACE_PY_VERSION = os.getenv("SQUAD_WORKSPACE_PY", f"{sys.version_info.major}.{sys.version_info.minor}")

# Agent boleh mendeklarasikan dependency lewat requirements.txt. Set 0 untuk
# mengunci venv workspace apa adanya.
ALLOW_DEPS = os.getenv("SQUAD_ALLOW_DEPS", "1") == "1"
DEPS_TIMEOUT = int(os.getenv("SQUAD_DEPS_TIMEOUT", "600"))

# Perintah test tidak pernah dikarang agent. Yang boleh dijalankan sudah
# ditetapkan di squad/stacks.py, dan yang dipilih ditentukan dari file penanda
# di workspace -- bukan dari string yang ditulis model.
#
# Isi SQUAD_TEST_COMMAND untuk memaksa satu perintah dan melewati deteksi.
# Itu keputusan manusia, jadi diizinkan.
TEST_COMMAND_OVERRIDE = os.getenv("SQUAD_TEST_COMMAND", "").strip()

# Perintah default saat workspace masih kosong, dipakai Tech Lead untuk
# menjelaskan cara test dijalankan.
TEST_COMMAND = TEST_COMMAND_OVERRIDE or f"{shlex.quote(str(VENV_PYTHON))} -m pytest -q"
TEST_TIMEOUT = int(os.getenv("SQUAD_TEST_TIMEOUT", "300"))
DEPLOY_TIMEOUT = int(os.getenv("SQUAD_DEPLOY_TIMEOUT", "900"))

MAX_QA_ROUNDS = int(os.getenv("SQUAD_MAX_QA_ROUNDS", "3"))
MAX_RPM = int(os.getenv("SQUAD_MAX_RPM", "10"))
MAX_FILE_BYTES = int(os.getenv("SQUAD_MAX_FILE_BYTES", "200000"))

# Saklar sementara: DevOps dimatikan default sampai deploy.sh benar-benar dikonfigurasi.
ENABLE_DEVOPS = os.getenv("SQUAD_ENABLE_DEVOPS", "0") == "1"

WORKSPACE.mkdir(parents=True, exist_ok=True)
OUTPUT.mkdir(parents=True, exist_ok=True)


def _llm(temperature: float, role_env: str) -> LLM:
    model = os.getenv(role_env) or os.getenv("NINE_ROUTER_MODEL")
    if not model:
        raise RuntimeError("NINE_ROUTER_MODEL belum diset di .env")
    return LLM(
        model=model,
        base_url=os.getenv("NINE_ROUTER_BASE_URL"),
        api_key=os.getenv("NINE_ROUTER_API_KEY"),
        temperature=temperature,
    )


# Semua peran default ke model yang sama. Begitu ada model berbayar, cukup isi
# SQUAD_MODEL_DEV di .env tanpa mengubah kode.
LLM_LEAD = _llm(0.3, "SQUAD_MODEL_LEAD")
LLM_DEV = _llm(0.2, "SQUAD_MODEL_DEV")
LLM_QA = _llm(0.5, "SQUAD_MODEL_QA")
LLM_OPS = _llm(0.2, "SQUAD_MODEL_OPS")
