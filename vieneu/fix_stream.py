"""Patches áp dụng lên VieNeu-TTS khi build image.

1. web_stream.py: dùng infer() (non-streaming) thay infer_stream().
   Nhánh streaming GGUF ghép audio từ các cửa sổ chồng lấp
   (lookback/lookforward) -> đôi khi nhân đôi đoạn audio, nghe thành
   LẶP CHỮ. infer() sinh 1 lần rồi decode nên không có lỗi ghép này.
   Kèm hạ temperature=0.7, top_k=30 để giảm lấy mẫu chệch vào vòng lặp.
   (Dùng regex vì upstream hay đổi khoảng trắng khiến khớp nguyên khối hỏng.)

2. standard.py: bật use_chat_format cho mọi model VieNeu-TTS.
   Heuristic gốc chỉ bật cho repo v1 'pnnbao-ump/VieNeu-TTS', khiến
   các bản fine-tune 0.3B (vd ngoc-huyen) đọc cả câu reference
   ("...tính chiến đấu, tính định hướng") và lặp lại.

3. web_stream.py: gộp thêm preset voices từ repo model gốc
   (pnnbao-ump/VieNeu-TTS-0.3B-q4-gguf — 6 giọng nam/nữ Bắc/Nam)
   vào model đang chạy. Bản fine-tune ngoc-huyen chỉ có 1 giọng
   trong voices.json; voice codes dùng chung codec nên nạp lẫn được.

4. standard.py: thêm repeat_penalty=1.3 vào lệnh sinh GGUF
   (_infer_ggml + _infer_stream_ggml). Model tự hồi quy đôi khi kẹt
   vòng lặp -> lặp cụm từ; lệnh sinh gốc không truyền repeat_penalty.
"""

import re

# ── Patch 1: web_stream.py — infer_stream -> infer (regex) ────
p1 = '/app/apps/web_stream.py'
c1 = open(p1).read()

# Thay cả khối từ "start = time.time()" tới hết except bằng 1 lần infer().
pat1 = re.compile(r"        start = time\.time\(\).*?        except Exception as e:", re.S)
new1 = (
    "        try:\n"
    "            wav = tts.infer(text, voice=voice_data, temperature=0.7, top_k=30)\n"
    "            yield float32_to_pcm16(wav)\n"
    "        except Exception as e:"
)

c1_new, n1 = pat1.subn(new1, c1)
if n1:
    open(p1, 'w').write(c1_new)
    print(f'Patched web_stream.py: infer_stream -> infer ({n1} site)')
else:
    print('WARN: web_stream.py infer_stream pattern not found')

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

# ── Patch 4: standard.py — chống lặp chữ khi sinh GGUF ────────
# Model tự hồi quy đôi khi kẹt vòng lặp -> lặp lại cụm từ.
# Lệnh sinh gốc không truyền repeat_penalty (mức mặc định quá nhẹ).
# Thêm repeat_penalty vào cả _infer_ggml và _infer_stream_ggml.
c2b = open(p2).read()

old4 = 'temperature=temperature, top_k=top_k, stop=["<|SPEECH_GENERATION_END|>"]'
new4 = 'temperature=temperature, top_k=top_k, repeat_penalty=1.3, stop=["<|SPEECH_GENERATION_END|>"]'

if old4 in c2b:
    open(p2, 'w').write(c2b.replace(old4, new4))
    print('Patched standard.py: added repeat_penalty=1.3 to GGUF generation')
else:
    print('WARN: standard.py repeat_penalty pattern not found')
