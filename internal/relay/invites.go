package relay

// This file re-exports invite operations from the Client.
// The actual implementations are in client.go.
// This file exists to match the build plan's file structure.

// InviteFlow encapsulates the full invite lifecycle.
type InviteFlow struct {
	client *Client
}

// NewInviteFlow creates an invite flow helper.
func NewInviteFlow(client *Client) *InviteFlow {
	return &InviteFlow{client: client}
}

// Create generates an invite and registers it on the relay.
func (f *InviteFlow) Create(req InviteRequest) error {
	return f.client.CreateInvite(req)
}

// Retrieve fetches an invite by its token hash.
func (f *InviteFlow) Retrieve(tokenHash string) (*InviteResponse, error) {
	return f.client.GetInvite(tokenHash)
}

// Consume redeems an invite and returns the invite details.
func (f *InviteFlow) Consume(tokenHash string) (*InviteResponse, error) {
	return f.client.ConsumeInvite(tokenHash)
}
