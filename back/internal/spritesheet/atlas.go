package spritesheet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// DatasetsServer is Hugging Face's dataset viewer API.
	DatasetsServer = "https://datasets-server.huggingface.co"
	// maxBody bounds what one response may weigh: a row's JSON or a sprite's PNG is a few KB.
	maxBody = 1 << 20
	// maxSide bounds a sprite's declared size, checked before decoding it.
	maxSide = 256
)

// HTTPAtlas reads rows of a dataset from the dataset viewer API, one request at a time. A failed request is tried
// again: the API rate-limits and sometimes times out.
type HTTPAtlas struct {
	client *http.Client
	base   *url.URL
	picks  Picks
	tries  int
	wait   time.Duration
	sleep  func(context.Context, time.Duration) error
	log    *slog.Logger
}

// AtlasOptions are how often and how patiently an HTTPAtlas tries a request. Zero values get the defaults:
// 5 tries, 5 s apart.
type AtlasOptions struct {
	Tries int
	Wait  time.Duration
	Sleep func(context.Context, time.Duration) error
}

func NewHTTPAtlas(client *http.Client, baseURL string, picks Picks, options AtlasOptions, log *slog.Logger) (*HTTPAtlas, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("NewHTTPAtlas: %w", err)
	}
	a := &HTTPAtlas{client: client, base: base, picks: picks, tries: options.Tries, wait: options.Wait, sleep: options.Sleep, log: log}
	if a.tries <= 0 {
		a.tries = 5
	}
	if a.wait <= 0 {
		a.wait = 5 * time.Second
	}
	if a.sleep == nil {
		a.sleep = sleep
	}
	return a, nil
}

type rowsIn struct {
	Rows []struct {
		Row struct {
			RowID int `json:"row_id"`
			Image struct {
				Src string `json:"src"`
			} `json:"image"`
		} `json:"row"`
	} `json:"rows"`
}

// Row asks the API for the row with that row_id, then downloads its sprite. It goes by row_id, the dataset's own
// id: a row's position in the split is not stable across the API's calls.
func (a *HTTPAtlas) Row(ctx context.Context, rowID int) (int, image.Image, error) {
	body, err := a.get(ctx, a.rowURL(rowID))
	if err != nil {
		return 0, nil, fmt.Errorf("Row %d: %w", rowID, err)
	}
	var rows rowsIn
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, nil, fmt.Errorf("Row %d: %w", rowID, err)
	}
	if len(rows.Rows) != 1 {
		return 0, nil, fmt.Errorf("Row %d: %d rows given, not 1", rowID, len(rows.Rows))
	}
	row := rows.Rows[0].Row
	img, err := a.sprite(ctx, row.Image.Src)
	if err != nil {
		return 0, nil, fmt.Errorf("Row %d: %w", rowID, err)
	}
	return row.RowID, img, nil
}

func (a *HTTPAtlas) rowURL(rowID int) string {
	query := url.Values{
		"dataset": {a.picks.Dataset}, "config": {a.picks.Config}, "split": {a.picks.Split},
		"where": {`"row_id"=` + strconv.Itoa(rowID)}, "offset": {"0"}, "length": {"1"},
	}
	return a.base.JoinPath("filter").String() + "?" + query.Encode()
}

// sprite downloads a PNG from the API's own host only, and checks its size before decoding it.
func (a *HTTPAtlas) sprite(ctx context.Context, src string) (image.Image, error) {
	u, err := url.Parse(src)
	if err != nil || u.Scheme != a.base.Scheme || u.Host != a.base.Host {
		return nil, fmt.Errorf("the sprite's URL is not on %s", a.base.Host)
	}
	body, err := a.get(ctx, src)
	if err != nil {
		return nil, err
	}
	config, err := png.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("not a PNG: %w", err)
	}
	if config.Width > maxSide || config.Height > maxSide {
		return nil, fmt.Errorf("the sprite is %dx%d, over %d px", config.Width, config.Height, maxSide)
	}
	return png.Decode(bytes.NewReader(body))
}

// get tries a GET up to a.tries times, a.wait apart, and gives the body of the first that answers 200.
func (a *HTTPAtlas) get(ctx context.Context, address string) ([]byte, error) {
	var err error
	for try := 1; try <= a.tries; try++ {
		if try > 1 {
			if sleepErr := a.sleep(ctx, a.wait); sleepErr != nil {
				return nil, sleepErr
			}
		}
		var body []byte
		if body, err = a.getOnce(ctx, address); err == nil {
			return body, nil
		}
		a.log.Warn("request failed", "try", try, "of", a.tries, "error", err)
	}
	return nil, fmt.Errorf("after %d tries: %w", a.tries, err)
}

func (a *HTTPAtlas) getOnce(ctx context.Context, address string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("response over 1 MB")
	}
	return body, nil
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
