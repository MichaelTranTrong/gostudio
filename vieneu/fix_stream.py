"""Patches áp dụng lên VieNeu-TTS khi build image.

1. web_stream.py: dùng infer() (non-streaming) thay infer_stream().
   Bản streaming GGUF bị lỗi ghép nhầm reference audio vào output.

2. standard.py: bật use_chat_format cho mọi model VieNeu-TTS.
   Heuristic gốc chỉ bật cho repo v1 'pnnbao-ump/VieNeu-TTS', khiến
   các bản fine-tune 0.3B (vd ngoc-huyen) đọc cả câu reference
   ("...tính chiến đấu, tính định hướng") và lặp lại.

3. web_stream.py: gộp thêm preset voices từ repo model gốc
   (pnnbao-ump/VieNeu-TTS-0.3B-q4-gguf — 6 giọng nam/nữ Bắc/Nam)
   vào model đang chạy. Bản fine-tune ngoc-huyen chỉ có 1 giọng
   trong voices.json; voice codes dùng chung codec nên nạp lẫn được.
"""

# ── Patch 1: web_stream.py ────────────────────────────────────
p1 = '/app/apps/web_stream.py'
c1 = open(p1).read()

old1 = '''    def audio_generator():
        header = io.BytesIO()
        with wave.open(header, 'wb') as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(24000)
            wav_file.setnframes(100_000_000)
        yield header.getvalue()

        start = time.time()
        count = 0
        try:
            for chunk in tts.infer_stream(text, voice=voice_data):
                if count == 0:
                     print(f"⚡ First sound in {time.time() - start:.3f}s")
                count += 1
                yield float32_to_pcm16(chunk)
                time.sleep(0.001)

        except Exception as e:
            print(f"Error during inference: {e}")'''

new1 = '''    def audio_generator():
        header = io.BytesIO()
        with wave.open(header, 'wb') as wav_file:
            wav_file.setnchannels(1)
            wav_file.setsampwidth(2)
            wav_file.setframerate(24000)
            wav_file.setnframes(100_000_000)
        yield header.getvalue()

        try:
            wav = tts.infer(text, voice=voice_data)
            yield float32_to_pcm16(wav)
        except Exception as e:
            print(f"Error during inference: {e}")'''

if old1 in c1:
    open(p1, 'w').write(c1.replace(old1, new1))
    print('Patched web_stream.py: infer_stream -> infer')
else:
    print('WARN: web_stream.py pattern not found')

# ── Patch 2: standard.py ──────────────────────────────────────
p2 = '/app/src/vieneu/standard.py'
c2 = open(p2).read()

old2 = 'self.use_chat_format = backbone_repo.rstrip("/").endswith("pnnbao-ump/VieNeu-TTS")'
new2 = 'self.use_chat_format = "VieNeu-TTS" in backbone_repo'

if old2 in c2:
    open(p2, 'w').write(c2.replace(old2, new2))
    print('Patched standard.py: use_chat_format heuristic')
else:
    print('WARN: standard.py pattern not found')

# ── Patch 3: web_stream.py — gộp thêm giọng từ repo gốc ───────
c1 = open(p1).read()

old3 = '''        tts = new_tts
        current_model_id = model_key
        print(f"✅ Model Loaded Successfully: {repo_id}")'''

new3 = '''        tts = new_tts
        current_model_id = model_key
        print(f"✅ Model Loaded Successfully: {repo_id}")
        extra_repo = "pnnbao-ump/VieNeu-TTS-0.3B-q4-gguf"
        if repo_id != extra_repo:
            try:
                keep_default = tts._default_voice
                tts._load_voices_from_repo(extra_repo)
                tts._default_voice = keep_default
                print(f"📢 Merged extra preset voices from {extra_repo}")
            except Exception as ve:
                print(f"⚠️ Could not merge extra voices: {ve}")'''

if old3 in c1:
    open(p1, 'w').write(c1.replace(old3, new3))
    print('Patched web_stream.py: merge extra preset voices')
else:
    print('WARN: web_stream.py merge-voices pattern not found')
