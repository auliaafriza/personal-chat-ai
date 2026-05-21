// FE tipis — semua backend logic ada di Go service (lihat ../backend/).
// FE cuma butuh tahu URL backend lewat NEXT_PUBLIC_API_BASE_URL.

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "**.googleusercontent.com" },
      { protocol: "https", hostname: "avatars.githubusercontent.com" },
    ],
  },
}

export default nextConfig
