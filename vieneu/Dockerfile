FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends git curl && rm -rf /var/lib/apt/lists/*

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

RUN git clone --depth=1 https://github.com/pnnbao97/VieNeu-TTS.git .

RUN sed -i 's/host="127\.0\.0\.1"/host="0.0.0.0"/g' apps/web_stream.py

RUN uv sync

EXPOSE 8001

CMD ["uv", "run", "vieneu-stream"]
