import { Component, input } from '@angular/core';
import { RunStatus } from '../../core/models/run';

/** Lencana status run. Warna dan label dipetakan sekali di sini saja. */
@Component({
  selector: 'app-status-badge',
  template: `<span class="badge" [class]="'badge ' + style()">{{ label() }}</span>`,
})
export class StatusBadge {
  readonly status = input.required<RunStatus>();

  private static readonly LABELS: Record<RunStatus, string> = {
    PENDING: 'Antre',
    RUNNING: 'Berjalan',
    PASSED: 'Lulus',
    FAILED: 'Gagal',
    CANCELLED: 'Dibatalkan',
    ERROR: 'Error',
  };

  // Kelas badge bawaan Velzon: latar lembut dengan teks berwarna senada.
  private static readonly STYLES: Record<RunStatus, string> = {
    PENDING: 'bg-warning-subtle text-warning',
    RUNNING: 'bg-primary-subtle text-primary',
    PASSED: 'bg-success-subtle text-success',
    FAILED: 'bg-danger-subtle text-danger',
    CANCELLED: 'bg-secondary-subtle text-secondary',
    ERROR: 'bg-danger-subtle text-danger',
  };

  protected label(): string {
    return StatusBadge.LABELS[this.status()] ?? this.status();
  }

  protected style(): string {
    return StatusBadge.STYLES[this.status()] ?? 'bg-secondary-subtle text-secondary';
  }
}
