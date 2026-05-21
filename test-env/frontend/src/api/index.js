const API_BASE = window.location.origin

async function request(url, options = {}) {
  const res = await fetch(`${API_BASE}${url}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(`HTTP ${res.status}: ${text}`)
  }
  return res.json()
}

export const api = {
  // Health & Config
  health: () => request('/api/health'),
  getConfig: () => request('/api/config'),
  updateConfig: (data) => request('/api/config', {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  resetConfig: () => request('/api/config/reset', { method: 'PUT' }),

  // Cameras
  getCameras: () => request('/api/cameras'),
  upsertCamera: (id, data) => request(`/api/cameras/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }),
  deleteCamera: (id) => request(`/api/cameras/${id}`, { method: 'DELETE' }),
  cameraStatus: (id) => request(`/api/cameras/${id}/status`),

  // Events
  getEvents: (limit = 50) => request(`/api/events?limit=${limit}`),
  getStats: () => request('/api/stats'),

  // Simulation
  simulate: (cameraId, data) => request(`/api/cameras/${cameraId}/simulate`, {
    method: 'POST',
    body: JSON.stringify(data),
  }),
  randomScenario: (count = 5) => request(`/api/scenarios/random?count=${count}`),
  presetScenario: (preset) => request('/api/scenarios/preset', {
    method: 'POST',
    body: JSON.stringify({ preset }),
  }),

  // People
  getPeople: () => request('/api/people'),
  addPerson: (name) => request('/api/people', {
    method: 'POST',
    body: JSON.stringify({ name }),
  }),
  removePerson: (name) => request(`/api/people?name=${encodeURIComponent(name)}`, {
    method: 'DELETE',
  }),

  // Faces (enrollment)
  getFaces: () => request('/api/faces'),
  enrollFace: async (name, studentId, imageFile) => {
    const form = new FormData()
    form.append('name', name)
    form.append('student_id', studentId)
    form.append('image', imageFile)
    const res = await fetch(`${API_BASE}/api/faces`, { method: 'POST', body: form })
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`)
    return res.json()
  },
  deleteFace: (name) => request(`/api/faces/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  }),
  getFaceImageUrl: (name) => `${API_BASE}/api/faces/${encodeURIComponent(name)}/image`,

  // Recognition
  recognitionStatus: () => request('/api/recognition/status'),
  getRecognitionResults: () => request('/api/recognition/results'),

  // Behavior status
  behaviorStatus: () => request('/api/behavior/status'),

  // Fake data
  toggleFakeData: (useFakeData) => request('/api/toggle-fake-data', {
    method: 'POST',
    body: JSON.stringify({ use_fake_data: useFakeData }),
  }),

  // Webcam
  webcamStart: (cameraId, deviceIndex) => request('/api/webcam/start', {
    method: 'POST',
    body: JSON.stringify({ camera_id: cameraId, device_index: deviceIndex }),
  }),
  webcamStop: (cameraId) => request('/api/webcam/stop', {
    method: 'POST',
    body: JSON.stringify({ camera_id: cameraId }),
  }),
  webcamStatus: () => request('/api/webcam/status', { method: 'POST' }),
  webcamStartAll: () => request('/api/webcam/start-all', { method: 'POST' }),
  webcamStopAll: () => request('/api/webcam/stop-all', { method: 'POST' }),

  frameUrl: (cameraId) => `${API_BASE}/api/cameras/${cameraId}/frame.jpg`,

  // SSE
  getSSEUrl: () => `${API_BASE}/api/events/stream`,
}
