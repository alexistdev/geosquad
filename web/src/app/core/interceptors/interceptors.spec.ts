import { TestBed } from '@angular/core/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';

import { credentialsInterceptor } from './credentials.interceptor';
import { errorInterceptor } from './error.interceptor';

describe('credentialsInterceptor', () => {
  it('menyalakan withCredentials supaya cookie SID ikut terkirim', () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([credentialsInterceptor])),
        provideHttpClientTesting(),
      ],
    });

    const http = TestBed.inject(HttpClient);
    const ctrl = TestBed.inject(HttpTestingController);

    http.get('/api/v1/auth/me').subscribe();
    const req = ctrl.expectOne('/api/v1/auth/me');

    expect(req.request.withCredentials).toBeTrue();
    req.flush({ status: true, messages: [] });
    ctrl.verify();
  });
});

describe('errorInterceptor', () => {
  let http: HttpClient;
  let ctrl: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideRouter([]),
      ],
    });
    http = TestBed.inject(HttpClient);
    ctrl = TestBed.inject(HttpTestingController);
  });

  afterEach(() => ctrl.verify());

  // API selalu membalas {status, messages, payload}. Pesan dari server itu
  // yang harus sampai ke user, bukan "Http failure response for ...".
  it('memakai pesan dari body API', (done) => {
    http.post('/api/v1/auth/login', {}).subscribe({
      error: (err: Error) => {
        expect(err.message).toBe('Email atau password salah.');
        done();
      },
    });

    ctrl.expectOne('/api/v1/auth/login').flush(
      { status: false, messages: ['Email atau password salah.'] },
      { status: 401, statusText: 'Unauthorized' },
    );
  });

  it('menggabungkan beberapa pesan validasi', (done) => {
    http.post('/api/v1/auth/register', {}).subscribe({
      error: (err: Error) => {
        expect(err.message).toContain('Email wajib diisi.');
        expect(err.message).toContain('Password minimal 8 karakter.');
        done();
      },
    });

    ctrl.expectOne('/api/v1/auth/register').flush(
      { status: false, messages: ['Email wajib diisi.', 'Password minimal 8 karakter.'] },
      { status: 400, statusText: 'Bad Request' },
    );
  });

  // Status 0 berarti request tidak pernah sampai. Pesan HTTP bawaan untuk
  // kasus ini tidak membantu siapa pun.
  it('menjelaskan saat server tidak bisa dihubungi', (done) => {
    http.get('/api/v1/runs').subscribe({
      error: (err: Error) => {
        expect(err.message).toContain('Tidak bisa menghubungi server');
        done();
      },
    });

    ctrl.expectOne('/api/v1/runs').error(new ProgressEvent('error'), { status: 0 });
  });
});
