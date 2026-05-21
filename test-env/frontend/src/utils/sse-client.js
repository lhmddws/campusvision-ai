/**
 * SSEClient — EventSource wrapper with auto-reconnect.
 * Connects to /api/events/stream and dispatches events.
 */
export class SSEClient {
  constructor(url, handlers = {}) {
    this.url = url || '/api/events/stream';
    this.handlers = {
      onRecognition: handlers.onRecognition || (() => {}),
      onBehavior: handlers.onBehavior || (() => {}),
      onHeartbeat: handlers.onHeartbeat || (() => {}),
      onError: handlers.onError || (() => {}),
      onConnected: handlers.onConnected || (() => {}),
    };
    this.eventSource = null;
    this.reconnectDelay = 1000;
    this.maxReconnectDelay = 30000;
    this.reconnectTimer = null;
  }

  connect() {
    this.disconnect();
    this.eventSource = new EventSource(this.url);

    this.eventSource.addEventListener('recognition', (e) => {
      try { this.handlers.onRecognition(JSON.parse(e.data)); } catch (err) {}
    });

    this.eventSource.addEventListener('behavior', (e) => {
      try { this.handlers.onBehavior(JSON.parse(e.data)); } catch (err) {}
    });

    this.eventSource.addEventListener('heartbeat', (e) => {
      try { this.handlers.onHeartbeat(JSON.parse(e.data)); } catch (err) {}
    });

    this.eventSource.onopen = () => {
      this.reconnectDelay = 1000;
      this.handlers.onConnected();
    };

    this.eventSource.onerror = (e) => {
      this.handlers.onError(e);
      this.disconnect();
      this.scheduleReconnect();
    };
  }

  disconnect() {
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
    if (this.eventSource) { this.eventSource.close(); this.eventSource = null; }
  }

  scheduleReconnect() {
    this.reconnectTimer = setTimeout(() => {
      this.connect();
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
    }, this.reconnectDelay);
  }
}

// Singleton factory
let instance = null;
export function createSSEClient(handlers) {
  if (!instance) { instance = new SSEClient('/api/events/stream', handlers); }
  return instance;
}
export function getSSEClient() { return instance; }
