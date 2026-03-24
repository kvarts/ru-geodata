package filterdat

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

type FilterStats struct {
	Type        string
	Total       int
	Kept        int
	Missing     []string
	MatchedKeys []string
}

func FilterGeoSite(data []byte, wanted []string) ([]byte, FilterStats, error) {
	return filterRootMessage(data, wanted, "geosite")
}

func FilterGeoIP(data []byte, wanted []string) ([]byte, FilterStats, error) {
	return filterRootMessage(data, wanted, "geoip")
}

func filterRootMessage(data []byte, wanted []string, kind string) ([]byte, FilterStats, error) {
	wantedSet := make(map[string]struct{}, len(wanted))
	for _, name := range wanted {
		wantedSet[name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(wanted))

	var out []byte
	stats := FilterStats{Type: kind}

	for len(data) > 0 {
		tagNum, tagType, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return nil, FilterStats{}, fmt.Errorf("%s root: read tag: %v", kind, protowire.ParseError(tagLen))
		}

		valueLen := protowire.ConsumeFieldValue(tagNum, tagType, data[tagLen:])
		if valueLen < 0 {
			return nil, FilterStats{}, fmt.Errorf("%s root: read field %d: %v", kind, tagNum, protowire.ParseError(valueLen))
		}

		fieldLen := tagLen + valueLen
		fieldBytes := data[:fieldLen]

		if tagNum == 1 && tagType == protowire.BytesType {
			entryBytes, n := protowire.ConsumeBytes(data[tagLen:])
			if n < 0 {
				return nil, FilterStats{}, fmt.Errorf("%s root: read entry bytes: %v", kind, protowire.ParseError(n))
			}
			stats.Total++

			key, err := extractEntryKey(entryBytes, kind)
			if err != nil {
				return nil, FilterStats{}, err
			}
			if _, keep := wantedSet[key]; keep {
				out = append(out, fieldBytes...)
				stats.Kept++
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					stats.MatchedKeys = append(stats.MatchedKeys, key)
				}
			}
		} else {
			out = append(out, fieldBytes...)
		}

		data = data[fieldLen:]
	}

	for _, name := range wanted {
		if _, ok := seen[name]; !ok {
			stats.Missing = append(stats.Missing, name)
		}
	}

	return out, stats, nil
}

func extractEntryKey(data []byte, kind string) (string, error) {
	var key string

	for len(data) > 0 {
		tagNum, tagType, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return "", fmt.Errorf("%s entry: read tag: %v", kind, protowire.ParseError(tagLen))
		}

		valueLen := protowire.ConsumeFieldValue(tagNum, tagType, data[tagLen:])
		if valueLen < 0 {
			return "", fmt.Errorf("%s entry: read field %d: %v", kind, tagNum, protowire.ParseError(valueLen))
		}

		if tagNum == 1 && tagType == protowire.BytesType {
			value, n := protowire.ConsumeString(data[tagLen:])
			if n < 0 {
				return "", fmt.Errorf("%s entry: read key field: %v", kind, protowire.ParseError(n))
			}
			key = value
		}

		data = data[tagLen+valueLen:]
	}

	if key == "" {
		return "", fmt.Errorf("%s entry: missing key field 1", kind)
	}

	return key, nil
}
