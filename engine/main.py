"""Orkestrator squad: Tech Lead -> Developer -> SQA -> DevOps.

Tiap crew di dalamnya tetap sequential. Yang tidak sequential hanya gerbangnya,
dan gerbang itu Python biasa di file ini, bukan fitur CrewAI. Konsekuensinya
mudah dibaca: DevOps tidak akan pernah jalan kalau verdict SQA bukan PASS.
"""

import re
import subprocess
import sys
from datetime import datetime

from crewai import Crew, Process

from squad.agents import developer, devops, sqa, tech_lead
from squad.config import (
    DEPLOY_SCRIPT,
    ENABLE_DEVOPS,
    MAX_QA_ROUNDS,
    MAX_RPM,
    OUTPUT,
    TICKET,
    VENV_DIR,
    WORKSPACE,
)
from squad.models import QAVerdict
from squad.tasks import deploy_task, dev_task, plan_task, qa_task
from squad.venv import ensure_venv

DEFAULT_REQUEST = (
    "Buat REST API sederhana untuk mencatat todo, dengan endpoint tambah, "
    "lihat semua, dan tandai selesai. Data cukup disimpan di memori."
)


def banner(text: str) -> None:
    print(f"\n{'=' * 70}\n{text}\n{'=' * 70}\n")


def write_output(name: str, content: str) -> None:
    """Simpan laporan satu tahap ke folder output milik tiket ini.

    Ditulis di sini, bukan lewat output_file milik CrewAI: CrewAI menganggap
    nilainya relatif terhadap cwd, sehingga path absolut mendarat di folder
    bersarang yang salah.
    """
    OUTPUT.mkdir(parents=True, exist_ok=True)
    target = OUTPUT / name
    try:
        target.write_text(content, encoding="utf-8")
        print(f"[output] {target}")
    except OSError as exc:
        # Laporan gagal disimpan tidak boleh menggagalkan run yang kodenya
        # sudah jadi. Cukup diberitahukan.
        print(f"[output] GAGAL menyimpan {target}: {exc}", file=sys.stderr)


def git(*args: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["git", *args], cwd=str(WORKSPACE), capture_output=True, text=True
    )


def git_init_branch() -> str:
    """Tiap run hidup di branch sendiri, supaya hasil buruk cukup dibuang."""
    if not (WORKSPACE / ".git").is_dir():
        git("init", "-q")
        git("commit", "-q", "--allow-empty", "-m", "baseline kosong")

    branch = f"squad/{datetime.now():%Y%m%d-%H%M%S}"
    git("checkout", "-q", "-b", branch)
    return branch


def git_snapshot(label: str) -> None:
    git("add", "-A")
    result = git("commit", "-q", "-m", label)
    if result.returncode == 0:
        print(f"[git] commit: {label}")


def parse_verdict(result) -> QAVerdict:
    """Ambil verdict terstruktur, dengan jaring pengaman kalau model gratis
    gagal menghasilkan JSON yang valid."""
    if getattr(result, "pydantic", None) is not None:
        return result.pydantic

    raw = str(result)
    status = "PASS" if re.search(r"\bPASS\b", raw) and not re.search(r"\bFAIL\b", raw) else "FAIL"
    return QAVerdict(
        status=status,
        failures=[] if status == "PASS" else [raw[:2000]],
        notes="Verdict dibaca dari teks bebas karena output terstruktur gagal diparse.",
    )


def ask_approval(prompt: str) -> bool:
    if not sys.stdin.isatty():
        print("Sesi non-interaktif, deploy dilewati demi keamanan.")
        return False
    return input(f"{prompt} ketik DEPLOY untuk lanjut: ").strip() == "DEPLOY"


def main() -> int:
    request = " ".join(sys.argv[1:]).strip() or DEFAULT_REQUEST

    banner(
        f"SQUAD START\ntiket: {TICKET}\nworkspace: {WORKSPACE}\n"
        f"venv workspace: {VENV_DIR}\noutput: {OUTPUT}"
    )
    print(f"Request: {request}\n")

    # Disiapkan di awal, bukan saat SQA pertama kali jalan, supaya kegagalan
    # setup terlihat sebelum satu token pun dibakar.
    created = ensure_venv()
    if created:
        print(created)

    branch = git_init_branch()
    print(f"[git] branch: {branch}")

    # Tahap 1: Tech Lead. Jalan sekali, hasilnya jadi kontrak untuk semua.
    banner("TAHAP 1 - TECH LEAD menyusun kontrak")
    plan_crew = Crew(
        agents=[tech_lead],
        tasks=[plan_task(request)],
        process=Process.sequential,
        max_rpm=MAX_RPM,
        verbose=True,
    )
    spec_result = plan_crew.kickoff()
    spec = str(spec_result)
    write_output("01-spec.md", spec)
    git_snapshot("tech lead: kontrak kerja")

    # Tahap 2: Developer dan SQA, sequential di dalam, diulang sampai lolos.
    feedback = "Belum ada temuan. Ini ronde pertama."
    verdict = None

    for round_no in range(1, MAX_QA_ROUNDS + 1):
        banner(f"TAHAP 2 - RONDE {round_no} dari {MAX_QA_ROUNDS}: DEVELOPER lalu SQA")

        build_crew = Crew(
            agents=[developer, sqa],
            tasks=[
                dev_task(spec, feedback, round_no),
                qa_task(spec, round_no),
            ],
            process=Process.sequential,
            max_rpm=MAX_RPM,
            verbose=True,
        )
        build_result = build_crew.kickoff()
        verdict = parse_verdict(build_result)

        # Crew ini berisi dua task berurutan: Developer lalu SQA. kickoff()
        # hanya mengembalikan keluaran task terakhir, jadi laporan Developer
        # diambil dari tasks_output.
        outputs = getattr(build_result, "tasks_output", []) or []
        if outputs:
            write_output(f"02-developer-round-{round_no}.md", str(outputs[0]))
        write_output(f"03-sqa-round-{round_no}.md", str(build_result))

        git_snapshot(f"ronde {round_no}: verdict {verdict.status}")

        print(f"\n>>> Verdict SQA ronde {round_no}: {verdict.status}")
        if verdict.status == "PASS":
            break

        for item in verdict.failures:
            print(f"    - {item}")
        feedback = "\n".join(f"- {item}" for item in verdict.failures) or "SQA menolak tanpa detail."

    if verdict is None or verdict.status != "PASS":
        banner(f"BERHENTI - SQA masih FAIL setelah {MAX_QA_ROUNDS} ronde. Deploy dibatalkan.")
        print(f"Hasil kerja tersimpan di branch {branch}. Periksa {OUTPUT}.")
        return 1

    if not ENABLE_DEVOPS:
        banner("SELESAI - DevOps sedang dimatikan (SQUAD_ENABLE_DEVOPS=0)")
        print(f"QA lulus. Deploy dilewati karena peran DevOps sedang dimatikan sementara.")
        print(f"Hasil kerja ada di branch {branch}. Set SQUAD_ENABLE_DEVOPS=1 di .env untuk mengaktifkan lagi.")
        return 0

    # Tahap 3: DevOps. Butuh verdict PASS dan persetujuan manusia.
    banner("TAHAP 3 - DEVOPS deploy ke production")

    if not DEPLOY_SCRIPT.is_file():
        print(f"deploy.sh tidak ada di {DEPLOY_SCRIPT}. Deploy dilewati.")
        return 0

    if not ask_approval(f"SQA lulus. Deploy branch {branch} ke production?"):
        print("Deploy dibatalkan oleh user. Kode tetap aman di branch.")
        return 0

    deploy_crew = Crew(
        agents=[devops],
        tasks=[deploy_task(spec, verdict.notes)],
        process=Process.sequential,
        max_rpm=MAX_RPM,
        verbose=True,
    )
    deploy_result = deploy_crew.kickoff()
    write_output("04-devops-deploy.md", str(deploy_result))
    git_snapshot("devops: deploy production")

    banner("SELESAI")
    print(deploy_result)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
