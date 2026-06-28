// ── Tab switching ──────────────────────────────────────────────
document.querySelectorAll('nav a[data-tab]').forEach(link => {
  link.addEventListener('click', e => {
    e.preventDefault();
    const tab = link.dataset.tab;
    document.querySelectorAll('nav a[data-tab]').forEach(a => a.classList.remove('active'));
    link.classList.add('active');
    ['mp4', 'tts', 'capture'].forEach(t => {
      document.getElementById('tab-' + t).classList.toggle('hidden', t !== tab);
    });
    if (tab === 'tts') loadVoices();
  });
});

// ── Tab con trong "Quay màn hình" (Chụp ảnh / Quay video) ──────
document.querySelectorAll('.subtab').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.subtab').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    const sub = btn.dataset.sub;
    document.getElementById('sub-shot').classList.toggle('hidden', sub !== 'shot');
    document.getElementById('sub-vid').classList.toggle('hidden', sub !== 'vid');
  });
});

// ── MP4 → MP3 ─────────────────────────────────────────────────
const dropZone    = document.getElementById('dropZone');
const fileInput   = document.getElementById('fileInput');
const fileNameEl  = document.getElementById('fileName');
const submitBtn   = document.getElementById('submitBtn');
const progressEl  = document.getElementById('progress');
const progressFill = document.getElementById('progressFill');
const statusText  = document.getElementById('statusText');
const resultEl    = document.getElementById('result');
const downloadLink = document.getElementById('downloadLink');

let selectedFile = null;

dropZone.addEventListener('click', () => fileInput.click());
dropZone.addEventListener('dragover', e => { e.preventDefault(); dropZone.classList.add('drag-over'); });
dropZone.addEventListener('dragleave', () => dropZone.classList.remove('drag-over'));
dropZone.addEventListener('drop', e => {
  e.preventDefault();
  dropZone.classList.remove('drag-over');
  const f = e.dataTransfer.files[0];
  if (f) setFile(f);
});
fileInput.addEventListener('change', () => { if (fileInput.files[0]) setFile(fileInput.files[0]); });

function setFile(f) {
  if (!f.name.toLowerCase().endsWith('.mp4')) { alert('Chỉ hỗ trợ file .mp4'); return; }
  selectedFile = f;
  fileNameEl.textContent = f.name + ' (' + (f.size / 1024 / 1024).toFixed(2) + ' MB)';
  submitBtn.disabled = false;
}

document.getElementById('uploadForm').addEventListener('submit', async e => {
  e.preventDefault();
  if (!selectedFile) return;

  submitBtn.disabled = true;
  progressEl.classList.remove('hidden');
  resultEl.classList.add('hidden');
  progressFill.style.width = '20%';
  statusText.textContent = 'Đang tải file lên…';

  const fd = new FormData();
  fd.append('file', selectedFile);

  let jobId;
  try {
    const res = await fetch('/api/convert/mp4-to-mp3', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Upload thất bại');
    jobId = data.job_id;
    progressFill.style.width = '40%';
    statusText.textContent = 'Đang chuyển đổi… (job #' + jobId + ')';
  } catch (err) {
    statusText.textContent = '❌ ' + err.message;
    submitBtn.disabled = false;
    return;
  }

  pollJob(jobId, progressFill, statusText, progressEl, resultEl, downloadLink, submitBtn);
});

// ── TTS ───────────────────────────────────────────────────────
const ttsText        = document.getElementById('ttsText');
const voiceSelect    = document.getElementById('voiceSelect');
const ttsSubmitBtn   = document.getElementById('ttsSubmitBtn');
const ttsProgressEl  = document.getElementById('ttsProgress');
const ttsProgressFill = document.getElementById('ttsProgressFill');
const ttsStatusText  = document.getElementById('ttsStatusText');
const ttsResultEl    = document.getElementById('ttsResult');
const ttsDownloadLink = document.getElementById('ttsDownloadLink');
const charCount      = document.getElementById('charCount');

ttsText.addEventListener('input', () => {
  const len = ttsText.value.length;
  charCount.textContent = len + ' / 3000';
  charCount.style.color = len > 2800 ? 'var(--accent2)' : '';
});

async function loadVoices() {
  voiceSelect.innerHTML = '<option value="">Đang tải…</option>';
  ttsSubmitBtn.disabled = true;
  try {
    const res = await fetch('/api/voices');
    if (!res.ok) throw new Error();
    const voices = await res.json();
    if (!Array.isArray(voices) || voices.length === 0) throw new Error();
    voiceSelect.innerHTML = voices.map(v =>
      `<option value="${v.id}">${v.name}</option>`
    ).join('');
    ttsSubmitBtn.disabled = ttsText.value.trim() === '';
  } catch {
    voiceSelect.innerHTML = '<option value="">VieNeu chưa sẵn sàng — thử lại sau</option>';
  }
}

ttsText.addEventListener('input', () => {
  ttsSubmitBtn.disabled = ttsText.value.trim() === '' || voiceSelect.options[0]?.value === '';
});

document.getElementById('ttsForm').addEventListener('submit', async e => {
  e.preventDefault();
  const text = ttsText.value.trim();
  if (!text) return;

  ttsSubmitBtn.disabled = true;
  ttsProgressEl.classList.remove('hidden');
  ttsResultEl.classList.add('hidden');
  ttsProgressFill.style.width = '20%';
  ttsStatusText.textContent = 'Đang gửi yêu cầu…';

  const fd = new FormData();
  fd.append('text', text);
  fd.append('voice_id', voiceSelect.value);

  let jobId;
  try {
    const res = await fetch('/api/tts', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Lỗi gửi yêu cầu');
    jobId = data.job_id;
    ttsProgressFill.style.width = '40%';
    ttsStatusText.textContent = 'Đang tạo giọng đọc… (job #' + jobId + ')';
  } catch (err) {
    ttsStatusText.textContent = '❌ ' + err.message;
    ttsSubmitBtn.disabled = false;
    return;
  }

  pollJob(jobId, ttsProgressFill, ttsStatusText, ttsProgressEl, ttsResultEl, ttsDownloadLink, ttsSubmitBtn);
});

// ── Chụp ảnh / Quay video (macOS native app) ──────────────────
function launchCapture(mode, paramObj, hintEl) {
  // Số job hiện tại để phát hiện job mới xuất hiện sau khi capture.
  fetch('/api/jobs').then(r => r.json()).then(jobs => {
    const before = (jobs || []).length;

    const params = new URLSearchParams({ mode, ...paramObj });
    window.location.href = 'gostudio://capture?' + params.toString();

    hintEl.classList.remove('hidden');
    hintEl.textContent = '📂 Đã mở app GoStudio Capture — bấm "Bắt đầu" trong app để ' +
      (mode === 'video' ? 'quay.' : 'chụp.') + ' File sẽ tự về lịch sử bên dưới.';

    // Theo dõi tới 3 phút (đủ thời gian cấp quyền + quay). Hết giờ chỉ nhắc nhẹ.
    let waited = 0;
    const iv = setInterval(async () => {
      waited += 2;
      try {
        const now = await (await fetch('/api/jobs')).json();
        if ((now || []).length > before) {
          clearInterval(iv);
          hintEl.textContent = '✅ Đã nhận file, xem trong lịch sử bên dưới.';
          loadHistory();
          return;
        }
      } catch {}
      if (waited >= 180) {
        clearInterval(iv);
        hintEl.innerHTML = 'ℹ️ Chưa nhận được file. Nếu app không mở, kiểm tra đã cài <strong>GoStudio Capture</strong> chưa.';
      }
    }, 2000);
  });
}

// Tab Chụp ảnh: phạm vi + ẩn con trỏ.
document.getElementById('shotBtn').addEventListener('click', () => {
  const p = { region: document.getElementById('shotRegion').value };
  if (document.getElementById('shotHideCursor').checked) p.cursor = 'hide';
  launchCapture('screenshot', p, document.getElementById('shotHint'));
});

// Tab Quay video: phạm vi + âm thanh (có đếm ngược trong app).
document.getElementById('vidBtn').addEventListener('click', () => {
  const p = {
    region: document.getElementById('vidRegion').value,
    audio: document.getElementById('vidAudio').value,
  };
  launchCapture('video', p, document.getElementById('vidHint'));
});

// ── Shared poll ───────────────────────────────────────────────
function pollJob(jobId, fill, statusEl, progressContainer, resultContainer, dlLink, btn) {
  const interval = setInterval(async () => {
    try {
      const res = await fetch('/api/jobs/' + jobId);
      const job = await res.json();
      if (job.status === 'done') {
        clearInterval(interval);
        fill.style.width = '100%';
        statusEl.textContent = 'Hoàn thành!';
        setTimeout(() => {
          progressContainer.classList.add('hidden');
          resultContainer.classList.remove('hidden');
          dlLink.href = '/api/download/' + jobId;
          btn.disabled = false;
          loadHistory();
        }, 600);
      } else if (job.status === 'failed') {
        clearInterval(interval);
        statusEl.textContent = '❌ Lỗi: ' + (job.error_msg || 'Không xác định');
        btn.disabled = false;
      } else {
        const cur = parseFloat(fill.style.width) || 40;
        if (cur < 90) fill.style.width = (cur + 2) + '%';
      }
    } catch {}
  }, 1500);
}

// ── History ───────────────────────────────────────────────────
const jobBody      = document.getElementById('jobBody');
const deleteAllBtn = document.getElementById('deleteAllBtn');

async function loadHistory() {
  try {
    const res = await fetch('/api/jobs');
    const jobs = await res.json();
    jobBody.innerHTML = '';
    (jobs || []).forEach(j => {
      const tr = document.createElement('tr');
      const meta = jobMeta(j);
      const displayName = meta.name;
      const typeLabel = meta.badge;
      const date = new Date(j.created_at).toLocaleString('vi-VN');
      const dl = j.status === 'done'
        ? `<a class="dl-link" href="/api/download/${j.id}">⬇ Tải về</a>`
        : '—';
      tr.innerHTML = `
        <td>${j.id}</td>
        <td title="${j.input_file}">${displayName}</td>
        <td>${typeLabel}</td>
        <td><span class="badge badge-${j.status}">${labelStatus(j.status)}</span></td>
        <td>${date}</td>
        <td>${dl}</td>
        <td><button class="btn-delete-row" title="Xóa" data-id="${j.id}">✕</button></td>
      `;
      jobBody.appendChild(tr);
    });
  } catch {}
}

jobBody.addEventListener('click', async e => {
  const btn = e.target.closest('.btn-delete-row');
  if (!btn) return;
  await fetch('/api/jobs/' + btn.dataset.id, { method: 'DELETE' });
  loadHistory();
});

deleteAllBtn.addEventListener('click', async () => {
  if (!confirm('Xóa toàn bộ lịch sử và reset ID?')) return;
  await fetch('/api/jobs', { method: 'DELETE' });
  loadHistory();
});

// Tên hiển thị + badge loại job tùy theo type.
function jobMeta(j) {
  switch (j.type) {
    case 'text_to_speech':
      return { name: j.input_file.replace('[TTS] ', ''), badge: '<span class="badge badge-tts">TTS</span>' };
    case 'screenshot':
      return { name: j.input_file.replace('[Ảnh] ', ''), badge: '<span class="badge badge-photo">Ảnh</span>' };
    case 'screen_record':
      return { name: j.input_file.replace('[Quay] ', ''), badge: '<span class="badge badge-screen">Quay</span>' };
    default:
      return { name: j.input_file.split('/').pop().replace(/^\d+_/, ''), badge: '<span class="badge badge-mp4">MP4</span>' };
  }
}

function labelStatus(s) {
  return { pending: 'Chờ', processing: 'Đang xử lý', done: 'Hoàn thành', failed: 'Lỗi' }[s] || s;
}

loadHistory();
setInterval(loadHistory, 8000);
