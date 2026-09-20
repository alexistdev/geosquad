import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-not-found',
  imports: [RouterLink],
  template: `
    <div class="text-center py-5">
      <h1 class="display-1 fw-semibold text-muted opacity-50 mb-0">404</h1>
      <h4 class="mt-3">Halaman tidak ditemukan</h4>
      <p class="text-muted mb-4">Alamat yang kamu buka tidak ada atau sudah dipindahkan.</p>
      <a class="btn btn-primary" [routerLink]="home">
        <i class="ri-home-4-line align-bottom me-1"></i> Kembali
      </a>
    </div>
  `,
})
export class NotFound {
  private readonly auth = inject(AuthService);
  // User yang sudah login dikembalikan ke dashboardnya, tamu ke login.
  protected readonly home = this.auth.isLoggedIn()
    ? this.auth.homeUrlFor(this.auth.user()?.role)
    : '/login';
}
