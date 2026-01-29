---
description: Convert GoMock tests to FakeSession pattern
---

# Convert GoMock to FakeSession

This workflow converts test files from using GoMock to the FakeSession pattern.

## Usage

When invoking this workflow, specify:
- **Phase:** The phase from REFACTOR_PLAN.md (2a, 2b, 3a, 3b, 4a, 4b, 5a, or Misc)
- **Module Path:** e.g., `app/guild/discord/setup/`
- **Files:** List of specific test files to convert

## Conversion Steps

### 1. Remove GoMock Imports

```go
// REMOVE these imports:
"go.uber.org/mock/gomock"
discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
```

### 2. Replace Mock Creation

```go
// BEFORE:
ctrl := gomock.NewController(t)
defer ctrl.Finish()
mockSession := discordmocks.NewMockSession(ctrl)

// AFTER:
fakeSession := discordgo.NewFakeSession()
```

### 3. Convert EXPECT() to Func Assignments

```go
// BEFORE:
mockSession.EXPECT().ChannelMessageSend(gomock.Any(), gomock.Any()).Return(&discordgo.Message{ID: "123"}, nil)

// AFTER:
fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
    return &discordgo.Message{ID: "123"}, nil
}
```

### 4. Handle gomock.Any() Matchers

With fakes, you capture actual arguments in the func:

```go
// If you need to verify arguments:
var capturedChannelID string
fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
    capturedChannelID = channelID
    return &discordgo.Message{ID: "123"}, nil
}
// Later assert: assert.Equal(t, "expected-channel", capturedChannelID)

// If you don't care about arguments, just return the expected value:
fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
    return &discordgo.Message{ID: "123"}, nil
}
```

### 5. Handle .Times() / .AnyTimes()

With fakes, track calls via counters or the Trace() method:

```go
// Using Trace():
fakeSession := discordgo.NewFakeSession()
// ... run test ...
trace := fakeSession.Trace()
assert.Contains(t, trace, "ChannelMessageSend")

// Using counter:
callCount := 0
fakeSession.ChannelMessageSendFunc = func(...) (*discordgo.Message, error) {
    callCount++
    return &discordgo.Message{}, nil
}
// ... run test ...
assert.Equal(t, 2, callCount)
```

### 6. Handle Error Returns

```go
// BEFORE:
mockSession.EXPECT().ChannelMessageSend(gomock.Any(), gomock.Any()).Return(nil, errors.New("discord error"))

// AFTER:
fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
    return nil, errors.New("discord error")
}
```

## Reference Files

- **FakeSession implementation:** `app/discordgo/fake_session.go`
- **Example conversion:** `app/round/discord/discord_test.go`
- **Handler fakes example:** `app/round/handlers/fake_test.go`

## Verification

After converting each file, run:

```bash
// turbo
go test ./app/[module]/discord/[submodule]/... -v -count=1
```

## On Completion

Update REFACTOR_PLAN.md to check off completed files with `[x]`.

---

## Quick Reference: Common Session Methods

| GoMock | FakeSession Func Field |
|--------|------------------------|
| `ChannelMessageSend` | `ChannelMessageSendFunc` |
| `ChannelMessageSendComplex` | `ChannelMessageSendComplexFunc` |
| `ChannelMessageEditComplex` | `ChannelMessageEditComplexFunc` |
| `ChannelMessageDelete` | `ChannelMessageDeleteFunc` |
| `InteractionRespond` | `InteractionRespondFunc` |
| `InteractionResponseEdit` | `InteractionResponseEditFunc` |
| `FollowupMessageCreate` | `FollowupMessageCreateFunc` |
| `GuildMember` | `GuildMemberFunc` |
| `UserChannelCreate` | `UserChannelCreateFunc` |
| `MessageThreadStartComplex` | `MessageThreadStartComplexFunc` |

See `app/discordgo/fake_session.go` for the complete list.
