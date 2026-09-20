"""Deteksi bahasa project dan perintah test yang sesuai.

Yang penting dari file ini bukan daftarnya, tapi dari mana perintahnya berasal.

Agent TIDAK boleh mengarang perintah test. Kalau boleh, "menulis file" berubah
jadi "menjalankan perintah apa pun", dan seluruh pembatasan di tools.py jadi
percuma. Yang bisa dilakukan agent hanyalah memilih file apa yang ia tulis;
perintahnya sendiri sudah ditetapkan di sini, ditulis dan direview manusia.

Konsekuensinya bahasa baru tidak bisa ditambahkan agent saat runtime. Itu
memang disengaja: menambah bahasa berarti menambah perintah yang boleh
dijalankan di mesin ini, dan keputusan itu milik manusia.
"""

import shutil
from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True)
class Stack:
    name: str
    # Nama file penanda di root workspace. Cukup satu yang cocok.
    markers: tuple[str, ...]
    # Executable yang harus ada di PATH. Kosong berarti tidak perlu dicek
    # (Python memakai interpreter venv yang path-nya sudah pasti).
    binary: str
    # Perintah yang dijalankan sekali sebelum test, misalnya memasang
    # dependency. Dipisah per perintah karena dieksekusi tanpa shell, jadi
    # "&&" tidak akan berfungsi.
    prepare: tuple[str, ...] = field(default_factory=tuple)
    # Perintah test. {python} diganti path interpreter venv workspace.
    test: str = ""


# Urutan menentukan. Penanda yang lebih spesifik diperiksa lebih dulu, dan
# Python ditaruh terakhir sebagai jaring terakhir karena file .py bisa saja
# ikut muncul di project bahasa lain (misalnya skrip pembantu).
STACKS: tuple[Stack, ...] = (
    Stack(
        name="Go",
        markers=("go.mod",),
        binary="go",
        # Dependency diambil otomatis oleh go build/test dari go.mod.
        test="go test ./...",
    ),
    Stack(
        name="Java (Maven)",
        markers=("pom.xml",),
        binary="mvn",
        test="mvn -q -B test",
    ),
    Stack(
        name="Rust",
        markers=("Cargo.toml",),
        binary="cargo",
        test="cargo test --quiet",
    ),
    Stack(
        name="PHP",
        markers=("composer.json",),
        binary="composer",
        prepare=("composer install --no-interaction --quiet",),
        test="vendor/bin/phpunit",
    ),
    Stack(
        name="Ruby",
        markers=("Gemfile",),
        binary="bundle",
        prepare=("bundle install --quiet",),
        test="bundle exec rspec",
    ),
    Stack(
        name="Node",
        markers=("package.json",),
        binary="npm",
        prepare=("npm install --no-audit --no-fund --silent",),
        test="npm test --silent",
    ),
    Stack(
        name="Python",
        markers=("pyproject.toml", "pytest.ini", "setup.py", "requirements.txt"),
        binary="",
        test="{python} -m pytest -q",
    ),
)

PYTHON = STACKS[-1]


def detect(workspace: Path) -> Stack:
    """Tebak stack dari file penanda di root workspace.

    Kembali ke Python kalau tidak ada penanda yang cocok. Python juga yang
    dipakai saat project masih kosong, yaitu saat Developer baru mulai menulis.
    """
    for stack in STACKS:
        if any((workspace / marker).is_file() for marker in stack.markers):
            return stack

    # Tidak ada penanda, tapi mungkin sudah ada file .py yang ditulis Developer.
    if any(workspace.rglob("*.py")):
        return PYTHON
    return PYTHON


def available(stack: Stack) -> bool:
    """Apakah toolchain stack ini terpasang di mesin."""
    return not stack.binary or shutil.which(stack.binary) is not None


def supported_names() -> list[str]:
    """Stack yang toolchain-nya benar-benar ada. Dipakai untuk memberi tahu
    Tech Lead pilihan yang realistis, bukan daftar teoretis."""
    return [s.name for s in STACKS if available(s)]


def resolve(stack: Stack, python: str) -> tuple[list[str], str]:
    """Kembalikan (perintah persiapan, perintah test) yang siap dijalankan."""
    prepare = [cmd.replace("{python}", python) for cmd in stack.prepare]
    return prepare, stack.test.replace("{python}", python)
