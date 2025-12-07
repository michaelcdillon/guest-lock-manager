# Feature Specification: Guest Lock PIN Manager

**Feature Branch**: `001-guest-lock-pins`  
**Created**: 2025-12-07  
**Status**: Draft  
**Input**: User description: "Home Assistant addon for managing IOT lock PINs via short-term rental calendars with static PIN scheduling and battery-efficient lock communication"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Calendar Guest PIN Automation (Priority: P1)

A short-term rental host wants to automatically provision door lock PINs for guests 
based on their rental calendar, without manually programming codes for each booking.

The host subscribes their Airbnb/VRBO calendar to the addon. When a new booking 
appears, the system automatically extracts the guest's phone number from the event 
details and programs the last 4 digits as a temporary PIN on the designated lock(s). 
The PIN activates at check-in time and deactivates at checkout.

**Why this priority**: This is the core value proposition—automating the most tedious
and error-prone task for rental hosts. Without this, the addon provides no unique value.

**Independent Test**: Can be fully tested by adding a calendar with a test event 
containing guest phone data, and verifying the lock receives the correct PIN with 
proper validity windows.

**Acceptance Scenarios**:

1. **Given** a subscribed calendar with a booking event containing "Last 4 Digits: 4567" 
   for Dec 15-18, **When** the system syncs the calendar, **Then** PIN "4567" is 
   programmed on selected locks with validity from Dec 15 check-in to Dec 18 checkout.

2. **Given** a booking event without phone digits in description for Dec 20-23, 
   **When** the system syncs, **Then** a date-based PIN is generated (e.g., "2023" 
   from day 20 + day 23) and programmed automatically.

3. **Given** an active guest PIN on a lock, **When** the checkout time passes, 
   **Then** the PIN is removed from the lock within 15 minutes.

4. **Given** overlapping bookings (same-day turnover), **When** both events exist, 
   **Then** the outgoing guest's PIN is removed at checkout and incoming guest's 
   PIN activates at check-in, with no overlap period.

5. **Given** a future reservation with a generated PIN, **When** the host triggers 
   PIN regeneration, **Then** a new PIN is generated and the lock is updated.

---

### User Story 2 - Static Recurring PINs (Priority: P2)

A host needs permanent or recurring access codes for cleaners, maintenance workers, 
or property managers who need regular access on specific days/times.

The host creates a "Cleaning Service" static PIN that only works on Saturdays 
from 10am-2pm. They create a "Property Manager" PIN that works 24/7. Each PIN 
has a friendly name for identification.

**Why this priority**: Static PINs complement guest PINs by handling the other 
major access use case. Many hosts need both automated guest codes and permanent 
service codes.

**Independent Test**: Can be fully tested by creating a time-restricted PIN and 
verifying it only grants access during the configured windows.

**Acceptance Scenarios**:

1. **Given** a static PIN configured for "Saturdays 10:00-14:00", **When** it is 
   Saturday at 11:00, **Then** the PIN is active on the lock.

2. **Given** a static PIN configured for "Saturdays 10:00-14:00", **When** it is 
   Friday at 11:00, **Then** the PIN is not active on the lock.

3. **Given** a static PIN with no time restrictions, **When** any time/day occurs, 
   **Then** the PIN remains continuously active.

4. **Given** an active static PIN, **When** the host edits the schedule, **Then** 
   the lock is updated to reflect new valid times within 5 minutes.

5. **Given** an existing static PIN named "Cleaner - Maria", **When** the host 
   attempts to create another PIN with the same name (case-insensitive), **Then** 
   the system rejects the request with a conflict error.

6. **Given** an existing static PIN named "Cleaner - Maria" with code "5678", 
   **When** the host creates a PIN named "Backup Cleaner" with code "5678", 
   **Then** the system allows it (same code, different name is permitted).

---

### User Story 3 - Lock Discovery and Management (Priority: P3)

A host with multiple properties or locks needs to see all compatible locks in their 
Home Assistant installation and assign them to the PIN management system.

The host opens the addon UI, sees a list of discovered locks, selects which locks 
should receive guest PINs and/or static PINs, and configures the PIN slot allocation 
for each lock.

**Why this priority**: Without lock discovery, the host cannot use the core features. 
However, this is foundational infrastructure rather than direct user value.

**Independent Test**: Can be fully tested by connecting to a Home Assistant instance 
with configured locks and verifying they appear in the UI with correct status.

**Acceptance Scenarios**:

1. **Given** Home Assistant has 3 locks configured (2 Z-Wave, 1 Zigbee), **When** 
   the user opens the addon, **Then** all 3 locks appear in the device list with 
   their protocol type displayed.

2. **Given** a discovered lock, **When** the user selects it for management, 
   **Then** they can see how many PIN slots the lock supports and allocate slots 
   between guest and static PINs.

3. **Given** a managed lock goes offline, **When** the user views the dashboard, 
   **Then** the lock shows an offline status with last-seen timestamp.

---

### User Story 4 - Multi-Calendar Support (Priority: P4)

A host with multiple rental units (or multiple platforms for one unit) needs to 
subscribe to several calendars and route each to the appropriate lock(s).

The host adds their Airbnb calendar for Unit A, VRBO calendar for Unit A, and 
Airbnb calendar for Unit B. They configure Unit A's lock to receive PINs from 
both Unit A calendars, and Unit B's lock from its calendar.

**Why this priority**: Many hosts have multiple units or list on multiple platforms. 
This extends P1's value to real-world multi-property scenarios.

**Independent Test**: Can be fully tested by adding two calendars and verifying 
each calendar's events generate PINs only on the assigned lock(s).

**Acceptance Scenarios**:

1. **Given** calendars A and B assigned to Lock 1, calendar C assigned to Lock 2, 
   **When** events appear in all calendars, **Then** Lock 1 receives PINs from 
   A and B only, Lock 2 receives PINs from C only.

2. **Given** the same guest books on both Airbnb and VRBO (duplicate booking), 
   **When** both events sync, **Then** the system creates only one PIN (deduplication 
   by phone number or date range).

---

### User Story 5 - Battery-Efficient Direct Protocol Integration (Priority: P5)

A technically-oriented host running Z-Wave JS UI or Zigbee2MQTT wants the addon 
to use direct protocol communication for maximum battery efficiency.

When the addon detects Z-Wave JS UI is installed, it uses the Z-Wave JS websocket 
API directly instead of going through Home Assistant's abstraction layer. This 
batches PIN operations and minimizes lock wake-ups.

**Why this priority**: Battery efficiency is explicitly critical per requirements, 
but this optimization layer can be added after core functionality works through 
standard Home Assistant APIs.

**Independent Test**: Can be fully tested by comparing lock battery drain and 
communication frequency between standard HA mode and direct protocol mode over 
a 1-week period.

**Acceptance Scenarios**:

1. **Given** Z-Wave JS UI is detected, **When** multiple PIN changes are queued 
   within 30 seconds, **Then** they are batched into a single lock communication.

2. **Given** Zigbee2MQTT is detected, **When** PIN operations occur, **Then** 
   the addon uses MQTT directly rather than HA entity services.

3. **Given** neither Z-Wave JS UI nor Zigbee2MQTT is detected, **When** PIN 
   operations occur, **Then** the addon falls back to standard HA services gracefully.

---

### Edge Cases

- **Duplicate PINs**: Two different guests generate the same PIN (e.g., same last-4 
  or same check-in/out pattern) for overlapping periods—system must detect conflict 
  and alert host, suggesting regeneration for one reservation.
- **Lock at PIN capacity**: All slots full when new guest booking arrives—system 
  must alert host and suggest cleanup of expired codes.
- **Calendar sync failure**: Calendar URL becomes unreachable—system must retain 
  existing PINs, retry with backoff, and alert host after sustained failure.
- **Lock offline during PIN change**: Lock unavailable when PIN update needed—system 
  must queue change and apply when lock reconnects.
- **Timezone handling**: Calendar in different timezone than property location—system 
  must normalize all times to property's local timezone.
- **Event description changes**: Guest event description modified after PIN generated 
  via description-random method—system should warn host that PIN may change on 
  next sync if regeneration is enabled, or preserve existing PIN if already 
  programmed to lock.
- **Event date changes**: Reservation dates modified after date-based PIN generated—
  system should handle gracefully: update PIN if future reservation, preserve if 
  guest already checked in.
- **Phone pattern not found**: Event description lacks "(Last 4 Digits): XXXX" 
  pattern—system falls back to next generation method without error.

## Requirements *(mandatory)*

### Functional Requirements

**Calendar Management**
- **FR-001**: System MUST accept iCal/ICS calendar URLs for subscription
- **FR-002**: System MUST sync calendars at a configurable interval (default: 15 minutes)
- **FR-003**: System MUST extract guest phone numbers from calendar event descriptions 
  or custom fields using configurable patterns
- **FR-004**: System MUST support multiple calendars with independent lock assignments

**PIN Generation**
- **FR-005**: System MUST support four PIN generation methods with the following 
  priority order (first successful method wins):
  1. **Owner Custom Value**: User-specified PIN for a specific reservation (highest priority)
  2. **Phone Last-4 Extraction**: Parse "(Last 4 Digits): XXXX" or "Last 4 Digits: XXXX" 
     patterns from event description (most stable automated method)
  3. **Description-Based Random**: Generate deterministic 4+ digit code derived from 
     event description content (changes if description changes)
  4. **Date-Based Code (Default/Fallback)**: Generate 4+ digit code from check-in day 
     combined with check-out day (always succeeds, changes if dates change)
- **FR-006**: System MUST use Date-Based Code as automatic fallback when all other 
  configured generation methods fail to produce a valid PIN
- **FR-007**: System MUST allow optional PIN regeneration for reservations where 
  start or end dates are at least 1 day in the future
- **FR-008**: System MUST detect and alert on PIN collisions (same PIN, overlapping dates)
- **FR-009**: System MUST allow configuring minimum PIN length (default: 4 digits, 
  configurable to 4-8 digits)

**Static PIN Management**  
- **FR-010**: System MUST support creating named static PINs with user-defined codes
- **FR-011**: System MUST support time-of-day restrictions (start time, end time)
- **FR-012**: System MUST support day-of-week restrictions (any combination of days)
- **FR-013**: System MUST support "always active" static PINs with no restrictions
- **FR-013a**: System MUST enforce unique static PIN names (case-insensitive) to prevent 
  confusion. Duplicate PIN codes are allowed across different names to support use cases 
  like giving multiple people the same code with different tracking names.

**Lock Integration**
- **FR-014**: System MUST discover locks through Home Assistant's device registry
- **FR-015**: System MUST support any lock exposing standard Home Assistant lock 
  entity services (protocol-agnostic via HA)
- **FR-016**: System MUST track available PIN slots per lock and prevent over-allocation
- **FR-017**: System MUST detect and utilize Z-Wave JS UI when present for direct 
  Z-Wave communication
- **FR-018**: System MUST detect and utilize Zigbee2MQTT when present for direct 
  MQTT communication

**Battery Efficiency**
- **FR-019**: System MUST batch PIN operations when multiple changes are pending 
  for the same lock
- **FR-020**: System MUST minimize lock wake-ups by scheduling non-urgent updates 
  during optimal windows (when lock is already awake for other operations, or 
  batched with other pending changes within a 30-second window)
- **FR-021**: System MUST track and display estimated battery impact of operations

**User Interface**
- **FR-022**: System MUST provide a web-based configuration UI accessible from 
  Home Assistant's addon panel
- **FR-023**: System MUST display current PIN status for all managed locks
- **FR-024**: System MUST provide notifications for errors and required actions
- **FR-025**: System MUST allow manual override/addition of PINs for edge cases

### Assumptions

- Date-based fallback ensures every reservation can receive a PIN even without 
  phone data or stable descriptions
- Locks support at least 4 programmable PIN slots (industry standard for smart locks)
- Home Assistant installation is version 2023.1 or newer (for current entity services)
- Property timezone is configured correctly in Home Assistant
- Calendar events contain check-in and check-out dates (required for date-based 
  PIN generation and validity windows)

### Key Entities

- **Calendar Subscription**: Represents a subscribed rental calendar (URL, name, 
  sync status, assigned locks, last sync timestamp)
- **Guest PIN**: Auto-generated temporary PIN derived from calendar event (source 
  event, PIN code, generation method used, validity start/end, assigned locks, 
  status, regeneration eligibility)
- **PIN Generation Method**: Strategy used to derive PIN code (owner-custom, 
  phone-last4, description-random, date-based). Includes success/failure status 
  and fallback chain.
- **Static PIN**: User-defined recurring PIN (name, code, day/time restrictions, 
  assigned locks, enabled status)
- **Managed Lock**: A lock under addon management (HA entity reference, protocol 
  type, total slots, allocated slots, online status, battery level)
- **PIN Slot**: A specific code position on a lock (slot number, assigned PIN 
  reference, current state)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Hosts can configure a calendar subscription and see guest PINs 
  auto-provisioned within 20 minutes of calendar sync
- **SC-002**: 95% of guest check-ins have correct PIN programmed before arrival 
  (measured as PIN active at check-in time)
- **SC-003**: Lock battery consumption increases by no more than 20% compared to 
  baseline with no PIN management activity
- **SC-004**: Static PIN schedule changes take effect within 5 minutes of configuration
- **SC-005**: System handles 10 calendars and 20 locks without performance degradation 
  for typical vacation rental operations
- **SC-006**: Host setup from addon install to first working guest PIN takes under 
  15 minutes for users familiar with Home Assistant
- **SC-007**: System achieves 99% uptime for PIN management (excluding Home Assistant 
  or lock hardware failures)
- **SC-008**: PIN conflicts and errors are surfaced to the user within 1 minute 
  of detection
