FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o nhsoauthclose .

FROM alpine:latest
RUN apk update && apk add --no-cache tzdata bash
RUN cp /usr/share/zoneinfo/Asia/Bangkok /etc/localtime
RUN echo "Asia/Bangkok" > /etc/timezone
ENV TZ=Asia/Bangkok

WORKDIR /app
COPY --from=builder /app/nhsoauthclose .
RUN chmod +x /app/nhsoauthclose

# Create logs directory
RUN mkdir -p /app/logs

# Create a script that runs every 10 minutes
RUN echo '#!/bin/bash' > /app/run.sh
RUN echo 'while true; do' >> /app/run.sh
RUN echo '    echo "$(date): Starting application..." >> /app/logs/run.log' >> /app/run.sh
RUN echo '    /app/nhsoauthclose >> /app/logs/app.log 2>&1' >> /app/run.sh
RUN echo '    echo "$(date): Application finished. Sleeping for 10 minutes..." >> /app/logs/run.log' >> /app/run.sh
RUN echo '    sleep 600' >> /app/run.sh
RUN echo 'done' >> /app/run.sh

RUN chmod +x /app/run.sh

CMD ["/bin/bash", "/app/run.sh"]