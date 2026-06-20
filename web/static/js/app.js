const dropZone   = document.getElementById('dropZone');
const fileInput  = document.getElementById('fileInput');
const fileNameEl = document.getElementById('fileName');
const submitBtn  = document.getElementById('submitBtn');
const progressEl = document.getElementById('progress');
const progressFill = document.getElementById('progressFill');
const statusText = document.getElementById('statusText');
const resultEl   = document.getElementById('result');
const downloadLink = document.getElementById('downloadLink');
const jobBody       = document.getElementById('jobBody');
const deleteAllBtn  = document.getElementById('deleteAllBtn');

let selectedFile = null;

dropZone.addEventListener('click', () => fileInput.click());
dropZone.addEventListener('dragover', e => { e.preventDefault(); dropZone.classList.add('drag-over'); });
dropZone.addEventListener('dragleave',  () => dropZone.classList.remove('drag-over'));
dropZone.addEventListener('drop', e => {
  e.preventDefault();
  dropZone.classList.remove('drag-over');
  const f = e.dataTransfer.files[0];
  if (f) setFile(f);
});
fileInput.addEventListener('change', () => { if (fileInput.files[0]) setFile(fileInput.files[0]); });

function setFile(f) {
  if (!f.name.toLowerCase().endsWith('.mp4')) {
    alert('Chỉ hỗ trợ file .mp4');
    return;
  }
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

  // Poll for completion
  const interval = setInterval(async () => {
    try {
      const res = await fetch('/api/jobs/' + jobId);
      const job = await res.json();
      if (job.status === 'done') {
        clearInterval(interval);
        progressFill.style.width = '100%';
        statusText.textContent = 'Hoàn thành!';
        setTimeout(() => {
          progressEl.classList.add('hidden');
          resultEl.classList.remove('hidden');
          downloadLink.href = '/api/download/' + jobId;
          submitBtn.disabled = false;
          loadHistory();
        }, 600);
      } else if (job.status === 'failed') {
        clearInterval(interval);
        statusText.textContent = '❌ Lỗi: ' + (job.error_msg || 'Không xác định');
        submitBtn.disabled = false;
      } else {
        // animate progress 40→90
        const cur = parseFloat(progressFill.style.width) || 40;
        if (cur < 90) progressFill.style.width = (cur + 3) + '%';
      }
    } catch {}
  }, 1500);
});

async function loadHistory() {
  try {
    const res = await fetch('/api/jobs');
    const jobs = await res.json();
    jobBody.innerHTML = '';
    (jobs || []).forEach(j => {
      const tr = document.createElement('tr');
      const fileName = j.input_file.split('/').pop().replace(/^\d+_/, '');
      const date = new Date(j.created_at).toLocaleString('vi-VN');
      const dl = j.status === 'done'
        ? `<a class="dl-link" href="/api/download/${j.id}">⬇ Tải về</a>`
        : '—';
      tr.innerHTML = `
        <td>${j.id}</td>
        <td title="${j.input_file}">${fileName}</td>
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
  const id = btn.dataset.id;
  await fetch('/api/jobs/' + id, { method: 'DELETE' });
  loadHistory();
});

deleteAllBtn.addEventListener('click', async () => {
  if (!confirm('Xóa toàn bộ lịch sử và reset ID?')) return;
  await fetch('/api/jobs', { method: 'DELETE' });
  loadHistory();
});

function labelStatus(s) {
  return { pending: 'Chờ', processing: 'Đang xử lý', done: 'Hoàn thành', failed: 'Lỗi' }[s] || s;
}

loadHistory();
setInterval(loadHistory, 8000);
