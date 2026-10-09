package message

import "regexp"

// FilterEmbeddedInline returns atts without the inline parts whose Content-ID
// bodyHTML embeds as an image source (src="cid:<id>"), the references the
// viewer resolves. Non-inline parts and inline parts the body never shows
// stay, so nothing without a rendering in the body is hidden.
func FilterEmbeddedInline(atts []*Attachment, bodyHTML string) []*Attachment {
	embedded := embeddedCIDs(bodyHTML)
	if len(embedded) == 0 {
		return atts
	}
	result := make([]*Attachment, 0, len(atts))
	for _, att := range atts {
		if _, ok := embedded[att.ContentID]; ok && att.IsInline && att.ContentID != "" {
			continue
		}
		result = append(result, att)
	}
	return result
}

// HasInlineCID reports whether any attachment is an inline part with a
// Content-ID, the only kind FilterEmbeddedInline can remove.
func HasInlineCID(atts []*Attachment) bool {
	for _, att := range atts {
		if att.IsInline && att.ContentID != "" {
			return true
		}
	}
	return false
}

// cidSrcPattern matches an image source cid reference the way the viewer's
// processCidReferences does (case-insensitive src= and cid:), also accepting
// the unquoted form found in raw, unsanitized HTML.
var cidSrcPattern = regexp.MustCompile(`(?i)\bsrc\s*=\s*["']?cid:([^"'\s>]+)`)

// embeddedCIDs returns the Content-IDs html uses as image sources. IDs are
// compared exactly, so "part1" does not match "cid:part10".
func embeddedCIDs(html string) map[string]struct{} {
	matches := cidSrcPattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		ids[m[1]] = struct{}{}
	}
	return ids
}
