FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl cmake build-essential g++ \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

# Ghim commit upstream (2026-07-06). KHÔNG dùng --depth=1 / HEAD:
# upstream đã viết lại lớn (V3TurboVieNeuTTS, đổi tên giọng, 48kHz) làm hỏng
# tích hợp + các patch trong fix_stream.py. Đây là bản mọi patch khớp.
RUN git clone https://github.com/pnnbao97/VieNeu-TTS.git . \
    && git checkout 9526b5ca17878d6fd0ec5327f1107af9cd963f39

RUN sed -i 's/host="127\.0\.0\.1"/host="0.0.0.0"/g' apps/web_stream.py
RUN sed -i 's/VieNeu-TTS-v2-Q4-K-M\.gguf/VieNeu-TTS-0.3B-ngoc-huyen-Q4_0.gguf/g' src/vieneu/standard.py
COPY fix_stream.py /tmp/fix_stream.py
RUN python3 /tmp/fix_stream.py

# --frozen: cài đúng theo uv.lock, KHÔNG resolve lại. uv.lock upstream trỏ tới
# wheel Windows trên GitHub release (llama_cpp_python-0.3.16-...-win_amd64.whl)
# đã bị xóa (404) → `uv sync` thường phải tải metadata wheel đó và fail.
RUN uv sync --frozen

# Ghim 0.3.34 (bản đang chạy ổn định) — không để tự lên bản mới.
RUN uv pip install trafilatura llama-cpp-python==0.3.34

EXPOSE 8001

# --no-sync: `uv run` mặc định sync lại env theo lock MỖI LẦN khởi động →
# dính 404 ở trên → container crash-loop; đồng thời sẽ gỡ llama-cpp-python
# cài thêm ở trên (không có trong lock).
CMD ["uv", "run", "--no-sync", "vieneu-stream"]
