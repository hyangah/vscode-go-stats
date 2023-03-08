// run_vsce_show runs `vsce show` and reports the stats in csv format.
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

var printHeader = flag.Bool("h", false, "print header")

func main() {
	ctx := context.Background()
	res, err := vsce(ctx, "golang", "go")
	if err != nil {
		exit(err)
	}
	if err := asCSVLine(os.Stdout, res, *printHeader); err != nil {
		exit(err)
	}
}

func vsce(ctx context.Context, publisher, extension string) (*VSCEResult, error) {
	cmd := exec.CommandContext(ctx, "npx", "vsce", "show", "--json", fmt.Sprintf("%v.%v", publisher, extension))
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("vsce failed: %v", err)
	}
	result := &VSCEResult{}
	dec := json.NewDecoder(bytes.NewBuffer(output))
	dec.UseNumber()
	if err := dec.Decode(result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return result, nil
}

func asCSVLine(w io.Writer, r *VSCEResult, printHeader bool) error {
	stats := map[string]any{}
	for _, s := range r.Statistics {
		stats[s.StatisticName] = s.Value
	}
	latestVersion := ""
	if len(r.Versions) > 0 {
		latestVersion = r.Versions[0].Version
	}

	type kv struct {
		key   string
		value any
	}
	cols := []kv{
		{"Date", time.Now().UTC()},
		{"LastUpdated", r.LastUpdated},
		{"PublishedDate", r.PublishedDate},
		{"ReleaseDate", r.ReleaseDate},
		{"LatestVersion", latestVersion},
		{"Versions", len(r.Versions)},
	}
	for _, name := range reportedStat {
		cols = append(cols, kv{name, stats[name]})
	}

	csvw := csv.NewWriter(w)
	defer csvw.Flush()

	if printHeader {
		array := make([]string, len(cols))
		for i, col := range cols {
			array[i] = col.key
		}
		if err := csvw.Write(array); err != nil {
			return err
		}
	}
	array := make([]string, len(cols))
	for i, col := range cols {
		array[i] = stringify(col.value)
	}
	return csvw.Write(array)
}

func stringify(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}

func exit(err error) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type VSCEResult struct {
	LastUpdated   time.Time    `json:"lastUpdated"`
	PublishedDate time.Time    `json:"publishedDate"`
	ReleaseDate   time.Time    `json:"releaseDate"`
	Versions      []Version    `json:"versions,omitempty"`
	Statistics    []Statistics `json:"statistics,omitempty"`
}

type Version struct {
	Version     string    `json:"version,omitempty"`
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
}

var reportedStat = []string{
	"install", "averagerating", "ratingcount", "trendingdaily",
	"trendingmonthly", "trendingweekly", "updateCount",
	"weightedRating", "downloadCount",
}

type Statistics struct {
	StatisticName string `json:"statisticName"`
	Value         any    `json:"value,omitempty"` // float64|int64
}
