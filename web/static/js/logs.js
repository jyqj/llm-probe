// ============ Logs ============
let logsOffset = 0;
const logsLimit = 30;

async function loadLogs() {
  try {
    const data = await api(`/api/logs?limit=${logsLimit}&offset=${logsOffset}`);

    document.getElementById('logs-total').textContent = data.total + ' logs';
    document.getElementById('logs-prev').disabled = logsOffset === 0;
    document.getElementById('logs-next').disabled = logsOffset + logsLimit >= data.total;

    const tbody = document.getElementById('logs-body');
    const logs = data.logs || [];
    if (logs.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:var(--text-dim)">No logs yet</td></tr>';
      return;
    }
    // latency_ms comes as Go time.Duration in nanoseconds via JSON
    // time.Duration.Milliseconds() is used in logger, but JSON serializes the raw ns value
    tbody.innerHTML = logs.map(log => {
      const statusClass = log.status_code < 400 ? 'badge-green' : 'badge-red';
      const time = new Date(log.timestamp).toLocaleTimeString();
      // latency_ms is serialized as nanoseconds (Go time.Duration), convert to ms
      const latencyMs = Math.round(log.latency_ms / 1000000);
      return `<tr>
        <td>${time}</td>
        <td><span class="badge ${statusClass}">${log.status_code}</span></td>
        <td>${esc(log.bedrock_model || log.model || '-')}</td>
        <td>${esc(log.route || log.path)}</td>
        <td>${log.stream ? '<span class="badge badge-accent">SSE</span>' : 'sync'}</td>
        <td>${latencyMs}ms</td>
        <td>${log.input_tokens}/${log.output_tokens}</td>
        <td>${log.sanitized ? '<span class="badge badge-yellow">yes</span>' : '-'}</td>
      </tr>`;
    }).join('');
  } catch(e) {
    console.error('loadLogs error:', e);
  }
}

function logsPage(dir) {
  logsOffset = Math.max(0, logsOffset + dir * logsLimit);
  loadLogs();
}
