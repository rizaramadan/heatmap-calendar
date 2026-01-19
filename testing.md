# Heatmap Internal - Browser MCP Testing Guide

This document provides instructions for an LLM with browser MCP capabilities to test the Heatmap Internal application.

## Prerequisites

- The service is already running at `http://localhost:8080`
- Browser MCP tools are available and ready to use
- API Key for protected endpoints: check `.env` file for `API_KEY` value

---

## Test Scenarios

### 1. Homepage & Navigation Tests

#### Test 1.1: Load Homepage
1. Navigate to `http://localhost:8080/`
2. **Expected**: Page loads successfully with the heatmap interface
3. **Verify**:
   - Page title contains "Heatmap"
   - Entity selector dropdown is present

```
Action: mcp_io_github_chr_new_page with url "http://localhost:8080/"
Verify: Take snapshot and confirm heatmap interface elements are present
```

#### Test 1.2: Entity Selection
1. On the homepage, locate the entity selector dropdown
2. Select an entity from the dropdown
3. **Expected**: Heatmap grid loads for the selected entity
4. **Verify**: Heatmap grid appears with day cells

---

### 2. Heatmap Functionality Tests

#### Test 2.1: View Heatmap for Entity
1. Navigate to `http://localhost:8080/?entity={entity_id}` (replace with valid entity)
2. **Expected**: Heatmap grid displays with colored cells indicating load levels
3. **Verify**:
   - Month headers are visible
   - Day cells are rendered with appropriate colors

#### Test 2.2: View Day Details
1. Load heatmap for an entity
2. Click on a specific day cell
3. **Expected**: Day details panel shows tasks/loads for that day
4. **Verify**: Task list or "no tasks" message appears

---

### 3. Public API Endpoint Tests

#### Test 3.1: List All Entities
1. Navigate to `http://localhost:8080/api/entities`
2. **Expected**: JSON response with array of entities
3. **Verify**: Response contains entity objects with `id`, `title`, `type` fields

```
Action: mcp_io_github_chr_new_page with url "http://localhost:8080/api/entities"
Verify: JSON response with entity list
```

#### Test 3.2: Filter Entities by Type (Persons)
1. Navigate to `http://localhost:8080/api/entities?type=person`
2. **Expected**: JSON response with only person entities
3. **Verify**: All entities have `type: "person"`

#### Test 3.3: Filter Entities by Type (Groups)
1. Navigate to `http://localhost:8080/api/entities?type=group`
2. **Expected**: JSON response with only group entities
3. **Verify**: All entities have `type: "group"`

#### Test 3.4: Get Single Entity
1. First, get entity list from `/api/entities`
2. Take an entity ID from the response
3. Navigate to `http://localhost:8080/api/entities/{id}`
4. **Expected**: JSON response with single entity details

#### Test 3.5: Get Heatmap Data (API)
1. Navigate to `http://localhost:8080/api/heatmap/{entity_id}`
2. **Expected**: HTML partial with heatmap grid for the entity

#### Test 3.6: Get Day Details (API)
1. Navigate to `http://localhost:8080/api/heatmap/{entity_id}/day/2026-01-19`
2. **Expected**: HTML partial with day task details

---

### 4. Protected API Tests (Require x-api-key Header)

> **Note**: These tests require the `x-api-key` header. Use `mcp_io_github_chr_evaluate_script` to make fetch requests with headers.

#### Test 4.1: Create a Person Entity
```javascript
// Execute via evaluate_script
async () => {
  const response = await fetch('http://localhost:8080/api/entities', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': 'YOUR_API_KEY'  // Replace with actual API key from .env
    },
    body: JSON.stringify({
      id: 'test.user@example.com',
      title: 'Test User',
      type: 'person',
      default_capacity: 1.0
    })
  });
  return await response.json();
}
```
**Expected**: 201 response with created entity

#### Test 4.2: Create a Group Entity
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/entities', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': 'YOUR_API_KEY'
    },
    body: JSON.stringify({
      id: 'test-group',
      title: 'Test Group',
      type: 'group',
      default_capacity: 5.0
    })
  });
  return await response.json();
}
```

#### Test 4.3: Add Member to Group
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/groups/test-group/members', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': 'YOUR_API_KEY'
    },
    body: JSON.stringify({
      person_email: 'test.user@example.com'
    })
  });
  return await response.json();
}
```

#### Test 4.4: Get Group Members
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/groups/test-group/members', {
    headers: {
      'x-api-key': 'YOUR_API_KEY'
    }
  });
  return await response.json();
}
```

#### Test 4.5: Upsert a Load (Create Task)
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/loads/upsert', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': 'YOUR_API_KEY'
    },
    body: JSON.stringify({
      external_id: 'test-task-001',
      title: 'Test Task for Heatmap',
      date: '2026-01-20',
      source: 'browser-test',
      assignees: [
        { email: 'test.user@example.com', weight: 1.0 }
      ]
    })
  });
  return await response.json();
}
```
**Expected**: 200 response with `{ success: true, load_id: <number> }`

#### Test 4.6: Add Assignee to Existing Load
```javascript
async () => {
  const loadId = 1;  // Replace with actual load_id from previous test
  const response = await fetch(`http://localhost:8080/api/loads/${loadId}/assignees`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-api-key': 'YOUR_API_KEY'
    },
    body: JSON.stringify({
      assignees: [
        { email: 'another.user@example.com', weight: 0.5 }
      ]
    })
  });
  return await response.json();
}
```

#### Test 4.7: Remove Assignee from Load
```javascript
async () => {
  const loadId = 1;
  const email = 'another.user@example.com';
  const response = await fetch(`http://localhost:8080/api/loads/${loadId}/assignees/${email}`, {
    method: 'DELETE',
    headers: {
      'x-api-key': 'YOUR_API_KEY'
    }
  });
  return await response.json();
}
```

#### Test 4.8: Remove Member from Group
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/groups/test-group/members/test.user@example.com', {
    method: 'DELETE',
    headers: {
      'x-api-key': 'YOUR_API_KEY'
    }
  });
  return await response.json();
}
```

#### Test 4.9: Delete Entity
```javascript
async () => {
  const response = await fetch('http://localhost:8080/api/entities/test.user@example.com', {
    method: 'DELETE',
    headers: {
      'x-api-key': 'YOUR_API_KEY'
    }
  });
  return await response.json();
}
```

---

### 5. Swagger Documentation Test

#### Test 5.1: Access Swagger UI
1. Navigate to `http://localhost:8080/api/doc/index.html`
2. **Expected**: Swagger UI loads with API documentation
3. **Verify**:
   - API title "Heatmap Internal API" is visible
   - Endpoint sections for Entities, Loads, Groups are present

```
Action: mcp_io_github_chr_new_page with url "http://localhost:8080/api/doc/index.html"
Verify: Swagger UI renders with API documentation
```

---

### 6. UI/UX Tests

#### Test 6.1: Responsive Layout Check
1. Load homepage at `http://localhost:8080/`
2. Resize the page to mobile dimensions (e.g., 375x667)
3. **Expected**: Page layout adapts to smaller screen
4. **Verify**: No horizontal scrolling, elements stack appropriately

---

## Test Execution Flow

For comprehensive testing, execute tests in this order:

```
1. Homepage Tests (1.1, 1.2)
   ↓
2. Public API Tests (3.1-3.6)
   ↓
3. Swagger Test (5.1)
   ↓
4. Protected API Tests - Create Data (4.1-4.5)
   ↓
5. Heatmap View Tests (2.1, 2.2) - verify created data shows
   ↓
6. Protected API Tests - Cleanup (4.6-4.9)
   ↓
7. UI/UX Tests (6.1)
```

---

## MCP Tool Usage Examples

### Navigate to a page
```
Tool: mcp_io_github_chr_new_page
Parameters: { "url": "http://localhost:8080/" }
```

### Take a snapshot (get page content)
```
Tool: activate_snapshot_capture_tools (first activate)
Then use snapshot tool to capture page content and get element UIDs
```

### Click an element
```
Tool: mcp_io_github_chr_click
Parameters: { "uid": "<element-uid-from-snapshot>" }
```

### Wait for content
```
Tool: mcp_io_github_chr_wait_for
Parameters: { "text": "expected text on page" }
```

### Execute API calls with headers (for protected endpoints)
```
Tool: mcp_io_github_chr_evaluate_script
Parameters: {
  "function": "async () => { const r = await fetch('http://localhost:8080/api/entities', { method: 'POST', headers: { 'Content-Type': 'application/json', 'x-api-key': 'YOUR_KEY' }, body: JSON.stringify({ id: 'test@example.com', title: 'Test', type: 'person' }) }); return await r.json(); }"
}
```

---

## Expected Test Results Summary

| Test | Expected Result |
|------|-----------------|
| 1.1 Homepage | Page loads with heatmap UI |
| 1.2 Entity Selection | Heatmap renders for entity |
| 2.1 View Heatmap | Colored grid with months |
| 2.2 Day Details | Task list or empty message |
| 3.1 List Entities | JSON array of entities |
| 3.2 Filter Persons | Only person entities |
| 3.3 Filter Groups | Only group entities |
| 3.4 Get Entity | Single entity JSON |
| 3.5 Heatmap API | HTML partial response |
| 3.6 Day Details API | HTML partial response |
| 4.1 Create Person | 201 with entity |
| 4.2 Create Group | 201 with entity |
| 4.3 Add Member | 200 success |
| 4.4 Get Members | JSON with members |
| 4.5 Upsert Load | 200 with load_id |
| 4.6 Add Assignee | 200 success |
| 4.7 Remove Assignee | 200 success |
| 4.8 Remove Member | 200 success |
| 4.9 Delete Entity | 200 success |
| 5.1 Swagger UI | API docs rendered |
| 6.1 Responsive | Mobile-friendly layout |

---

## Troubleshooting

### Page Not Loading
- Verify service is running on port 8080
- Check if `make dev` command succeeded
- Look for error messages in terminal

### API Returns 401 Unauthorized
- Check that `x-api-key` header is included
- Verify the API key matches the `API_KEY` value in `.env` file

### API Returns 500
- Database might not be initialized
- Check server logs for migration errors

### Element Not Found
- Use snapshot to get current page structure
- UIDs change between page loads - refresh snapshot before interactions

### Entity/Load Not Found (404)
- Ensure the entity/load was created first
- Check IDs match exactly (case-sensitive)
