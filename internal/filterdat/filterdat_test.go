package filterdat

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestLoadCategories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "category-for-save.txt")
	content := strings.Join([]string{
		"geosite:RU",
		"",
		"geoip:US",
		"geosite:RU",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cats, err := LoadCategories(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := strings.Join(cats.GeoSite, ","), "RU"; got != want {
		t.Fatalf("GeoSite mismatch: got %q want %q", got, want)
	}
	if got, want := strings.Join(cats.GeoIP, ","), "US"; got != want {
		t.Fatalf("GeoIP mismatch: got %q want %q", got, want)
	}
}

func TestLoadCategoriesRejectsUnknownPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "category-for-save.txt")
	if err := os.WriteFile(path, []byte("unknown:value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadCategories(path); err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}

func TestFilterGeoSite(t *testing.T) {
	data := encodeRootEntries(
		encodeGeoSiteEntry("KEEP"),
		encodeGeoSiteEntry("DROP"),
	)

	filtered, stats, err := FilterGeoSite(data, []string{"KEEP", "MISSING"})
	if err != nil {
		t.Fatal(err)
	}

	keys, err := collectKeys(filtered)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(keys, ","), "KEEP"; got != want {
		t.Fatalf("filtered keys mismatch: got %q want %q", got, want)
	}
	if got, want := stats.Total, 2; got != want {
		t.Fatalf("total mismatch: got %d want %d", got, want)
	}
	if got, want := stats.Kept, 1; got != want {
		t.Fatalf("kept mismatch: got %d want %d", got, want)
	}
	if got, want := strings.Join(stats.Missing, ","), "MISSING"; got != want {
		t.Fatalf("missing mismatch: got %q want %q", got, want)
	}
}

func TestLoadSourceHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	data, err := LoadSource(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "payload"; got != want {
		t.Fatalf("payload mismatch: got %q want %q", got, want)
	}
}

func TestRunWithRepositoryFixtures(t *testing.T) {
	dir := t.TempDir()
	fixturesDir := filepath.Join(dir, "fixtures")
	geositeOut := filepath.Join(dir, "geosite.filtered.dat")
	geoipOut := filepath.Join(dir, "geoip.filtered.dat")

	if err := os.MkdirAll(fixturesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	categoryFile := filepath.Join(fixturesDir, "category-for-save.txt")
	geositeInput := filepath.Join(fixturesDir, "geosite.dat")
	geoipInput := filepath.Join(fixturesDir, "geoip.dat")

	if err := os.WriteFile(categoryFile, []byte("geosite:KEEP\ngeosite:EXTRA\ngeoip:RU\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(geositeInput, encodeRootEntries(
		encodeGeoSiteEntry("KEEP"),
		encodeGeoSiteEntry("EXTRA"),
		encodeGeoSiteEntry("DROP"),
	), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(geoipInput, encodeRootEntries(
		encodeGeoIPEntry("RU"),
		encodeGeoIPEntry("US"),
	), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		CategoryFile:  categoryFile,
		GeoSiteInput:  geositeInput,
		GeoSiteOutput: geositeOut,
		GeoIPInput:    geoipInput,
		GeoIPOutput:   geoipOut,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run(context.Background(), cfg, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}

	geositeData, err := os.ReadFile(geositeOut)
	if err != nil {
		t.Fatal(err)
	}
	geoipData, err := os.ReadFile(geoipOut)
	if err != nil {
		t.Fatal(err)
	}

	geositeKeys, err := collectKeys(geositeData)
	if err != nil {
		t.Fatal(err)
	}
	geoipKeys, err := collectKeys(geoipData)
	if err != nil {
		t.Fatal(err)
	}

	if len(geositeKeys) != 2 {
		t.Fatalf("expected 2 geosite entries, got %d", len(geositeKeys))
	}
	if got, want := strings.Join(geositeKeys, ","), "KEEP,EXTRA"; got != want {
		t.Fatalf("unexpected geosite keys: got %q want %q", got, want)
	}
	if len(geoipKeys) != 1 || geoipKeys[0] != "RU" {
		t.Fatalf("unexpected geoip keys: %v", geoipKeys)
	}
	if strings.Contains(stderr.String(), "warning:") {
		t.Fatalf("unexpected warnings: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "geosite: kept 2") {
		t.Fatalf("missing geosite summary: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "geoip: kept 1") {
		t.Fatalf("missing geoip summary: %s", stdout.String())
	}
}

func encodeRootEntries(entries ...[]byte) []byte {
	var out []byte
	for _, entry := range entries {
		out = protowire.AppendTag(out, 1, protowire.BytesType)
		out = protowire.AppendBytes(out, entry)
	}
	return out
}

func encodeGeoSiteEntry(code string) []byte {
	var entry []byte
	entry = protowire.AppendTag(entry, 1, protowire.BytesType)
	entry = protowire.AppendString(entry, code)
	return entry
}

func encodeGeoIPEntry(code string) []byte {
	var entry []byte
	entry = protowire.AppendTag(entry, 1, protowire.BytesType)
	entry = protowire.AppendString(entry, code)
	return entry
}

func collectKeys(data []byte) ([]string, error) {
	var keys []string
	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return nil, protowire.ParseError(tagLen)
		}
		if num != 1 || typ != protowire.BytesType {
			return nil, nil
		}

		entry, valueLen := protowire.ConsumeBytes(data[tagLen:])
		if valueLen < 0 {
			return nil, protowire.ParseError(valueLen)
		}

		key, err := extractEntryKey(entry, "test")
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		data = data[tagLen+valueLen:]
	}
	return keys, nil
}
