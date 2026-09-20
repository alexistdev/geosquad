import {
  AfterViewChecked,
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { DatePipe } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subscription } from 'rxjs';

import { RunService } from '../../core/services/run.service';
import { Run, RunLog, isTerminal } from '../../core/models/run';
import { StatusBadge } from '../../shared/ui/status-badge';

@Component({
  selector: 'app-run-detail',
  imports: [RouterLink, DatePipe, StatusBadge],
  templateUrl: './run-detail.html',
})
export class RunDetail implements OnInit, OnDestroy, AfterViewChecked {
  private readonly runs = inject(RunService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  private readonly logBox = viewChild<ElementRef<HTMLElement>>('logBox');

  protected readonly run = signal<Run | null>(null);
  protected readonly logs = signal<RunLog[]>([]);
  protected readonly loading = signal(true);
  protected readonly error = signal('');
  protected readonly streaming = signal(false);
  protected readonly cancelling = signal(false);

  /** Menggulung otomatis ke bawah, kecuali user sedang membaca ke atas. */
  protected readonly followTail = signal(true);
  private shouldScroll = false;

  private stream?: Subscription;
  private runId = '';

  ngOnInit(): void {
    this.runId = this.route.snapshot.paramMap.get('id') ?? '';
    this.loadRun();
  }

  ngOnDestroy(): void {
    this.stream?.unsubscribe();
  }

  ngAfterViewChecked(): void {
    if (this.shouldScroll && this.followTail()) {
      const box = this.logBox()?.nativeElement;
      if (box) box.scrollTop = box.scrollHeight;
      this.shouldScroll = false;
    }
  }

  private loadRun(): void {
    this.runs.get(this.runId).subscribe({
      next: (run) => {
        this.run.set(run);
        this.loading.set(false);
        // Run yang sudah selesai cukup dibaca sekali dari database. Hanya yang
        // masih hidup yang perlu koneksi SSE.
        if (isTerminal(run.status)) {
          this.loadAllLogs();
        } else {
          this.connect();
        }
      },
      error: (err: Error) => {
        this.loading.set(false);
        this.error.set(err.message);
      },
    });
  }

  private loadAllLogs(): void {
    this.runs.logs(this.runId, 0, 2000).subscribe({
      next: (logs) => {
        this.logs.set(logs);
        this.shouldScroll = true;
      },
      error: (err: Error) => this.error.set(err.message),
    });
  }

  private connect(): void {
    this.streaming.set(true);
    // afterSeq=0: server mengirim riwayat dulu, baru menyusul yang live.
    // Jadi tidak perlu memuat log terpisah sebelum menyambung.
    this.stream = this.runs.stream(this.runId, 0).subscribe({
      next: (event) => {
        if (event.kind === 'log' && event.seq !== undefined) {
          this.logs.update((current) => [
            ...current,
            {
              seq: event.seq!,
              stream: (event.stream as RunLog['stream']) ?? 'STDOUT',
              line: event.line ?? '',
              loggedAt: new Date().toISOString(),
            },
          ]);
          this.shouldScroll = true;
        }
        if (event.kind === 'done') {
          this.streaming.set(false);
          // Ambil ulang run untuk mendapat exit code, branch, dan waktu selesai.
          this.runs.get(this.runId).subscribe({ next: (run) => this.run.set(run) });
        }
      },
      error: (err: Error) => {
        this.streaming.set(false);
        this.error.set(err.message);
      },
      complete: () => this.streaming.set(false),
    });
  }

  protected onScroll(event: Event): void {
    const box = event.target as HTMLElement;
    // Toleransi 40px supaya pembulatan piksel tidak mematikan auto-scroll.
    const atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
    this.followTail.set(atBottom);
  }

  protected cancel(): void {
    this.cancelling.set(true);
    this.runs.cancel(this.runId).subscribe({
      next: () => this.cancelling.set(false),
      error: (err: Error) => {
        this.cancelling.set(false);
        this.error.set(err.message);
      },
    });
  }

  protected remove(): void {
    if (!confirm('Hapus run ini beserta seluruh lognya?')) return;

    this.runs.delete(this.runId).subscribe({
      next: () => this.router.navigate(['/user/dashboard']),
      error: (err: Error) => this.error.set(err.message),
    });
  }

  protected isTerminal = isTerminal;
}
