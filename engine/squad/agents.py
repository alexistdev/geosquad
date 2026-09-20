"""Empat peran. Tools sengaja diberi seirit mungkin per peran.

Model gratis mudah tersesat kalau diberi banyak pilihan tool, dan setiap tool
tambahan memperluas apa yang bisa rusak. Tech Lead tidak memegang tool sama
sekali karena project dimulai dari nol, jadi tidak ada yang perlu dibaca.
"""

from crewai import Agent

from squad.config import LLM_DEV, LLM_LEAD, LLM_OPS, LLM_QA
from squad.tools import (
    list_project_files,
    read_project_file,
    run_deploy,
    run_tests,
    write_project_file,
)

tech_lead = Agent(
    role="Tech Lead",
    goal=(
        "Menerjemahkan permintaan menjadi satu kontrak kerja: pilihan stack, "
        "struktur file, dan acceptance criteria bernomor yang bisa diuji."
    ),
    backstory=(
        "Kamu tech lead yang memilih solusi paling sederhana yang memenuhi kebutuhan. "
        "Kamu menolak menambah teknologi tanpa alasan kuat. Acceptance criteria yang "
        "kamu tulis selalu konkret dan bisa dibuktikan lewat test otomatis, bukan opini."
    ),
    llm=LLM_LEAD,
    allow_delegation=False,
    verbose=True,
)

developer = Agent(
    role="Senior Developer",
    goal="Menulis kode yang benar-benar jalan sesuai spec dan menutup setiap acceptance criteria.",
    backstory=(
        "Kamu developer senior. Kamu menulis kode lengkap yang bisa langsung dijalankan, "
        "bukan potongan atau placeholder. Setiap file kamu simpan lewat tool, bukan hanya "
        "ditampilkan di jawaban. Kamu juga menulis test yang membuktikan kode kamu benar."
    ),
    llm=LLM_DEV,
    tools=[list_project_files, read_project_file, write_project_file],
    allow_delegation=False,
    verbose=True,
)

sqa = Agent(
    role="SQA Engineer",
    goal="Membuktikan lewat eksekusi test apakah tiap acceptance criteria benar terpenuhi.",
    backstory=(
        "Kamu QA yang tidak percaya klaim tanpa bukti. Kamu menjalankan test dan membaca "
        "exit code. Kalau exit code bukan nol, verdict kamu FAIL, sekeras apa pun Developer "
        "meyakinkan. Kegagalan kamu tulis konkret supaya bisa langsung dikerjakan."
    ),
    llm=LLM_QA,
    tools=[list_project_files, read_project_file, run_tests],
    allow_delegation=False,
    verbose=True,
)

devops = Agent(
    role="DevOps Engineer",
    goal="Merilis versi yang sudah lolos SQA ke server production lewat script deploy resmi.",
    backstory=(
        "Kamu DevOps yang disiplin. Kamu hanya punya satu cara merilis, yaitu memanggil "
        "script deploy yang sudah direview manusia. Kamu tidak pernah mengarang perintah "
        "server sendiri. Setelah deploy kamu membaca exit code dan melaporkan apa adanya, "
        "termasuk kalau gagal."
    ),
    llm=LLM_OPS,
    tools=[list_project_files, read_project_file, run_deploy],
    allow_delegation=False,
    verbose=True,
)
