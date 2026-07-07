package middleware

import "net/http"

// SecurityHeaders (Minggu 12) — pasang security-related response headers.
//
// Rationale singkat per header:
//   - CSP: batasi asal script/img/dll supaya XSS payload nggak bisa fetch/exec
//     eksternal. Value konservatif; kalau butuh CDN tambahin domain-nya.
//   - HSTS: force browser pakai HTTPS ke domain ini selama 1 tahun. Aman
//     karena FE production selalu HTTPS.
//   - X-Frame-Options DENY: cegah clickjacking (nggak boleh di-iframe).
//   - X-Content-Type-Options nosniff: cegah MIME sniffing attacks.
//   - Referrer-Policy: privacy — jangan leak URL asal ke third-party.
//   - Permissions-Policy: disable APIs yang nggak dipakai (camera, mic, dll).
//
// PENTING: HSTS + CSP di production only. Di dev localhost bisa mengganggu
// (hot reload script inline, dll). Kita apply semua kecuali HSTS terlalu
// ketat — set max-age lebih pendek + preload=false. Toggle via ENV kalau perlu.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Content-Security-Policy — restrictive baseline. FE (Next.js) di domain
		// terpisah; BE cuma serve JSON + streams. Jadi CSP di sini strict.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// HSTS — 1 year, preload OK karena kita expect HTTPS-only production.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Legacy but useful — nggak boleh di-iframe.
		h.Set("X-Frame-Options", "DENY")

		// MIME sniffing block
		h.Set("X-Content-Type-Options", "nosniff")

		// Privacy — jangan leak URL path ke referrer.
		h.Set("Referrer-Policy", "no-referrer")

		// Feature/permissions policy — disable browser APIs yang nggak dipakai.
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		next.ServeHTTP(w, r)
	})
}
