/**
 * Static PIN List Component
 * 
 * Displays and manages static/recurring PINs for service personnel.
 */

import { api, type StaticPin } from '../services/api';

const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/**
 * Render the static PIN list page.
 */
export async function StaticPinList(): Promise<string> {
  let pins: StaticPin[] = [];
  let error: string | null = null;

  try {
    pins = await api.getStaticPins();
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load static PINs';
  }

  if (error) {
    return `
      <div class="alert alert-danger">
        <h4>Error Loading Static PINs</h4>
        <p>${error}</p>
      </div>
    `;
  }

  return `
    <div class="static-pin-list">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1>Static PINs</h1>
        <button class="btn btn-primary" onclick="showAddPinModal()">
          + Add Static PIN
        </button>
      </div>

      ${pins.length === 0 ? `
        <div class="card">
          <div class="card-body">
            <div class="empty-state">
              <div class="empty-icon">🔢</div>
              <div class="empty-title">No Static PINs</div>
              <div class="empty-description">
                Create static PINs for cleaners, maintenance staff, or family members with optional time restrictions.
              </div>
              <button class="btn btn-primary" onclick="showAddPinModal()">
                Create Static PIN
              </button>
            </div>
          </div>
        </div>
      ` : `
        <div class="row g-4">
          ${pins.map(pin => renderPinCard(pin)).join('')}
        </div>
      `}

      <!-- Add/Edit PIN Modal -->
      <div class="modal fade" id="pinModal" tabindex="-1">
        <div class="modal-dialog modal-lg">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title" id="pinModalTitle">Add Static PIN</h5>
              <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              <form id="pinForm">
                <input type="hidden" id="pinId">
                <div class="row">
                  <div class="col-md-6">
                    <div class="mb-3">
                      <label class="form-label" for="pinName">Name</label>
                      <input type="text" class="form-control" id="pinName" 
                             placeholder="e.g., Cleaner - Maria" required>
                    </div>
                    <div class="mb-3">
                      <label class="form-label" for="pinCode">PIN Code</label>
                      <input type="text" class="form-control" id="pinCode" 
                             placeholder="4-8 digits" pattern="[0-9]{4,8}" required>
                    </div>
                  <div class="mb-3">
                    <label class="form-label" for="pinSlot">Slot Number</label>
                    <input type="number" class="form-control" id="pinSlot" min="1" max="50" value="1">
                    <div class="form-text">PIN slot to program on each managed lock.</div>
                  </div>
                    <div class="form-check mb-3">
                      <input class="form-check-input" type="checkbox" id="alwaysActive">
                      <label class="form-check-label" for="alwaysActive">
                        Always Active (no time restrictions)
                      </label>
                    </div>
                  </div>
                  <div class="col-md-6">
                    <div id="scheduleSection">
                      <label class="form-label">Access Schedule</label>
                      <div class="form-text mb-2">Select days and times when this PIN is valid</div>
                      ${DAYS.map((day, i) => `
                        <div class="d-flex align-items-center mb-2">
                          <div class="form-check me-3" style="width: 100px;">
                            <input class="form-check-input day-check" type="checkbox" 
                                   id="day${i}" data-day="${i}">
                            <label class="form-check-label" for="day${i}">${day.substring(0, 3)}</label>
                          </div>
                          <input type="time" class="form-control form-control-sm me-2" 
                                 id="start${i}" value="08:00" style="width: 110px;">
                          <span class="me-2">-</span>
                          <input type="time" class="form-control form-control-sm" 
                                 id="end${i}" value="17:00" style="width: 110px;">
                        </div>
                      `).join('')}
                    </div>
                  </div>
                </div>
              </form>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
              <button type="button" class="btn btn-primary" onclick="savePin()">Save PIN</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

/**
 * Render a single static PIN card.
 */
function renderPinCard(pin: StaticPin): string {
  const scheduleText = pin.always_active 
    ? 'Always Active' 
    : formatSchedule(pin.schedules || []);

  const schedulesJson = JSON.stringify(pin.schedules || []).replace(/"/g, '&quot;');
  const slotText = pin.slot_number ? `Slot ${pin.slot_number}` : 'Slot not set';

  return `
    <div class="col-md-6 col-lg-4">
      <div class="card h-100 ${!pin.enabled ? 'opacity-50' : ''}">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span>${pin.name}</span>
          <span class="badge ${pin.enabled ? 'bg-success' : 'bg-secondary'}">
            ${pin.enabled ? 'Active' : 'Disabled'}
          </span>
        </div>
        <div class="card-body">
          <div class="text-center mb-3">
            <span class="pin-code">${pin.pin_code}</span>
          </div>
          <div class="mb-3">
            <small class="text-muted d-block">Schedule</small>
            <span>${scheduleText}</span>
          </div>
          <div class="mb-3">
            <small class="text-muted d-block">Slot</small>
            <span>${slotText}</span>
          </div>
          <div class="d-flex gap-2">
            <button class="btn btn-sm btn-outline-primary flex-grow-1" 
                    onclick="editPin('${pin.id}', '${pin.name}', '${pin.pin_code}', ${pin.always_active}, ${schedulesJson}, ${pin.slot_number ?? 1})">
              Edit
            </button>
            <button class="btn btn-sm btn-outline-danger" 
                    onclick="deletePin('${pin.id}', '${pin.name}')">
              🗑
            </button>
          </div>
        </div>
        <div class="card-footer">
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" id="enabled-${pin.id}"
                   ${pin.enabled ? 'checked' : ''} 
                   onchange="togglePin('${pin.id}', this.checked)">
            <label class="form-check-label" for="enabled-${pin.id}">
              Enabled
            </label>
          </div>
        </div>
      </div>
    </div>
  `;
}

/**
 * Format schedule for display.
 */
function formatSchedule(schedules: { day_of_week: number; start_time: string; end_time: string }[]): string {
  if (schedules.length === 0) {
    return 'No schedule configured';
  }

  // Group by time range
  const timeGroups = new Map<string, number[]>();
  schedules.forEach(s => {
    const key = `${s.start_time}-${s.end_time}`;
    if (!timeGroups.has(key)) {
      timeGroups.set(key, []);
    }
    timeGroups.get(key)!.push(s.day_of_week);
  });

  const parts: string[] = [];
  timeGroups.forEach((days, time) => {
    const dayStr = days.map(d => DAYS[d].substring(0, 3)).join(', ');
    parts.push(`${dayStr}: ${time.replace('-', ' - ')}`);
  });

  return parts.join('; ');
}

