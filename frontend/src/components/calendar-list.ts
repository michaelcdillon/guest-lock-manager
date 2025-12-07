/**
 * Calendar List Component
 * 
 * Displays and manages calendar subscriptions.
 */

import { api, type Calendar } from '../services/api';

/**
 * Render the calendar list page.
 */
export async function CalendarList(): Promise<string> {
  let calendars: Calendar[] = [];
  let error: string | null = null;

  try {
    calendars = await api.getCalendars();
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load calendars';
  }

  if (error) {
    return `
      <div class="alert alert-danger">
        <h4>Error Loading Calendars</h4>
        <p>${error}</p>
      </div>
    `;
  }

  return `
    <div class="calendar-list">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1>Calendars</h1>
        <button class="btn btn-primary" onclick="showAddCalendarModal()">
          + Add Calendar
        </button>
      </div>

      ${calendars.length === 0 ? `
        <div class="card">
          <div class="card-body">
            <div class="empty-state">
              <div class="empty-icon">📅</div>
              <div class="empty-title">No Calendars Configured</div>
              <div class="empty-description">
                Add a calendar subscription to automatically generate guest PINs from your rental bookings.
              </div>
              <button class="btn btn-primary" onclick="showAddCalendarModal()">
                Add Your First Calendar
              </button>
            </div>
          </div>
        </div>
      ` : `
        <div class="row g-4">
          ${calendars.map(calendar => renderCalendarCard(calendar)).join('')}
        </div>
      `}

      <!-- Add Calendar Modal -->
      <div class="modal fade" id="addCalendarModal" tabindex="-1">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title">Add Calendar</h5>
              <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              <form id="addCalendarForm">
                <div class="mb-3">
                  <label class="form-label" for="calendarName">Calendar Name</label>
                  <input type="text" class="form-control" id="calendarName" 
                         placeholder="e.g., Airbnb - Beach House" required>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="calendarUrl">iCal URL</label>
                  <input type="url" class="form-control" id="calendarUrl" 
                         placeholder="https://..." required>
                  <div class="form-text">
                    Get this from your Airbnb, VRBO, or other rental platform settings.
                  </div>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="syncInterval">Sync Interval (minutes)</label>
                  <select class="form-select" id="syncInterval">
                    <option value="15">Every 15 minutes</option>
                    <option value="30">Every 30 minutes</option>
                    <option value="60">Every hour</option>
                    <option value="120">Every 2 hours</option>
                  </select>
                </div>
              </form>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
              <button type="button" class="btn btn-primary" onclick="submitAddCalendar()">Add Calendar</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

/**
 * Render a single calendar card.
 */
function renderCalendarCard(calendar: Calendar): string {
  const syncStatusClass = getSyncStatusClass(calendar.sync_status);
  const lastSync = calendar.last_sync_at 
    ? new Date(calendar.last_sync_at).toLocaleString() 
    : 'Never';

  return `
    <div class="col-md-6 col-lg-4">
      <div class="card h-100">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span>${calendar.name}</span>
          <span class="badge status-badge ${syncStatusClass}">${calendar.sync_status}</span>
        </div>
        <div class="card-body">
          <div class="mb-3">
            <small class="text-muted d-block">Last Sync</small>
            <span>${lastSync}</span>
          </div>
          <div class="mb-3">
            <small class="text-muted d-block">Sync Interval</small>
            <span>Every ${calendar.sync_interval_min} minutes</span>
          </div>
          ${calendar.sync_error ? `
            <div class="alert alert-danger py-2 mb-3">
              <small>${calendar.sync_error}</small>
            </div>
          ` : ''}
          <div class="d-flex gap-2">
            <button class="btn btn-sm btn-outline-primary flex-grow-1" 
                    onclick="syncCalendar('${calendar.id}')">
              Sync Now
            </button>
            <button class="btn btn-sm btn-outline-danger" 
                    onclick="deleteCalendar('${calendar.id}', '${calendar.name}')">
              🗑
            </button>
          </div>
        </div>
        <div class="card-footer">
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" id="enabled-${calendar.id}"
                   ${calendar.enabled ? 'checked' : ''}>
            <label class="form-check-label" for="enabled-${calendar.id}">
              ${calendar.enabled ? 'Enabled' : 'Disabled'}
            </label>
          </div>
        </div>
      </div>
    </div>
  `;
}

/**
 * Get CSS class for sync status.
 */
function getSyncStatusClass(status: string): string {
  switch (status) {
    case 'success': return 'online';
    case 'error': return 'offline';
    case 'syncing': return 'syncing';
    default: return 'pending';
  }
}

