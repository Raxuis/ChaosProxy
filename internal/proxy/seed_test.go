package proxy

import "testing"

func TestDeriveSeedIsStableAndSpecific(t *testing.T) {
	t.Parallel()

	const globalSeed int64 = 42
	first := deriveSeed(globalSeed, "flaky", 0)
	if got := deriveSeed(globalSeed, "flaky", 0); got != first {
		t.Fatalf("deriveSeed(42, flaky, 0) = %d, want stable value %d", got, first)
	}
	for name, other := range map[string]uint64{
		"next request":     deriveSeed(globalSeed, "flaky", 1),
		"other rule":       deriveSeed(globalSeed, "twin", 0),
		"other seed":       deriveSeed(globalSeed+1, "flaky", 0),
		"empty rule name":  deriveSeed(globalSeed, "", 0),
		"swapped sequence": deriveSeed(globalSeed, "twin", 1),
	} {
		if other == first {
			t.Errorf("%s produced the same derived seed %d", name, first)
		}
	}
}
