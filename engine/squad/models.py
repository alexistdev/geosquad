"""Bentuk output terstruktur.

Hanya QAVerdict yang benar-benar ditegakkan lewat output_pydantic, karena
main.py membaca field-nya untuk memutuskan lanjut atau berhenti.

Spec dan DevReport TIDAK ditegakkan. Keduanya sempat dipakai sebagai
output_pydantic, tapi hasilnya tidak pernah dibaca terstruktur -- main.py cuma
melakukan str() atasnya. Memaksakan JSON di sana membuat model gratis kerap
berhenti sebelum menutup objeknya, dan exception-nya mematikan seluruh run.
Keduanya dibiarkan di sini sebagai dokumentasi bentuk yang diharapkan, dan
supaya gampang dinyalakan lagi kalau nanti memakai model yang andal untuk
structured output.
"""

from typing import List, Literal

from pydantic import BaseModel, Field


class AcceptanceCriterion(BaseModel):
    id: str = Field(description="Nomor kriteria, contoh AC-1")
    description: str = Field(description="Perilaku yang harus terpenuhi")
    verify: str = Field(description="Cara membuktikannya, sebisa mungkin lewat test")


class Spec(BaseModel):
    summary: str
    stack: List[str] = Field(description="Bahasa, framework, dan library utama")
    files: List[str] = Field(description="Daftar file yang harus dibuat, path relatif")
    acceptance_criteria: List[AcceptanceCriterion]


class DevReport(BaseModel):
    files_written: List[str]
    covered: List[str] = Field(default_factory=list, description="ID AC yang diklaim selesai")
    notes: str = ""


class QAVerdict(BaseModel):
    status: Literal["PASS", "FAIL"]
    passed: List[str] = Field(default_factory=list, description="ID AC yang lolos")
    failures: List[str] = Field(
        default_factory=list,
        description="Kegagalan konkret yang bisa langsung dikerjakan Developer",
    )
    notes: str = ""
