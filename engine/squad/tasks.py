"""Task dibangun lewat fungsi, bukan variabel modul.

Tidak ada output_file di sini. CrewAI memperlakukan nilainya sebagai path
relatif terhadap direktori kerja, sehingga path absolut ditempelkan ke cwd dan
laporan mendarat di folder bersarang yang salah. Nama filenya juga tidak
mengandung tiket, jadi tiap run menimpa laporan run sebelumnya. Laporan
sekarang ditulis main.py sendiri ke OUTPUT milik tiket yang bersangkutan.

Alasannya ronde QA. Setiap ronde perlu menyuntikkan feedback berbeda ke
Developer, jadi task harus dibuat ulang tiap ronde. Isi spec dan feedback
ditempel langsung ke deskripsi, bukan lewat placeholder CrewAI, supaya tidak
ada interpolasi yang bisa gagal di tengah run.
"""

from crewai import Task

from squad.agents import developer, devops, sqa, tech_lead
from squad import stacks
from squad.config import TEST_COMMAND_OVERRIDE


def _supported_stacks() -> str:
    """Daftar bahasa yang toolchain-nya benar-benar terpasang.

    Diambil saat runtime, bukan ditulis tetap di prompt: mesin yang tidak punya
    Maven tidak boleh menawarkan Java, karena Tech Lead akan memilihnya dan
    seluruh run terbuang."""
    if TEST_COMMAND_OVERRIDE:
        return (
            f"- Perintah test sudah ditetapkan manusia: {TEST_COMMAND_OVERRIDE}\n"
            "  Pilih bahasa yang cocok dengan perintah itu."
        )
    return "\n".join(f"- {name}" for name in stacks.supported_names())
from squad.models import QAVerdict


def plan_task(request: str) -> Task:
    return Task(
        description=(
            "Permintaan dari user:\n\n"
            f"{request}\n\n"
            "Susun kontrak kerja untuk squad. Isinya:\n"
            "1. Ringkasan singkat apa yang dibangun.\n"
            "2. Stack yang dipilih. Pilih yang paling sederhana dan umum. "
            "Jangan menambah teknologi tanpa alasan.\n"
            "3. Daftar file yang harus dibuat, pakai path relatif ke root project. "
            "Sertakan file test.\n"
            "4. Acceptance criteria bernomor AC-1, AC-2, dan seterusnya. Tiap kriteria "
            "harus konkret dan bisa dibuktikan dengan menjalankan test otomatis. "
            "Tulis juga cara membuktikannya.\n\n"
            "Bahasa yang test-nya bisa dijalankan otomatis di mesin ini:\n"
            f"{_supported_stacks()}\n"
            "PILIH SALAH SATU DARI DAFTAR ITU. Bahasa di luar daftar tidak punya "
            "cara menjalankan test, sehingga SQA tidak akan pernah bisa "
            "meluluskannya.\n"
            "Jangan membuat pembungkus dalam bahasa lain untuk mengakali ini; "
            "lapisan tambahan itu justru yang paling sering gagal.\n\n"
            "Buat scope sekecil mungkin yang masih memenuhi permintaan."
        ),
        expected_output=(
            "Kontrak kerja berisi ringkasan, stack, daftar file, dan acceptance "
            "criteria bernomor AC-1, AC-2, dan seterusnya."
        ),
        agent=tech_lead,
        # Sengaja tanpa output_pydantic. Hasilnya hanya dipakai sebagai teks
        # (main.py melakukan str() atasnya, tidak pernah membaca field-nya),
        # jadi memaksa JSON terstruktur menuntut model melakukan hal sulit lalu
        # membuang hasilnya. Model gratis kerap berhenti sebelum menutup objek,
        # dan kegagalannya melempar exception yang mematikan seluruh run.
    )


def dev_task(spec: str, feedback: str, round_no: int) -> Task:
    if round_no == 1:
        opening = "Ini ronde pertama. Bangun project dari nol sesuai kontrak di bawah."
    else:
        opening = (
            f"Ini ronde perbaikan ke-{round_no}. SQA menolak hasil ronde sebelumnya. "
            "Perbaiki kegagalan yang disebut, jangan menulis ulang bagian yang sudah lolos."
        )

    return Task(
        description=(
            f"{opening}\n\n"
            "KONTRAK KERJA DARI TECH LEAD:\n"
            f"{spec}\n\n"
            "TEMUAN SQA YANG HARUS DIPERBAIKI:\n"
            f"{feedback}\n\n"
            "Cara kerja:\n"
            "- Panggil list_project_files dulu untuk melihat apa yang sudah ada.\n"
            "- Tulis setiap file lewat tool write_project_file. File yang hanya kamu "
            "tampilkan di jawaban dianggap tidak pernah dibuat.\n"
            "- Tulis kode lengkap yang bisa dijalankan. Dilarang memakai placeholder, "
            "TODO, atau komentar yang menggantikan implementasi.\n"
            "- Sertakan file test yang membuktikan setiap acceptance criteria.\n"
            "- Test dijalankan lewat tool run_tests. Perintahnya dipilih otomatis "
            "sesuai bahasa project, jadi kamu tidak perlu menentukannya.\n\n"
            "Laporkan file apa saja yang kamu tulis dan AC nomor berapa yang sudah tertutup."
        ),
        expected_output="Laporan berisi daftar file yang ditulis dan acceptance criteria yang tertutup.",
        agent=developer,
        # Alasannya sama seperti plan_task: laporan ini tidak pernah dibaca
        # terstruktur. Yang menentukan lanjut atau tidak adalah verdict SQA.
    )


def qa_task(spec: str, round_no: int) -> Task:
    return Task(
        description=(
            "Uji hasil kerja Developer terhadap kontrak berikut.\n\n"
            "KONTRAK KERJA DARI TECH LEAD:\n"
            f"{spec}\n\n"
            "Cara kerja:\n"
            "- Panggil run_tests untuk menjalankan test suite.\n"
            "- Baca exit code. Nol berarti lolos, selain nol berarti gagal.\n"
            "- Baca file yang relevan kalau perlu memastikan sebuah AC benar tertutup.\n\n"
            "Aturan verdict yang tidak bisa ditawar:\n"
            "- Kalau exit code bukan nol, status wajib FAIL.\n"
            "- Kalau ada acceptance criteria yang tidak punya test yang membuktikannya, "
            "status wajib FAIL.\n"
            "- Status PASS hanya kalau test lolos dan semua AC terbukti.\n\n"
            "Untuk setiap kegagalan, tulis kalimat konkret yang bisa langsung dikerjakan "
            "Developer. Sebutkan nama file dan penyebabnya, bukan keluhan umum."
        ),
        expected_output="Verdict PASS atau FAIL, daftar AC yang lolos, dan daftar kegagalan yang konkret.",
        agent=sqa,
        # Satu-satunya task yang benar-benar butuh output terstruktur: main.py
        # membaca verdict.status untuk memutuskan lanjut atau berhenti, dan
        # verdict.failures untuk disuntikkan ke ronde berikutnya. Kalau JSON-nya
        # gagal, parse_verdict punya jaring pengaman yang membaca PASS/FAIL dari
        # teks bebas, sehingga kegagalan di sini tidak mematikan run.
        output_pydantic=QAVerdict,
    )


def deploy_task(spec: str, verdict_notes: str) -> Task:
    return Task(
        description=(
            "SQA sudah meluluskan build ini dan manusia sudah menyetujui rilis.\n\n"
            "KONTRAK KERJA:\n"
            f"{spec}\n\n"
            "CATATAN SQA:\n"
            f"{verdict_notes}\n\n"
            "Tugas kamu:\n"
            "- Panggil run_deploy satu kali untuk merilis ke server production.\n"
            "- Baca exit code hasilnya. Nol berarti deploy sukses.\n"
            "- Kalau exit code bukan nol, jangan mengulang deploy. Laporkan kegagalannya "
            "dan tulis langkah rollback yang harus dilakukan manusia.\n\n"
            "Tutup dengan ringkasan rilis: apa yang dideploy, hasilnya, apa yang perlu "
            "dipantau setelah ini, dan cara rollback kalau ada masalah."
        ),
        expected_output="Laporan hasil deploy berisi status, hal yang perlu dipantau, dan langkah rollback.",
        agent=devops,
    )
