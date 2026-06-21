FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl cmake build-essential g++ \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

RUN git clone --depth=1 https://github.com/pnnbao97/VieNeu-TTS.git .

RUN sed -i 's/host="127\.0\.0\.1"/host="0.0.0.0"/g' apps/web_stream.py
RUN sed -i 's/VieNeu-TTS-v2-Q4-K-M\.gguf/VieNeu-TTS-0.3B-ngoc-huyen-Q4_0.gguf/g' src/vieneu/standard.py
COPY fix_stream.py /tmp/fix_stream.py
RUN python3 /tmp/fix_stream.py

RUN uv sync

RUN uv pip install trafilatura llama-cpp-python

EXPOSE 8001

CMD ["uv", "run", "vieneu-stream"]
