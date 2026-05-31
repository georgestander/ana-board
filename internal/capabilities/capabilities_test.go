package capabilities

import (
	"strings"
	"testing"
)

func TestCurrentDescribesNativeEmojiAndPerTileColor(t *testing.T) {
	caps := Current()

	if !caps.NativeEmoji {
		t.Fatal("NativeEmoji = false, want true")
	}

	if len(caps.EmojiAliases) == 0 {
		t.Fatal("EmojiAliases is empty")
	}

	if !strings.Contains(caps.Text.EmojiSupport, "no emoji whitelist") {
		t.Fatalf("EmojiSupport = %q, want no-whitelist guidance", caps.Text.EmojiSupport)
	}

	if !strings.Contains(caps.Text.ColorSyntax, "default for untagged tiles") {
		t.Fatalf("ColorSyntax = %q, want default-color guidance", caps.Text.ColorSyntax)
	}
}
