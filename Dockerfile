# Frontend build stage
FROM node:24-alpine AS frontend-builder
WORKDIR /app
COPY . .
WORKDIR /app/ui
RUN corepack enable && corepack prepare yarn@4.18.1+sha512.b2e1e7524f654f2749d32b4ebcb4622473cb5bcbc485df2007e12a154e50162a4d795526768bc5f5b8f81717bfd79deb2472813d86fb5ae2eb551fa9c872b08f --activate
RUN yarn install --frozen-lockfile
RUN yarn build

# Go build stage
FROM golang:1.27-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
COPY main.go ./
COPY ui/ui.go ./ui/
COPY --from=frontend-builder /app/ui/dist ./ui/dist
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o echohttp

# Final stage
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache sqlite-libs
COPY --from=backend-builder /app/echohttp .
RUN mkdir -p /data
ENV HTTP_PORT=8025
EXPOSE 8025
CMD ["./echohttp"]
