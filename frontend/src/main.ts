/**
 * Guest Lock Manager - Main Application Entry Point
 * 
 * Bootstraps the application, initializes services, and sets up navigation.
 */

import 'bootstrap/dist/js/bootstrap.bundle.min.js';
import './styles/main.scss';

import { api } from './services/api';
import { websocket } from './services/websocket';
import { Dashboard } from './components/dashboard';
import { CalendarList } from './components/calendar-list';
import { LockList } from './components/lock-list';
import { StaticPinList } from './components/static-pin-list';
import { Settings } from './components/settings';
import { Notifications, type NotificationPayload } from './components/notifications';
import { initializePageHandlers, initializePageListeners } from './services/page-handlers';

// Page components registry
const pages: Record<string, () => Promise<string>> = {
  dashboard: Dashboard,
  calendars: CalendarList,
  locks: LockList,
  'static-pins': StaticPinList,
  settings: Settings,
};

// Initialize notifications
const notifications = new Notifications();

/**
 * Navigate to a page by name.
 */
async function navigateTo(pageName: string): Promise<void> {
  const pageContent = document.getElementById('page-content');
  if (!pageContent) return;

  // Show loading state
  pageContent.innerHTML = `
    <div class="text-center py-5">
      <div class="spinner-border text-primary" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>
  `;

  // Update navigation active state
  document.querySelectorAll('.nav-link').forEach(link => {
    link.classList.toggle('active', link.getAttribute('data-page') === pageName);
  });

  // Load page content
  const pageLoader = pages[pageName];
  if (pageLoader) {
    try {
      pageContent.innerHTML = await pageLoader();
      setupPageListeners(pageName);
    } catch (error) {
      console.error(`Error loading page ${pageName}:`, error);
      pageContent.innerHTML = `
        <div class="alert alert-danger">
          <h4>Error Loading Page</h4>
          <p>Failed to load ${pageName}. Please try again.</p>
        </div>
      `;
    }
  } else {
    pageContent.innerHTML = `
      <div class="alert alert-warning">
        <h4>Page Not Found</h4>
        <p>The page "${pageName}" does not exist.</p>
      </div>
    `;
  }
}

/**
 * Initialize event handlers for the current page.
 */
function setupPageListeners(pageName: string): void {
  // Initialize page-specific event listeners
  initializePageListeners(pageName);
  console.log(`Initialized handlers for page: ${pageName}`);
}

/**
 * Update connection status indicator.
 */
function updateConnectionStatus(connected: boolean): void {
  const statusEl = document.getElementById('connection-status');
  if (!statusEl) return;

  if (connected) {
    statusEl.className = 'badge bg-success';
    statusEl.innerHTML = '<span class="status-dot connected"></span> Connected';
  } else {
    statusEl.className = 'badge bg-danger';
    statusEl.innerHTML = '<span class="status-dot disconnected"></span> Disconnected';
  }
}

/**
 * Initialize the application.
 */
async function init(): Promise<void> {
  console.log('Guest Lock Manager starting...');

  // Initialize global page handlers for inline onclick events
  initializePageHandlers();

  // Set up navigation handlers
  document.querySelectorAll('[data-page]').forEach(link => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      const pageName = (e.currentTarget as HTMLElement).getAttribute('data-page');
      if (pageName) {
        navigateTo(pageName);
        history.pushState({ page: pageName }, '', `#${pageName}`);
      }
    });
  });

  // Handle browser back/forward
  window.addEventListener('popstate', (e) => {
    const pageName = e.state?.page || 'dashboard';
    navigateTo(pageName);
  });

  // Initialize WebSocket connection
  websocket.onConnect(() => updateConnectionStatus(true));
  websocket.onDisconnect(() => updateConnectionStatus(false));
  websocket.onMessage((msg) => {
    console.log('WebSocket message:', msg);
    // Handle real-time updates
    if (msg.type === 'notification') {
      notifications.show(msg.payload as NotificationPayload);
    }
  });
  websocket.connect();

  // Check API health
  try {
    const health = await api.getHealth();
    console.log('API health:', health);
  } catch (error) {
    console.error('API health check failed:', error);
    notifications.show({
      level: 'error',
      title: 'Connection Error',
      message: 'Failed to connect to the API server.',
      dismissible: true,
    });
  }

  // Load initial page
  const initialPage = window.location.hash.slice(1) || 'dashboard';
  navigateTo(initialPage);

  console.log('Guest Lock Manager initialized');
}

// Start the application
document.addEventListener('DOMContentLoaded', init);

