package embedpagination

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	maxEmbedFields            = 25
	maxEmbedFieldValueLength  = 1024
	maxFooterTextLength       = 2048
	defaultParticipantField   = "👥 Participants"
	placeholderNoParticipants = "*No participants*"

	pagerPrefix      = "round_page|"
	pagerCustomIDFmt = "round_page|%s|%d"
)

type SnapshotKind string

const (
	SnapshotKindLines  SnapshotKind = "lines"
	SnapshotKindFields SnapshotKind = "fields"
)

type Snapshot struct {
	MessageID string

	Kind                 SnapshotKind
	Title                string
	Description          string
	Color                int
	Timestamp            string
	Footer               *discordgo.MessageEmbedFooter
	StaticFields         []*discordgo.MessageEmbedField
	ParticipantFieldName string
	LineItems            []string
	FieldItems           []*discordgo.MessageEmbedField
	BaseComponents       []discordgo.MessageComponent
	CurrentPage          int
}

var snapshotStore = struct {
	mu    sync.RWMutex
	items map[string]*Snapshot
}{
	items: make(map[string]*Snapshot),
}

func NewLineSnapshot(
	messageID string,
	embed *discordgo.MessageEmbed,
	components []discordgo.MessageComponent,
	staticFields []*discordgo.MessageEmbedField,
	participantFieldName string,
	lines []string,
) *Snapshot {
	if participantFieldName == "" {
		participantFieldName = defaultParticipantField
	}

	return &Snapshot{
		MessageID:            messageID,
		Kind:                 SnapshotKindLines,
		Title:                safeEmbedTitle(embed),
		Description:          safeEmbedDescription(embed),
		Color:                safeEmbedColor(embed),
		Timestamp:            safeEmbedTimestamp(embed),
		Footer:               cloneFooter(embedFooter(embed)),
		StaticFields:         cloneFields(staticFields),
		ParticipantFieldName: participantFieldName,
		LineItems:            cloneLines(lines),
		FieldItems:           nil,
		BaseComponents:       stripPagerComponents(components),
		CurrentPage:          0,
	}
}

func NewFieldSnapshot(
	messageID string,
	embed *discordgo.MessageEmbed,
	components []discordgo.MessageComponent,
	staticFields []*discordgo.MessageEmbedField,
	participantFields []*discordgo.MessageEmbedField,
) *Snapshot {
	return &Snapshot{
		MessageID:            messageID,
		Kind:                 SnapshotKindFields,
		Title:                safeEmbedTitle(embed),
		Description:          safeEmbedDescription(embed),
		Color:                safeEmbedColor(embed),
		Timestamp:            safeEmbedTimestamp(embed),
		Footer:               cloneFooter(embedFooter(embed)),
		StaticFields:         cloneFields(staticFields),
		ParticipantFieldName: "",
		LineItems:            nil,
		FieldItems:           cloneFields(participantFields),
		BaseComponents:       stripPagerComponents(components),
		CurrentPage:          0,
	}
}

func setInMemory(snapshot *Snapshot) {
	if snapshot == nil || snapshot.MessageID == "" {
		return
	}

	snapshotStore.mu.Lock()
	defer snapshotStore.mu.Unlock()

	cloned := cloneSnapshot(snapshot)
	if existing, ok := snapshotStore.items[snapshot.MessageID]; ok {
		cloned.CurrentPage = existing.CurrentPage
	}
	snapshotStore.items[snapshot.MessageID] = cloned
}

func deleteInMemory(messageID string) {
	if messageID == "" {
		return
	}

	snapshotStore.mu.Lock()
	defer snapshotStore.mu.Unlock()
	delete(snapshotStore.items, messageID)
}

func getInMemory(messageID string) (*Snapshot, bool) {
	if messageID == "" {
		return nil, false
	}

	snapshotStore.mu.RLock()
	defer snapshotStore.mu.RUnlock()

	snapshot, ok := snapshotStore.items[messageID]
	if !ok {
		return nil, false
	}

	return cloneSnapshot(snapshot), true
}

func updateInMemory(messageID string, mutate func(snapshot *Snapshot) bool) (*Snapshot, bool) {
	if messageID == "" || mutate == nil {
		return nil, false
	}

	snapshotStore.mu.Lock()
	defer snapshotStore.mu.Unlock()

	existing, ok := snapshotStore.items[messageID]
	if !ok {
		return nil, false
	}

	clone := cloneSnapshot(existing)
	if !mutate(clone) {
		return clone, true
	}

	snapshotStore.items[messageID] = clone
	return cloneSnapshot(clone), true
}

func renderPageInMemory(messageID string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, int, int, error) {
	snapshotStore.mu.Lock()
	defer snapshotStore.mu.Unlock()

	snapshot, ok := snapshotStore.items[messageID]
	if !ok {
		return nil, nil, 0, 0, fmt.Errorf("pagination snapshot not found for message %s", messageID)
	}

	embed, components, actualPage, totalPages := renderSnapshot(cloneSnapshot(snapshot), page)
	snapshot.CurrentPage = actualPage

	return embed, components, actualPage, totalPages, nil
}

func IsPagerCustomID(customID string) bool {
	return strings.HasPrefix(customID, pagerPrefix)
}

func ParsePagerCustomID(customID string) (messageID string, page int, ok bool) {
	parts := strings.Split(customID, "|")
	if len(parts) != 3 || parts[0] != "round_page" || parts[1] == "" {
		return "", 0, false
	}

	pageValue, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, false
	}

	return parts[1], pageValue, true
}

func ParticipantLinesFromFieldValue(value string) []string {
	if strings.TrimSpace(value) == "" || value == placeholderNoParticipants || value == "-" {
		return nil
	}

	parts := strings.Split(value, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}

	return lines
}

func IsParticipantFieldName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "accepted") ||
		strings.Contains(lower, "tentative") ||
		strings.Contains(lower, "declined") ||
		strings.Contains(lower, "participants") ||
		strings.Contains(name, "✅") ||
		strings.Contains(name, "❓") ||
		strings.Contains(name, "❌") ||
		strings.Contains(name, "👥")
}

func renderSnapshot(snapshot *Snapshot, requestedPage int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, int, int) {
	if snapshot == nil {
		return &discordgo.MessageEmbed{}, nil, 0, 1
	}

	staticFields := normalizeStaticFields(cloneFields(snapshot.StaticFields), snapshot.Kind)

	var (
		participantFields []*discordgo.MessageEmbedField
		actualPage        int
		totalPages        int
		rangeLabel        string
	)

	switch snapshot.Kind {
	case SnapshotKindFields:
		pages := chunkFields(snapshot.FieldItems, max(1, maxEmbedFields-len(staticFields)))
		actualPage = clampPage(requestedPage, len(pages))
		totalPages = len(pages)
		participantFields = cloneFields(pages[actualPage])
		rangeLabel = buildRangeLabelForFields(pages, actualPage)
	default:
		pages := chunkLines(snapshot.LineItems, maxEmbedFieldValueLength)
		actualPage = clampPage(requestedPage, len(pages))
		totalPages = len(pages)
		participantFields = []*discordgo.MessageEmbedField{buildParticipantLinesField(snapshot.ParticipantFieldName, pages[actualPage])}
		rangeLabel = buildRangeLabelForLines(pages, actualPage)
	}

	allFields := append(staticFields, participantFields...)
	if len(allFields) > maxEmbedFields {
		allFields = allFields[:maxEmbedFields]
	}

	embed := &discordgo.MessageEmbed{
		Title:       snapshot.Title,
		Description: snapshot.Description,
		Color:       snapshot.Color,
		Fields:      allFields,
		Footer:      buildFooter(snapshot.Footer, actualPage, totalPages, rangeLabel),
		Timestamp:   snapshot.Timestamp,
	}

	components := buildComponents(snapshot.BaseComponents, snapshot.MessageID, actualPage, totalPages)
	return embed, components, actualPage, totalPages
}

func buildParticipantLinesField(name string, lines []string) *discordgo.MessageEmbedField {
	fieldName := name
	if fieldName == "" {
		fieldName = defaultParticipantField
	}

	value := placeholderNoParticipants
	if len(lines) > 0 {
		value = strings.Join(lines, "\n")
	}

	return &discordgo.MessageEmbedField{
		Name:   fieldName,
		Value:  value,
		Inline: false,
	}
}

func buildFooter(base *discordgo.MessageEmbedFooter, page, totalPages int, rangeLabel string) *discordgo.MessageEmbedFooter {
	baseText := ""
	if base != nil {
		baseText = strings.TrimSpace(base.Text)

		// Strip off any existing pagination suffix so it doesn't duplicate
		// The suffix format is " | Page X/Y" or " | Page X/Y • Z-W of V"
		if idx := strings.Index(baseText, " | Page "); idx != -1 {
			baseText = strings.TrimSpace(baseText[:idx])
		}
	}

	// Only show page label when there is actually more than one page.
	var footerText string
	if totalPages > 1 {
		pagerText := fmt.Sprintf("Page %d/%d", page+1, totalPages)
		if rangeLabel != "" {
			pagerText = fmt.Sprintf("%s • %s", pagerText, rangeLabel)
		}
		if baseText != "" {
			footerText = fmt.Sprintf("%s | %s", baseText, pagerText)
		} else {
			footerText = pagerText
		}
	} else {
		footerText = baseText
	}

	if len(footerText) > maxFooterTextLength {
		footerText = footerText[:maxFooterTextLength]
	}

	footer := cloneFooter(base)
	if footer == nil {
		if footerText == "" {
			return nil
		}
		footer = &discordgo.MessageEmbedFooter{}
	}
	footer.Text = footerText
	return footer
}

func buildComponents(base []discordgo.MessageComponent, messageID string, page, totalPages int) []discordgo.MessageComponent {
	components := cloneComponents(base)
	if totalPages <= 1 {
		return components
	}

	pagerRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Prev",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf(pagerCustomIDFmt, messageID, page-1),
				Disabled: page <= 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf(pagerCustomIDFmt, messageID, page+1),
				Disabled: page >= totalPages-1,
			},
		},
	}

	if len(components) < 5 {
		return append(components, pagerRow)
	}

	for i := len(components) - 1; i >= 0; i-- {
		row, ok := asActionsRow(components[i])
		if !ok {
			continue
		}
		if len(row.Components)+len(pagerRow.Components) > 5 {
			continue
		}
		row.Components = append(row.Components, pagerRow.Components...)
		components[i] = row
		return components
	}

	return components
}

func buildRangeLabelForLines(pages [][]string, page int) string {
	total := 0
	for _, p := range pages {
		total += len(p)
	}
	if total == 0 || page < 0 || page >= len(pages) || len(pages[page]) == 0 {
		return ""
	}

	start := 1
	for i := 0; i < page; i++ {
		start += len(pages[i])
	}
	end := start + len(pages[page]) - 1
	return fmt.Sprintf("%d-%d of %d", start, end, total)
}

func buildRangeLabelForFields(pages [][]*discordgo.MessageEmbedField, page int) string {
	total := 0
	for _, p := range pages {
		total += len(p)
	}
	if total == 0 || page < 0 || page >= len(pages) || len(pages[page]) == 0 {
		return ""
	}

	start := 1
	for i := 0; i < page; i++ {
		start += len(pages[i])
	}
	end := start + len(pages[page]) - 1
	return fmt.Sprintf("%d-%d of %d", start, end, total)
}

func chunkLines(lines []string, maxValueLen int) [][]string {
	if len(lines) == 0 {
		return [][]string{{}}
	}

	pages := make([][]string, 0, 1)
	current := make([]string, 0)
	currentLen := 0

	for _, line := range lines {
		normalized := strings.TrimSpace(line)
		if normalized == "" {
			continue
		}
		if len(normalized) > maxValueLen {
			normalized = normalized[:maxValueLen]
		}

		additionalLen := len(normalized)
		if len(current) > 0 {
			additionalLen++
		}

		if currentLen+additionalLen > maxValueLen && len(current) > 0 {
			pages = append(pages, current)
			current = []string{normalized}
			currentLen = len(normalized)
			continue
		}

		current = append(current, normalized)
		currentLen += additionalLen
	}

	if len(current) > 0 {
		pages = append(pages, current)
	}
	if len(pages) == 0 {
		return [][]string{{}}
	}

	return pages
}

func chunkFields(fields []*discordgo.MessageEmbedField, pageSize int) [][]*discordgo.MessageEmbedField {
	if pageSize <= 0 {
		pageSize = 1
	}
	if len(fields) == 0 {
		return [][]*discordgo.MessageEmbedField{{}}
	}

	pages := make([][]*discordgo.MessageEmbedField, 0, (len(fields)+pageSize-1)/pageSize)
	for i := 0; i < len(fields); i += pageSize {
		end := i + pageSize
		if end > len(fields) {
			end = len(fields)
		}
		pages = append(pages, cloneFields(fields[i:end]))
	}
	return pages
}

func normalizeStaticFields(fields []*discordgo.MessageEmbedField, kind SnapshotKind) []*discordgo.MessageEmbedField {
	if len(fields) == 0 {
		return fields
	}

	maxStatic := maxEmbedFields - 1
	if kind == SnapshotKindFields {
		maxStatic = maxEmbedFields - 1
	}

	if maxStatic < 0 {
		maxStatic = 0
	}
	if len(fields) <= maxStatic {
		return fields
	}
	return fields[:maxStatic]
}

func stripPagerComponents(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	if len(components) == 0 {
		return nil
	}

	cloned := cloneComponents(components)
	filtered := make([]discordgo.MessageComponent, 0, len(cloned))

	for _, component := range cloned {
		row, ok := asActionsRow(component)
		if !ok {
			filtered = append(filtered, component)
			continue
		}

		rowComponents := make([]discordgo.MessageComponent, 0, len(row.Components))
		for _, rowComponent := range row.Components {
			button, isButton := asButton(rowComponent)
			if isButton && IsPagerCustomID(button.CustomID) {
				continue
			}
			rowComponents = append(rowComponents, rowComponent)
		}

		if len(rowComponents) == 0 {
			continue
		}
		row.Components = rowComponents
		filtered = append(filtered, row)
	}

	return filtered
}

func clampPage(requested, total int) int {
	if total <= 0 {
		return 0
	}
	if requested < 0 {
		return 0
	}
	if requested >= total {
		return total - 1
	}
	return requested
}

func cloneSnapshot(snapshot *Snapshot) *Snapshot {
	if snapshot == nil {
		return nil
	}

	return &Snapshot{
		MessageID:            snapshot.MessageID,
		Kind:                 snapshot.Kind,
		Title:                snapshot.Title,
		Description:          snapshot.Description,
		Color:                snapshot.Color,
		Timestamp:            snapshot.Timestamp,
		Footer:               cloneFooter(snapshot.Footer),
		StaticFields:         cloneFields(snapshot.StaticFields),
		ParticipantFieldName: snapshot.ParticipantFieldName,
		LineItems:            cloneLines(snapshot.LineItems),
		FieldItems:           cloneFields(snapshot.FieldItems),
		BaseComponents:       cloneComponents(snapshot.BaseComponents),
		CurrentPage:          snapshot.CurrentPage,
	}
}

func cloneFields(fields []*discordgo.MessageEmbedField) []*discordgo.MessageEmbedField {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]*discordgo.MessageEmbedField, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		copyField := *field
		cloned = append(cloned, &copyField)
	}
	return cloned
}

func cloneLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}

	cloned := make([]string, len(lines))
	copy(cloned, lines)
	return cloned
}

func cloneFooter(footer *discordgo.MessageEmbedFooter) *discordgo.MessageEmbedFooter {
	if footer == nil {
		return nil
	}
	copyFooter := *footer
	return &copyFooter
}

func cloneComponents(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	if len(components) == 0 {
		return nil
	}

	cloned := make([]discordgo.MessageComponent, 0, len(components))
	for _, component := range components {
		switch typed := component.(type) {
		case discordgo.ActionsRow:
			row := discordgo.ActionsRow{Components: cloneRowComponents(typed.Components)}
			cloned = append(cloned, row)
		case *discordgo.ActionsRow:
			if typed == nil {
				continue
			}
			row := discordgo.ActionsRow{Components: cloneRowComponents(typed.Components)}
			cloned = append(cloned, row)
		default:
			cloned = append(cloned, component)
		}
	}
	return cloned
}

func cloneRowComponents(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	if len(components) == 0 {
		return nil
	}

	cloned := make([]discordgo.MessageComponent, 0, len(components))
	for _, component := range components {
		switch typed := component.(type) {
		case discordgo.Button:
			button := typed
			cloned = append(cloned, button)
		case *discordgo.Button:
			if typed == nil {
				continue
			}
			button := *typed
			cloned = append(cloned, button)
		default:
			cloned = append(cloned, component)
		}
	}
	return cloned
}

func asActionsRow(component discordgo.MessageComponent) (discordgo.ActionsRow, bool) {
	switch typed := component.(type) {
	case discordgo.ActionsRow:
		return typed, true
	case *discordgo.ActionsRow:
		if typed == nil {
			return discordgo.ActionsRow{}, false
		}
		return *typed, true
	default:
		return discordgo.ActionsRow{}, false
	}
}

func asButton(component discordgo.MessageComponent) (discordgo.Button, bool) {
	switch typed := component.(type) {
	case discordgo.Button:
		return typed, true
	case *discordgo.Button:
		if typed == nil {
			return discordgo.Button{}, false
		}
		return *typed, true
	default:
		return discordgo.Button{}, false
	}
}

func safeEmbedTitle(embed *discordgo.MessageEmbed) string {
	if embed == nil {
		return ""
	}
	return embed.Title
}

func safeEmbedDescription(embed *discordgo.MessageEmbed) string {
	if embed == nil {
		return ""
	}
	return embed.Description
}

func safeEmbedColor(embed *discordgo.MessageEmbed) int {
	if embed == nil {
		return 0
	}
	return embed.Color
}

func safeEmbedTimestamp(embed *discordgo.MessageEmbed) string {
	if embed == nil {
		return ""
	}
	return embed.Timestamp
}

func embedFooter(embed *discordgo.MessageEmbed) *discordgo.MessageEmbedFooter {
	if embed == nil {
		return nil
	}
	return embed.Footer
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
