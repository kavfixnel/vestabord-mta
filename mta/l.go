package mta

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

const noteRowWidth = 15

const (
	LFeedURL = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l"

	// 1st Ave on the L line (GTFS stop IDs from the realtime feed).
	Stop1stAveNorth = "L06N" // toward 8 Av
	Stop1stAveSouth = "L06S" // toward Brooklyn
)

type Direction string

const (
	DirectionBrooklyn Direction = "Brooklyn"
	Direction8Av      Direction = "8 Av"
)

type Arrival struct {
	Direction Direction
	When      time.Time
}

func (a Arrival) MinutesFrom(now time.Time) int {
	if !a.When.After(now) {
		return 0
	}
	secs := a.When.Sub(now).Seconds()
	return int((secs + 59) / 60)
}

func (a Arrival) Due(now time.Time) bool {
	return !a.When.After(now.Add(30 * time.Second))
}

type NextArrivals struct {
	Brooklyn []Arrival
	EightAv  []Arrival
}

func FetchLArrivals1stAve(now time.Time) (*NextArrivals, error) {
	resp, err := http.Get(LFeedURL)
	if err != nil {
		return nil, fmt.Errorf("fetch L feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch L feed: status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read L feed: %w", err)
	}

	var feed gtfs.FeedMessage
	if err := proto.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("decode L feed: %w", err)
	}

	brooklyn := collectArrivals(&feed, Stop1stAveSouth, now)
	eightAv := collectArrivals(&feed, Stop1stAveNorth, now)

	result := &NextArrivals{
		Brooklyn: takeUpTo(brooklyn, 2),
		EightAv:  takeUpTo(eightAv, 2),
	}

	return result, nil
}

func takeUpTo(arrivals []Arrival, n int) []Arrival {
	if len(arrivals) <= n {
		return arrivals
	}
	return arrivals[:n]
}

func collectArrivals(feed *gtfs.FeedMessage, stopID string, now time.Time) []Arrival {
	var arrivals []Arrival

	for _, entity := range feed.GetEntity() {
		tripUpdate := entity.GetTripUpdate()
		if tripUpdate == nil {
			continue
		}

		for _, stopUpdate := range tripUpdate.GetStopTimeUpdate() {
			if stopUpdate.GetStopId() != stopID {
				continue
			}

			when := stopTime(stopUpdate)
			if when.IsZero() || !when.After(now.Add(-30*time.Second)) {
				continue
			}

			arrivals = append(arrivals, Arrival{
				Direction: directionForStop(stopID),
				When:      when,
			})
		}
	}

	sort.Slice(arrivals, func(i, j int) bool {
		return arrivals[i].When.Before(arrivals[j].When)
	})

	return arrivals
}

func stopTime(update *gtfs.TripUpdate_StopTimeUpdate) time.Time {
	if arrival := update.GetArrival(); arrival != nil && arrival.GetTime() != 0 {
		return time.Unix(int64(arrival.GetTime()), 0)
	}
	if departure := update.GetDeparture(); departure != nil && departure.GetTime() != 0 {
		return time.Unix(int64(departure.GetTime()), 0)
	}
	return time.Time{}
}

func directionForStop(stopID string) Direction {
	switch stopID {
	case Stop1stAveSouth:
		return DirectionBrooklyn
	default:
		return Direction8Av
	}
}

func formatMinutes(minutes int) string {
	if minutes == 1 {
		return "1 MIN"
	}
	return fmt.Sprintf("%d MINS", minutes)
}

func minuteValue(now time.Time, arrival Arrival) int {
	if arrival.Due(now) {
		return 0
	}
	return arrival.MinutesFrom(now)
}

func formatArrivalTimes(now time.Time, arrivals []Arrival) string {
	if len(arrivals) == 0 {
		return "--"
	}
	if len(arrivals) == 1 {
		m := minuteValue(now, arrivals[0])
		if m == 0 {
			return "0 MIN"
		}
		return formatMinutes(m)
	}

	m1 := minuteValue(now, arrivals[0])
	m2 := minuteValue(now, arrivals[1])
	times := fmt.Sprintf("%d,%d", m1, m2)
	suffix := " MINS"
	if m1 <= 1 && m2 <= 1 {
		suffix = " MIN"
	}
	return times + suffix
}

func formatDirectionLine(now time.Time, label string, arrivals []Arrival) string {
	prefix := label + " "
	available := noteRowWidth - len(prefix)

	if len(arrivals) == 0 {
		return prefix + "--"
	}

	if len(arrivals) >= 2 {
		two := formatArrivalTimes(now, arrivals[:2])
		if len(two) <= available {
			return prefix + two
		}
	}

	one := formatArrivalTimes(now, arrivals[:1])
	if len(one) <= available {
		return prefix + one
	}

	// Last resort: truncate label (shouldn't happen with normal arrival times).
	return strings.TrimSpace(prefix) + " " + one
}

func FormatBoardMessage(now time.Time, arrivals *NextArrivals) string {
	if arrivals == nil {
		return "L @ 1 AVE\nNO DATA"
	}

	bk := formatDirectionLine(now, "BKLYN", arrivals.Brooklyn)
	av := formatDirectionLine(now, "8 AV", arrivals.EightAv)

	return fmt.Sprintf("L @ 1 AVE\n%s\n%s", bk, av)
}
