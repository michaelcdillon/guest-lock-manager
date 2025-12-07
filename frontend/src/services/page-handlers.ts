/**
 * Page Handlers Service
 * 
 * Registers global functions and initializes page-specific event handlers.
 * This bridges the gap between inline HTML event handlers and ES modules.
 */

import { Modal } from 'bootstrap';
import { api } from './api';

// Store modal instances for reuse
const modals = new Map<string, Modal>();

/**
 * Get or create a Bootstrap Modal instance.
 */
function getModal(id: string): Modal | null {
  const element = document.getElementById(id);
  if (!element) return null;

  if (!modals.has(id)) {
    modals.set(id, new Modal(element));
  }
  return modals.get(id)!;
}

/**
 * Show a modal by ID.
 */
function showModal(id: string): void {
  const modal = getModal(id);
  if (modal) modal.show();
}

/**
 * Hide a modal by ID.
 */
function hideModal(id: string): void {
  const modal = getModal(id);
  if (modal) modal.hide();
}

// ============================================================================
// Calendar Page Handlers
// ============================================================================

function showAddCalendarModal(): void {
  const form = document.getElementById('addCalendarForm') as HTMLFormElement | null;
  if (form) form.reset();
  showModal('addCalendarModal');
}

async function submitAddCalendar(): Promise<void> {
  const name = (document.getElementById('calendarName') as HTMLInputElement).value;
  const url = (document.getElementById('calendarUrl') as HTMLInputElement).value;
  const interval = parseInt((document.getElementById('syncInterval') as HTMLSelectElement).value);

  try {
    await api.createCalendar({
      name,
      url,
      sync_interval_min: interval,
      enabled: true,
    });

    hideModal('addCalendarModal');
    location.reload();
  } catch (error) {
    alert('Error: ' + (error instanceof Error ? error.message : 'Failed to add calendar'));
  }
}

async function syncCalendar(id: string): Promise<void> {
  const btn = event?.target as HTMLButtonElement;
  if (btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> Syncing...';
  }

  try {
    await api.syncCalendar(id);
    location.reload();
  } catch (error) {
    alert('Sync failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    if (btn) {
      btn.disabled = false;
      btn.textContent = 'Sync Now';
    }
  }
}

async function deleteCalendar(id: string, name: string): Promise<void> {
  if (!confirm(`Delete calendar "${name}"? This will also remove all associated guest PINs.`)) {
    return;
  }

  try {
    await api.deleteCalendar(id);
    location.reload();
  } catch (error) {
    alert('Delete failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
  }
}

// ============================================================================
// Lock Page Handlers
// ============================================================================

async function discoverLocks(): Promise<void> {
  const btn = event?.target as HTMLButtonElement;
  if (btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> Discovering...';
  }

  try {
    await api.discoverLocks();
    location.reload();
  } catch (error) {
    alert('Discovery failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    if (btn) {
      btn.disabled = false;
      btn.textContent = '🔍 Discover Locks';
    }
  }
}

function editLock(id: string, name: string, totalSlots: number, guestSlots: number, staticSlots: number): void {
  (document.getElementById('editLockId') as HTMLInputElement).value = id;
  (document.getElementById('lockName') as HTMLInputElement).value = name;
  (document.getElementById('totalSlots') as HTMLInputElement).value = String(totalSlots || 0);
  (document.getElementById('guestSlots') as HTMLInputElement).value = String(guestSlots);
  (document.getElementById('staticSlots') as HTMLInputElement).value = String(staticSlots);
  showModal('editLockModal');
}

async function saveLock(): Promise<void> {
  const id = (document.getElementById('editLockId') as HTMLInputElement).value;
  const name = (document.getElementById('lockName') as HTMLInputElement).value;
  const totalSlots = parseInt((document.getElementById('totalSlots') as HTMLInputElement).value);
  const guestSlots = parseInt((document.getElementById('guestSlots') as HTMLInputElement).value);
  const staticSlots = parseInt((document.getElementById('staticSlots') as HTMLInputElement).value);

  if (guestSlots + staticSlots > totalSlots) {
    alert('Guest slots + Static slots cannot exceed Total slots.');
    return;
  }

  try {
    await api.updateLock(id, { name, total_slots: totalSlots, guest_slots: guestSlots, static_slots: staticSlots });
    hideModal('editLockModal');
    location.reload();
  } catch (error) {
    alert('Error: ' + (error instanceof Error ? error.message : 'Failed to update lock'));
  }
}

async function deleteLock(id: string, name: string): Promise<void> {
  if (!confirm(`Remove lock "${name}" from management?`)) {
    return;
  }

  try {
    await api.deleteLock(id);
    location.reload();
  } catch (error) {
    alert('Delete failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
  }
}

// ============================================================================
// Static PIN Page Handlers
// ============================================================================

function showAddPinModal(): void {
  (document.getElementById('pinModalTitle') as HTMLElement).textContent = 'Add Static PIN';
  (document.getElementById('pinForm') as HTMLFormElement).reset();
  (document.getElementById('pinId') as HTMLInputElement).value = '';
  (document.getElementById('scheduleSection') as HTMLElement).style.display = 'block';
  showModal('pinModal');
}

interface Schedule {
  day_of_week: number;
  start_time: string;
  end_time: string;
}

function editPin(id: string, name: string, code: string, alwaysActive: boolean, schedules: Schedule[]): void {
  (document.getElementById('pinModalTitle') as HTMLElement).textContent = 'Edit Static PIN';
  (document.getElementById('pinId') as HTMLInputElement).value = id;
  (document.getElementById('pinName') as HTMLInputElement).value = name;
  (document.getElementById('pinCode') as HTMLInputElement).value = code;
  (document.getElementById('alwaysActive') as HTMLInputElement).checked = alwaysActive;
  (document.getElementById('scheduleSection') as HTMLElement).style.display = alwaysActive ? 'none' : 'block';

  // Clear all day checkboxes
  document.querySelectorAll<HTMLInputElement>('.day-check').forEach(cb => cb.checked = false);

  // Set schedule values
  if (schedules) {
    schedules.forEach(s => {
      (document.getElementById('day' + s.day_of_week) as HTMLInputElement).checked = true;
      (document.getElementById('start' + s.day_of_week) as HTMLInputElement).value = s.start_time;
      (document.getElementById('end' + s.day_of_week) as HTMLInputElement).value = s.end_time;
    });
  }

  showModal('pinModal');
}

async function savePin(): Promise<void> {
  const id = (document.getElementById('pinId') as HTMLInputElement).value;
  const name = (document.getElementById('pinName') as HTMLInputElement).value;
  const code = (document.getElementById('pinCode') as HTMLInputElement).value;
  const alwaysActive = (document.getElementById('alwaysActive') as HTMLInputElement).checked;

  // Build schedules
  const schedules: Schedule[] = [];
  if (!alwaysActive) {
    document.querySelectorAll<HTMLInputElement>('.day-check:checked').forEach(cb => {
      const day = parseInt(cb.dataset.day || '0');
      schedules.push({
        day_of_week: day,
        start_time: (document.getElementById('start' + day) as HTMLInputElement).value,
        end_time: (document.getElementById('end' + day) as HTMLInputElement).value,
      });
    });
  }

  const data: Partial<import('./api').StaticPin> = {
    name,
    pin_code: code,
    enabled: true,
    always_active: alwaysActive,
  };

  try {
    if (id) {
      await api.updateStaticPin(id, data);
    } else {
      await api.createStaticPin(data);
    }
    hideModal('pinModal');
    location.reload();
  } catch (error) {
    alert('Error: ' + (error instanceof Error ? error.message : 'Failed to save PIN'));
  }
}

async function togglePin(id: string, enabled: boolean): Promise<void> {
  try {
    await api.updateStaticPin(id, { enabled });
  } catch (error) {
    alert('Failed to update PIN: ' + (error instanceof Error ? error.message : 'Unknown error'));
    location.reload();
  }
}

async function deletePin(id: string, name: string): Promise<void> {
  if (!confirm(`Delete static PIN "${name}"?`)) {
    return;
  }

  try {
    await api.deleteStaticPin(id);
    location.reload();
  } catch (error) {
    alert('Delete failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
  }
}

// ============================================================================
// Settings Page Handlers
// ============================================================================

async function saveSettings(): Promise<void> {
  const settings = {
    min_pin_length: (document.getElementById('minPinLength') as HTMLSelectElement).value,
    max_pin_length: (document.getElementById('maxPinLength') as HTMLSelectElement).value,
    checkin_time: (document.getElementById('checkinTime') as HTMLInputElement).value,
    checkout_time: (document.getElementById('checkoutTime') as HTMLInputElement).value,
    default_sync_interval_min: (document.getElementById('syncInterval') as HTMLSelectElement).value,
    battery_efficient_mode: (document.getElementById('batteryMode') as HTMLInputElement).checked ? 'true' : 'false',
    batch_window_seconds: (document.getElementById('batchWindow') as HTMLSelectElement).value,
  };

  try {
    await api.updateSettings(settings);

    // Show success message
    const btn = event?.target as HTMLButtonElement;
    if (btn) {
      const original = btn.textContent;
      btn.textContent = '✓ Saved!';
      btn.classList.add('btn-success');
      btn.classList.remove('btn-primary');

      setTimeout(() => {
        btn.textContent = original;
        btn.classList.remove('btn-success');
        btn.classList.add('btn-primary');
      }, 2000);
    }
  } catch (error) {
    alert('Error: ' + (error instanceof Error ? error.message : 'Failed to save settings'));
  }
}

// ============================================================================
// Register Global Functions
// ============================================================================

/**
 * Initialize page handlers by exposing functions to the global window object.
 * This allows inline onclick handlers to work with ES modules.
 */
export function initializePageHandlers(): void {
  // Calendar handlers
  (window as any).showAddCalendarModal = showAddCalendarModal;
  (window as any).submitAddCalendar = submitAddCalendar;
  (window as any).syncCalendar = syncCalendar;
  (window as any).deleteCalendar = deleteCalendar;

  // Lock handlers
  (window as any).discoverLocks = discoverLocks;
  (window as any).editLock = editLock;
  (window as any).saveLock = saveLock;
  (window as any).deleteLock = deleteLock;

  // Static PIN handlers
  (window as any).showAddPinModal = showAddPinModal;
  (window as any).editPin = editPin;
  (window as any).savePin = savePin;
  (window as any).togglePin = togglePin;
  (window as any).deletePin = deletePin;

  // Settings handlers
  (window as any).saveSettings = saveSettings;

  console.log('Page handlers initialized');
}

/**
 * Initialize page-specific event listeners.
 * Called after page content is loaded.
 */
export function initializePageListeners(pageName: string): void {
  // Clean up old modal instances when changing pages
  modals.forEach((modal) => {
    try {
      modal.dispose();
    } catch {
      // Ignore disposal errors
    }
  });
  modals.clear();

  // Page-specific initialization
  switch (pageName) {
    case 'static-pins':
      // Set up always-active toggle
      const alwaysActiveCheckbox = document.getElementById('alwaysActive');
      if (alwaysActiveCheckbox) {
        alwaysActiveCheckbox.addEventListener('change', function(this: HTMLInputElement) {
          const section = document.getElementById('scheduleSection');
          if (section) {
            section.style.display = this.checked ? 'none' : 'block';
          }
        });
      }
      break;

    case 'settings':
      // Check integration status
      api.getStatus().then(status => {
        const zwaveStatus = document.getElementById('zwaveStatus');
        const zigbeeStatus = document.getElementById('zigbeeStatus');

        if (zwaveStatus) {
          zwaveStatus.textContent = status.zwave_js_ui_available ? 'Available' : 'Not Found';
          zwaveStatus.className = 'badge ' + (status.zwave_js_ui_available ? 'bg-success' : 'bg-secondary');
        }

        if (zigbeeStatus) {
          zigbeeStatus.textContent = status.zigbee2mqtt_available ? 'Available' : 'Not Found';
          zigbeeStatus.className = 'badge ' + (status.zigbee2mqtt_available ? 'bg-success' : 'bg-secondary');
        }
      }).catch(() => {
        // Ignore status check errors
      });
      break;
  }
}

