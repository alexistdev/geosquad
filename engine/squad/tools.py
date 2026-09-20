"""Tool untuk agent.

Dua aturan yang membentuk file ini:

1. Semua operasi file dikurung di dalam WORKSPACE. Path di luar itu ditolak.
2. Tool yang menjalankan perintah tidak menerima argumen apa pun dari LLM.
   Perintah test dipilih dari daftar tetap di squad/stacks.py berdasarkan file
   penanda yang ada di workspace -- agent memengaruhinya lewat file yang ia
   tulis, bukan lewat string perintah. Ini menutup command injection, dan
   sekaligus meringankan model gratis yang sering keliru memformat argumen.
"""

import shlex
import subprocess
from pathlib import Path

from crewai.tools import tool

from squad import stacks
from squad.config import (
    DEPLOY_SCRIPT,
    DEPLOY_TIMEOUT,
    DEPS_TIMEOUT,
    MAX_FILE_BYTES,
    TEST_COMMAND_OVERRIDE,
    TEST_TIMEOUT,
    VENV_PYTHON,
    WORKSPACE,
)
from squad.venv import ensure_ready

SKIP_DIRS = {".git", "node_modules", "__pycache__", ".venv", "venv", "dist", "build"}


def _resolve(relative_path: str) -> Path:
    """Ubah path relatif jadi absolut, dan tolak apa pun yang keluar workspace."""
    cleaned = relative_path.strip()
    if not cleaned:
        raise ValueError("Ditolak: path kosong.")
    if Path(cleaned).is_absolute():
        # Ditolak, bukan dipotong jadi relatif. Agent harus tahu path-nya salah
        # daripada menemukan file muncul di tempat yang tidak dia duga.
        raise ValueError(
            f"Ditolak: {relative_path} adalah path absolut. "
            "Pakai path relatif ke root project."
        )

    candidate = (WORKSPACE / cleaned).resolve()
    if candidate != WORKSPACE and WORKSPACE not in candidate.parents:
        raise ValueError(
            f"Ditolak: {relative_path} berada di luar workspace {WORKSPACE}"
        )
    return candidate


def _run(command: str, cwd: Path, timeout: int) -> str:
    try:
        proc = subprocess.run(
            shlex.split(command),
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except FileNotFoundError:
        return f"EXIT_CODE: 127\nPerintah tidak ditemukan: {command}"
    except subprocess.TimeoutExpired:
        return f"EXIT_CODE: 124\nPerintah melewati batas {timeout} detik: {command}"

    output = (proc.stdout + proc.stderr).strip()
    if len(output) > 12000:
        output = output[:6000] + "\n...[dipotong]...\n" + output[-6000:]
    return f"EXIT_CODE: {proc.returncode}\n{output}"


@tool("write_project_file")
def write_project_file(path: str, content: str) -> str:
    """Tulis file di dalam project. path relatif ke root project, content isi lengkap file.
    Menimpa file yang sudah ada. Hanya bisa menulis di dalam workspace project."""
    try:
        target = _resolve(path)
    except ValueError as exc:
        return str(exc)

    if any(part in SKIP_DIRS for part in target.relative_to(WORKSPACE).parts):
        return (
            f"Ditolak: {path} berada di direktori infrastruktur "
            f"({', '.join(sorted(SKIP_DIRS))}). Tulis kode aplikasi saja."
        )

    if len(content.encode("utf-8")) > MAX_FILE_BYTES:
        return f"Ditolak: isi file melebihi batas {MAX_FILE_BYTES} byte."

    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8")
    return f"Tersimpan: {target.relative_to(WORKSPACE)} ({len(content)} karakter)"


@tool("read_project_file")
def read_project_file(path: str) -> str:
    """Baca isi satu file di dalam project. path relatif ke root project."""
    try:
        target = _resolve(path)
    except ValueError as exc:
        return str(exc)

    if not target.is_file():
        return f"File tidak ada: {path}"
    return target.read_text(encoding="utf-8", errors="replace")[:MAX_FILE_BYTES]


@tool("list_project_files")
def list_project_files() -> str:
    """Tampilkan semua file yang sudah ada di dalam project. Tidak butuh argumen."""
    found = []
    for item in sorted(WORKSPACE.rglob("*")):
        if any(part in SKIP_DIRS for part in item.parts):
            continue
        if item.is_file():
            found.append(str(item.relative_to(WORKSPACE)))
    return "\n".join(found) if found else "Project masih kosong."


@tool("run_tests")
def run_tests() -> str:
    """Jalankan test suite project dan kembalikan exit code beserta outputnya.
    Perintahnya dipilih otomatis sesuai bahasa project, dan dependency dipasang
    lebih dulu. Exit code 0 berarti semua test lolos. Tidak butuh argumen."""
    stack = stacks.detect(WORKSPACE)
    notes: list[str] = []

    if TEST_COMMAND_OVERRIDE:
        # Manusia sudah menentukan perintahnya lewat .env. Deteksi dilewati.
        prepare, test_cmd = [], TEST_COMMAND_OVERRIDE
        notes.append(f"[test] perintah dari SQUAD_TEST_COMMAND: {test_cmd}")
    else:
        if not stacks.available(stack):
            return (
                f"EXIT_CODE: 127\nProject terdeteksi sebagai {stack.name}, "
                f"tapi '{stack.binary}' tidak terpasang di mesin ini. "
                "Pakai bahasa lain, atau minta administrator memasang toolchain itu."
            )
        prepare, test_cmd = stacks.resolve(stack, str(VENV_PYTHON))
        notes.append(f"[test] stack terdeteksi: {stack.name}")

    # Venv Python hanya disiapkan untuk project Python. Project Go, Java, atau
    # Rust mengambil dependency-nya lewat toolchain sendiri.
    if stack is stacks.PYTHON and not TEST_COMMAND_OVERRIDE:
        try:
            prelude = ensure_ready()
        except RuntimeError as exc:
            return f"EXIT_CODE: 1\nGagal menyiapkan venv workspace: {exc}"
        if prelude:
            notes.append(prelude)

    for cmd in prepare:
        outcome = _run(cmd, WORKSPACE, DEPS_TIMEOUT)
        if not outcome.startswith("EXIT_CODE: 0"):
            # Gagal memasang dependency dilaporkan sebagai kegagalan test yang
            # bisa ditindak SQA, bukan disembunyikan.
            return f"{chr(10).join(notes)}\n[deps] GAGAL: {cmd}\n{outcome}"
        notes.append(f"[deps] {cmd}")

    result = _run(test_cmd, WORKSPACE, TEST_TIMEOUT)
    return f"{chr(10).join(notes)}\n{result}"


@tool("run_deploy")
def run_deploy() -> str:
    """Jalankan script deploy resmi ke server production. Tidak butuh argumen.
    Script ini sudah ditulis dan direview manusia. Kamu tidak bisa mengubah isinya."""
    if not DEPLOY_SCRIPT.is_file():
        return f"Deploy dibatalkan: {DEPLOY_SCRIPT} tidak ditemukan."
    return _run(f"bash {shlex.quote(str(DEPLOY_SCRIPT))}", WORKSPACE, DEPLOY_TIMEOUT)
