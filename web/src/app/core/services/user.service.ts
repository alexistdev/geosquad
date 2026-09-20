import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';

import { environment } from '../../../environments/environment';
import { BaseResponse, Paged } from '../models/response';
import { CreateUserRequest, Role, UpdateUserRequest, User } from '../models/user';

/** Endpoint di bawah /admin. Hanya dipanggil dari halaman yang dijaga roleGuard. */
@Injectable({ providedIn: 'root' })
export class UserService {
  private readonly http = inject(HttpClient);
  private readonly base = `${environment.apiUrl}/admin/users`;

  list(opts: { page?: number; perPage?: number; search?: string; role?: Role | '' } = {}): Observable<Paged<User>> {
    let params = new HttpParams()
      .set('page', String(opts.page ?? 1))
      .set('perPage', String(opts.perPage ?? 20));
    if (opts.search) params = params.set('search', opts.search);
    if (opts.role) params = params.set('role', opts.role);

    return this.http
      .get<BaseResponse<Paged<User>>>(this.base, { params })
      .pipe(map((res) => res.payload ?? { items: [], meta: { page: 1, perPage: 20, total: 0, totalPages: 0 } }));
  }

  create(request: CreateUserRequest): Observable<User> {
    return this.http
      .post<BaseResponse<User>>(this.base, request)
      .pipe(map((res) => res.payload!));
  }

  update(id: string, request: UpdateUserRequest): Observable<User> {
    return this.http
      .patch<BaseResponse<User>>(`${this.base}/${id}`, request)
      .pipe(map((res) => res.payload!));
  }

  delete(id: string): Observable<void> {
    return this.http
      .delete<BaseResponse<void>>(`${this.base}/${id}`)
      .pipe(map(() => void 0));
  }
}
