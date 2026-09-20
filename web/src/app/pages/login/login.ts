import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';

import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-login',
  imports: [FormsModule, RouterLink],
  templateUrl: './login.html',
})
export class Login {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected email = '';
  protected password = '';

  protected readonly loading = signal(false);
  protected readonly error = signal('');
  /** Ditandai interceptor saat sesi mati di tengah pemakaian. */
  protected readonly sessionExpired = signal(
    this.route.snapshot.queryParamMap.get('expired') === '1',
  );

  protected submit(): void {
    this.error.set('');
    this.sessionExpired.set(false);
    this.loading.set(true);

    this.auth.login({ email: this.email.trim(), password: this.password }).subscribe({
      next: (res) => {
        this.loading.set(false);
        // Kembali ke halaman yang tadi diminta, kalau ada. Kalau tidak,
        // ikuti tujuan bawaan yang ditentukan server sesuai peran.
        const redirect = this.route.snapshot.queryParamMap.get('redirect');
        this.router.navigateByUrl(redirect || res.defaultHomeUrl);
      },
      error: (err: Error) => {
        this.loading.set(false);
        this.error.set(err.message);
      },
    });
  }
}
