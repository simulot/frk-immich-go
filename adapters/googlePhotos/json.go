package gp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/tzone"
)

type GoogleMetaData struct {
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Category           string             `json:"category"`
	Date               *googTimeObject    `json:"date,omitempty"`
	PhotoTakenTime     *googTimeObject    `json:"photoTakenTime"`
	GeoDataExif        *googGeoData       `json:"geoDataExif"`
	GeoData            *googGeoData       `json:"geoData"`
	Trashed            bool               `json:"trashed,omitempty"`
	Archived           bool               `json:"archived,omitempty"`
	URLPresent         googIsPresent      `json:"url,omitempty"`         // true when the file is an asset metadata
	Favorited          bool               `json:"favorited,omitempty"`   // true when starred in GP
	Enrichments        *googleEnrichments `json:"enrichments,omitempty"` // Album enrichments
	GooglePhotosOrigin struct {
		FromPartnerSharing googIsPresent `json:"fromPartnerSharing,omitempty"` // true when this is a partner's asset
	} `json:"googlePhotosOrigin"`
}

func (gmd *GoogleMetaData) UnmarshalJSON(data []byte) error {
	// test the presence of the key albumData
	type md GoogleMetaData
	type album struct {
		AlbumData *md `json:"albumData"`
	}

	var t album
	err := json.Unmarshal(data, &t)
	if err == nil && t.AlbumData != nil {
		*gmd = GoogleMetaData(*(t.AlbumData))
		return nil
	}

	var gg md
	err = json.Unmarshal(data, &gg)
	if err != nil {
		return err
	}

	*gmd = GoogleMetaData(gg)
	return nil
}

func (gmd GoogleMetaData) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("Title", gmd.Title),
		slog.String("Description", gmd.Description),
		slog.String("Category", gmd.Category),
		slog.Any("Date", gmd.Date),
		slog.Any("PhotoTakenTime", gmd.PhotoTakenTime),
		slog.Any("GeoDataExif", gmd.GeoDataExif),
		slog.Any("GeoData", gmd.GeoData),
		slog.Bool("Trashed", gmd.Trashed),
		slog.Bool("Archived", gmd.Archived),
		slog.Bool("URLPresent", bool(gmd.URLPresent)),
		slog.Bool("Favorited", gmd.Favorited),
		slog.Any("Enrichments", gmd.Enrichments),
		slog.Bool("FromPartnerSharing", bool(gmd.GooglePhotosOrigin.FromPartnerSharing)),
	)
}

func (gmd GoogleMetaData) AsMetadata() *metadata.Metadata {
	latitude, longitude := gmd.GeoDataExif.Latitude, gmd.GeoDataExif.Longitude
	if latitude == 0 && longitude == 0 {
		latitude, longitude = gmd.GeoData.Latitude, gmd.GeoData.Longitude
	}

	t := time.Time{}
	if gmd.PhotoTakenTime != nil && gmd.PhotoTakenTime.Timestamp != "" && gmd.PhotoTakenTime.Timestamp != "0" {
		t = gmd.PhotoTakenTime.Time()
	}

	return &metadata.Metadata{
		FileName:    gmd.Title,
		Description: gmd.Description,
		DateTaken:   t,
		Latitude:    latitude,
		Longitude:   longitude,
		Trashed:     gmd.Trashed,
		Archived:    gmd.Archived,
		Favorited:   gmd.Favorited,
		FromPartner: gmd.isPartner(),
	}
}

func (gmd *GoogleMetaData) isAlbum() bool {
	if gmd == nil || gmd.Date == nil {
		return false
	}
	return gmd.Date.Timestamp != ""
}

func (gmd *GoogleMetaData) isAsset() bool {
	if gmd == nil || gmd.PhotoTakenTime == nil {
		return false
	}
	return gmd.PhotoTakenTime.Timestamp != ""
}

func (gmd *GoogleMetaData) isPartner() bool {
	if gmd == nil {
		return false
	}
	return bool(gmd.GooglePhotosOrigin.FromPartnerSharing)
}

// Key return an expected unique key for the asset
// based on the title and the timestamp
func (gmd GoogleMetaData) Key() string {
	return fmt.Sprintf("%s,%s", gmd.Title, gmd.PhotoTakenTime.Timestamp)
}

// googIsPresent is set when the field is present. The content of the field is not relevant
type googIsPresent bool

func (p *googIsPresent) UnmarshalJSON(b []byte) error {
	var bl bool
	err := json.Unmarshal(b, &bl)
	if err == nil {
		return nil
	}

	*p = len(b) > 0
	return nil
}

func (p googIsPresent) MarshalJSON() ([]byte, error) {
	if p {
		return json.Marshal("present")
	}
	return json.Marshal(struct{}{})
}

// googGeoData contains GPS coordinates
type googGeoData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

func (ggd *googGeoData) LogValue() slog.Value {
	if ggd == nil {
		return slog.Value{}
	}
	return slog.GroupValue(
		slog.Float64("Latitude", ggd.Latitude),
		slog.Float64("Longitude", ggd.Longitude),
		slog.Float64("Altitude", ggd.Altitude),
	)
}

// googTimeObject to handle the epoch timestamp
type googTimeObject struct {
	Timestamp string `json:"timestamp"`
	// Formatted string    `json:"formatted"`
}

func (gt *googTimeObject) LogValue() slog.Value {
	if gt == nil {
		return slog.Value{}
	}
	return slog.TimeValue(gt.Time())
}

// Time return the time.Time of the epoch
func (gt googTimeObject) Time() time.Time {
	ts, _ := strconv.ParseInt(gt.Timestamp, 10, 64)
	if ts == 0 {
		return time.Time{}
	}
	t := time.Unix(ts, 0)
	local, _ := tzone.Local()
	//	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return t.In(local)
}

type googleEnrichments struct {
	Text      string
	Latitude  float64
	Longitude float64
}

func (ge *googleEnrichments) LogValue() slog.Value {
	if ge == nil {
		return slog.Value{}
	}
	return slog.GroupValue(
		slog.String("Text", ge.Text),
		slog.Float64("Latitude", ge.Latitude),
		slog.Float64("Longitude", ge.Longitude),
	)
}

func (ge *googleEnrichments) UnmarshalJSON(b []byte) error {
	type googleEnrichment struct {
		NarrativeEnrichment struct {
			Text string `json:"text"`
		} `json:"narrativeEnrichment,omitempty"`
		LocationEnrichment struct {
			Location []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				LatitudeE7  int    `json:"latitudeE7"`
				LongitudeE7 int    `json:"longitudeE7"`
			} `json:"location"`
		} `json:"locationEnrichment,omitempty"`
	}

	var enrichments []googleEnrichment

	err := json.Unmarshal(b, &enrichments)
	if err != nil {
		return err
	}

	for _, e := range enrichments {
		if e.NarrativeEnrichment.Text != "" {
			ge.Text = addString(ge.Text, "\n", e.NarrativeEnrichment.Text)
		}
		if e.LocationEnrichment.Location != nil {
			for _, l := range e.LocationEnrichment.Location {
				if l.Name != "" {
					ge.Text = addString(ge.Text, "\n", l.Name)
				}
				if l.Description != "" {
					ge.Text = addString(ge.Text, " - ", l.Description)
				}
				ge.Latitude = float64(l.LatitudeE7) / 10e6
				ge.Longitude = float64(l.LongitudeE7) / 10e6
			}
		}
	}
	return err
}

func addString(s string, sep string, t string) string {
	if s != "" {
		return s + sep + t
	}
	return t
}
