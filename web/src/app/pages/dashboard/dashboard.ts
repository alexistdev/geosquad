import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DatePipe } from '@angular/common';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../core/services/auth.service';
import { RunService } from '../../core/services/run.service';
import { Run, RunStatus, isTerminal } from '../../core/models/run';
import { StatusBadge } from '../../shared/ui/status-badge';

@Component({
  selector: 'app-dashboard',
  imports: [FormsModule, RouterLink, DatePipe, StatusBadge],
  templateUrl: './dashboard.html',
})
export class Dashboard implements OnInit, OnDestroy {
  private readonly runs = inject(RunService);
  private readonly router = inject(Router);
  protected readonly auth = inject(AuthService);

  protected readonly items = signal<Run[]>([]);
  protected readonly total = signal(0);
  protected readonly loading = signal(true);
  protected readonly error = signal('');

  protected readonly submitting = signal(false);
  protected newRequest = '';
  protected statusFilter: RunStatus | '' = '';

  private poller?: ReturnType<typeof setInterval>;

  ngOnInit(): void {
    this.load();
    // Daftar run tidak punya aliran realtime sendiri -- SSE dipasang per run,
    // bukan per daftar. Polling ringan sudah cukup untuk membuat kolom status
    // bergerak, dan berhenti sendiri saat tidak ada yang sedang berjalan.
    this.poller = setInterval(() => {
      if (this.items().some((run) => !isTerminal(run.status))) {
        this.load(true);
      }
    }, 5000);
  }

  ngOnDestroy(): void {
    clearInterval(this.poller);
  }

  protected load(silent = false): void {
    if (!silent) this.loading.set(true);

    this.runs.list({ perPage: 25, status: this.statusFilter }).subscribe({
      next: (page) => {
        this.items.set(page.items);
        this.total.set(page.meta.total);
        this.loading.set(false);
      },
      error: (err: Error) => {
        this.loading.set(false);
        if (!silent) this.error.set(err.message);
      },
    });
  }

  protected submit(): void {
    const request = this.newRequest.trim();
    if (request.length < 10) return;

    this.error.set('');
    this.submitting.set(true);

    this.runs.create(request).subscribe({
      next: (run) => {
        this.submitting.set(false);
        this.newRequest = '';
        // Langsung ke halaman detail: di situ log-nya mengalir realtime.
        this.router.navigate(['/runs', run.id]);
      },
      error: (err: Error) => {
        this.submitting.set(false);
        this.error.set(err.message);
      },
    });
  }

  protected isTerminal = isTerminal;
}
