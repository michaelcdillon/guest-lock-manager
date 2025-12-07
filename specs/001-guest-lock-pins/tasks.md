# Tasks: Guest Lock PIN Manager

**Input**: Design documents from `/specs/001-guest-lock-pins/`  
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Tests**: Not explicitly requested - tests omitted per specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `backend/` with Go packages in `internal/`
- **Frontend**: `frontend/src/` with TypeScript components
- **Shared**: Root-level Dockerfile, config.yaml, README.md

---

## Phase 1: Setup (Shared Infrastructure) ✅

**Purpose**: Project initialization and basic structure

- [x] T001 Create backend directory structure per plan.md in backend/
- [x] T002 Initialize Go module with `go mod init` in backend/go.mod
- [x] T003 [P] Add Go dependencies (gorilla/websocket, go-sqlite3, robfig/cron) in backend/go.mod
- [x] T004 [P] Create frontend directory structure per plan.md in frontend/
- [x] T005 [P] Initialize npm project with TypeScript and Vite in frontend/package.json
- [x] T006 [P] Add frontend dependencies (bootstrap) in frontend/package.json
- [x] T007 [P] Configure TypeScript in frontend/tsconfig.json
- [x] T008 [P] Configure Vite build in frontend/vite.config.ts
- [x] T009 Create Home Assistant addon config in config.yaml
- [x] T010 [P] Create multi-stage Dockerfile with distroless base in Dockerfile
- [x] T011 [P] Create project README with architecture overview in README.md

---

## Phase 2: Foundational (Blocking Prerequisites) ✅

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Database & Storage

- [x] T012 Create SQLite connection wrapper in backend/internal/storage/sqlite.go
- [x] T013 Create migration runner in backend/internal/storage/migrations.go
- [x] T014 Create initial schema migration in backend/internal/storage/migrations/001_initial_schema.sql
- [x] T015 Create base repository interface in backend/internal/storage/repository.go

### API Infrastructure

- [x] T016 Create HTTP server entrypoint in backend/cmd/server/main.go
- [x] T017 Create API router with gorilla/mux in backend/internal/api/router.go
- [x] T018 [P] Create error response middleware in backend/internal/api/middleware/error.go
- [x] T019 [P] Create logging middleware in backend/internal/api/middleware/logging.go
- [x] T020 [P] Create health check handler in backend/internal/api/handlers/health.go

### WebSocket Infrastructure

- [x] T021 Create WebSocket hub for connection management in backend/internal/websocket/hub.go
- [x] T022 Create WebSocket message types in backend/internal/websocket/messages.go
- [x] T023 Create WebSocket upgrade handler in backend/internal/api/handlers/websocket.go

### Home Assistant Integration Base

- [x] T024 Create HA client configuration in backend/internal/lock/config.go
- [x] T025 Create HA API client wrapper in backend/internal/lock/homeassistant.go
- [x] T026 Create lock entity discovery via HA API in backend/internal/lock/discovery.go

### Core Entities (Shared Across Stories)

- [x] T027 Create ManagedLock model in backend/internal/storage/models/lock.go
- [x] T028 Create ManagedLock repository in backend/internal/storage/lock_repository.go

### Frontend Infrastructure

- [x] T029 Create HTML shell with Bootstrap in frontend/src/index.html
- [x] T030 Create application bootstrap in frontend/src/main.ts
- [x] T031 [P] Create API client service in frontend/src/services/api.ts
- [x] T032 [P] Create WebSocket client service in frontend/src/services/websocket.ts
- [x] T033 [P] Create Bootstrap theme customization in frontend/src/styles/main.scss

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Calendar Guest PIN Automation (Priority: P1) 🎯 MVP

**Goal**: Automatically provision door lock PINs for guests based on rental calendar events

**Independent Test**: Add a calendar with a test event containing guest phone data, verify the lock receives the correct PIN with proper validity windows

### Backend - Models & Repository

- [x] T034 [P] [US1] Create CalendarSubscription model in backend/internal/storage/models/calendar.go
- [x] T035 [P] [US1] Create GuestPIN model in backend/internal/storage/models/guest_pin.go
- [x] T036 [P] [US1] Create CalendarLockMapping model in backend/internal/storage/models/calendar_lock.go
- [x] T037 [P] [US1] Create GuestPINLock junction model in backend/internal/storage/models/guest_pin_lock.go
- [x] T038 [US1] Create calendar repository in backend/internal/storage/calendar_repository.go
- [x] T039 [US1] Create guest PIN repository in backend/internal/storage/guest_pin_repository.go

### Backend - Calendar Sync

- [x] T040 [US1] Create iCal parser in backend/internal/calendar/ical.go
- [x] T041 [US1] Create PIN extractor with 4 generation methods in backend/internal/calendar/extractor.go
- [x] T042 [US1] Create calendar sync service in backend/internal/calendar/sync.go
- [ ] T043 [US1] Create sync scheduler with robfig/cron in backend/internal/calendar/scheduler.go

### Backend - PIN Management

- [x] T044 [US1] Create PIN generator (date-based, phone-last4, description-random, custom) in backend/internal/pin/generator.go
- [x] T045 [US1] Create PIN conflict detector in backend/internal/pin/conflict.go
- [ ] T046 [US1] Create PIN activation/deactivation scheduler in backend/internal/pin/scheduler.go

### Backend - Lock Operations

- [x] T047 [US1] Create lock manager orchestration in backend/internal/lock/manager.go
- [x] T048 [US1] Create PIN sync to lock via HA API in backend/internal/lock/pin_sync.go

### Backend - API Handlers

- [x] T049 [P] [US1] Create calendar CRUD handlers in backend/internal/api/handlers/calendars.go
- [x] T050 [P] [US1] Create guest PIN handlers in backend/internal/api/handlers/guest_pins.go
- [x] T051 [US1] Create calendar sync trigger handler in backend/internal/api/handlers/calendars.go
- [x] T052 [US1] Create PIN regeneration handler in backend/internal/api/handlers/guest_pins.go

### Backend - WebSocket Events

- [x] T053 [US1] Add calendar.sync_completed event broadcast in backend/internal/websocket/events.go
- [x] T054 [US1] Add pin.status_changed event broadcast in backend/internal/websocket/events.go

### Frontend - Calendar Management UI

- [x] T055 [P] [US1] Create calendar list component in frontend/src/components/calendar-list.ts
- [x] T056 [P] [US1] Create calendar add/edit form in frontend/src/components/calendar-form.ts
- [x] T057 [US1] Create PIN table component in frontend/src/components/pin-table.ts
- [x] T058 [US1] Create dashboard overview in frontend/src/components/dashboard.ts

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Static Recurring PINs (Priority: P2)

**Goal**: Create permanent or recurring access codes for service personnel with time/day restrictions

**Independent Test**: Create a time-restricted PIN and verify it only grants access during configured windows

### Backend - Models & Repository

- [ ] T059 [P] [US2] Create StaticPIN model in backend/internal/storage/models/static_pin.go
- [ ] T060 [P] [US2] Create StaticPINSchedule model in backend/internal/storage/models/static_pin_schedule.go
- [ ] T061 [P] [US2] Create StaticPINLock junction model in backend/internal/storage/models/static_pin_lock.go
- [ ] T062 [US2] Create static PIN repository in backend/internal/storage/static_pin_repository.go

### Backend - Schedule Management

- [ ] T063 [US2] Create schedule evaluator (is PIN active now?) in backend/internal/pin/schedule.go
- [ ] T064 [US2] Create static PIN activation scheduler in backend/internal/pin/static_scheduler.go
- [ ] T065 [US2] Integrate static PIN sync with lock manager in backend/internal/lock/manager.go

### Backend - API Handlers

- [x] T066 [US2] Create static PIN CRUD handlers in backend/internal/api/handlers/static_pins.go
- [x] T067 [US2] Create schedule management within static PIN handlers in backend/internal/api/handlers/static_pins.go
- [x] T067a [US2] Add unique name validation (case-insensitive) for static PINs in create/update handlers

### Frontend - Static PIN UI

- [ ] T068 [P] [US2] Create static PIN list component in frontend/src/components/static-pin-list.ts
- [ ] T069 [US2] Create static PIN form with schedule builder in frontend/src/components/static-pin-form.ts
- [ ] T070 [US2] Create day/time picker component in frontend/src/components/schedule-picker.ts

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Lock Discovery and Management (Priority: P3)

**Goal**: Discover and configure compatible locks from Home Assistant for PIN management

**Independent Test**: Connect to a Home Assistant instance with configured locks and verify they appear in the UI with correct status

### Backend - Lock Discovery Enhancement

- [ ] T071 [US3] Enhance lock discovery with protocol detection in backend/internal/lock/discovery.go
- [ ] T072 [US3] Create lock slot capacity detection in backend/internal/lock/capacity.go
- [ ] T073 [US3] Create lock online status monitor via HA WebSocket in backend/internal/lock/monitor.go

### Backend - API Handlers

- [ ] T074 [P] [US3] Create lock CRUD handlers in backend/internal/api/handlers/locks.go
- [ ] T075 [P] [US3] Create lock discovery trigger handler in backend/internal/api/handlers/locks.go
- [ ] T076 [US3] Create lock PIN listing handler in backend/internal/api/handlers/locks.go

### Backend - WebSocket Events

- [ ] T077 [US3] Add lock.status_changed event broadcast in backend/internal/websocket/events.go

### Frontend - Lock Management UI

- [ ] T078 [P] [US3] Create lock list component in frontend/src/components/lock-list.ts
- [ ] T079 [US3] Create lock configuration form in frontend/src/components/lock-form.ts
- [ ] T080 [US3] Create lock status indicator component in frontend/src/components/lock-status.ts
- [ ] T081 [US3] Add lock selection to calendar form in frontend/src/components/calendar-form.ts

**Checkpoint**: At this point, User Stories 1, 2 AND 3 should all work independently

---

## Phase 6: User Story 4 - Multi-Calendar Support (Priority: P4)

**Goal**: Subscribe to multiple calendars and route each to appropriate locks

**Independent Test**: Add two calendars and verify each calendar's events generate PINs only on assigned locks

### Backend - Multi-Calendar Routing

- [ ] T082 [US4] Enhance calendar-lock mapping with M:N support in backend/internal/storage/calendar_repository.go
- [ ] T083 [US4] Create PIN deduplication logic (same guest on multiple platforms) in backend/internal/pin/deduplication.go
- [ ] T084 [US4] Add calendar lock assignment API in backend/internal/api/handlers/calendars.go

### Frontend - Multi-Calendar UI

- [ ] T085 [US4] Enhance calendar list with lock assignment display in frontend/src/components/calendar-list.ts
- [ ] T086 [US4] Create lock assignment picker in calendar form in frontend/src/components/calendar-form.ts
- [ ] T087 [US4] Create multi-calendar dashboard view in frontend/src/components/dashboard.ts

**Checkpoint**: At this point, User Stories 1-4 should all work independently

---

## Phase 7: User Story 5 - Battery-Efficient Direct Protocol Integration (Priority: P5)

**Goal**: Use direct Z-Wave JS UI or Zigbee2MQTT communication for battery efficiency

**Independent Test**: Compare lock communication frequency between HA mode and direct protocol mode

### Backend - Z-Wave JS UI Integration

- [ ] T088 [P] [US5] Create Z-Wave JS UI WebSocket client in backend/internal/lock/zwave.go
- [ ] T089 [US5] Create Z-Wave JS UI detection via HA addons API in backend/internal/lock/detection.go
- [ ] T090 [US5] Implement PIN batching for Z-Wave in backend/internal/lock/zwave.go

### Backend - Zigbee2MQTT Integration

- [ ] T091 [P] [US5] Create MQTT client for Zigbee2MQTT in backend/internal/lock/zigbee.go
- [ ] T092 [US5] Create Zigbee2MQTT detection in backend/internal/lock/detection.go
- [ ] T093 [US5] Implement PIN batching for Zigbee in backend/internal/lock/zigbee.go

### Backend - Batching & Efficiency

- [ ] T094 [US5] Create PIN operation batcher in backend/internal/lock/batcher.go
- [ ] T095 [US5] Create battery impact estimator in backend/internal/lock/battery.go
- [ ] T096 [US5] Integrate direct protocols with lock manager fallback in backend/internal/lock/manager.go

### Backend - Status API

- [ ] T097 [US5] Create system status handler with integration detection in backend/internal/api/handlers/status.go
- [ ] T098 [US5] Create settings handler (battery mode, PIN length 4-8, check-in/out times) in backend/internal/api/handlers/settings.go

### Frontend - System Status UI

- [ ] T099 [US5] Create system status display in frontend/src/components/system-status.ts
- [ ] T100 [US5] Create settings page with battery mode toggle in frontend/src/components/settings.ts

**Checkpoint**: All user stories should now be independently functional

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T101 [P] Create notification system for errors and alerts in backend/internal/websocket/notifications.go
- [ ] T102 [P] Add notification display component in frontend/src/components/notifications.ts
- [ ] T103 [P] Create ADR documentation in docs/adr/001-architecture.md
- [ ] T104 Code cleanup and refactoring pass across backend/
- [ ] T105 [P] Add GoDoc comments to all public functions in backend/
- [ ] T106 Performance optimization for calendar sync in backend/internal/calendar/sync.go
- [ ] T107 Security review for API endpoints in backend/internal/api/
- [ ] T108 Validate quickstart.md instructions work end-to-end
- [ ] T109 [P] Create CHANGELOG.md in CHANGELOG.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - can start immediately
- **Phase 2 (Foundational)**: Depends on Setup completion - BLOCKS all user stories
- **Phase 3-7 (User Stories)**: All depend on Foundational phase completion
  - Stories can proceed in parallel (if staffed) or sequentially
- **Phase 8 (Polish)**: Depends on all desired user stories being complete

### User Story Dependencies

| Story | Depends On | Can Start After |
|-------|------------|-----------------|
| US1 (Calendar Guest PIN) | Phase 2 | Phase 2 complete |
| US2 (Static PINs) | Phase 2 | Phase 2 complete |
| US3 (Lock Discovery) | Phase 2 | Phase 2 complete |
| US4 (Multi-Calendar) | US1 (extends it) | US1 complete |
| US5 (Direct Protocol) | US1 or US2 (needs PIN sync) | US1 or US2 complete |

### Within Each User Story

1. Models before repositories
2. Repositories before services
3. Services before API handlers
4. Backend before frontend components
5. Core features before enhancements

### Parallel Opportunities

**Phase 2 (Foundational)**:
```
T018, T019, T020 can run in parallel (different middleware files)
T031, T032, T033 can run in parallel (different frontend services)
```

**Phase 3 (US1)**:
```
T034, T035, T036, T037 can run in parallel (different model files)
T049, T050 can run in parallel (different handler files)
T055, T056 can run in parallel (different components)
```

**Across Stories** (after Phase 2):
```
US1, US2, US3 can all start in parallel
- Developer A: US1 (Calendar Guest PIN)
- Developer B: US2 (Static PINs)  
- Developer C: US3 (Lock Discovery)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Calendar Guest PIN Automation)
4. **STOP and VALIDATE**: Add a test calendar, verify PIN creation and lock sync
5. Deploy to Home Assistant for real-world testing

**MVP Scope**: 58 tasks (T001-T058)

### Incremental Delivery

1. Setup + Foundational → Foundation ready (T001-T033)
2. Add US1 → Test independently → Deploy (T034-T058) **← MVP!**
3. Add US2 → Test independently → Deploy (T059-T070)
4. Add US3 → Test independently → Deploy (T071-T081)
5. Add US4 → Test independently → Deploy (T082-T087)
6. Add US5 → Test independently → Deploy (T088-T100)
7. Polish → Final release (T101-T109)

### Parallel Team Strategy

With 3 developers after Phase 2:

```
Developer A: US1 (Calendar Guest PIN) - Core value
Developer B: US2 (Static PINs) - Complementary feature
Developer C: US3 (Lock Discovery) - Infrastructure enhancement
```

---

## Task Summary

| Phase | Story | Task Count | Parallel Tasks |
|-------|-------|------------|----------------|
| 1 | Setup | 11 | 8 |
| 2 | Foundational | 22 | 5 |
| 3 | US1 (P1) | 25 | 10 |
| 4 | US2 (P2) | 12 | 3 |
| 5 | US3 (P3) | 11 | 3 |
| 6 | US4 (P4) | 6 | 0 |
| 7 | US5 (P5) | 13 | 2 |
| 8 | Polish | 9 | 5 |
| **Total** | | **109** | **36** |

---

## Notes

- [P] tasks = different files, no dependencies within same phase
- [Story] label maps task to specific user story for traceability
- Each user story checkpoint allows independent validation
- Commit after each task or logical group
- File paths are relative to repository root
- Backend uses Go 1.22+, Frontend uses TypeScript 5.x
- All API endpoints follow OpenAPI spec in contracts/openapi.yaml
- WebSocket events follow spec in contracts/websocket.md

