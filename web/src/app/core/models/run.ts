export type RunStatus =
  | 'PENDING'
  | 'RUNNING'
  | 'PASSED'
  | 'FAILED'
  | 'CANCELLED'
  | 'ERROR';

export interface Run {
  id: string;
  /** Nomor tiket yang dilihat manusia (geo-1001), sekaligus nama folder hasil kerja. */
  ticket: string;
  userId: string;
  request: string;
  status: RunStatus;
  branch?: string;
  exitCode?: number;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  createdDate: string;
}

export interface RunLog {
  seq: number;
  stream: 'STDOUT' | 'STDERR' | 'SYSTEM';
  line: string;
  loggedAt: string;
}

/** Event yang dikirim endpoint SSE. */
export interface RunEvent {
  kind: 'log' | 'done';
  seq?: number;
  stream?: string;
  line?: string;
  status?: RunStatus;
}

/** Status yang tidak akan berubah lagi. */
export const TERMINAL_STATUSES: RunStatus[] = ['PASSED', 'FAILED', 'CANCELLED', 'ERROR'];

export function isTerminal(status: RunStatus): boolean {
  return TERMINAL_STATUSES.includes(status);
}
