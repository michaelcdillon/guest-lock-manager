/**
 * Settings Component
 * 
 * System configuration and preferences.
 */

import { api, type Settings as SettingsType } from '../services/api';

/**
 * Render the settings page.
 */
export async function Settings(): Promise<string> {
  let settings: SettingsType | null = null;
  let error: string | null = null;

  try {
    settings = await api.getSettings();
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load settings';
  }

  if (error || !settings) {
    return `
      <div class="alert alert-danger">
        <h4>Error Loading Settings</h4>
        <p>${error || 'Unknown error'}</p>
      </div>
    `;
  }

  return `
    <div class="settings">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1>Settings</h1>
      </div>

      <div class="row g-4">
        <!-- PIN Configuration -->
        <div class="col-lg-6">
          <div class="card">
            <div class="card-header">PIN Configuration</div>
            <div class="card-body">
              <form id="pinSettingsForm">
                <div class="row mb-3">
                  <div class="col-6">
                    <label class="form-label" for="minPinLength">Minimum PIN Length</label>
                    <select class="form-select" id="minPinLength">
                      ${[4, 5, 6, 7, 8].map(n => `
                        <option value="${n}" ${settings.min_pin_length === String(n) ? 'selected' : ''}>
                          ${n} digits
                        </option>
                      `).join('')}
                    </select>
                  </div>
                  <div class="col-6">
                    <label class="form-label" for="maxPinLength">Maximum PIN Length</label>
                    <select class="form-select" id="maxPinLength">
                      ${[4, 5, 6, 7, 8].map(n => `
                        <option value="${n}" ${settings.max_pin_length === String(n) ? 'selected' : ''}>
                          ${n} digits
                        </option>
                      `).join('')}
                    </select>
                  </div>
                </div>
                <p class="form-text">
                  PIN length applies to automatically generated PINs. Custom PINs may vary.
                </p>
              </form>
            </div>
          </div>
        </div>

        <!-- Timing Configuration -->
        <div class="col-lg-6">
          <div class="card">
            <div class="card-header">Timing</div>
            <div class="card-body">
              <form id="timingSettingsForm">
                <div class="row mb-3">
                  <div class="col-6">
                    <label class="form-label" for="checkinTime">Default Check-in Time</label>
                    <input type="time" class="form-control" id="checkinTime" 
                           value="${settings.checkin_time}">
                  </div>
                  <div class="col-6">
                    <label class="form-label" for="checkoutTime">Default Check-out Time</label>
                    <input type="time" class="form-control" id="checkoutTime" 
                           value="${settings.checkout_time}">
                  </div>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="syncInterval">Default Sync Interval</label>
                  <select class="form-select" id="syncInterval">
                    ${[15, 30, 60, 120].map(n => `
                      <option value="${n}" ${settings.default_sync_interval_min === String(n) ? 'selected' : ''}>
                        Every ${n} minutes
                      </option>
                    `).join('')}
                  </select>
                </div>
              </form>
            </div>
          </div>
        </div>

        <!-- Battery Efficiency -->
        <div class="col-lg-6">
          <div class="card">
            <div class="card-header">Battery Efficiency</div>
            <div class="card-body">
              <form id="batterySettingsForm">
                <div class="form-check form-switch mb-3">
                  <input class="form-check-input" type="checkbox" id="batteryMode" 
                         ${settings.battery_efficient_mode === 'true' ? 'checked' : ''}>
                  <label class="form-check-label" for="batteryMode">
                    Battery Efficient Mode
                  </label>
                </div>
                <p class="form-text">
                  When enabled, PIN updates are batched to minimize lock wake-ups. 
                  This can significantly extend lock battery life.
                </p>
                <div class="mb-3">
                  <label class="form-label" for="batchWindow">Batch Window</label>
                  <select class="form-select" id="batchWindow">
                    ${[15, 30, 60, 120].map(n => `
                      <option value="${n}" ${settings.batch_window_seconds === String(n) ? 'selected' : ''}>
                        ${n} seconds
                      </option>
                    `).join('')}
                  </select>
                  <div class="form-text">
                    How long to wait before sending batched PIN updates to locks.
                  </div>
                </div>
              </form>
            </div>
          </div>
        </div>

        <!-- Integrations -->
        <div class="col-lg-6">
          <div class="card">
            <div class="card-header">Integrations</div>
            <div class="card-body">
              <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center mb-2">
                  <span>Home Assistant</span>
                  <span class="badge bg-success">Connected</span>
                </div>
                <p class="form-text mb-0">Required for lock control</p>
              </div>
              <hr>
              <div class="mb-3">
                <div class="d-flex justify-content-between align-items-center mb-2">
                  <span>Z-Wave JS UI</span>
                  <span class="badge bg-secondary" id="zwaveStatus">Detecting...</span>
                </div>
                <p class="form-text mb-0">Direct Z-Wave communication for battery efficiency</p>
              </div>
              <hr>
              <div class="mb-0">
                <div class="d-flex justify-content-between align-items-center mb-2">
                  <span>Zigbee2MQTT</span>
                  <span class="badge bg-secondary" id="zigbeeStatus">Detecting...</span>
                </div>
                <p class="form-text mb-0">Direct Zigbee communication for battery efficiency</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Save Button -->
      <div class="mt-4 d-flex justify-content-end">
        <button class="btn btn-primary btn-lg" onclick="saveSettings()">
          Save Settings
        </button>
      </div>
    </div>
  `;
}

