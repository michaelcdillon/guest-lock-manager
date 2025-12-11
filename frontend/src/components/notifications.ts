/**
 * Notifications Component
 * 
 * Displays toast notifications for real-time events.
 */

export interface NotificationPayload {
  level: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message: string;
  dismissible?: boolean;
  action?: {
    type: string;
    label: string;
    url: string;
  };
}

export class Notifications {
  private container: HTMLElement | null;

  constructor() {
    this.container = document.getElementById('notifications');
  }

  /**
   * Show a toast notification.
   */
  show(notification: NotificationPayload): void {
    if (!this.container) return;

    const toast = this.createToast(notification);
    this.container.appendChild(toast);

    // Initialize and show Bootstrap toast
    const bsToast = new (window as unknown as { bootstrap: { Toast: new (el: Element) => { show(): void } } }).bootstrap.Toast(toast);
    bsToast.show();

    // Remove from DOM after hiding
    toast.addEventListener('hidden.bs.toast', () => {
      toast.remove();
    });
  }

  /**
   * Create a toast element.
   */
  private createToast(notification: NotificationPayload): HTMLElement {
    const toast = document.createElement('div');
    toast.className = `toast toast-${notification.level}`;
    toast.setAttribute('role', 'alert');
    toast.setAttribute('aria-live', 'assertive');
    toast.setAttribute('aria-atomic', 'true');

    const iconMap: Record<string, string> = {
      info: 'ℹ️',
      success: '✅',
      warning: '⚠️',
      error: '❌',
    };

    const colorMap: Record<string, string> = {
      info: 'text-info',
      success: 'text-success',
      warning: 'text-warning',
      error: 'text-danger',
    };

    toast.innerHTML = `
      <div class="toast-header">
        <span class="${colorMap[notification.level]} me-2">${iconMap[notification.level]}</span>
        <strong class="me-auto">${notification.title}</strong>
        <small>just now</small>
        ${notification.dismissible !== false ? '<button type="button" class="btn-close" data-bs-dismiss="toast"></button>' : ''}
      </div>
      <div class="toast-body">
        ${notification.message}
        ${notification.action ? `
          <div class="mt-2">
            <a href="${notification.action.url}" class="btn btn-sm btn-primary">
              ${notification.action.label}
            </a>
          </div>
        ` : ''}
      </div>
    `;

    return toast;
  }

  /**
   * Show an info notification.
   */
  info(title: string, message: string): void {
    this.show({ level: 'info', title, message, dismissible: true });
  }

  /**
   * Show a success notification.
   */
  success(title: string, message: string): void {
    this.show({ level: 'success', title, message, dismissible: true });
  }

  /**
   * Show a warning notification.
   */
  warning(title: string, message: string): void {
    this.show({ level: 'warning', title, message, dismissible: true });
  }

  /**
   * Show an error notification.
   */
  error(title: string, message: string): void {
    this.show({ level: 'error', title, message, dismissible: true });
  }
}



