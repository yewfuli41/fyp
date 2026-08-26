# frontend build
FROM node:25-bookworm AS frontend
WORKDIR /app/client

COPY client/package.json client/package-lock.json ./
RUN npm ci

COPY client ./
RUN npm run build

# backend build
FROM golang:1.25.6-alpine AS backend
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o fyp-fuli-booking .

# runtime image
FROM alpine:latest
WORKDIR /app

COPY --from=backend /app/fyp-fuli-booking .
COPY --from=backend /app/config ./config
COPY --from=frontend /app/client/dist ./client

EXPOSE 8080

CMD ["./fyp-fuli-booking", "server"]