import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink],
  templateUrl: './register.html',
})
export class Register {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected fullName = '';
  protected email = '';
  protected password = '';

  protected readonly loading = signal(false);
  protected readonly error = signal('');

  protected submit(): void {
    this.error.set('');
    this.loading.set(true);

    const credentials = { email: this.email.trim(), password: this.password };

    this.auth
      .register({ fullName: this.fullName.trim(), ...credentials })
      .subscribe({
        next: () => {
          // Registrasi tidak membuat sesi, jadi user langsung dilogin-kan
          // supaya tidak perlu mengetik ulang apa yang baru saja diketik.
          this.auth.login(credentials).subscribe({
            next: (res) => {
              this.loading.set(false);
              this.router.navigateByUrl(res.defaultHomeUrl);
            },
            error: () => {
              // Akunnya sudah jadi, hanya auto-login yang gagal. Arahkan ke
              // login daripada membuat user mengira pendaftarannya batal.
              this.loading.set(false);
              this.router.navigate(['/login']);
            },
          });
        },
        error: (err: Error) => {
          this.loading.set(false);
          this.error.set(err.message);
        },
      });
  }
}
