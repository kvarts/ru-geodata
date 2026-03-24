package filterdat

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Categories struct {
	GeoSite []string
	GeoIP   []string

	geoSiteSet map[string]struct{}
	geoIPSet   map[string]struct{}
}

func LoadCategories(path string) (Categories, error) {
	file, err := os.Open(path)
	if err != nil {
		return Categories{}, fmt.Errorf("open category file: %w", err)
	}
	defer file.Close()

	var cats Categories
	cats.geoSiteSet = make(map[string]struct{})
	cats.geoIPSet = make(map[string]struct{})

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		kind, value, ok := strings.Cut(line, ":")
		if !ok {
			return Categories{}, fmt.Errorf("invalid category format at line %d: %q", lineNo, line)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return Categories{}, fmt.Errorf("empty category name at line %d", lineNo)
		}

		switch strings.TrimSpace(kind) {
		case "geosite":
			if _, exists := cats.geoSiteSet[value]; exists {
				continue
			}
			cats.geoSiteSet[value] = struct{}{}
			cats.GeoSite = append(cats.GeoSite, value)
		case "geoip":
			if _, exists := cats.geoIPSet[value]; exists {
				continue
			}
			cats.geoIPSet[value] = struct{}{}
			cats.GeoIP = append(cats.GeoIP, value)
		default:
			return Categories{}, fmt.Errorf("unknown category type %q at line %d", kind, lineNo)
		}
	}

	if err := scanner.Err(); err != nil {
		return Categories{}, fmt.Errorf("read category file: %w", err)
	}

	return cats, nil
}
