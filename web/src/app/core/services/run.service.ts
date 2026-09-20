import { Injectable, NgZone, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';

import { environment } from '../../../environments/environment';
import { BaseResponse, Paged } from '../models/response';
import { Run, RunEvent, RunLog, RunStatus } from '../models/run';

@Injectable({ providedIn: 'root' })
export class RunService {
  private readonly http = inject(HttpClient);
  private readonly zone = inject(NgZone);
  private readonly base = `${environment.apiUrl}/runs`;

  list(opts: { page?: number; perPage?: number; status?: RunStatus | '' } = {}): Observable<Paged<Run>> {
    let params = new HttpParams()
      .set('page', String(opts.page ?? 1))
      .set('perPage', String(opts.perPage ?? 20));
    if (opts.status) {
      params = params.set('status', opts.status);
    }

    return this.http
      .get<BaseResponse<Paged<Run>>>(this.base, { params })
      .pipe(map((res) => res.payload ?? { items: [], meta: { page: 1, perPage: 20, total: 0, totalPages: 0 } }));
  }

  get(id: string): Observable<Run> {
    return this.http
      .get<BaseResponse<Run>>(`${this.base}/${id}`)
      .pipe(map((res) => res.payload!));
  }

  create(request: string): Observable<Run> {
    return this.http
      .post<BaseResponse<Run>>(this.base, { request })
      .pipe(map((res) => res.payload!));
  }

  logs(id: string, afterSeq = 0, limit = 500): Observable<RunLog[]> {
    const params = new HttpParams()
      .set('afterSeq', String(afterSeq))
      .set('limit', String(limit));

    return this.http
      .get<BaseResponse<RunLog[]>>(`${this.base}/${id}/logs`, { params })
      .pipe(map((res) => res.payload ?? []));
  }

  cancel(id: string): Observable<void> {
    return this.http
      .post<BaseResponse<void>>(`${this.base}/${id}/cancel`, {})
      .pipe(map(() => void 0));
  }

  delete(id: string): Observable<void> {
    return this.http
      .delete<BaseResponse<void>>(`${this.base}/${id}`)
      .pipe(map(() => void 0));
  }

  /**
   * Mendengarkan log realtime lewat Server-Sent Events.
   *
   * Tidak memakai HttpClient: Angular menunggu response selesai sebelum
   * memberikan hasilnya, sedangkan SSE justru response yang sengaja tidak
   * pernah selesai. EventSource bawaan browser menangani ini, termasuk
   * menyambung ulang sendiri saat koneksi putus.
   *
   * `afterSeq` membuat penyambungan ulang tidak mengulang baris yang sudah
   * tampil di layar.
   */
  stream(id: string, afterSeq = 0): Observable<RunEvent> {
    return new Observable<RunEvent>((subscriber) => {
      const url = `${this.base}/${id}/stream?afterSeq=${afterSeq}`;
      // withCredentials wajib supaya cookie SID ikut terkirim.
      const source = new EventSource(url, { withCredentials: true });

      const emit = (event: MessageEvent) => {
        try {
          const parsed = JSON.parse(event.data) as RunEvent;
          // EventSource memanggil balik di luar zona Angular, jadi perubahan
          // yang dipicunya tidak akan terlihat kalau tidak dikembalikan.
          this.zone.run(() => subscriber.next(parsed));
        } catch {
          // Baris yang tidak bisa diparse dilewati, bukan menjatuhkan aliran.
        }
      };

      source.addEventListener('log', emit);
      source.addEventListener('done', (event) => {
        emit(event as MessageEvent);
        this.zone.run(() => subscriber.complete());
      });

      source.onerror = () => {
        // EventSource menyambung ulang sendiri selama state-nya bukan CLOSED.
        // Yang benar-benar tertutup baru dilaporkan sebagai error.
        if (source.readyState === EventSource.CLOSED) {
          this.zone.run(() => subscriber.error(new Error('Koneksi log terputus.')));
        }
      };

      return () => source.close();
    });
  }
}
