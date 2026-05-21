# Kageos Web Next

Next.js App Router frontend experiment for the Hub marketplace and future refreshed workspace UI.

## Stack

- Next.js App Router
- TypeScript
- Tailwind CSS
- shadcn-style local UI components

## Run

Start Hub first:

```bash
go run ../hub/cmd/hub -catalog ../hub/catalog.example.json -addr :8090 -allow-origin http://localhost:4000
```

Then run this app:

```bash
npm run dev
```

Open `http://localhost:4000`.

## Environment

Copy `.env.example` to `.env.local` if you need custom endpoints.

`NEXT_PUBLIC_HUB_API_URL` points to the closed-source Hub service.

`NEXT_PUBLIC_APP_SERVER_API_URL` points to the existing app-server. If it is not set, the install panel still downloads the bundle and stops before calling the open-source installer.
