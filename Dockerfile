# Frontend build stage
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY . .
WORKDIR /app/ui
RUN corepack enable && corepack prepare yarn@4.7.0 --activate
RUN yarn install --frozen-lockfile
RUN yarn build

# Go build stage
FROM golang:1.24-alpine AS backend-builder
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
