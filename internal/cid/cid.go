// Package cid finds the Content-IDs an HTML body embeds as image sources.
// It has no dependencies, so the database migrations can share it with the
// message package.
package cid

import "regexp"

// srcPattern matches an image source cid reference the way the viewer's
// processCidReferences does (case-insensitive src= and cid:), also accepting
// the unquoted form found in raw, unsanitized HTML. An unquoted value drops
// the "/" of a self-closing tag, so <img src=cid:a/> yields "a".
var srcPattern = regexp.MustCompile(`(?i)\bsrc\s*=\s*(?:["']cid:([^"'\s>]+)|cid:([^"'\s>]+?)/?(?:[\s>]|$))`)

// Embedded returns the Content-IDs html uses as image sources. IDs are
// compared exactly, so "part1" does not match "cid:part10".
func Embedded(html string) map[string]struct{} {
	matches := srcPattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		ids[m[1]+m[2]] = struct{}{}
	}
	return ids
}
