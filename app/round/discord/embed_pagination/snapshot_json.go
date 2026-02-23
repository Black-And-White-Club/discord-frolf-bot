package embedpagination

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type snapshotJSON struct {
	MessageID            string                         `json:"message_id"`
	Kind                 SnapshotKind                   `json:"kind"`
	Title                string                         `json:"title"`
	Description          string                         `json:"description"`
	Color                int                            `json:"color"`
	Timestamp            string                         `json:"timestamp"`
	Footer               *discordgo.MessageEmbedFooter  `json:"footer,omitempty"`
	StaticFields         []*discordgo.MessageEmbedField `json:"static_fields,omitempty"`
	ParticipantFieldName string                         `json:"participant_field_name,omitempty"`
	LineItems            []string                       `json:"line_items,omitempty"`
	FieldItems           []*discordgo.MessageEmbedField `json:"field_items,omitempty"`
	BaseComponents       []json.RawMessage              `json:"base_components,omitempty"`
	CurrentPage          int                            `json:"current_page"`
}

// MarshalJSON ensures Snapshot can be safely persisted as JSON.
func (s *Snapshot) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}

	baseComponents := make([]json.RawMessage, 0, len(s.BaseComponents))
	for _, component := range s.BaseComponents {
		if component == nil {
			continue
		}
		raw, err := json.Marshal(component)
		if err != nil {
			return nil, fmt.Errorf("marshal component: %w", err)
		}
		baseComponents = append(baseComponents, raw)
	}

	return json.Marshal(snapshotJSON{
		MessageID:            s.MessageID,
		Kind:                 s.Kind,
		Title:                s.Title,
		Description:          s.Description,
		Color:                s.Color,
		Timestamp:            s.Timestamp,
		Footer:               cloneFooter(s.Footer),
		StaticFields:         cloneFields(s.StaticFields),
		ParticipantFieldName: s.ParticipantFieldName,
		LineItems:            cloneLines(s.LineItems),
		FieldItems:           cloneFields(s.FieldItems),
		BaseComponents:       baseComponents,
		CurrentPage:          s.CurrentPage,
	})
}

// UnmarshalJSON ensures Snapshot can be restored from persisted JSON.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	var wire snapshotJSON
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	components := make([]discordgo.MessageComponent, 0, len(wire.BaseComponents))
	for _, raw := range wire.BaseComponents {
		component, err := discordgo.MessageComponentFromJSON(raw)
		if err != nil {
			return fmt.Errorf("unmarshal component: %w", err)
		}
		components = append(components, component)
	}

	*s = Snapshot{
		MessageID:            wire.MessageID,
		Kind:                 wire.Kind,
		Title:                wire.Title,
		Description:          wire.Description,
		Color:                wire.Color,
		Timestamp:            wire.Timestamp,
		Footer:               cloneFooter(wire.Footer),
		StaticFields:         cloneFields(wire.StaticFields),
		ParticipantFieldName: wire.ParticipantFieldName,
		LineItems:            cloneLines(wire.LineItems),
		FieldItems:           cloneFields(wire.FieldItems),
		BaseComponents:       cloneComponents(components),
		CurrentPage:          wire.CurrentPage,
	}

	return nil
}
