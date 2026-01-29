# Discord Frolf Bot Refactoring Plan

## 🚦 Progress Tracking

**Last Updated:** 2026-01-29

### Main Phases (Handler Layer)
| Phase | Module | Status | Notes |
|-------|--------|--------|-------|
| **Phase 1** | `discordgo` (Foundation) | ✅ **COMPLETE** | FakeSession created, tests converted |
| **Phase 2** | `guild` (handlers) | ✅ **COMPLETE** | Handlers restructured |
| **Phase 3** | `user` (handlers) | ✅ **COMPLETE** | Handlers restructured |
| **Phase 4** | `leaderboard` (handlers) | ✅ **COMPLETE** | Handlers restructured |
| **Phase 5** | `round` (handlers) | ⏳ **PENDING** | Handlers restructured |
| **Phase 5.5** | `auth` (new module) | ✅ **COMPLETE** | PWA→Auth migration + tests |
| **Phase 6** | Integration & Cleanup | 🔲 Not Started | Full test suite, CI/CD, docs |

### Implementation Layer (GoMock → FakeSession Conversion)
| Phase | Module | Files | Status |
|-------|--------|-------|--------|
| **Phase 2a** | `guild/discord` | 5 files | ✅ **COMPLETE** |
| **Phase 2b** | `guild/handlers` (remaining mocks) | 2 files | ✅ Completed |
| **Phase 3a** | `user/discord` | 9 files | ✅ **COMPLETE** |
| **Phase 3b** | `user/handlers` + `module_test` | 2 files | ✅ **COMPLETE** |
| **Phase 4a** | `leaderboard/discord` | 5 files | 🔲 Not Started |
| **Phase 4b** | `leaderboard/handlers` (remaining mocks) | 2 files | 🔲 Not Started |
| **Phase 5a** | `round/discord` | 30 files | 🔲 Not Started |
| **Phase 5.5** | `auth` (new module with tests) | 4 files | ✅ **COMPLETE** |
| **Misc** | `bot/bot_test.go` | 1 file | 🔲 Not Started |

**Total remaining GoMock files:** 38 files

---

### Phase 5 Details (Completed)

**Summary:** Handler layer fully refactored. All 25 handler test files use fakes.

**Files Created:**
- `app/round/handlers/fake_test.go` (475 lines) - Fakes for all 10 managers

**Files Modified:**
- `app/round/module.go` - Fixed import naming collision
- `app/round/discord/discord_test.go` - Converted to FakeSession
- `app/bot/bot.go` - Updated roundrouter import path

**Directories Restructured:**
- `app/round/watermill/handlers/` → `app/round/handlers/` ✅
- `app/round/watermill/router.go` → `app/round/router/router.go` ✅
- `app/round/watermill/` → Deleted (empty) ✅
- `app/round/mocks/` → Deleted ✅

**Fakes Created:**
- `FakeRoundDiscord` - Main interface fake
- `FakeCreateRoundManager`
- `FakeRoundRsvpManager`
- `FakeRoundReminderManager`
- `FakeStartRoundManager`
- `FakeScoreRoundManager`
- `FakeFinalizeRoundManager`
- `FakeDeleteRoundManager`
- `FakeUpdateRoundManager`
- `FakeTagUpdateManager`
- `FakeScorecardUploadManager`

---

### Phase 5a: Round Discord Layer (Implementation Tests)

**Goal:** Convert all `app/round/discord/` test files from GoMock to FakeSession.

**Pattern:** Replace `discordmocks.NewMockSession(ctrl)` with `discordgo.NewFakeSession()`

#### 5a.1 - `create_round/` (4 files)
- [x] `create_round_test.go` ✅ Converted
- [x] `embed_round_test.go` ✅ Converted
- [x] `interactions_test.go` ✅ Converted
- [x] `modal_test.go` ✅ Converted

#### 5a.2 - `delete_round/` (3 files)
- [x] `delete_embed_test.go` ✅ Converted
- [x] `delete_round_test.go` ✅ Converted
- [x] `interactions_test.go` ✅ Converted

#### 5a.3 - `finalize_round/` (3 files)
- [x] `embed_transform_test.go` ✅ Converted
- [x] `finalize_round_test.go` ✅ Converted
- [x] `update_embed_test.go` ✅ Converted

#### 5a.4 - `round_reminder/` (2 files)
- [x] `round_reminder_test.go` ✅ Converted
- [x] `thread_reminder_test.go` ✅ Converted

#### 5a.5 - `round_rsvp/` (3 files)
- [x] `embed_test.go` ✅ Converted
- [x] `interactions_test.go` ✅ Converted
- [x] `round_rsvp_test.go` ✅ Converted

#### 5a.6 - `score_round/` (3 files)
- [x] `messages_test.go` ✅ Converted
- [x] `score_round_test.go` ✅ Converted
- [x] `update_embed_test.go` ✅ Converted

#### 5a.7 - `scorecard_upload/` (4 files)
- [x] `helpers_test.go` ✅ Converted
- [x] `interactions_test.go` ✅ Converted
- [x] `publish_test.go` ✅ Converted
- [x] `register_handlers_test.go` ✅ Converted

#### 5a.8 - `start_round/` (3 files)
- [x] `embed_transform_test.go` ✅ Converted
- [x] `start_round_test.go` ✅ Converted
- [x] `update_embed_test.go` ✅ Converted

#### 5a.9 - `tag_updates/` (1 file)
- [x] `tag_updates_test.go` ✅ Converted

#### 5a.10 - `update_round/` (4 files)
- [x] `interactions_test.go` ✅ Converted
- [x] `modal_test.go` ✅ Converted
- [x] `update_embed_test.go` ✅ Converted
- [x] `update_round_test.go` ✅ Converted

#### 5a.11 - Root `discord/` (1 file)
- [x] `discord_test.go` ✅ Already converted

**Total:** 31 test files ✅ All Converted

---

### Phase 2a: Guild Discord Layer

**Goal:** Convert `app/guild/discord/` test files from GoMock to FakeSession.

#### 2a.1 - Root `discord/` (1 file)
- [x] `discord_test.go` ✅ Converted

#### 2a.2 - `setup/` (4 files)
- [x] `modal_test.go` ✅ Converted
- [x] `perform_custom_setup_test.go` ✅ Converted
- [x] `setup_additional_test.go` ✅ Converted
- [x] `setup_handlers_wrap_test.go` ✅ Converted

**Total:** 5 files

---

### Phase 2b: Guild Handlers (Remaining Mocks)

**Goal:** Remove remaining GoMock usage from guild handler tests.

- [x] `guild_config_creation_handler_test.go` ✅ Converted
- [x] `guild_config_deletion_handler_test.go` ✅ Converted

**Total:** 2 files

---

### Phase 3a: User Discord Layer

**Goal:** Convert `app/user/discord/` test files from GoMock to FakeSession.

#### 3a.1 - `role/` (3 files)
- [x] `interactions_test.go` ✅ Converted
- [x] `role_management_test.go` ✅ Converted
- [x] `role_test.go` ✅ Converted

#### 3a.2 - `signup/` (4 files)
- [x] `interactions_test.go` ✅ Converted
- [x] `modal_test.go` ✅ Converted
- [x] `responses_test.go` ✅ Converted
- [x] `signup_test.go` ✅ Converted

#### 3a.3 - `udisc/` (2 files)
- [x] `interactions_test.go` ✅ Converted
- [x] `udisc_manager_test.go` ✅ Converted

**Total:** 9 files

---

### Phase 3b: User Handlers + Module (Remaining Mocks)

**Goal:** Remove remaining GoMock usage from user tests.

- [x] `handlers/handlers_test.go` ✅ Converted
- [x] `module_test.go` ✅ Converted

**Total:** 2 files

---

### Phase 4a: Leaderboard Discord Layer

**Goal:** Convert `app/leaderboard/discord/` test files from GoMock to FakeSession.

#### 4a.1 - Root `discord/` (1 file)
- [x] `discord_test.go` ✅ Converted

#### 4a.2 - `claim_tag/` (1 file)
- [x] `interactions_test.go` ✅ Converted

#### 4a.3 - `leaderboard_updated/` (3 files)
- [x] `interactions_test.go` ✅ Converted
- [x] `leaderboard_embed_test.go` ✅ Converted
- [x] `leaderboard_updated_test.go` ✅ Converted

**Total:** 5 files

---

### Phase 4b: Leaderboard Handlers (Remaining Mocks)

**Goal:** Remove remaining GoMock usage from leaderboard handler tests.

- [x] `handlers_test.go` ✅ Converted
- [x] `leaderboard_update_test.go` ✅ Converted

**Total:** 2 files

---

### Misc: Other Files

**Goal:** Convert remaining scattered GoMock usages.

- [x] `app/bot/bot_test.go` ✅ Converted

**Total:** 1 file

---

## 📊 Summary: All Remaining GoMock Files (51 total)

| Module | Files |
|--------|-------|
| `round/discord` | ✅ Complete |
| `user/discord` | ✅ Complete |
| `guild/discord` | ✅ Complete |
| `leaderboard/discord` | ✅ Complete |
| `guild/handlers` | ✅ Complete |
| `user/handlers+module` | ✅ Complete |
| `leaderboard/handlers` | ✅ Complete |
| `bot` | ✅ Complete |
| **Total Remaining** | **0** |

---

### Phase 1 Details (Completed)

**Files Created:**
- `app/discordgo/fake_session.go` - FakeSession implementing the Fake/Stub pattern

**Files Modified:**
- `app/discordgo/discord.go` - Session interface reorganized into logical sections
- `app/discordgo/commands_test.go` - Converted from GoMock to FakeSession

**Files Deleted:**
- `app/discordgo/mocks/` directory (was already removed)

**Key Pattern Established:**
```go
// FakeSession pattern - see app/discordgo/fake_session.go
type FakeSession struct {
    trace []string
    MethodNameFunc func(...) (ReturnType, error)  // For each interface method
}
```

---

## Overview

This document outlines a comprehensive refactoring plan to migrate `discord-frolf-bot` (Target Repo) to match the Hexagonal/DDD architecture and testing patterns of `frolf-bot` (Source Repo).

**Source Repo:** `frolf-bot` (backend)  
**Target Repo:** `discord-frolf-bot` (discord gateway/UI layer)


---

## Key Architectural Differences

### Current State (Target Repo: discord-frolf-bot)

```
app/
├── discordgo/           # Discord session interface + GoMock mocks
│   └── mocks/           # Generated mocks (mockgen)
├── guild/
│   ├── discord/         # Discord-specific handlers
│   ├── mocks/           # Generated mocks (mockgen)
│   └── watermill/       # Event handlers
├── round/
│   ├── discord/
│   ├── mocks/           # 11 mock files!
│   └── watermill/
├── leaderboard/
│   ├── discord/
│   ├── mocks/
│   └── watermill/
└── user/
    ├── discord/
    ├── mocks/
    └── watermill/
```

### Target State (Source Repo Pattern: frolf-bot)

```
app/
├── modules/
│   ├── guild/
│   │   ├── application/     # Service layer (no payload awareness)
│   │   │   ├── interface.go
│   │   │   ├── service.go
│   │   │   ├── fake_test.go # Hand-written fakes
│   │   │   └── *_test.go
│   │   └── infrastructure/
│   │       ├── handlers/    # Pure transformation handlers
│   │       │   ├── interface.go
│   │       │   ├── fake_test.go
│   │       │   └── *_test.go
│   │       ├── repositories/
│   │       └── router/
│   ├── round/
│   ├── leaderboard/
│   └── user/
```

---

## Part 1: Testing Pattern Changes - Fakes/Stubs Over Mocks

### 1.1 What Is the Fakes/Stubs Pattern?

The source repo uses **hand-written fakes** instead of auto-generated mocks (like GoMock). Each fake:

1. **Implements the interface** it's faking
2. Has **programmable function fields** for each method
3. Maintains a **trace** of method calls for verification
4. Provides **sensible defaults** when no function is configured

**Example from Source Repo:**

```go
// app/modules/guild/application/fake_test.go
type FakeGuildRepository struct {
    trace []string

    GetConfigFunc               func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
    SaveConfigFunc              func(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error
    // ... other methods
}

func (f *FakeGuildRepository) GetConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
    f.record("GetConfig")
    if f.GetConfigFunc != nil {
        return f.GetConfigFunc(ctx, db, guildID)
    }
    // Default: Return ErrNotFound to simulate a clean state
    return nil, guilddb.ErrNotFound
}

// Interface assertion
var _ guilddb.Repository = (*FakeGuildRepository)(nil)
```

### 1.2 Benefits Over Mocks

| Aspect | GoMock | Fakes/Stubs |
|--------|--------|-------------|
| Setup complexity | High (EXPECT chains) | Low (just set func) |
| Readability | Poor (mock syntax) | Clear (plain Go) |
| Brittleness | High | Low |
| Maintenance | regenerate on interface change | update manually |
| Verification | Order-sensitive | Trace-based (flexible) |
| Code ownership | Generated | Hand-written |

### 1.3 Files to Delete (Mocks)

These mock files will be replaced by hand-written fakes:

| Module | Files to Delete |
|--------|-----------------|
| `discordgo` | `mocks/mock_discord.go`, `mocks/mock_discord_operations.go` |
| `guild` | `mocks/mock_guild_discord.go`, `mocks/mock_guild_handlers.go`, `mocks/mock_reset_manager.go`, `mocks/mock_setup_manager.go`, `mocks/mock_setup_session.go` |
| `round` | `mocks/mock_create_round_manager.go`, `mocks/mock_delete_round_manager.go`, `mocks/mock_finalize_round_manager.go`, `mocks/mock_handlers.go`, `mocks/mock_round_discord.go`, `mocks/mock_round_reminder_manager.go`, `mocks/mock_round_rsvp_manager.go`, `mocks/mock_score_round_manager.go`, `mocks/mock_start_round_manager.go`, `mocks/mock_tag_update_manager.go`, `mocks/mock_update_round_manager.go` |
| `leaderboard` | `mocks/mock_claim_tag.go`, `mocks/mock_handlers.go`, `mocks/mock_leaderboard_discord.go`, `mocks/mock_leaderboard_updated_manager.go` |
| `user` | `mocks/mock_handlers.go`, `mocks/mock_role_manager.go`, `mocks/mock_signup_manager.go`, `mocks/mock_user_discord.go` |

**Total: ~26 mock files to delete**

---

## Part 2: Discord Interface Refactoring

### 2.1 Current Problem

The target repo has a custom `discord.Session` interface wrapper with 40+ methods, created primarily to enable mocking:

```go
// app/discordgo/discord.go - Current (295 lines!)
type Session interface {
    UserChannelCreate(...)
    ChannelMessageSend(...)
    GuildMember(...)
    // ... 40+ more methods
}

type DiscordSession struct {
    session *discordgo.Session
}
```

### 2.2 Analysis: Can We Remove This?

**Keep the interface, use fakes.** Since Discord has no test environment or sandbox API, integration tests against a real Discord server are not feasible. This makes the fake/stub pattern **essential** - unit tests with fakes are the primary testing strategy for this repo.

Replace `mock_discord.go` with a hand-written `FakeSession`:

```go
// app/discordgo/fake_session.go
type FakeSession struct {
    trace []string
    
    ChannelMessageSendFunc func(channelID, content string, ...) (*discordgo.Message, error)
    GuildMemberFunc        func(guildID, userID string, ...) (*discordgo.Member, error)
    // Only stub the methods you actually use
}

func (f *FakeSession) ChannelMessageSend(channelID, content string, ...) (*discordgo.Message, error) {
    f.record("ChannelMessageSend")
    if f.ChannelMessageSendFunc != nil {
        return f.ChannelMessageSendFunc(channelID, content)
    }
    return &discordgo.Message{ID: "fake-msg-123"}, nil
}
```

**Benefits:**
- Minimal code changes to existing handlers
- Tests remain isolated from real Discord API
- Interface documents which methods are used
- **Only viable testing approach** given Discord's lack of test environment

**Recommendation:** Keep the interface but **slim it down** to only include methods actually used.

### 2.3 Interface Audit

Audit `app/discordgo/discord.go` to identify which methods are actually used across the codebase. Remove unused methods from the interface.

---

## Part 3: Service Layer Refactoring - Payload-Agnostic Services

### 3.1 Current Problem

The target repo's handlers are tightly coupled to event payloads and Discord-specific concerns:

```go
// Current: Handler does everything including Discord API calls
func (h *GuildHandlers) HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error) {
    // Discord API calls directly in handler
    h.session.InteractionResponseEdit(...)
    h.service.RegisterAllCommands(...)
    h.config.UpdateGuildConfig(...)
}
```

### 3.2 Target Pattern

The source repo separates concerns:

1. **Handlers** - Pure transformation: payload → domain object → service → result → event
2. **Services** - Business logic only, no knowledge of events/payloads
3. **Discord Operations** - Separate layer for Discord API interactions

```go
// Source Pattern: Handler is a thin transformation layer
func (h *GuildHandlers) HandleCreateGuildConfig(ctx context.Context, payload *guildevents.GuildConfigCreationRequestedPayloadV1) ([]handlerwrapper.Result, error) {
    // 1. Convert payload to domain model
    config := &guildtypes.GuildConfig{
        GuildID: payload.GuildID,
        // ...
    }
    
    // 2. Call service with domain model (NOT payload)
    result, err := h.service.CreateGuildConfig(ctx, config)
    
    // 3. Transform result to events
    if result.Success != nil {
        return []handlerwrapper.Result{{
            Topic: guildevents.GuildConfigCreatedV1,
            Payload: guildevents.GuildConfigCreatedPayloadV1{...},
        }}, nil
    }
}
```

### 3.3 Key Principle: Services Don't Know About Payloads

**Before (Target Repo):**
```go
// Service method tied to event structure
func (s *GuildService) HandleConfigCreated(payload *guildevents.GuildConfigCreatedPayloadV1) error
```

**After (Source Repo Pattern):**
```go
// Service method uses domain types only
func (s *GuildService) CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildConfigResult, error)
```

---

## Part 4: Typed Result Structs

### 4.1 Current Problem

The target repo doesn't consistently use typed result structs.

### 4.2 Target Pattern

The source repo defines **type aliases** for operation results:

```go
// app/modules/guild/application/interface.go
type GuildConfigResult = results.OperationResult[*guildtypes.GuildConfig, error]

// app/modules/round/application/interface.go
type CreateRoundResult = results.OperationResult[*roundtypes.CreateRoundResult, error]
type FinalizeRoundResult = results.OperationResult[*roundtypes.FinalizeRoundResult, error]
type ScoreUpdateResult = results.OperationResult[*roundtypes.ScoreUpdateResult, error]
// ... many more
```

This provides:
- **Type safety** at compile time
- **Self-documenting** return types
- **Consistent error handling** pattern

---

## Part 5: Module-by-Module Refactoring Tasks

---

### Module: Guild

#### Current Structure

```
app/guild/
├── discord/
│   ├── discord.go
│   ├── permissions.go
│   ├── reset/
│   └── setup/
├── mocks/                  # DELETE
│   ├── mock_guild_discord.go
│   ├── mock_guild_handlers.go
│   ├── mock_reset_manager.go
│   ├── mock_setup_manager.go
│   └── mock_setup_session.go
├── module.go
└── watermill/
    ├── handlers/
    └── router.go
```

#### Target Structure

```
app/guild/
├── discord/
│   ├── discord.go
│   ├── permissions.go
│   ├── reset/
│   └── setup/
├── handlers/               # NEW: move from watermill/handlers
│   ├── interface.go
│   ├── handlers.go
│   ├── guild_config_creation_handler.go
│   ├── guild_config_deletion_handler.go
│   ├── guild_config_update_handler.go
│   ├── fake_test.go       # NEW
│   └── *_test.go
├── module.go
└── router/                 # NEW: move from watermill/
    └── router.go
```

#### Tasks

| # | Task | Files Affected |
|---|------|----------------|
| G1 | Delete all mock files | `mocks/*` |
| G2 | Create `handlers/fake_test.go` with `FakeGuildDiscord` | NEW |
| G3 | Rename `watermill/handlers/` to `handlers/` | Move files |
| G4 | Rename `watermill/router.go` to `router/router.go` | Move file |
| G5 | Update handler to be payload-agnostic | `handlers/*.go` |
| G6 | Update all tests to use fakes | `handlers/*_test.go` |
| G7 | Remove unused methods from `discord/discord.go` interface | `discord/discord.go` |

---

### Module: Round

#### Current Structure

```
app/round/
├── discord/                # 81 files - complex Discord managers
├── mocks/                  # DELETE - 11 mock files!
├── module.go
└── watermill/
    └── handlers/
```

#### Tasks

| # | Task | Files Affected |
|---|------|----------------|
| R1 | Delete all 11 mock files | `mocks/*` |
| R2 | Create `handlers/fake_test.go` with fakes for all managers | NEW |
| R3 | Audit `discord/` - many managers may need simplification | `discord/*.go` |
| R4 | Rename `watermill/handlers/` to `handlers/` | Move files |
| R5 | Create typed result structs for discord operations | NEW |
| R6 | Update all tests to use fakes | `*_test.go` |

**Note:** Round module has the most complexity. Consider breaking `discord/` into smaller, focused components.

---

### Module: Leaderboard

#### Current Structure

```
app/leaderboard/
├── discord/
├── mocks/                  # DELETE
├── module.go
└── watermill/
```

#### Tasks

| # | Task | Files Affected |
|---|------|----------------|
| L1 | Delete all mock files | `mocks/*` |
| L2 | Create `handlers/fake_test.go` | NEW |
| L3 | Rename `watermill/handlers/` to `handlers/` | Move files |
| L4 | Update all tests to use fakes | `*_test.go` |

---

### Module: User

#### Current Structure

```
app/user/
├── discord/
├── mocks/                  # DELETE
├── module.go
├── module_test.go
└── watermill/
```

#### Tasks

| # | Task | Files Affected |
|---|------|----------------|
| U1 | Delete all mock files | `mocks/*` |
| U2 | Create `handlers/fake_test.go` | NEW |
| U3 | Rename `watermill/handlers/` to `handlers/` | Move files |
| U4 | Update all tests to use fakes | `*_test.go` |

---

### Module: DiscordGo Wrapper

#### Current Structure

```
app/discordgo/
├── commands.go
├── commands_test.go
├── discord.go              # 295 lines - interface + implementation
├── messaging.go
├── mocks/                  # DELETE
│   ├── mock_discord.go     # 908 lines!
│   └── mock_discord_operations.go
└── operations.go
```

#### Tasks

| # | Task | Files Affected |
|---|------|----------------|
| D1 | Delete all mock files | `mocks/*` |
| D2 | Audit interface - remove unused methods | `discord.go` |
| D3 | Create `fake_session.go` | NEW |
| D4 | Slim down interface to only used methods | `discord.go` |
| D5 | Update tests to use fakes | `*_test.go` |

---

## Part 6: Implementation Order

### Phase 1: Foundation (Week 1)

1. **Create fake patterns** - Start with `discordgo/fake_session.go`
2. **Delete mock files** - All 26 mock files
3. **Update `discordgo` tests** - Validate fake pattern works

### Phase 2: Guild Module (Week 1-2)

1. Restructure directories
2. Create `FakeGuildDiscord`
3. Update handlers to be payload-agnostic
4. Update all tests

### Phase 3: User Module (Week 2)

1. Restructure directories
2. Create fakes
3. Update tests

### Phase 4: Leaderboard Module (Week 2-3)

1. Restructure directories
2. Create fakes
3. Update tests

### Phase 5: Round Module (Week 3-4)

1. Audit and simplify `discord/` managers
2. Restructure directories
3. Create fakes (most complex - 11 mocks to replace)
4. Update tests

### Phase 6: Integration & Cleanup (Week 4)

1. Run full test suite
2. Update CI/CD
3. Documentation updates

---

## Part 7: Code Examples

### Example: Creating a Fake

```go
// app/guild/handlers/fake_test.go
package handlers

import (
    "context"
    guildconfig "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
)

// FakeGuildDiscord provides a programmable stub for guild Discord operations.
type FakeGuildDiscord struct {
    trace []string

    RegisterAllCommandsFunc func(guildID string) error
    // Add other methods as needed
}

func NewFakeGuildDiscord() *FakeGuildDiscord {
    return &FakeGuildDiscord{
        trace: []string{},
    }
}

func (f *FakeGuildDiscord) record(step string) {
    f.trace = append(f.trace, step)
}

func (f *FakeGuildDiscord) Trace() []string {
    out := make([]string, len(f.trace))
    copy(out, f.trace)
    return out
}

func (f *FakeGuildDiscord) RegisterAllCommands(guildID string) error {
    f.record("RegisterAllCommands")
    if f.RegisterAllCommandsFunc != nil {
        return f.RegisterAllCommandsFunc(guildID)
    }
    return nil
}

// Interface assertion
var _ GuildDiscord = (*FakeGuildDiscord)(nil)
```

### Example: Using a Fake in Tests

```go
// Before (with mocks)
func TestHandleGuildConfigCreated(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockService := mocks.NewMockGuildDiscord(ctrl)
    mockService.EXPECT().RegisterAllCommands("guild-1").Return(nil)
    
    h := NewGuildHandlers(mockService, ...)
    // ...
}

// After (with fakes)
func TestHandleGuildConfigCreated(t *testing.T) {
    fakeService := NewFakeGuildDiscord()
    fakeService.RegisterAllCommandsFunc = func(guildID string) error {
        return nil
    }
    
    h := NewGuildHandlers(fakeService, ...)
    res, err := h.HandleGuildConfigCreated(ctx, payload)
    
    // Verify calls
    if !slices.Contains(fakeService.Trace(), "RegisterAllCommands") {
        t.Error("expected RegisterAllCommands to be called")
    }
}
```

---

## Part 8: Decision Points

### Question 1: Keep `app/discordgo` Interface?

**Recommendation:** Yes, but slim it down significantly.

- Audit actual usage across codebase
- Remove methods not called anywhere
- The interface serves as documentation of Discord API surface area

### Question 2: Separate Discord Operations Layer?

**Recommendation:** Consider for Round module.

The Round module has 11 different "managers" in `discord/`. These could potentially be:
1. Consolidated into fewer, cohesive handlers
2. Separated by concern (messaging, events, threads)

### Question 3: Keep Watermill Naming?

**Recommendation:** Rename to just `handlers/` and `router/`.

The "watermill" prefix is an implementation detail. The source repo doesn't expose this.

---

## Part 9: Metrics & Verification

### Success Criteria

| Metric | Before | Target |
|--------|--------|--------|
| Mock files | ~26 | 0 |
| Generated code | ~2000 lines | 0 |
| Test readability | Low | High |
| Handler complexity | Mixed | Pure transformation |
| Service payload awareness | Yes | No |

### Verification Steps

1. All existing tests pass
2. No `gomock` imports remain
3. No `mockgen` directives remain
4. Each fake has interface assertion (`var _ Interface = (*)nil`)
5. Handlers follow pure transformation pattern

---

## Appendix A: Mock Files to Delete (Full List)

```
app/discordgo/mocks/mock_discord.go
app/discordgo/mocks/mock_discord_operations.go
app/guild/mocks/mock_guild_discord.go
app/guild/mocks/mock_guild_handlers.go
app/guild/mocks/mock_reset_manager.go
app/guild/mocks/mock_setup_manager.go
app/guild/mocks/mock_setup_session.go
app/leaderboard/mocks/mock_claim_tag.go
app/leaderboard/mocks/mock_handlers.go
app/leaderboard/mocks/mock_leaderboard_discord.go
app/leaderboard/mocks/mock_leaderboard_updated_manager.go
app/round/mocks/mock_create_round_manager.go
app/round/mocks/mock_delete_round_manager.go
app/round/mocks/mock_finalize_round_manager.go
app/round/mocks/mock_handlers.go
app/round/mocks/mock_round_discord.go
app/round/mocks/mock_round_reminder_manager.go
app/round/mocks/mock_round_rsvp_manager.go
app/round/mocks/mock_score_round_manager.go
app/round/mocks/mock_start_round_manager.go
app/round/mocks/mock_tag_update_manager.go
app/round/mocks/mock_update_round_manager.go
app/user/mocks/mock_handlers.go
app/user/mocks/mock_role_manager.go
app/user/mocks/mock_signup_manager.go
app/user/mocks/mock_user_discord.go
```

---

## Appendix B: Adding Typical Fake Structure

Each fake should follow this pattern:

```go
// FakeXXX provides a programmable stub for the XXX interface.
type FakeXXX struct {
    trace []string
    
    // For each interface method, add a corresponding Func field
    MethodOneFunc func(args...) (returnType, error)
    MethodTwoFunc func(args...) (returnType, error)
}

// NewFakeXXX initializes a new FakeXXX with an empty trace.
func NewFakeXXX() *FakeXXX {
    return &FakeXXX{
        trace: []string{},
    }
}

func (f *FakeXXX) record(step string) {
    f.trace = append(f.trace, step)
}

// Trace returns the sequence of method calls made to the fake.
func (f *FakeXXX) Trace() []string {
    out := make([]string, len(f.trace))
    copy(out, f.trace)
    return out
}

// --- Interface Implementation ---

func (f *FakeXXX) MethodOne(args...) (returnType, error) {
    f.record("MethodOne")
    if f.MethodOneFunc != nil {
        return f.MethodOneFunc(args...)
    }
    // Default behavior
    return defaultValue, nil
}

// Interface assertion - compile time check
var _ XXXInterface = (*FakeXXX)(nil)
```

---

## Appendix C: Handler Pure Transformation Pattern

Each handler should follow this pattern:

```go
func (h *Handlers) HandleXXX(ctx context.Context, payload *events.XXXPayloadV1) ([]handlerwrapper.Result, error) {
    // 1. Validate payload
    if payload == nil {
        return nil, errors.New("payload cannot be nil")
    }
    
    // 2. Convert payload to domain model
    domainModel := &types.Model{
        Field1: payload.Field1,
        Field2: payload.Field2,
    }
    
    // 3. Call service with domain model
    result, err := h.service.Operation(ctx, domainModel)
    if err != nil {
        return nil, err // Infrastructure error
    }
    
    // 4. Transform result to events
    if result.Success != nil {
        return []handlerwrapper.Result{{
            Topic:   events.XXXSucceededV1,
            Payload: events.XXXSucceededPayloadV1{...},
        }}, nil
    }
    
    if result.Failure != nil {
        return []handlerwrapper.Result{{
            Topic:   events.XXXFailedV1,
            Payload: events.XXXFailedPayloadV1{
                Reason: (*result.Failure).Error(),
            },
        }}, nil
    }
    
    return nil, nil
}
```

---

## Summary

This refactoring will:

1. **Delete ~26 mock files** (thousands of lines of generated code)
2. **Replace with ~5-10 hand-written fake files** (readable, maintainable)
3. **Restructure directories** to match source repo pattern
4. **Simplify handlers** to pure transformation functions
5. **Improve test readability** significantly

The estimated effort is **3-4 weeks** for a complete migration, with the Round module being the most complex due to its many Discord managers.
