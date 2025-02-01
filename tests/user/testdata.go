package tests

type TestMessage struct {
	Content   string
	Author    TestUser
	ChannelID string
	ID        string
}

type TestUser struct {
	ID string
}

type TestSession struct {
	User TestUser
}
