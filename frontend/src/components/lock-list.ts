/**
 * Lock List Component
 * 
 * Displays and manages IoT locks.
 */

import { api, type Lock } from '../services/api';

/**
 * Render the lock list page.
 */
export async function LockList(): Promise<string> {
  let locks: Lock[] = [];
  let error: string | null = null;

  try {
    locks = await api.getLocks();
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load locks';
  }

  if (error) {
    return `
      <div class="alert alert-danger">
        <h4>Error Loading Locks</h4>
        <p>${error}</p>
      </div>
    `;
  }

  return `
    <div class="lock-list">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h1>Locks</h1>
        <button class="btn btn-primary" onclick="discoverLocks()">
          🔍 Discover Locks
        </button>
      </div>

      ${locks.length === 0 ? `
        <div class="card">
          <div class="card-body">
            <div class="empty-state">
              <div class="empty-icon">🔒</div>
              <div class="empty-title">No Locks Configured</div>
              <div class="empty-description">
                Discover locks from your Home Assistant installation to start managing PINs.
              </div>
              <button class="btn btn-primary" onclick="discoverLocks()">
                Discover Locks Now
              </button>
            </div>
          </div>
        </div>
      ` : `
        <div class="table-responsive">
          <table class="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Protocol</th>
                <th>Status</th>
                <th>Battery</th>
                <th>Slots (Guest/Static/Total)</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              ${locks.map(lock => renderLockRow(lock)).join('')}
            </tbody>
          </table>
        </div>
      `}

      <!-- Edit Lock Modal -->
      <div class="modal fade" id="editLockModal" tabindex="-1">
        <div class="modal-dialog">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title">Configure Lock</h5>
              <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
              <form id="editLockForm">
                <input type="hidden" id="editLockId">
                <div class="mb-3">
                  <label class="form-label" for="lockName">Display Name</label>
                  <input type="text" class="form-control" id="lockName" required>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="totalSlots">Total PIN Slots</label>
                  <input type="number" class="form-control" id="totalSlots" min="1" max="50">
                  <div class="form-text">Total programmable PIN slots on the lock.</div>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="guestSlots">Guest PIN Slots</label>
                  <input type="number" class="form-control" id="guestSlots" min="0" max="20">
                  <div class="form-text">Number of slots reserved for guest PINs</div>
                </div>
                <div class="mb-3">
                  <label class="form-label" for="staticSlots">Static PIN Slots</label>
                  <input type="number" class="form-control" id="staticSlots" min="0" max="20">
                  <div class="form-text">Number of slots reserved for static PINs</div>
                </div>
              </form>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
              <button type="button" class="btn btn-primary" id="saveLockBtn">Save Changes</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

/**
 * Render a single lock table row.
 */
function renderLockRow(lock: Lock): string {
  const batteryDisplay = lock.battery_level != null 
    ? `${lock.battery_level}%` 
    : 'N/A';
  
  const batteryClass = lock.battery_level != null && lock.battery_level < 20 
    ? 'text-danger' 
    : '';

  const directIntegration = lock.direct_integration
    ? `<span class="badge bg-success ms-1">${lock.direct_integration}</span>`
    : '';

  return `
    <tr>
      <td>
        <div>
          <strong>${lock.name}</strong>
          <br>
          <small class="text-muted">${lock.entity_id}</small>
        </div>
      </td>
      <td>
        <span class="badge protocol-badge ${lock.protocol}">${lock.protocol}</span>
        ${directIntegration}
      </td>
      <td>
        <div class="d-flex gap-1">
          <span class="badge status-badge ${lock.online ? 'online' : 'offline'}">
            ${lock.online ? 'Online' : 'Offline'}
          </span>
          <span class="badge ${lock.state === 'locked' ? 'bg-success' : lock.state === 'unlocked' ? 'bg-warning text-dark' : 'bg-secondary'}">
            ${lock.state ? lock.state : 'unknown'}
          </span>
        </div>
      </td>
      <td class="${batteryClass}">
        ${lock.battery_level != null ? '🔋' : ''} ${batteryDisplay}
      </td>
      <td>
        <span class="text-muted">
          ${lock.guest_slots} / ${lock.static_slots} / ${lock.total_slots}
        </span>
      </td>
      <td>
        <button class="btn btn-sm btn-outline-primary me-1"
                onclick="editLock('${lock.id}', '${lock.name}', ${lock.total_slots}, ${lock.guest_slots}, ${lock.static_slots})">
          ⚙️
        </button>
        <button class="btn btn-sm btn-outline-danger" 
                onclick="deleteLock('${lock.id}', '${lock.name}')">
          🗑
        </button>
      </td>
    </tr>
  `;
}

