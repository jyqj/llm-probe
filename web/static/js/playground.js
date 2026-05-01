// ============ Playground ============
async function sendPlayground() {
  const route = document.getElementById('pg-route').value;
  const model = document.getElementById('pg-model').value;
  const maxTokens = parseInt(document.getElementById('pg-max-tokens').value) || 1024;
  const stream = document.getElementById('pg-stream').value === 'true';
  const system = document.getElementById('pg-system').value.trim();
  const message = document.getElementById('pg-message').value.trim();
  const apiKey = document.getElementById('pg-apikey').value.trim();

  if (!message) { alert('Please enter a message'); return; }
  if (!apiKey) { alert('Please enter your API key'); return; }

  const respArea = document.getElementById('pg-response');
  const metaDiv = document.getElementById('pg-meta');
  respArea.textContent = '';
  metaDiv.style.display = 'none';

  const body = { model, max_tokens: maxTokens, stream, messages: [{ role: 'user', content: message }] };
  if (system) body.system = system;

  const btn = document.getElementById('pg-send');
  btn.disabled = true;
  btn.textContent = 'Sending...';
  const startTime = Date.now();

  try {
    const resp = await fetch(route, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'x-api-key': apiKey, 'anthropic-version': '2023-06-01' },
      body: JSON.stringify(body)
    });

    if (stream && resp.ok) {
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '', fullText = '', usage = null, stopReason = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop(); // keep incomplete last line

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          try {
            const event = JSON.parse(line.slice(6));
            if (event.type === 'content_block_delta') {
              if (event.delta?.text) { fullText += event.delta.text; respArea.textContent = fullText; respArea.scrollTop = respArea.scrollHeight; }
              else if (event.delta?.thinking) { fullText += '[thinking] ' + event.delta.thinking; respArea.textContent = fullText; }
            } else if (event.type === 'message_delta') {
              stopReason = event.delta?.stop_reason || '';
              usage = event.usage;
            } else if (event.type === 'message_start' && event.message?.usage) {
              usage = event.message.usage;
            }
          } catch(e) { /* ignore malformed SSE lines */ }
        }
      }

      // Process any remaining buffer after stream ends
      if (buffer.trim()) {
        const remaining = buffer.trim();
        if (remaining.startsWith('data: ')) {
          try {
            const event = JSON.parse(remaining.slice(6));
            if (event.type === 'content_block_delta' && event.delta?.text) {
              fullText += event.delta.text;
              respArea.textContent = fullText;
            } else if (event.type === 'message_delta') {
              stopReason = event.delta?.stop_reason || '';
              usage = event.usage;
            }
          } catch(e) { /* ignore */ }
        }
      }

      const latency = Date.now() - startTime;
      metaDiv.style.display = 'flex';
      document.getElementById('pg-meta-status').innerHTML = '<span class="badge badge-green">200 OK</span>';
      document.getElementById('pg-meta-latency').textContent = `${latency}ms`;
      document.getElementById('pg-meta-tokens').textContent = usage ? `tokens: ${usage.output_tokens || '?'}` : '';
      document.getElementById('pg-meta-model').textContent = stopReason ? `stop: ${stopReason}` : '';
    } else {
      const data = await resp.json();
      const latency = Date.now() - startTime;
      respArea.textContent = JSON.stringify(data, null, 2);
      metaDiv.style.display = 'flex';
      const statusBadge = resp.ok ? 'badge-green' : 'badge-red';
      document.getElementById('pg-meta-status').innerHTML = `<span class="badge ${statusBadge}">${resp.status}</span>`;
      document.getElementById('pg-meta-latency').textContent = `${latency}ms`;
      if (data.usage) document.getElementById('pg-meta-tokens').textContent = `in:${data.usage.input_tokens} out:${data.usage.output_tokens}`;
      document.getElementById('pg-meta-model').textContent = data.model || '';
    }
  } catch(e) {
    respArea.textContent = 'Error: ' + e.message;
  }

  btn.disabled = false;
  btn.textContent = 'Send Request';
}

function clearPlayground() {
  document.getElementById('pg-response').textContent = 'Waiting for request...';
  document.getElementById('pg-meta').style.display = 'none';
  document.getElementById('pg-message').value = '';
  document.getElementById('pg-system').value = '';
}
