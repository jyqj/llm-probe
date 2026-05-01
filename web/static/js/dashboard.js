// ============ Dashboard ============
async function loadStats() {
  try {
    const data = await api('/api/stats');

    document.getElementById('stat-total').textContent = fmt(data.total_requests);
    document.getElementById('stat-success').textContent = fmt(data.success_requests);
    document.getElementById('stat-errors').textContent = fmt(data.error_requests);
    document.getElementById('stat-streams').textContent = fmt(data.active_streams);
    document.getElementById('stat-input-toks').textContent = fmt(data.total_input_tokens);
    document.getElementById('stat-output-toks').textContent = fmt(data.total_output_tokens);
    document.getElementById('stat-latency').textContent = Math.round(data.avg_latency_ms) + 'ms';
    document.getElementById('stat-rpm').textContent = data.requests_per_min.toFixed(1);

    // Timeline chart
    const chart = document.getElementById('timeline-chart');
    chart.innerHTML = '';
    const timeline = data.requests_timeline || [];
    const counts = timeline.map(t => t.count);
    const maxCount = counts.length > 0 ? Math.max(1, ...counts) : 1;
    timeline.forEach(t => {
      const bar = document.createElement('div');
      bar.className = 'chart-bar';
      bar.style.height = (t.count / maxCount * 100) + '%';
      bar.title = new Date(t.time).toLocaleTimeString() + ': ' + t.count + ' req';
      chart.appendChild(bar);
    });

    // Model stats — build HTML in one pass, then assign once
    const tbody = document.getElementById('model-stats-body');
    const models = data.model_stats || {};
    const entries = Object.entries(models);
    if (entries.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--text-dim)">No data yet</td></tr>';
    } else {
      tbody.innerHTML = entries.map(([model, ms]) =>
        `<tr>
          <td>${esc(model)}</td>
          <td>${fmt(ms.requests)}</td>
          <td>${fmt(ms.input_tokens)}</td>
          <td>${fmt(ms.output_tokens)}</td>
          <td>${Math.round(ms.avg_latency_ms)}ms</td>
          <td>${ms.errors > 0 ? `<span class="badge badge-red">${ms.errors}</span>` : '0'}</td>
        </tr>`
      ).join('');
    }
  } catch(e) {
    console.error('loadStats error:', e);
  }
}
