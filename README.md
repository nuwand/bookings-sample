# bookings-sample

Mock server for the Property Bookings API defined in `openapi.yaml`.

## Run locally

```
cd src
go run .
```

The server listens on `:8080` by default (override with `PORT`). Seed data contains a couple of bookings so `GET /bookings` works immediately.
