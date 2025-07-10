/*
convert UTC RFC339 format to several time zone

how to:
./utc2local -utc="2025-07-09T00:00:00Z"

./utc2local -help
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	utcFlag := flag.String("utc", "", "UTC time in RFC339 format (e.g., 2025-07-09T00:00:00Z)")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help || *utcFlag == "" {
		showHelp()
		return
	}

	// parse UTC
	utcTime, err := time.Parse(time.RFC3339, *utcFlag)
	if err != nil {
		log.Err(err).Msgf("error parsing UTC time '%s'", *utcFlag)
		log.Info().Msg("please use RFC3339 format like: 2025-07-09T00:00:00Z")
		os.Exit(1)
	}

	// convert and display times
	fmt.Printf("UTC time: %s\n", utcTime.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("singapore: %s\n", convertToTimezone(utcTime, "Asia/Singapore").Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("jakarta: %s\n", convertToTimezone(utcTime, "Asia/Jakarta").Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("kuala lumpur: %s\n", convertToTimezone(utcTime, "Asia/Kuala_Lumpur").Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("tokyo: %s\n", convertToTimezone(utcTime, "Asia/Tokyo").Format("2006-01-02 15:04:05 MST"))
}

func showHelp() {
	fmt.Println("utc2local - Convert UTC time to local timezones")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  utc2local -utc=\"2025-07-09T00:00:00Z\"")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -utc string    UTC time in RFC3339 format")
	fmt.Println("  -help          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  utc2local -utc=\"2025-07-09T00:00:00Z\"")
	fmt.Println("  utc2local -utc=\"2025-07-09T08:00:00Z\"")
	fmt.Println()
	fmt.Println("Supported timezones:")
	fmt.Println("  - Singapore (UTC+8)")
	fmt.Println("  - Jakarta (UTC+7)")
	fmt.Println("  - Kuala Lumpur (UTC+8)")
	fmt.Println("  - Tokyo (UTC+9)")
}

func convertToTimezone(t time.Time, timezone string) time.Time {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		log.Err(err).Msgf("error loading timezone: %s", timezone)
		return t
	}

	return t.In(location)
}
