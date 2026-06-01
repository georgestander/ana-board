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

	if !strings.Contains(caps.ExactFrame.PlacementSyntax, "Rows are 0-9") {
		t.Fatalf("PlacementSyntax = %q, want row/column guidance", caps.ExactFrame.PlacementSyntax)
	}

	if caps.BlockArt.PixelSymbol != "█" {
		t.Fatalf("PixelSymbol = %q, want block pixel", caps.BlockArt.PixelSymbol)
	}

	if len(caps.BlockArt.Sprites) == 0 {
		t.Fatal("BlockArt.Sprites is empty")
	}
}
