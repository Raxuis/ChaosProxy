package proxy

import "testing"

func TestDeriveSeedIsStableAndRequestSpecific(t *testing.T) {
	t.Parallel()

	const globalSeed int64 = 42
	first := deriveSeed(globalSeed, 0)
	if got := deriveSeed(globalSeed, 0); got != first {
		t.Fatalf("deriveSeed(42, 0) = %d, want stable value %d", got, first)
	}
	if second := deriveSeed(globalSeed, 1); second == first {
		t.Fatalf("consecutive requests received the same derived seed %d", first)
	}
	if other := deriveSeed(globalSeed+1, 0); other == first {
		t.Fatalf("different global seeds produced the same derived seed %d", first)
	}
}
