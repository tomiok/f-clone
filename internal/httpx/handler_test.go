package httpx

import "testing"

func TestZoneLabel(t *testing.T) {
	if got := labelForZone("America/Argentina/Cordoba"); got != "BS AS/Argentina" {
		t.Fatalf("cordoba = %q", got)
	}
	if got := labelForZone("America/Argentina/Buenos_Aires"); got != "BS AS/Argentina" {
		t.Fatalf("buenos aires = %q", got)
	}
	if got := labelForZone("Europe/Madrid"); got != "Madrid/Europe" {
		t.Fatalf("madrid = %q", got)
	}
}
