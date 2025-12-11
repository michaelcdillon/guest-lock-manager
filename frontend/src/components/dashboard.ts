/**
 * Dashboard Component
 * 
 * Shows system overview with key metrics and recent activity.
 */

import { api, type StatusResponse, type GuestPin } from '../services/api';

/**
 * Render the dashboard page.
 */
export async function Dashboard(): Promise<string> {
  let status: StatusResponse | null = null;
  let recentPins: GuestPin[] = [];
  let error: string | null = null;

  try {
    [status, recentPins] = await Promise.all([
      api.getStatus(),
      api.getGuestPins({ status: 'active' }),
    ]);
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load dashboard';
  }

  if (error || !status) {
    return `
      <div class="alert alert-danger">
        <h4>Error Loading Dashboard</h4>
        <p>${error || 'Unknown error'}</p>
      </div>
    `;
  }

  return `
    <div class="dashboard">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1>Dashboard</h1>
        <div class="d-flex align-items-center gap-2">
          <span class="badge ${status.ha_connected ? 'bg-success' : 'bg-danger'}">
            HA ${status.ha_connected ? 'Connected' : 'Disconnected'}
          </span>
          ${status.zwave_js_ui_available ? '<span class="badge protocol-badge zwave">Z-Wave JS UI</span>' : ''}
          ${status.zigbee2mqtt_available ? '<span class="badge protocol-badge zigbee">Zigbee2MQTT</span>' : ''}
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="row g-4 mb-4">
        <div class="col-6 col-md-3">
          <div class="card stat-card">
            <div class="stat-value">${status.calendars_count}</div>
            <div class="stat-label">Calendars</div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card stat-card">
            <div class="stat-value">${status.locks_count}</div>
            <div class="stat-label">Locks</div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card stat-card">
            <div class="stat-value">${status.active_guest_pins}</div>
            <div class="stat-label">Active Guest PINs</div>
          </div>
        </div>
        <div class="col-6 col-md-3">
          <div class="card stat-card">
            <div class="stat-value">${status.active_static_pins}</div>
            <div class="stat-label">Static PINs</div>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="row g-4 mb-4">
        <div class="col-md-6">
          <div class="card h-100">
            <div class="card-header">Quick Actions</div>
            <div class="card-body">
              <div class="d-grid gap-2">
                <a href="#calendars" class="btn btn-outline-primary" data-page="calendars">
                  📅 Add Calendar
                </a>
                <a href="#locks" class="btn btn-outline-primary" data-page="locks">
                  🔒 Discover Locks
                </a>
                <a href="#static-pins" class="btn btn-outline-primary" data-page="static-pins">
                  🔢 Create Static PIN
                </a>
              </div>
            </div>
          </div>
        </div>
        <div class="col-md-6">
          <div class="card h-100">
            <div class="card-header">System Status</div>
            <div class="card-body">
              <div class="mb-3">
                <div class="d-flex justify-content-between mb-1">
                  <span class="text-muted">Pending Operations</span>
                  <span class="badge ${status.pending_operations > 0 ? 'bg-warning' : 'bg-success'}">
                    ${status.pending_operations}
                  </span>
                </div>
              </div>
              ${status.next_sync_at ? `
                <div class="mb-3">
                  <div class="d-flex justify-content-between mb-1">
                    <span class="text-muted">Next Calendar Sync</span>
                    <span class="badge bg-info">${formatDate(status.next_sync_at)}</span>
                  </div>
                </div>
              ` : ''}
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Guest PINs -->
      <div class="card">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span>Active Guest PINs</span>
          <a href="#calendars" class="btn btn-sm btn-outline-primary" data-page="calendars">View All</a>
        </div>
        <div class="card-body">
          ${recentPins.length === 0 ? `
            <div class="empty-state">
              <div class="empty-icon">📅</div>
              <div class="empty-title">No Active Guest PINs</div>
              <div class="empty-description">
                Add a calendar to automatically generate guest PINs from bookings.
              </div>
            </div>
          ` : `
            <div class="table-responsive">
              <table class="table">
                <thead>
                  <tr>
                    <th>Guest</th>
                    <th>PIN</th>
                    <th>Valid From</th>
                    <th>Valid Until</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  ${recentPins.slice(0, 5).map(pin => `
                    <tr>
                      <td>${pin.event_summary || 'Guest'}</td>
                      <td><span class="pin-code">${pin.pin_code}</span></td>
                      <td>${formatDate(pin.valid_from)}</td>
                      <td>${formatDate(pin.valid_until)}</td>
                      <td>
                        <span class="badge status-badge ${pin.status}">
                          ${pin.status}
                        </span>
                      </td>
                    </tr>
                  `).join('')}
                </tbody>
              </table>
            </div>
          `}
        </div>
      </div>
    </div>
  `;
}

/**
 * Format an ISO date string for display.
 */
function formatDate(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}



