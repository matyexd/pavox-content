FROM golang:1.27.1-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/pavox-content .

FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup --system app \
    && adduser --system --ingroup app app \
    && mkdir -p /app/downloads \
    && chown -R app:app /app

WORKDIR /app

COPY --from=build --chown=app:app /out/pavox-content ./pavox-content

USER app

ENTRYPOINT ["./pavox-content"]
