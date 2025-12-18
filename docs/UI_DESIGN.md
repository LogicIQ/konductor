# Konductor UI Design Specification

## Overview
A web-based dashboard for monitoring and managing konductor synchronization primitives. Provides real-time visibility into coordination state, debugging capabilities, and operational controls.

## Core Value Proposition

### Problems UI Solves
- **Debugging**: "Why is my job stuck waiting?"
- **Monitoring**: "How many permits are in use?"
- **Operations**: "I need to force-release a stuck lease"
- **Visibility**: "What coordination is happening in my cluster?"

### UI vs CLI/kubectl
| Task | CLI | UI |
|------|-----|-----|
| **Quick Status Check** | `kondctl status semaphore api-quota` | Visual dashboard |
| **Debugging Stuck Jobs** | Multiple kubectl commands | Single view with relationships |
| **Historical Trends** | Not available | Charts and metrics |
| **Force Operations** | `kondctl force-release` | Point-and-click |
| **Multi-Primitive View** | Multiple commands | Single dashboard |

## UI Architecture

### Technology Stack
```
┌─────────────────────────────────────────┐
│              Frontend                   │
│  React/Vue.js + WebSocket for real-time│
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│              Backend                    │
│  Go HTTP Server + K8s Client + WebSocket│
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│           Kubernetes API                │
│     CRDs + Events + Watch Streams       │
└─────────────────────────────────────────┘
```

### Deployment Options
1. **Standalone Pod**: Separate UI deployment
2. **Operator Extension**: Built into konductor operator
3. **kubectl Plugin**: Local UI server

## Core UI Views

### 1. Dashboard Overview
```
┌─────────────────────────────────────────────────────────┐
│ Konductor Dashboard                    🔄 Auto-refresh  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │
│ │ Semaphores  │ │  Barriers   │ │   Leases    │        │
│ │     12      │ │      3      │ │      5      │        │
│ │  (8 active) │ │ (1 waiting) │ │ (3 active)  │        │
│ └─────────────┘ └─────────────┘ └─────────────┘        │
│                                                         │
│ Recent Activity                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 14:32 | api-quota      | Permit acquired by job-123 │ │
│ │ 14:31 | stage-2        | Barrier opened (10/10)     │ │
│ │ 14:30 | db-migration   | Lease acquired by pod-xyz  │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 2. Semaphores View
```
┌─────────────────────────────────────────────────────────┐
│ Semaphores                                              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌─ api-quota ──────────────────────────────────────────┐ │
│ │ Permits: 3/5 used    Status: Active    TTL: 5m      │ │
│ │                                                      │ │
│ │ Active Permits:                                      │ │
│ │ • job-worker-123  (2m remaining)  [Release]         │ │
│ │ • job-worker-456  (4m remaining)  [Release]         │ │
│ │ • job-worker-789  (1m remaining)  [Release]         │ │
│ │                                                      │ │
│ │ Waiting Queue: 2 jobs                               │ │
│ │ • job-worker-abc  (waiting 30s)                     │ │
│ │ • job-worker-def  (waiting 15s)                     │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─ db-connections ─────────────────────────────────────┐ │
│ │ Permits: 8/10 used   Status: Active    TTL: 10m     │ │
│ │ [View Details] [Edit] [Force Release All]           │ │
│ └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 3. Barriers View
```
┌─────────────────────────────────────────────────────────┐
│ Barriers                                                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌─ stage-1-complete ──────────────────────────────────┐ │
│ │ Status: Open ✅     Expected: 10    Arrived: 10     │ │
│ │ Opened: 2m ago      Timeout: 30m                    │ │
│ │                                                      │ │
│ │ Arrivals:                                            │ │
│ │ ✅ job-extract-1   ✅ job-extract-2   ✅ job-extract-3 │ │
│ │ ✅ job-extract-4   ✅ job-extract-5   ✅ job-extract-6 │ │
│ │ ✅ job-extract-7   ✅ job-extract-8   ✅ job-extract-9 │ │
│ │ ✅ job-extract-10                                    │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─ stage-2-ready ─────────────────────────────────────┐ │
│ │ Status: Waiting ⏳   Expected: 5     Arrived: 3      │ │
│ │ Waiting: 5m         Timeout: 1h (55m remaining)     │ │
│ │                                                      │ │
│ │ Arrivals:                                            │ │
│ │ ✅ transform-job-1  ✅ transform-job-2  ✅ transform-job-3 │ │
│ │ ⏳ transform-job-4  ⏳ transform-job-5               │ │
│ │                                                      │ │
│ │ [Force Open] [Reset]                                 │ │
│ └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 4. Leases View
```
┌─────────────────────────────────────────────────────────┐
│ Leases                                                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ ┌─ service-leader ────────────────────────────────────┐ │
│ │ Status: Held 🔒      Holder: pod-service-abc-123    │ │
│ │ Acquired: 5m ago     TTL: 30s (auto-renewing)       │ │
│ │ Priority: 1          Renewals: 10                    │ │
│ │                                                      │ │
│ │ [Force Release] [View Holder Details]                │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─ db-migration ──────────────────────────────────────┐ │
│ │ Status: Available 🔓  Last Holder: migration-job-456 │ │
│ │ Released: 2h ago     Duration: 15m                   │ │
│ │                                                      │ │
│ │ Waiting Queue: 0                                     │ │
│ └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 5. Events & Logs View
```
┌─────────────────────────────────────────────────────────┐
│ Events & Activity Log                    [Filter] [⬇️]   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ 🟢 14:35:22 | semaphore/api-quota      | Permit released │
│              by job-worker-123 (normal completion)      │
│                                                         │
│ 🟡 14:34:15 | barrier/stage-2-ready    | Timeout warning │
│              (3/5 arrived, 5m remaining)                │
│                                                         │
│ 🔵 14:33:45 | lease/service-leader     | Lease renewed   │
│              by pod-service-abc-123 (renewal #10)       │
│                                                         │
│ 🟢 14:32:10 | semaphore/db-connections | Permit acquired │
│              by batch-processor-789                     │
│                                                         │
│ 🔴 14:30:55 | lease/migration-lock     | Force released  │
│              by admin (stuck holder cleanup)            │
│                                                         │
│ 🟢 14:29:30 | barrier/stage-1-complete | Barrier opened  │
│              (10/10 arrivals reached)                   │
└─────────────────────────────────────────────────────────┘
```

## Real-Time Features

### WebSocket Updates
```javascript
// Real-time status updates
const ws = new WebSocket('ws://konductor-ui/events');
ws.onmessage = (event) => {
    const update = JSON.parse(event.data);
    switch(update.type) {
        case 'semaphore_permit_acquired':
            updateSemaphoreView(update.semaphore, update.permit);
            break;
        case 'barrier_arrival':
            updateBarrierProgress(update.barrier, update.arrival);
            break;
        case 'lease_acquired':
            updateLeaseStatus(update.lease, update.holder);
            break;
    }
};
```

### Auto-Refresh Indicators
- 🔄 Live updates via WebSocket
- ⏱️ Last updated timestamps
- 🟢 Connected / 🔴 Disconnected status
- 📊 Update frequency controls

## Administrative Features

### Force Operations
```
┌─────────────────────────────────────────┐
│ Force Release Permit                    │
├─────────────────────────────────────────┤
│ Semaphore: api-quota                    │
│ Permit ID: job-worker-123               │
│ Holder: job-worker-123                  │
│ Acquired: 15m ago                       │
│                                         │
│ ⚠️  This will forcibly release the      │
│    permit. The holder may not expect    │
│    this and could cause issues.         │
│                                         │
│ Reason: [Stuck job cleanup        ▼]   │
│                                         │
│ [Cancel]              [Force Release]   │
└─────────────────────────────────────────┘
```

### Bulk Operations
- Force release all permits for a semaphore
- Reset barrier (clear all arrivals)
- Bulk lease cleanup
- Emergency coordination reset

## Metrics & Analytics

### Historical Charts
```
Semaphore Usage Over Time
┌─────────────────────────────────────────┐
│ api-quota                               │
│                                         │
│ 5 ┤                     ╭─╮             │
│ 4 ┤           ╭─╮       │ │    ╭─╮      │
│ 3 ┤     ╭─╮   │ │   ╭─╮ │ │    │ │      │
│ 2 ┤ ╭─╮ │ │   │ │   │ │ │ │    │ │      │
│ 1 ┤ │ │ │ │   │ │   │ │ │ │    │ │      │
│ 0 └─┴─┴─┴─┴───┴─┴───┴─┴─┴─┴────┴─┴──────│
│   09:00   12:00   15:00   18:00   21:00 │
└─────────────────────────────────────────┘
```

### Performance Metrics
- Average permit hold time
- Barrier completion rates
- Lease renewal success rates
- Queue wait times
- Coordination bottlenecks

## Implementation Phases

### Phase 1: Basic UI (MVP)
- Dashboard overview
- Semaphore list view with basic details
- Real-time status updates
- Simple force operations

### Phase 2: Full Primitives
- Barrier view with arrival tracking
- Lease management interface
- Events log with filtering
- Administrative controls

### Phase 3: Advanced Features
- Historical metrics and charts
- Performance analytics
- Bulk operations
- Advanced filtering and search

### Phase 4: Integration
- Alerting integration (Slack, email)
- Export capabilities (metrics, logs)
- API for external tools
- Custom dashboards

## Technical Implementation

### Backend API
```go
// REST API for UI backend
type UIServer struct {
    kubeClient kubernetes.Interface
    wsHub      *WebSocketHub
}

// Endpoints
// GET  /api/semaphores
// GET  /api/barriers  
// GET  /api/leases
// GET  /api/events
// POST /api/semaphores/{name}/force-release
// POST /api/barriers/{name}/reset
// WebSocket /ws/events
```

### Frontend Components
```
src/
├── components/
│   ├── Dashboard.jsx
│   ├── SemaphoreList.jsx
│   ├── BarrierList.jsx
│   ├── LeaseList.jsx
│   ├── EventLog.jsx
│   └── AdminActions.jsx
├── hooks/
│   ├── useWebSocket.js
│   ├── useKonductorAPI.js
│   └── useRealTimeUpdates.js
└── utils/
    ├── api.js
    └── formatters.js
```

## Deployment Options

### Option 1: Sidecar to Operator
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: konductor-operator
spec:
  template:
    spec:
      containers:
      - name: operator
        image: konductor/operator:latest
      - name: ui
        image: konductor/ui:latest
        ports:
        - containerPort: 8080
```

### Option 2: Standalone Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: konductor-ui
spec:
  template:
    spec:
      containers:
      - name: ui
        image: konductor/ui:latest
        env:
        - name: KUBE_CONFIG
          value: "in-cluster"
---
apiVersion: v1
kind: Service
metadata:
  name: konductor-ui
spec:
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: konductor-ui
```

## Security Considerations

### RBAC Requirements
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: konductor-ui
rules:
- apiGroups: ["sync.konductor.io"]
  resources: ["semaphores", "barriers", "leases", "gates"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch"]
```

### Authentication Options
- Kubernetes ServiceAccount (in-cluster)
- OIDC integration (external access)
- Basic auth (development)
- No auth (internal networks only)

## Value Proposition Summary

**UI transforms konductor from a CLI tool to an observable platform:**

✅ **Debugging**: Visual representation of coordination state
✅ **Operations**: Point-and-click administrative actions  
✅ **Monitoring**: Real-time status and historical trends
✅ **Adoption**: Easier for teams to understand and trust the system

**This positions konductor as enterprise-ready coordination infrastructure with full observability.**