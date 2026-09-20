export const environment = {
  production: false,
  // Dev server memakai proxy (lihat proxy.conf.json), jadi cukup path relatif.
  // Dengan begitu request satu origin dan cookie SID terkirim tanpa urusan CORS.
  apiUrl: '/api/v1',
};
